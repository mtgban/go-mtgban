package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPokemonMetalCard pins the Ultra-Premium Collection metal cards, named
// for the metal and the set they copy, to the metal printings, with or
// without the note saying so; and a card really named for its metal to that
// card.
func TestPokemonMetalCard(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	for _, tt := range []struct {
		name, edition, notes, wantID string
	}{
		{"Metal Base Set Charizard - 4/102", "Celebrations", "Metal Card from Celebrations - Ultra Premium Collection", "004-102_252517_holofoil"},
		{"Metal Base Set Pikachu - 58/102", "Celebrations", "Metal Card from Celebrations - Ultra Premium Collection", "058-102_252516_holofoil"},
		{"Metal Mew ex - 205/165", "SV 151", "Metal Card from 151 Ultra-Premium Collection", "205-165_519481"},
		{"Metal Mew ex - 205/165", "SV 151", "", "205-165_519481"},
		{"Metal Energy - 112/114", "Black and White", "", "112-114_87352"},
		{"Metal Energy (Secret Rare) - 163/149", "Sun & Moon", "", "163-149_127202_holofoil"},
	} {
		t.Run(tt.name+"/"+tt.notes, func(t *testing.T) {
			card := pokemonListing(b, tt.name, tt.edition, tt.notes, false)
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
