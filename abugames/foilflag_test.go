package abugames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestFoilFlag pins the flag this storefront writes both with its space and
// without. Reading only the spaced form priced a foil as a nonfoil, and put
// its price beside the nonfoil's on one uuid.
func TestFoilFlag(t *testing.T) {
	for _, test := range []struct {
		desc  string
		title string
		want  bool
	}{
		{"the flag written without its space", "Island (261) -FOIL", true},
		{"and with it", "Island (261) - FOIL", true},
		{"a card that names no finish", "Island (261)", false},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := ABUCard{DisplayTitle: test.title, Edition: "Ravnica Allegiance", Number: "261"}
			in, err := preprocess(&card)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", test.title, err)
			}
			if in.Foil != test.want {
				t.Errorf("preprocess(%q).Foil = %v, want %v", test.title, in.Foil, test.want)
			}
			if _, err := mtgmatcher.Match(in); err != nil {
				t.Errorf("Match(%q) = %v", in, err)
			}
		})
	}
}
