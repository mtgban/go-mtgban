package pokemon

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPromoLabelDepth pins that a printing sharing a label with its siblings
// is told from them by the labels it does not share. Four Cosmos Holo
// printings of one number differ only in the retailer that stamped them, so
// the wording naming a retailer has to outrank the wording naming none.
func TestPromoLabelDepth(t *testing.T) {
	path := os.Getenv("POKEMON_PATH")
	if path == "" {
		t.Skip("POKEMON_PATH not set; skipping the promo label depth")
	}
	f, err := datastore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Load(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	mtgmatcher.SetGlobalDatastore(b)

	for _, tt := range []struct{ desc, variation, want string }{
		{"the retailer is what tells the stampings apart", "117 GameStop Cosmos Holo", "117-159_626640_holo"},
		{"and the other retailer likewise", "117 EB Games Cosmos Holo", "117-159_629648_holo"},
		// Naming only the shared label names no one of them, and saying so
		// beats answering with whichever came first.
		{"the shared label alone still aliases", "117 Cosmos Holo", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := mtgmatcher.Match(&mtgmatcher.InputCard{Name: "Hop's Snorlax", Variation: tt.variation})
			if id != tt.want {
				t.Errorf("Match(%q) = %q (err %v), want %q", tt.variation, id, err, tt.want)
			}
		})
	}
}
