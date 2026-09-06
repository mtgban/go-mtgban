package cardmarket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// IDMapProduct is one product of the published catalog: what the marketplace
// calls it, where it files it, and the printings it stands for.
type IDMapProduct struct {
	ExpansionID int      `json:"expansionId"`
	Name        string   `json:"name"`
	Number      string   `json:"number,omitempty"`
	UUIDs       []string `json:"uuids,omitempty"`
}

// IDMapExpansion names one expansion of the catalog and the set codes it
// covers. Code is the marketplace's own abbreviation, which is where the
// foreign catalogs say what they are; MTGJSON's file does not carry it.
type IDMapExpansion struct {
	Name     string   `json:"name"`
	Code     string   `json:"code,omitempty"`
	SetCodes []string `json:"setCodes,omitempty"`
}

// IDMap is the published Cardmarket catalog: every singles product mapped to
// the printings it sells, which is what the index prices from instead of
// walking the API. MTGJSON publishes the Magic one; mkmcatalog builds the
// other games' by walking the API once a day.
type IDMap struct {
	Products   map[int]IDMapProduct
	Expansions map[int]IDMapExpansion
}

// LoadIDMap reads a published catalog. The file keys both tables by the
// stringified ids JSON forces on it; the map keys them by the numbers they
// are.
func LoadIDMap(reader io.Reader) (*IDMap, error) {
	var payload struct {
		Data struct {
			Expansions map[string]IDMapExpansion `json:"expansions"`
			Products   map[string]IDMapProduct   `json:"products"`
		} `json:"data"`
	}
	err := json.NewDecoder(reader).Decode(&payload)
	if err != nil {
		return nil, err
	}
	if len(payload.Data.Products) == 0 {
		return nil, errors.New("empty id map")
	}

	out := IDMap{
		Products:   make(map[int]IDMapProduct, len(payload.Data.Products)),
		Expansions: make(map[int]IDMapExpansion, len(payload.Data.Expansions)),
	}
	for key, product := range payload.Data.Products {
		id, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("product id %q: %w", key, err)
		}
		out.Products[id] = product
	}
	for key, expansion := range payload.Data.Expansions {
		id, err := strconv.Atoi(key)
		if err != nil {
			return nil, fmt.Errorf("expansion id %q: %w", key, err)
		}
		out.Expansions[id] = expansion
	}
	return &out, nil
}

// resolveUUIDs answers a product from the printings its map entry lists,
// splitting them by finish the way the guide's columns are split. Ids the
// datastore does not carry are passed over - a double-faced card lists its
// back face too, and the index knows only fronts. Within a finish the
// printing whose number agrees with the product's wins, the way Fallback
// already prefers it; a pick between printings the number cannot settle is
// said out loud. Both ids empty means the entry decided nothing.
func (mkm *Index) resolveUUIDs(product *MKMProduct, uuids []string) (string, string) {
	var plain, foil []string
	var plainMatched, foilMatched bool
	for _, uuid := range uuids {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil {
			continue
		}
		sameNumber := strings.EqualFold(co.OriginalNumber, product.Number)
		if co.Foil || co.Etched {
			if sameNumber && !foilMatched {
				foil = append([]string{uuid}, foil...)
				foilMatched = true
			} else {
				foil = append(foil, uuid)
			}
		} else {
			if sameNumber && !plainMatched {
				plain = append([]string{uuid}, plain...)
				plainMatched = true
			} else {
				plain = append(plain, uuid)
			}
		}
	}

	if len(plain) > 1 && !plainMatched {
		mkm.printf("id %d %q lists %d plain printings and the number settles none; keeping %s",
			product.IDProduct, product.Name, len(plain), plain[0])
	}
	if len(foil) > 1 && !foilMatched {
		mkm.printf("id %d %q lists %d foil printings and the number settles none; keeping %s",
			product.IDProduct, product.Name, len(foil), foil[0])
	}

	var cardID, cardIDFoil string
	switch {
	case len(plain) > 0:
		cardID = plain[0]
		if len(foil) > 0 {
			cardIDFoil = foil[0]
		} else {
			// The entry lists no foil printing, but the datastore may
			// still carry one, the way processProduct probes for it.
			cardIDFoil, _ = mtgmatcher.MatchID(cardID, true)
		}
	case len(foil) > 0:
		// A foil-only product prices through its own columns; both ids
		// point to it, the way Fallback answers a single printing.
		cardID = foil[0]
		cardIDFoil = foil[0]
	}
	return cardID, cardIDFoil
}

// resolved is what one product of the walk answered with, held until its
// expansion is read whole, so a product can be judged beside its siblings.
type resolved struct {
	product    *MKMProduct
	cardID     string
	cardIDFoil string
	byName     bool
	err        error
}

// resolveMapped answers one product of the id map. The map answers first;
// what it left unmapped is answered from what the catalog says of it, the
// way processProduct does, so a product the file does not know yet is
// matched rather than lost.
func (mkm *Index) resolveMapped(id int, mapped IDMapProduct, expansionName string) resolved {
	product := &MKMProduct{
		IDProduct:     id,
		Name:          mapped.Name,
		Number:        mapped.Number,
		ExpansionName: expansionName,
	}

	cardID, cardIDFoil := mkm.resolveUUIDs(product, mapped.UUIDs)
	if cardID != "" {
		return resolved{product: product, cardID: cardID, cardIDFoil: cardIDFoil}
	}
	cardID, cardIDFoil, byName, err := mkm.resolveProduct(product)
	return resolved{product: product, cardID: cardID, cardIDFoil: cardIDFoil, byName: byName, err: err}
}

// checkIDMap reports whether the id map can be walked. For the games that
// shelve whole foreign catalogs, the map says which shelves those are only
// through the expansion codes: a map written before it carried them cannot
// be walked safely, and the run refuses rather than price the foreign
// printings onto the English ones.
func (mkm *Index) checkIDMap() error {
	if mkm.IDMap == nil {
		return errors.New("no id map to price from")
	}
	switch mkm.gameID {
	case GameOnePiece, GameYuGiOh:
	default:
		return nil
	}
	for _, expansion := range mkm.IDMap.Expansions {
		if expansion.Code != "" {
			return nil
		}
	}
	return errors.New("the id map carries no expansion codes to tell the foreign shelves by")
}

// walkIDMap prices every product of the id map, and of the product list
// beside it, expansion by expansion.
func (mkm *Index) walkIDMap(ctx context.Context) error {
	// The map knows only what MTGJSON has linked; the published product list
	// knows everything on sale today. Products it names that the map does
	// not - several thousand for Magic - are priced from what the catalog
	// says of them, with the one thing the list never carries left empty:
	// their collector number.
	products := make(map[int]IDMapProduct, len(mkm.IDMap.Products))
	for id, product := range mkm.IDMap.Products {
		products[id] = product
	}
	list, err := GetProductListSingles(ctx, mkm.gameID)
	if err != nil {
		return err
	}
	var unmapped int
	for _, entry := range list {
		_, found := products[entry.IDProduct]
		if found {
			continue
		}
		products[entry.IDProduct] = IDMapProduct{ExpansionID: entry.ExpansionID, Name: entry.Name}
		unmapped++
	}
	mkm.printf("%d products of the list are not in the map and resolve by name", unmapped)

	byExpansion := map[int][]int{}
	for id, product := range products {
		byExpansion[product.ExpansionID] = append(byExpansion[product.ExpansionID], id)
	}

	var items []MKMExpansion
	for expansionID := range byExpansion {
		entry := mkm.IDMap.Expansions[expansionID]
		name := entry.Name
		if name == "" {
			name = fmt.Sprintf("expansion %d", expansionID)
		}
		if mkm.TargetEdition != "" && name != mkm.TargetEdition {
			continue
		}
		items = append(items, MKMExpansion{IDExpansion: expansionID, Name: name, SetCode: entry.Code})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].IDExpansion < items[j].IDExpansion })

	// The non-English programs are whole separate catalogs (OP01-JP beside
	// OP01, "Metal Raiders (Korean)" beside Metal Raiders) whose prices must
	// not land on the English printings the datastore carries. Yu-Gi-Oh
	// shelves them the same way, and one more besides: the PMT tail marks
	// the European multi-language print of a set, which is a catalog of its
	// own for the same reason. One Piece's duplicate shelves resolve through
	// the shelved table; see offShelf.
	switch mkm.gameID {
	case GameOnePiece, GameYuGiOh:
		kept := items[:0]
		for _, exp := range items {
			if strings.HasSuffix(exp.SetCode, "-JP") || foreignShelf(exp.Name) {
				continue
			}
			kept = append(kept, exp)
		}
		items = kept
		if mkm.gameID == GameOnePiece {
			mkm.shelved = shelvedSets(items)
		}
	}

	mkm.printf("Parsing %d expansion ids from the id map", len(items))

	walked, refused := mkm.collectPrices(ctx, items,
		func(ctx context.Context, exp MKMExpansion, channel chan<- responseChan) error {
			mkm.printf("Processing %s (%d)", exp.Name, exp.IDExpansion)
			ids := byExpansion[exp.IDExpansion]
			sort.Ints(ids)

			results := make([]resolved, 0, len(ids))
			for _, id := range ids {
				results = append(results, mkm.resolveMapped(id, products[id], exp.Name))
			}
			if mkm.gameID == GamePokemon {
				pokemonTwins(results)
			}

			// A refusal is named once per name and number: the same
			// card sold as several products refuses as one card.
			var refused []string
			named := map[string]int{}
			var twins, foreign, refusals int
			for i := range results {
				r := &results[i]
				id, mapped := r.product.IDProduct, products[r.product.IDProduct]
				err := r.err
				if err == nil && r.cardID != "" {
					err = mkm.emitPrices(channel, r.product, r.cardID, r.cardIDFoil, r.byName)
				}
				switch {
				case errors.Is(err, errTwin):
					twins++
				case errors.Is(err, errForeign):
					foreign++
				case errors.Is(err, errNoPrinting):
					refusals++
					key := fmt.Sprintf("%q (%s) in %s", pokemonName(mapped.Name), mapped.Number, exp.Name)
					if at, seen := named[key]; seen {
						refused[at] += "+"
						continue
					}
					named[key] = len(refused)
					refused = append(refused, fmt.Sprintf("%d %s", id, key))
				case err != nil:
					mkm.printf("product id %d returned %s", id, err)
				}
			}

			mkm.reportRefused(exp.Name, len(ids), refused, twins, foreign)
			channel <- responseChan{tally: true, walked: len(ids), refused: refusals + twins + foreign}
			return nil
		})

	mkm.printf("Walked %d products, %d of which named no printing of ours", walked, refused)
	mkm.inventoryDate = time.Now()
	return nil
}
