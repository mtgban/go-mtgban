package coolstuffinc

import (
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestPokemonDeckHoloRedirectPrefersSetCode pins a print-run note past the
// plain printing of the same number: Undaunted's Espeon and Umbreon are
// each sold both as the set's own nonfoil and as the theme-deck
// cracked-ice pull a "Shattered Holo from Theme Deck" note names, tied on
// number and edition alone. The probe asks for the catalog's "Cracked Ice
// Holo" label so the cracked-ice twin wins the match over the plain one;
// the redirect is then accepted once that match lands on the marker's own
// set code, not because the landing itself is checked for the label.
func TestPokemonDeckHoloRedirectPrefersSetCode(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	for _, tt := range []struct {
		row    CSIPriceEntry
		wantID string
	}{
		{CSIPriceEntry{Name: "Espeon - 2/90", ItemSet: "HS Undaunted", Notes: "Shattered Holo from Theme Deck", Number: "2/90", RarityName: "Rare Holo"}, "002-090_125042_holofoil"},
		{CSIPriceEntry{Name: "Umbreon - 10/90", ItemSet: "HS Undaunted", Notes: "Shattered Holo from Theme Deck", Number: "10/90", RarityName: "Rare Holo"}, "010-090_125044_holofoil"},
		// A number the redirect's own set prints only once is unaffected
		// by the label: there is no plain twin to prefer over it.
		{CSIPriceEntry{Name: "Charizard - 14/181", ItemSet: "SM Team Up", Notes: "Shattered Holo Theme Deck Version", Number: "14/181", RarityName: "Rare Holo"}, "014-181_184217_holofoil"},
	} {
		t.Run(tt.row.Name, func(t *testing.T) {
			card, run := pokemonBuylistCard(b, tt.row)
			var id string
			var err error
			if run != nil {
				id, err = matchRun(b, card, run)
			} else {
				id, err = b.Match(card)
			}
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

// TestPokemonDeckHoloRedirectSkipsANonHoloListing pins that a "(Non-Holo)"
// listing is never handed to pokemonDeckHoloRedirect. The bracket sits in
// the number's own tail here (CSI's "Chandelure - 16/116 (NON-HOLO)"
// shape), not the name's head, because that is what leaves pokemonListing's
// local name clean enough to reach the redirect at all - without the
// guard, Theme Deck's own note then forces Espeon and Umbreon (where a
// plain nonfoil and a cracked-ice twin both exist) onto the wrong one.
func TestPokemonDeckHoloRedirectSkipsANonHoloListing(t *testing.T) {
	b := readGameDatastore(t, "pokemon", "POKEMON_PATH")
	for _, tt := range []struct {
		row    CSIPriceEntry
		wantID string
	}{
		{CSIPriceEntry{Name: "Team Magma's Groudon - 9/95 (Non-Holo)", ItemSet: "Ex Team Magma vs. Team Aqua", Notes: "Non-Holo Theme Deck Version", Number: "9", RarityName: "Rare"}, "009-095_125255"},
		{CSIPriceEntry{Name: "Espeon - 2/90 (Non-Holo)", ItemSet: "HS Undaunted", Notes: "Non-Holo Version from Theme Deck", Number: "2", RarityName: "Rare"}, "002-090_125043"},
		{CSIPriceEntry{Name: "Umbreon - 10/90 (Non-Holo)", ItemSet: "HS Undaunted", Notes: "Non-Holo Version from Theme Deck", Number: "10", RarityName: "Rare"}, "010-090_125045"},
	} {
		t.Run(tt.row.Name, func(t *testing.T) {
			card, run := pokemonBuylistCard(b, tt.row)
			var id string
			var err error
			if run != nil {
				id, err = matchRun(b, card, run)
			} else {
				id, err = b.Match(card)
			}
			if err != nil {
				t.Fatalf("match(%q) = %v", card, err)
			}
			co, _ := b.GetUUID(id)
			if id != tt.wantID {
				t.Errorf("match(%q) = %q (%s/%s/%s), want %q", card, id, co.SetCode, co.Number, co.Finish, tt.wantID)
			}
			if co.Foil || co.Etched {
				t.Errorf("a (Non-Holo) listing landed on a foil printing: %q", co)
			}
		})
	}
}
