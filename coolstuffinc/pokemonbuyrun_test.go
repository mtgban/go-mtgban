package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPokemonBuylistCardReadsTheRun pins the print run on the buy side. The
// sell listings read the run out of the shelf's title; the buy rows did not,
// so every first-edition row matched the set spelled in the title and was
// published against the unlimited printing - the whole Base Set shelf at the
// first edition's price, which reads as arbitrage against every ordinary
// listing of the card.
func TestPokemonBuylistCardReadsTheRun(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")

	for _, tt := range []struct {
		row    CSIPriceEntry
		wantID string
	}{
		// Base Set files its runs under a set of its own, so the shelf has
		// to name that set and not the one it is titled after.
		{CSIPriceEntry{Name: "Abra - 43/102", ItemSet: "1st Edition Base Set", Notes: "1st Edition", Number: "43/102", RarityName: "Common"}, "043-102_107040_1stedition"},
		{CSIPriceEntry{Name: "Charizard - 4/102", ItemSet: "1st Edition Base Set", Notes: "1st Edition", Number: "4/102", RarityName: "Holo Rare"}, "004-102_106999_1steditionholofoil"},
		// The other shelves carry their run as a finish of the set beside
		// them, and keep answering with it.
		{CSIPriceEntry{Name: "Lapras - 10/62", ItemSet: "1st Edition Fossil", Notes: "1st Edition", Number: "10/62", RarityName: "Holo Rare"}, "10-62_44419_1steditionholofoil"},
		// A shelf naming no run is matched as it always was.
		{CSIPriceEntry{Name: "Charizard - 4/102", ItemSet: "Base Set", Notes: "Unlimited Edition", Number: "4/102", RarityName: "Holo Rare"}, "004-102_42382_holofoil"},
	} {
		t.Run(tt.row.ItemSet+" "+tt.row.Name, func(t *testing.T) {
			card, run := pokemonBuylistCard(b, tt.row)
			if card == nil {
				t.Fatal("the row preprocessed to nothing")
			}
			var id string
			var err error
			if run != nil {
				id, err = matchRun(b, card, run)
			} else {
				id, err = b.Match(card)
			}
			if err != nil {
				t.Fatalf("match = %v", err)
			}
			if id != tt.wantID {
				co, _ := b.GetUUID(id)
				t.Errorf("match = %q (%s/%s), want %q", id, co.SetCode, co.Finish, tt.wantID)
			}
		})
	}
}
