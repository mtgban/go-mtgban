package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func TestExtractWCDNumber(t *testing.T) {
	for _, tt := range []struct {
		str, prefix string
		sideboard   bool
		want        string
	}{
		{"Kai Budde kb112", "kb", false, "kb112"},
		{"Kai Budde (kb112sb)", "kb", true, "kb112sb"},
		{"Jon Finkel jf0012", "jf", false, "jf12"},
		{"Jon Finkel jf12a", "jf", false, "jf12a"},
		// A field holding nothing past the deck code, its zeros or the
		// sideboard mark is passed over, and the next one still read
		{"Kai Budde kb kb112", "kb", false, "kb112"},
		{"Kai Budde (kb)", "kb", false, ""},
		{"Leon Lindback ll", "ll", false, ""},
		{"Leon Lindback llsb", "ll", true, ""},
		{"0", "", false, ""},
		{"00", "", false, ""},
		{"Not Tournament Legal ()", "", false, ""},
	} {
		if got := ExtractWCDNumber(tt.str, tt.prefix, tt.sideboard); got != tt.want {
			t.Errorf("ExtractWCDNumber(%q, %q, %v) = %q, want %q", tt.str, tt.prefix, tt.sideboard, got, tt.want)
		}
	}
}

// "City of Brass (0)" is a plain card name with a stray number, and every
// World Championship printing of it ran wcdNumberCompare on the "0": the
// empty number was indexed and Match panicked, which ended a scraper run.
func TestWorldChampVariationWithNoNumber(t *testing.T) {
	b := &mtgmatcher.Backend{Sets: map[string]*mtgmatcher.Set{
		"WC01": {Name: "World Championship Decks 2001"},
	}}
	card := &mtgmatcher.Card{Name: "City of Brass", SetCode: "WC01", Number: "ll112sb"}
	for _, variation := range []string{"0", "00", "()", "Leon Lindback ll", "Kai Budde (kb)"} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("variation %q: %v", variation, r)
				}
			}()
			in := &mtgmatcher.InputCard{Name: "City of Brass", Edition: "World Championship Decks 2001", Variation: variation}
			wcdNumberCompare(b, in, card)
		}()
	}
}
