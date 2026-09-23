package cardmarket

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// sameProduct says whether two products of a game's shelves are the same
// card sold twice, for the games whose shelves do that; nil for the rest.
func sameProduct(gameID cm.Game) func(a, b *cm.Product) bool {
	switch gameID {
	case cm.GamePokemon:
		return pokemonSameProduct
	case cm.GameYuGiOh:
		return yugiohSameProduct
	case cm.GameOnePiece:
		return onePieceSameProduct
	case cm.GameFleshAndBlood:
		return fabSameProduct
	}
	return nil
}

// faceOf answers the rule telling a product that names one face of a fused
// printing, for the games whose shelves sell a card face by face.
func faceOf(b *mtgmatcher.Backend, gameID cm.Game) func(product *cm.Product, cardID string) bool {
	if gameID == cm.GameFleshAndBlood {
		return func(product *cm.Product, cardID string) bool {
			return fabFaceOf(b, product, cardID)
		}
	}
	return nil
}

// onePieceSameProduct reports whether two One Piece products are the same
// card sold twice: the same name, code tag included, once the version index
// is off it. The shelves sell each event copy of a promo as a version, and
// the name reaches the same family for every one of them.
func onePieceSameProduct(a, b *cm.Product) bool {
	return mtgmatcher.Normalize(versionTail.ReplaceAllString(a.Name, "")) == mtgmatcher.Normalize(versionTail.ReplaceAllString(b.Name, ""))
}

// resolved is what one product of the walk answered with, held until its
// expansion is read whole, so a product can be judged beside its siblings.
type resolved struct {
	product    *cm.Product
	cardID     string
	cardIDFoil string
	byName     bool
	err        error
}

// walkCatalog prices every product of the id map, and of the product list
// beside it, expansion by expansion.
func (mkm *Index) walkCatalog(ctx context.Context) error {
	// The map knows only what MTGJSON has linked; the published product list
	// knows everything on sale today. Products it names that the map does
	// not - several thousand for Magic - are priced from what the catalog
	// says of them, with the one thing the list never carries left empty:
	// their collector number.
	products := make(map[int]cm.CatalogProduct, len(mkm.catalog.Data.Products))
	for id, product := range mkm.catalog.Data.Products {
		products[id] = product
	}
	list, err := cm.DownloadProductListSingles(ctx, mkm.gameID)
	if err != nil {
		return err
	}
	var unmapped int
	for _, entry := range list {
		_, found := products[entry.IDProduct]
		if found {
			continue
		}
		products[entry.IDProduct] = cm.CatalogProduct{ExpansionID: entry.ExpansionID, Name: entry.Name}
		unmapped++
	}
	mkm.printf("%d products of the list are not in the map and resolve by name", unmapped)

	byExpansion := map[int][]int{}
	for id, product := range products {
		byExpansion[product.ExpansionID] = append(byExpansion[product.ExpansionID], id)
	}

	var items []cm.Expansion
	for expansionID := range byExpansion {
		entry := mkm.catalog.Data.Expansions[expansionID]
		name := entry.Name
		if name == "" {
			name = fmt.Sprintf("expansion %d", expansionID)
		}
		if mkm.targetEdition != "" && name != mkm.targetEdition {
			continue
		}
		items = append(items, cm.Expansion{IDExpansion: expansionID, Name: name, SetCode: entry.Code})
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
	case cm.GameOnePiece, cm.GameYuGiOh:
		kept := items[:0]
		for _, exp := range items {
			if strings.HasSuffix(exp.SetCode, "-JP") || foreignShelf(exp.Name) {
				continue
			}
			kept = append(kept, exp)
		}
		items = kept
		if mkm.gameID == cm.GameOnePiece {
			mkm.shelved = shelvedSets(mkm.backend, items)
		}
	}

	mkm.printf("Parsing %d expansion ids from the id map", len(items))

	walked, refused, foreign := mkm.collectPrices(ctx, items,
		func(ctx context.Context, exp cm.Expansion, channel chan<- responseChan) error {
			mkm.printf("Processing %s (%d)", exp.Name, exp.IDExpansion)
			ids := byExpansion[exp.IDExpansion]
			sort.Ints(ids)

			results := make([]resolved, 0, len(ids))
			for _, id := range ids {
				results = append(results, mkm.resolveMapped(id, products[id], exp))
			}
			if mkm.gameID == cm.GameFleshAndBlood {
				mkm.disownBridged(results)
			}
			if same := sameProduct(mkm.gameID); same != nil {
				twinsAmong(results, same, faceOf(mkm.backend, mkm.gameID))
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
					key := fmt.Sprintf("%q (%s) in %s", mkm.refusalName(mapped.Name), mapped.Number, exp.Name)
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
			channel <- responseChan{tally: true, walked: len(ids), refused: refusals + twins + foreign, foreign: foreign}
			return nil
		})

	mkm.printf("Walked %d products, %d of which named no printing of ours", walked, refused)
	if foreign > 0 {
		mkm.printf("%d of those were products of a catalog we do not carry", foreign)
	}
	mkm.inventoryDate = time.Now()
	return nil
}
