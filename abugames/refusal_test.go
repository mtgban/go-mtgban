package abugames

import (
	"errors"
	"testing"
)

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
