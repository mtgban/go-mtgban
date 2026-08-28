package magiccorner

import (
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestMain loads the datastore when one is configured. The rest of this
// package's tests read no cards, so a checkout without it still runs them;
// the promo-edition test asks the datastore whether a name is a set, and
// says so when it cannot.
func TestMain(m *testing.M) {
	if path := os.Getenv("ALLPRINTINGS5_PATH"); path != "" {
		if err := datastore.Load(path); err != nil {
			log.Fatalln(err)
		}
	}
	os.Exit(m.Run())
}

// TestPromoSetBase pins which editions are read as an expansion's promos.
// The store spells the same thing with and without the colon, so the colon
// cannot decide; what follows from getting this wrong is that every card
// with a promo pack printing anywhere gets stamped as one.
func TestPromoSetBase(t *testing.T) {
	if len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("ALLPRINTINGS5_PATH not set; skipping the promo edition suite")
	}

	for _, tt := range []struct {
		desc, edition, wantBase string
		wantPromo               bool
	}{
		{"the store's own colon spelling", "Tarkir: Dragonstorm: Promos", "Tarkir: Dragonstorm", true},
		{"and the same edition without it", "Tarkir: Dragonstorm Promos", "Tarkir: Dragonstorm", true},
		{"a core set reads the same way", "Foundations Promos", "Foundations", true},
		{"an edition naming an event names no expansion", "Game Day Promos", "", false},
		{"nor does this one", "Judge Rewards Promos", "", false},
		{"nor this one", "Friday Night Magic Promos", "", false},
		// Normalize drops a standalone "s", so this reaches the promo set
		// spelled "Store Championships" unless the names compare exactly.
		{"an event whose name is a promo set but for one letter", "Store Championship Promos", "", false},
		// Already a promo set: decorating it again sends its cards away.
		{"an edition that is itself a promo set", "San Diego Comic-Con 2013 Promos", "", false},
		{"an edition that is not promos at all", "Tarkir: Dragonstorm", "", false},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			base, _, promo := promoSetBase(tt.edition)
			if promo != tt.wantPromo || base != tt.wantBase {
				t.Errorf("promoSetBase(%q) = (%q, %v), want (%q, %v)", tt.edition, base, promo, tt.wantBase, tt.wantPromo)
			}
		})
	}
}
