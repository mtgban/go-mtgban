package cardtrader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const ctExpansionsExportURL = "https://api.cardtrader.com/api/v2/expansions/export"

// ctExportWorkers is how many expansions' blueprints are fetched at once.
const ctExportWorkers = 4

// expansionBlueprints is one expansion's fetch, or the error it gave.
type expansionBlueprints struct {
	id         int
	blueprints []Blueprint
	err        error
}

// Inventory is a seller's own stock as Card Trader exports it: every listing
// in every game, with the blueprints and expansions the listings refer to.
// Nothing is matched, filtered or converted here.
type Inventory struct {
	Products   []Product
	Blueprints map[int]*Blueprint // keyed by blueprint id, every category
	Expansions map[int]Expansion  // the seller's expansions, keyed by id
	// Unavailable names the expansions whose blueprints could not be
	// fetched, with the error each gave; their listings are still in
	// Products, without a blueprint to read them by.
	Unavailable map[int]error
}

// ExpansionsExport returns the expansions the seller has listings in.
func (ct *CTAuthClient) ExpansionsExport(ctx context.Context) ([]Expansion, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ctExpansionsExportURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := ct.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// A refused token answers with a status, which the caller needs to
	// tell from the catalog being down.
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expansions export: %s", resp.Status)
	}

	var out []Expansion
	err = json.NewDecoder(resp.Body).Decode(&out)
	if err != nil {
		return nil, fmt.Errorf("unmarshal error for expansions export, got: %w", err)
	}

	return out, nil
}

// ExportInventory returns the seller's listings together with the catalog
// entries they are sold against. The blueprints come per expansion, a few
// at once, so an expansion that fails is reported rather than failing the lot.
func (ct *CTAuthClient) ExportInventory(ctx context.Context) (*Inventory, error) {
	expansions, err := ct.ExpansionsExport(ctx)
	if err != nil {
		return nil, err
	}

	inv := &Inventory{
		Blueprints:  map[int]*Blueprint{},
		Expansions:  make(map[int]Expansion, len(expansions)),
		Unavailable: map[int]error{},
	}
	for _, exp := range expansions {
		inv.Expansions[exp.ID] = exp
	}
	mtgban.WorkerPool(ctx, ctExportWorkers, expansions,
		func(ctx context.Context, exp Expansion, results chan<- expansionBlueprints) error {
			blueprints, err := ct.Blueprints(ctx, exp.ID)
			results <- expansionBlueprints{id: exp.ID, blueprints: blueprints, err: err}
			return nil
		},
		func(res expansionBlueprints) {
			if res.err != nil {
				inv.Unavailable[res.id] = res.err
				return
			}
			for i := range res.blueprints {
				inv.Blueprints[res.blueprints[i].ID] = &res.blueprints[i]
			}
		},
		nil,
	)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	inv.Products, err = ct.ProductsExport(ctx)
	if err != nil {
		return nil, err
	}
	return inv, nil
}

// GameID returns Card Trader's catalog number for a game, and whether the
// catalog carries it.
func GameID(game mtgmatcher.Game) (int, bool) {
	id, found := ctGames[game]
	return id, found
}
