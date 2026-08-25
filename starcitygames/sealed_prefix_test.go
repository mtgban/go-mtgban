package starcitygames

import (
	"io"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
	"github.com/mtgban/go-mtgban/mtgmatcher/riftbound"
)

// TestSealedUntrimmedGamePrefixResolves covers the two games whose sealed
// names keep the game words the prefix trim was meant to take off: the
// catalog's game field is not how their names open, so nothing is trimmed and
// the resolver is handed the game words along with the product.
//
// That is only safe because the rule forgives words the vendor says over the
// datastore's name, and nothing in this package said so. If it ever stops
// forgiving them, these two games lose their whole sealed catalog silently -
// every product refused, which reads exactly like a storefront that stopped
// stocking them.
func TestSealedUntrimmedGamePrefixResolves(t *testing.T) {
	for _, tt := range []struct {
		game, env string
		load      func(io.Reader) (*mtgmatcher.Backend, error)
		names     []string
	}{
		{"Riftbound", "RIFTBOUND_PATH", riftbound.Load, []string{
			"Riftbound: League of Legends TCG - Origins Booster Box",
			"Riftbound: League of Legends TCG - Spiritforged Booster Box",
		}},
		{"Lorcana", "LORCANA_PATH", lorcana.Load, []string{
			"Lorcana: The First Chapter Booster Box",
			"Lorcana: Into the Inklands Booster Box",
		}},
	} {
		t.Run(tt.game, func(t *testing.T) {
			withGameDatastore(t, tt.env, tt.load)
			for _, catalogName := range tt.names {
				product := CatalogProduct{Name: catalogName, Game: tt.game}
				name := sealedProductName(product)
				if name != catalogName {
					t.Errorf("sealedProductName(%q) = %q, want it left alone",
						catalogName, name)
				}
				// The product without its game words is the baseline: a
				// datastore that cannot place it has no sealed catalog for
				// this game to begin with (a checkout's copy may carry only
				// singles), and there is nothing here to assert either way.
				bare := trimGameWords(tt.game, name)
				stripped, err := mtgmatcher.ResolveSealed(bare)
				if err != nil {
					t.Skipf("the installed %s datastore does not carry %q", tt.game, bare)
				}
				// The name as the catalog spells it has to land on that same
				// printing: the game words are extra, not part of what picked
				// it.
				uuid, err := mtgmatcher.ResolveSealed(name)
				if err != nil {
					t.Errorf("%q: %v", name, err)
					continue
				}
				if uuid != stripped {
					t.Errorf("%q resolved to %s, and %s without its game words",
						name, uuid, stripped)
				}
			}
		})
	}
}

// trimGameWords takes off the game words a sealed name really opens with, as
// opposed to the ones the catalog's game field spells. It exists to say what
// the trim would take off if it fired, and nothing in the scraper needs it.
func trimGameWords(game, name string) string {
	switch game {
	case "Riftbound":
		return name[len("Riftbound: League of Legends TCG - "):]
	case "Lorcana":
		return name[len("Lorcana: "):]
	}
	return name
}
