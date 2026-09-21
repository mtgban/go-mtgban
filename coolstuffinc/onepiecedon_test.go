package coolstuffinc

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestOnePieceDonNameReachesTheCard pins that a DON!! listing is handed
// over under the name the catalog files it by. All 238 of the game's
// DON!! cards are named "DON!! Card" and told apart by promo type,
// which the matcher already reads out of a listing's wording - but the
// storefront writes those words into the product name, so the name
// reached nothing and the wording was never read.
func TestOnePieceDonNameReachesTheCard(t *testing.T) {
	b := readGameDatastore(t, "onepiece", "ONEPIECE_PATH")

	for _, tt := range []struct {
		desc, name, shelf string
		foil              bool
		wantID            string
	}{
		// The storefront numbers a booster's DON!! cards in the name,
		// and the character is what tells them apart.
		{"a numbered character", "DON!! (03) - Uta", "PRB01 - Premium Booster", true, "don_593826_foil"},
		{"another at the same shelf", "DON!! (13) - Sakazuki", "PRB01 - Premium Booster", true, "don_593824_foil"},
		// The gold border is a promo type of its own, so the gold and
		// the plain printing of one character must not be confused.
		{"a gold border", "DON!! (07) - Kaido (GOLD)", "PRB01 - Premium Booster", true, "don_586552_foil"},
		// The second booster spells the same shape differently.
		{"the second booster's wording", "Don!! Card (33) (Buggy)", "PRB02 - Premium Booster 2", false, "don_655130"},
		{"its gold wording", "Don!! Card (35) (Carrot) (Gold Foil)", "PRB02 - Premium Booster 2", true, "don_655138_foil"},
		// A set's own DON!! carries no number at all.
		{"an unnumbered character", "DON!! - Nico Robin", "EB03 - Heroines Edition", true, "don_677568_foil"},
		{"an unnumbered gold", "DON!! - Nami (GOLD)", "EB03 - Heroines Edition", true, "don_677559_foil"},
		// The double packs label theirs by the pair and the volume.
		{"a double pack", "DON!! - Katakuri (Double Pack Vol. 7)", "OP11 - A Fist Of Divine Speed", true, "don_636745_foil"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			name, description, isDon := onePieceDonName(tt.name)
			if !isDon {
				t.Fatalf("onePieceDonName(%q) did not read a DON!! listing", tt.name)
			}
			card := &mtgmatcher.InputCard{
				Name:      name,
				Edition:   onePieceShelf(tt.shelf, tt.name),
				Variation: description,
				Foil:      tt.foil,
			}
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%v) = %v", card, err)
			}
			if id != tt.wantID {
				co, _ := b.GetUUID(id)
				t.Errorf("Match(%v) = %q (%s), want %q", card, id, co, tt.wantID)
			}
		})
	}
}

// TestOnePieceDonNameTellsTheGoldApart pins the printing the description
// must keep separate: one character's gold border and its plain printing
// share a name, a set and a number, and differ by a single promo type.
func TestOnePieceDonNameTellsTheGoldApart(t *testing.T) {
	b := readGameDatastore(t, "onepiece", "ONEPIECE_PATH")

	match := func(listing string) string {
		t.Helper()
		name, description, isDon := onePieceDonName(listing)
		if !isDon {
			t.Fatalf("onePieceDonName(%q) did not read a DON!! listing", listing)
		}
		id, err := b.Match(&mtgmatcher.InputCard{
			Name:      name,
			Edition:   onePieceShelf("EB03 - Heroines Edition", listing),
			Variation: description,
			Foil:      true,
		})
		if err != nil {
			t.Fatalf("Match(%q) = %v", listing, err)
		}
		return id
	}

	plain := match("DON!! - Nami")
	gold := match("DON!! - Nami (GOLD)")
	if plain == gold {
		t.Errorf("both listings answered %q; the border told two printings apart", plain)
	}
}

// TestOnePieceDonNameLeavesOtherListingsAlone pins that the ordinary
// cards keep their own name: only a listing the storefront opens with
// DON!! is one.
func TestOnePieceDonNameLeavesOtherListingsAlone(t *testing.T) {
	for _, name := range []string{
		"Monkey.D.Luffy (119) (Parallel)",
		"Donquixote Rosinante",
		"Don Accino",
		"",
	} {
		if _, _, isDon := onePieceDonName(name); isDon {
			t.Errorf("onePieceDonName(%q) read a DON!! listing", name)
		}
	}
}
