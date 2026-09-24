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
