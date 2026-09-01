package abugames

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestNeverPrinted pins the listings naming a card the catalog does not hold.
// Each was answered with the nearest printing wearing another identity - a
// tutorial card, or the same card reprinted out of a different set - and
// carried the storefront's price there, beside the price already on it.
func TestNeverPrinted(t *testing.T) {
	for _, test := range []struct {
		desc string
		card ABUCard
	}{
		{"a nonfoil of a card its set sells only in foil", ABUCard{
			DisplayTitle: "Aang's Defense (211)", Edition: "Avatar: The Last Airbender", Number: "211"}},
		{"and one whose set keeps a tutorial card at the other number", ABUCard{
			DisplayTitle: "Katara, Heroic Healer (215)", Edition: "Avatar: The Last Airbender Eternal", Number: "215"}},
		{"a reprint's nonfoil, where that set reprinted it in foil", ABUCard{
			DisplayTitle: "Entreat the Angels (The List)", Edition: "Avacyn Restored", Number: "20"}},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := test.card
			in, err := preprocess(&card)
			if err == nil {
				_, err = mtgmatcher.Match(in)
			}
			if err == nil {
				t.Errorf("preprocess(%q) = %v, want a refusal", card.DisplayTitle, in)
				return
			}
			if !errors.Is(err, errUnprintedFinish) {
				t.Logf("%q refused as %v", card.DisplayTitle, err)
			}
		})
	}
}
