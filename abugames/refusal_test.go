package abugames

import (
	"errors"
	"testing"
)

// TestUnprintedFinish pins the listings this storefront prices in a finish
// the catalog never sold the printing in. There is no printing of their own
// to price them against, and answering with the finish that was printed puts
// two of the storefront's prices on one uuid.
func TestUnprintedFinish(t *testing.T) {
	for _, test := range []struct {
		desc string
		card ABUCard
	}{
		{"a commander deck's extended art, which is sold in no foil", ABUCard{
			DisplayTitle: "Bladewing, Deathless Tyrant (Extended Art) - FOIL",
			Edition:      "Dominaria United Commander", Number: "9"}},
		{"and another set's", ABUCard{
			DisplayTitle: "Tributary Instructor (Extended Art) - FOIL",
			Edition:      "The Lost Caverns of Ixalan Commander", Number: "64"}},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := test.card
			in, err := preprocess(&card)
			if !errors.Is(err, errUnprintedFinish) {
				t.Errorf("preprocess(%q) = %v, %v, want %v", card.DisplayTitle, in, err, errUnprintedFinish)
			}
		})
	}
}

// TestForeignListing pins the listings priced in a language the catalog never
// printed the card in. The match falls back on the English printing, so the
// storefront's Italian and Japanese prices land beside its English one.
func TestForeignListing(t *testing.T) {
	for _, test := range []struct {
		desc string
		card ABUCard
		want error
	}{
		{"an Italian printing this set never had", ABUCard{
			DisplayTitle: "Elvish Piper - FOIL", Edition: "9th Edition", Number: "239",
			Title: "Non-English - Italian", Language: []string{"Italian"}}, errForeignListing},
		{"a Japanese one it never had either", ABUCard{
			DisplayTitle: "Elvish Piper - FOIL", Edition: "9th Edition", Number: "239",
			Title: "Non-English - Japanese", Language: []string{"Japanese"}}, errForeignListing},
		{"a set that does hold the Japanese printing is kept", ABUCard{
			DisplayTitle: "Urza's Mine (d Tower)", Edition: "Chronicles", Number: "114d",
			Title: "Non-English - Japanese", Language: []string{"Japanese"}}, nil},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := test.card
			_, err := preprocess(&card)
			if test.want == nil {
				if err != nil {
					t.Errorf("preprocess(%q) = %v, want no error", card.DisplayTitle, err)
				}
				return
			}
			if !errors.Is(err, test.want) {
				t.Errorf("preprocess(%q) = %v, want %v", card.DisplayTitle, err, test.want)
			}
		})
	}
}
