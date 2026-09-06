package mtgmatcher_test

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// A Secret Lair sold etched is its own product, and the bonus card it lists
// carries the plain foil flag - etched is the only foil that card comes in,
// and mtgjson has nowhere else to say so. Resolved on the flag alone the card
// lands on a printing the drop does not contain, and where no foil exists it
// lands back on the nonfoil one.
func TestEtchedProductHoldsEtchedCards(t *testing.T) {
	for _, code := range mtgmatcher.GetAllSets() {
		set, err := mtgmatcher.GetSet(code)
		if err != nil {
			continue
		}
		for _, product := range set.SealedProduct {
			words := strings.Fields(product.Name)
			var namesEtched bool
			for _, word := range words {
				if strings.EqualFold(word, "Etched") {
					namesEtched = true
					break
				}
			}
			if !namesEtched || !mtgmatcher.SealedHasDecklist(code, product.UUID) {
				continue
			}

			picks, err := mtgmatcher.GetDecklist(code, product.UUID)
			if err != nil {
				t.Errorf("%s %q: %v", code, product.Name, err)
				continue
			}
			for _, id := range picks {
				co, err := mtgmatcher.GetUUID(id)
				if err != nil {
					t.Errorf("%s %q: pick %s does not resolve", code, product.Name, id)
					continue
				}
				// Only where the card is sold etched at all: a printing
				// without an etched sibling keeps whatever it has.
				var sold bool
				for _, sibling := range mtgmatcher.FinishSiblings(id) {
					sco, err := mtgmatcher.GetUUID(sibling)
					if err == nil && sco.Etched {
						sold = true
						break
					}
				}
				if sold && !co.Etched {
					t.Errorf("%s %q holds %s #%s as %s, but it is sold etched",
						code, product.Name, co.Name, co.Number, co.Finish)
				}
			}
		}
	}
}
