package mtgmatcher_test

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// saysEtched reports whether a product's name says its cards are etched,
// the way the decklist reads it.
func saysEtched(name string) bool {
	for _, word := range strings.Fields(name) {
		if strings.EqualFold(word, "Etched") {
			return true
		}
	}
	return false
}

// A Secret Lair sold etched is its own product, and the bonus card it lists
// carries the plain foil flag - etched is the only foil that card comes in,
// and mtgjson has nowhere else to say so. Resolved on the flag alone the card
// lands on a printing the drop does not contain, and where no foil exists it
// lands back on the nonfoil one.
func TestEtchedProductHoldsEtchedCards(t *testing.T) {
	realDatastore(t)
	for _, code := range mtgmatcher.GetAllSets() {
		set, err := mtgmatcher.GetSet(code)
		if err != nil {
			continue
		}
		for _, product := range set.SealedProduct {
			if !saysEtched(product.Name) || !mtgmatcher.SealedHasDecklist(code, product.UUID) {
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

// Not every card an etched drop lists is sold etched, and the flag the
// product carries for it is the plain foil one. Asked for etched, a card sold
// in foil and nonfoil only used to answer with the nonfoil: the etched flag
// was dropped for a finish the card does not have, and the foil flag went
// with it.
func TestEtchedProductKeepsItsFoils(t *testing.T) {
	realDatastore(t)
	var checked int
	for _, code := range mtgmatcher.GetAllSets() {
		set, err := mtgmatcher.GetSet(code)
		if err != nil {
			continue
		}
		for _, product := range set.SealedProduct {
			if !saysEtched(product.Name) {
				continue
			}
			for _, content := range product.Contents["card"] {
				if !content.Foil {
					continue
				}
				id, err := mtgmatcher.MatchID(content.UUID, true, true)
				if err != nil {
					t.Errorf("%s %q: %s: %v", code, product.Name, content.UUID, err)
					continue
				}
				co, err := mtgmatcher.GetUUID(id)
				if err != nil {
					t.Errorf("%s %q: %s resolves to nothing", code, product.Name, id)
					continue
				}
				checked++
				if !co.Foil && !co.Etched {
					t.Errorf("%s %q holds %s #%s as %s, but the product lists it foil",
						code, product.Name, co.Name, co.Number, co.Finish)
				}
			}
		}
	}
	if checked == 0 {
		t.Skip("no etched product lists a foil card")
	}
}
