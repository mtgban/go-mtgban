package cardmarket

import (
	"fmt"
	"strings"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// onePieceShelves spells the shelves Cardmarket names apart from the set of
// ours they sell.
var onePieceShelves = map[string]string{
	"Reprints":   "Revision Pack Cards",
	"Demo Decks": "One Piece Demo Deck Cards",
}

// onePieceShelfLabels names the event the products of a promo shelf were
// handed out at, which the datastore labels the printing with.
var onePieceShelfLabels = map[string]string{
	"Store Tournament Promos": "Tournament Pack",
	"Winner Cards":            "Winner Pack",
}

// onePieceDons spells, by product id, the DON!! cards whose product name
// reads apart from the datastore's label.
var onePieceDons = map[int]struct{ edition, label string }{
	873734: {"", "Nico Robin"},
	873735: {"", "Nico Robin Gold"},
	873738: {"", "Boa Hancock"},
	873739: {"", "Boa Hancock Gold"},
	877822: {"One Piece Promotion Cards", "Japanese Version 3rd Anniversary Set"},
	882449: {"One Piece Promotion Cards", "Japanese Version 3rd Anniversary Set"},
	906864: {"One Piece Promotion Cards", "English Version 3rd Anniversary Set Ace, Luffy, Sabo"},
}

// onePieceByID answers a One Piece product from the bridge's TCGplayer id,
// naming its printing outright. An empty id under a nil error leaves the
// product to its wording.
func (r *resolver) onePieceByID(product *cm.Product) (string, error) {
	tcgID, found := r.tcgBridge[product.IDProduct]
	if !found {
		return "", nil
	}
	id, err := r.backend.MatchID(fmt.Sprint(tcgID), false)
	if err != nil || offCode(r.backend, product, id) {
		return "", nil
	}
	if r.offShelf(product, id) || r.notPreErrata(product, id) {
		return "", errNoPrinting
	}
	return id, nil
}

// offCode reports whether the product's name carries a collector number
// other than the printing's. The bridge's links are another marketplace's,
// and one pointing at the neighbouring product names the wrong card.
func offCode(b *mtgmatcher.Backend, product *cm.Product, cardID string) bool {
	fields := nameCode.FindStringSubmatch(versionTail.ReplaceAllString(product.Name, ""))
	if fields == nil {
		return false
	}
	co, err := b.GetUUID(cardID)
	return err == nil && !strings.EqualFold(co.Number, fields[1])
}

// claimByID fills claimed with the printings the walk's products are
// answered with by id, before any product is named by its wording.
func (r *resolver) claimByID(byExpansion map[int][]int, products map[int]cm.CatalogProduct, items []cm.Expansion) {
	r.claimed = map[string]bool{}
	for _, exp := range items {
		for _, id := range byExpansion[exp.IDExpansion] {
			product := &cm.Product{IDProduct: id, Name: products[id].Name, ExpansionName: exp.Name}
			cardID, err := r.onePieceByID(product)
			if err != nil || cardID == "" {
				continue
			}
			r.claimed[cardID] = true
			foilID, err := r.backend.MatchID(cardID, true)
			if err == nil {
				r.claimed[foilID] = true
			}
		}
	}
}

// giveWay refuses a One Piece product whose wording landed on a printing
// another product is answered with by id. Cardmarket sells the treasure
// rares, winner copies and event stamps the datastore does not carry beside
// the printing they reprint, and the wording reaches that printing for all
// of them, publishing a second price for it.
func (r *resolver) giveWay(results []resolved) {
	for i, res := range results {
		if res.err != nil || !r.claimed[res.cardID] {
			continue
		}
		id, err := r.onePieceByID(res.product)
		if err == nil && id != "" {
			continue
		}
		results[i] = resolved{product: res.product, err: errTwin}
	}
}
