package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPokemonGoldStar pins a Gold Star spelled the storefront's way, "Mew *
// (Star)", to the catalog's "Mew Star" - the highest-value refusals on the
// buylist, Mew Star alone worth $1,800.
func TestPokemonGoldStar(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	for _, tt := range []struct {
		name, edition, wantID string
	}{
		{"Mew * (Star) - 101/101", "Ex Dragon Frontiers", "101-101_87408_holofoil"},
		{"Pikachu * (Star) - 104/110", "Ex Holon Phantoms", "104-110_88111_holofoil"},
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

// TestPokemonUnownListing pins an EX Unseen Forces Unown, sold under the
// storefront's own index alongside a repeat of the letter, to the catalog's
// plain "Unown" numbered by the letter alone.
func TestPokemonUnownListing(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	card := pokemonListing(b, "Unown A - A/28", "Ex Unseen Forces", "", false)
	id, err := b.Match(card)
	if err != nil {
		t.Fatalf("match(%q) = %v", card, err)
	}
	const want = "a-28_90168_holofoil"
	if id != want {
		co, _ := b.GetUUID(id)
		t.Errorf("match(%q) = %q (%s/%s/%s), want %q", card, id, co.SetCode, co.Number, co.Finish, want)
	}
}
