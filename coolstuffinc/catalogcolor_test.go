package coolstuffinc

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestCatalogColor pins the four colours Duelist League 9 prints one number
// in. Nothing but the colour tells them apart, and this storefront names one
// of them differently: "Light Blue" says the word blue, so it answered with
// the blue printing and two products met on one id.
func TestCatalogColor(t *testing.T) {
	path := os.Getenv("YUGIOH_PATH")
	if path == "" {
		t.Skip("Need YUGIOH_PATH variable set to run this test")
	}
	if err := datastore.Load(path); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name  string
		promo string
	}{
		{"Kuriboh (Blue)", "blue"},
		{"Kuriboh (Bronze)", "bronze"},
		{"Kuriboh (Light Green)", "green"},
		{"Kuriboh (Light Blue)", "silver"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			card := &mtgmatcher.InputCard{
				Name:      catalogColor(test.name),
				Edition:   "Duelist League 9",
				Variation: "DL09-EN003 Rare",
			}
			id, err := mtgmatcher.Match(card)
			if err != nil {
				t.Fatalf("Match(%q) = %v", test.name, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			var found bool
			for _, promoType := range co.PromoTypes {
				if promoType == test.promo {
					found = true
				}
			}
			if !found {
				t.Errorf("Match(%q) = %s %v, want one of them to be %q", test.name, co.Number, co.PromoTypes, test.promo)
			}
		})
	}
}
