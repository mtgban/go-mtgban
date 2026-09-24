package coolstuffinc

import (
	"slices"
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPokemonClassicCollectionReprintNote pins a Celebrations Classic
// Collection retail listing past two traps at once: the "(Classic
// Collection)" bracket that arrives in the name's own numbered tail rather
// than the note field the buylist side reads, and the note itself - "25th
// Anniversary Stamp Base Set Reprint" - which the stamped-promo redirect
// otherwise reads as a real stamp demand and blanks the edition entirely.
func TestPokemonClassicCollectionReprintNote(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	for _, tt := range []struct {
		name, notes, wantID string
	}{
		{"Blastoise - 2/102 (Classic Collection)", "25th Anniversary Stamp Base Set Reprint", "2-102_250319_holofoil"},
		{"Zekrom - 114/114 (Classic Collection)", "25th Anniversary Stamp Base Set Reprint", "114-114_250338_holofoil"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			card := pokemonListing(b, tt.name, "Celebrations", tt.notes, false)
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

// TestPokemonReprintNoteLeavesARealStamp pins that a genuine stamped promo,
// whose note carries no reprint wording, still redirects to the Promo shelf.
func TestPokemonReprintNoteLeavesARealStamp(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	card := pokemonListing(b, "Psyduck (Detective Pikachu Stamp) - SM199", "SM Promos", "", false)
	id, err := b.Match(card)
	if err != nil {
		t.Fatalf("match(%q) = %v", card, err)
	}
	co, err := b.GetUUID(id)
	if err != nil || !slices.Contains(co.PromoTypes, "stamped") {
		t.Errorf("match(%q) = %q, want the stamped printing", card, id)
	}
}
