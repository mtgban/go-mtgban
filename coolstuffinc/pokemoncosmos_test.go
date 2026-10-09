package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPokemonCosmosHolo pins the collection-box holos sold under a main set:
// the catalog's miscellaneous shelf answers where it holds the printing at the
// listing's number, and the listing keeps its own answer where it does not.
func TestPokemonCosmosHolo(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")

	for _, tt := range []struct {
		desc, name, edition, notes, rarity, wantID string
	}{
		{"a holo promo with a cosmos holo of its number",
			"Charmeleon (Holo Promo) - 005/165", "SV 151", "", "Promo", "005-165_586829_holofoil"},
		{"a holo promo of any rarity is the collection box printing",
			"Squirtle - 33/214 (Holo Promo)", "SM Unbroken Bonds", "", "Common", "033-214_193275_holofoil"},
		{"a bare holo bracket beside a cosmos holo note",
			"Mimikyu (Holo) - 081/189", "SWSH Darkness Ablaze", "Cosmos Holo", "Holo Rare", "081-189_247469_holofoil"},
		{"a dashed cosmos holo on the blister shelf",
			"Dragonite - 5/20 - Cosmos Holo", "Dragon Vault", "", "Fixed", "005-020_233835_holofoil"},
		{"a bare holo bracket on a holo rare is the holo rare",
			"Munkidori (Holo) - 095/167", "SV Twilight Masquerade", "", "Holo Rare", "095-167_550139_holofoil"},
		{"a bare holo bracket beside a note saying it may be cosmos holo",
			"Mimikyu (Holo) - 081/189", "SWSH Darkness Ablaze", "Can be Regular or Cosmos Holo", "Holo Rare", "081-189_219467"},
		{"a printing the miscellaneous shelf lacks stays where it was",
			"Greavard (Holo Promo) - 100/197", "SV Obsidian Flames", "", "Promo", "100-197_509947"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card := pokemonCosmosHolo(b, pokemonListing(b, tt.name, tt.edition, tt.notes, false), tt.rarity)
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match(%v) = %v", card, err)
			}
			if id != tt.wantID {
				co, _ := b.GetUUID(id)
				t.Errorf("Match(%v) = %q (%s %s), want %q", card, id, co.SetCode, co.Number, tt.wantID)
			}
		})
	}
}
