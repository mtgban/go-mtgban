package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestNumberedListingReachesTheNameRules pins the rules that read a Pokemon
// listing by name. Both sides of this storefront spell a card with its
// collector number on the end, and the matcher reads the two apart itself, so
// the listings went on matching and nothing said that every rule keyed on a
// name was being asked a spelling no name ever has.
func TestNumberedListingReachesTheNameRules(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")

	for _, tt := range []struct {
		name, edition, variation, wantID string
	}{
		// The Elite Four cards of the Platinum sets.
		{"Bronzong 4 - 16/111", "Platinum Rising Rivals", "16", "16-111_83999"},
		// The printing is written behind the number, and neither side's
		// foil column says so, so it has to survive the read.
		{"Bronzong 4 - 16/111 (Reverse Foil)", "Platinum Rising Rivals", "16", "16-111_83999_reverseholofoil"},
		// The special energies, named for the catalog's energy labelled
		// special.
		{"Special Darkness Energy - 79/90", "HS Undaunted", "79", "79-90_84693"},
		// An energy whose type the catalog letters.
		{"Heat Fire Energy - 174/189 (Reverse Foil)", "SWSH Darkness Ablaze", "174/189", "174-189_219291_reverseholofoil"},
		// A misspelling this storefront reads off the printing.
		{"Sprigattito - MEP061", "ME Promos", "MEP061", "061_709975_holofoil"},
		// A Team Galactic invention, sold under the invention alone.
		{"SP Radar - 96/111", "Platinum Rising Rivals", "96", "96-111_89809"},
		// Nidoran, sold without the sex the catalog names it by.
		{"Nidoran - 57/101", "Ex Dragon Frontiers", "57", "57-101_87731"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card := pokemonListing(b, tt.name, tt.edition, tt.variation, false)
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%v) = %v", card, err)
			}
			if id != tt.wantID {
				co, _ := b.GetUUID(id)
				t.Errorf("Match = %q (%s %s %s), want %q", id, co.Name, co.Number, co.Finish, tt.wantID)
			}
		})
	}
}

// A tail that names no number is none: the storefront hangs plain wording off
// a dash as well, and a name read apart there would be handed on without it.
func TestNumberedListingLeavesOtherTails(t *testing.T) {
	for _, tt := range []struct{ in, name, number string }{
		{"Bronzong 4 - 16/111", "Bronzong 4", "16/111"},
		{"Bronzong 4 - 16/111 (Reverse Foil)", "Bronzong 4", "16/111 (Reverse Foil)"},
		{"Sprigattito - MEP061", "Sprigattito", "MEP061"},
		{"Ancient Mew - Movie Promo", "Ancient Mew - Movie Promo", ""},
		{"Mew ex (Blue) - B/RGB", "Mew ex (Blue) - B/RGB", ""},
		{"Unown ! - !/28", "Unown ! - !/28", ""},
		{"Charizard", "Charizard", ""},
	} {
		name, number := numberedListing(tt.in)
		if name != tt.name || number != tt.number {
			t.Errorf("numberedListing(%q) = %q, %q; want %q, %q", tt.in, name, number, tt.name, tt.number)
		}
	}
}
