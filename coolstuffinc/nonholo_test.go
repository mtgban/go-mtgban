package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPokemonNonHoloDeckExclusive pins a "(Non-Holo)" listing to the plain
// printing PR-1840 Deck Exclusives actually carries at that number, rather
// than the set's own holo the bare name and number already match on their
// own - a $4.99 Team Aqua's Kyogre was served at $126.27, the price Game
// Nerdz buys the real holo for.
func TestPokemonNonHoloDeckExclusive(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	for _, tt := range []struct {
		name, edition, wantID string
	}{
		{"Jirachi (Non-Holo) - 99/181", "SM Team Up", "099-181_200377"},
		{"Team Aqua's Kyogre (Non-Holo) - 3/95", "Ex Team Magma vs. Team Aqua", "003-095_125256"},
		// The bracket's case varies by print era.
		{"Chandelure - 16/116 (NON-HOLO)", "BW Plasma Freeze", "016-116_135041"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card := pokemonListing(b, tt.name, tt.edition, "", false)
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("match(%q) = %v", card, err)
			}
			if id != tt.wantID {
				co, _ := b.GetUUID(id)
				t.Errorf("match(%q) = %q (%s/%s/%s), want %q", card, id, co.SetCode, co.Number, co.Finish, tt.wantID)
			}
		})
	}
}

// TestPokemonNonHoloLeavesAPromoShelf pins that the redirect never fires for
// a listing already on a promo shelf - pokemonPromoShelf owns that case, and
// probing Deck Exclusives underneath it would only add a second collision.
func TestPokemonNonHoloLeavesAPromoShelf(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	card := pokemonListing(b, "Some Card (Non-Holo) - 1/1", "SM Promos", "", false)
	if card.Edition == "Deck Exclusives" {
		t.Errorf("a promo shelf was redirected: %q", card)
	}
}

// TestPokemonNonHoloRefusesUnnumbered pins that a "(Non-Holo)" basic energy
// carrying no real number, or only a year, refuses rather than landing on
// PR-1840's lone unnumbered energy of that name.
func TestPokemonNonHoloRefusesUnnumbered(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	for _, tt := range []string{
		"Water Energy (Non-Holo)",
		"Lightning Energy (Non-Holo) - 2023",
	} {
		t.Run(tt, func(t *testing.T) {
			card := pokemonListing(b, tt, "SV Paldea Evolved", "", false)
			id, err := b.Match(card)
			if err == nil {
				co, _ := b.GetUUID(id)
				t.Errorf("match(%q) = %q (%s/%s), want a refusal", card, id, co.SetCode, co.Number)
			}
		})
	}
}
