package mtgmatcher_test

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
	"github.com/mtgban/go-mtgban/mtgmatcher/gundam"
	"github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
	"github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
	"github.com/mtgban/go-mtgban/mtgmatcher/palworld"
	"github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
	"github.com/mtgban/go-mtgban/mtgmatcher/riftbound"
	"github.com/mtgban/go-mtgban/mtgmatcher/yugioh"
)

// TestEveryGameNamesAnAddedPrinting pins that no game refuses a printing
// TCGplayer has added to its category. The builders carry such a printing
// under its own name rather than dropping it, and a game whose vocabulary is
// a closed list here would fold it to "" - which files a CardObject under
// the empty string and strands a real, priced sku.
//
// Refusing a name is still how a caller learns it asked for something this
// datastore does not sell: MatchIDFinish checks the finish against the ones
// the datastore actually carries, so the guard lives there rather than in a
// list each game keeps.
func TestEveryGameNamesAnAddedPrinting(t *testing.T) {
	for _, g := range []struct {
		name  string
		rules mtgmatcher.GameRules
	}{
		{"pokemon", pokemon.Rules{}},
		{"onepiece", onepiece.Rules{}},
		{"yugioh", yugioh.Rules{}},
		{"fleshandblood", fleshandblood.Rules{}},
		{"gundam", gundam.Rules{}},
		{"palworld", palworld.Rules{}},
		{"lorcana", lorcana.Rules{}},
		{"riftbound", riftbound.Rules{}},
	} {
		t.Run(g.name, func(t *testing.T) {
			if got := g.rules.CanonicalFinish("Prismatic Foil"); got != "prismaticfoil" {
				t.Errorf("CanonicalFinish(%q) = %q, want it named rather than refused",
					"Prismatic Foil", got)
			}
			// The shared names keep their shared meaning, and an empty
			// name is still nothing.
			if got := g.rules.CanonicalFinish(""); got != "" {
				t.Errorf("CanonicalFinish(%q) = %q, want %q", "", got, "")
			}
		})
	}
}
