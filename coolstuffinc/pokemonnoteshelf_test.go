package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPokemonNoteShelfFollowsTheNote pins the listings this storefront files
// on a main or promo shelf while its note names the catalog's own shelf.
func TestPokemonNoteShelfFollowsTheNote(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")

	for _, tt := range []struct {
		row     CSIPriceEntry
		wantSet string
	}{
		{CSIPriceEntry{Name: "Cynthia - 119a/156 (Pokemon Regional Stamp)", ItemSet: "SM Promos", Number: "119a/156", Notes: "Pokemon Regional Championship Promo", RarityName: "Promo"}, "PR-1539"},
		{CSIPriceEntry{Name: "Misty's Gyarados - 049/182 (Destined Rivals Stamp)", ItemSet: "SV Destined Rivals", Number: "049/182", Notes: "Destined Rivals Stamp from Build & Battle Kits", RarityName: "Promo"}, "MCAP"},
		{CSIPriceEntry{Name: "Charizard - 11/108 (XY Evolutions Stamp)", ItemSet: "XY Promos", Number: "11/108", Notes: "XY Evolutions Prerelease", RarityName: "Holo Rare"}, "PR-1451"},
		{CSIPriceEntry{Name: "Latias - 9/20 - Dragon Vault Stamped Mirror Holo", ItemSet: "Dragon Vault", Number: "9/20", Notes: "From Dragon Vault Blister Pack.", RarityName: "Fixed"}, "BLE"},
	} {
		t.Run(tt.row.Name, func(t *testing.T) {
			card, _ := pokemonBuylistCard(b, tt.row)
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%v) = %v", card, err)
			}
			co, _ := b.GetUUID(id)
			if co.SetCode != tt.wantSet {
				t.Errorf("Match = %q (%s %s), want set %s", id, co.SetCode, co.Number, tt.wantSet)
			}
		})
	}
}
