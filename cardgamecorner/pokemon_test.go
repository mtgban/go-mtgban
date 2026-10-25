package cardgamecorner

import (
	"os"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
)

var (
	pokemonOnce    sync.Once
	pokemonErr     error
	pokemonBackend *mtgmatcher.Backend
)

// pokemonDatastore reads the Pokemon datastore the first time a test asks
// for it, and skips where the run carries none.
func pokemonDatastore(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	pokemonOnce.Do(func() {
		path := os.Getenv("POKEMON_PATH")
		if path == "" {
			return
		}
		pokemonBackend, pokemonErr = datastore.Read(mtgmatcher.GamePokemon, path)
	})
	if pokemonErr != nil {
		t.Fatal(pokemonErr)
	}
	if pokemonBackend == nil {
		t.Skip("Need POKEMON_PATH set to run this test")
	}
	return pokemonBackend
}

// TestPokemonFirstEdition pins a first edition copy to its first edition
// printing: a Base copy is the Shadowless product's, a Shining card is its
// holofoil, and a set printed in no first edition keeps its one printing.
func TestPokemonFirstEdition(t *testing.T) {
	b := pokemonDatastore(t)
	for _, tt := range []struct {
		card Card
		want string
	}{
		{Card{Name: "Shining Charizard", Edition: "Neo Destiny", Number: "107", Rarity: "Rare Secret"}, "107-105_89163_1steditionholofoil"},
		{Card{Name: "Wartortle ", Edition: "Base", Number: "42", Rarity: "Uncommon"}, "042-102_107039_1stedition"},
		{Card{Name: "Bebe's Search", Edition: "Mysterious Treasures", Number: "109", Rarity: "Uncommon"}, "109-123_83761"},
	} {
		card := tt.card
		card.Variants = []Variant{{Print: "First Edition", Language: "EN", Foil: "Non Foil"}}
		in, err := preprocessGame(b, &card, 0)
		if err != nil {
			t.Errorf("%s: preprocess: %v", card.Name, err)
			continue
		}
		id, err := b.Match(in)
		if err != nil || id != tt.want {
			t.Errorf("%s (%s): Match = %q, %v, want %q", card.Name, card.Edition, id, err, tt.want)
		}
	}
}

// TestEnergyName pins an energy's type letter to the spelling the catalog
// gives that energy: spelled out where that names a card, kept where not.
func TestEnergyName(t *testing.T) {
	b := pokemonDatastore(t)
	for _, tt := range []struct{ in, want string }{
		{"Wash [W] Energy", "Wash Water Energy"},
		{"Bubbly [W] Energy", "Bubbly [W] Energy"},
		{"Unown [A] Lv.15", "Unown [A] Lv.15"},
	} {
		got := energyName(b, tt.in)
		if got != tt.want {
			t.Errorf("energyName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
