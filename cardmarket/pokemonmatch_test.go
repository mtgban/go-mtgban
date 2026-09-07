package cardmarket

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
)

// pokemonDatastore is the published Pokemon datastore cut down to the
// printings these tests turn on, every row copied verbatim from it: Marowak
// of Jungle, the set Japan's "Pokémon Jungle" would trim down to, and the
// XY95 Pikachu of XY Promos, the set whose own name is the Japanese XY promo
// expansion's. Three rows rather than the whole file keeps the answers below
// facts about data the test states.
const pokemonDatastore = `{
 "game": "pokemon",
 "sets": {"JU": {"abbreviation": "JU", "name": "Jungle", "releaseDate": "1999-06-16"}, "PR-1451": {"abbreviation": "PR", "name": "XY Promos", "releaseDate": "2013-12-16", "type": "promo"}},
 "cards": [
  {"externalLinks": {"tcgPlayerId": 45142}, "finish": "1st Edition", "id": "39-64_45142_1e", "name": "Marowak", "number": "39/64", "rarity": "Uncommon", "setCode": "JU"},
  {"externalLinks": {"tcgPlayerId": 45142}, "finish": "Unlimited", "id": "39-64_45142_unl", "name": "Marowak", "number": "39/64", "rarity": "Uncommon", "setCode": "JU"},
  {"externalLinks": {"tcgPlayerId": 114004}, "finish": "Holofoil", "id": "xy95_114004_holo", "name": "Pikachu", "number": "XY95", "rarity": "Promo", "setCode": "PR-1451"},
  {"externalLinks": {"tcgPlayerId": 268261}, "finish": "Normal", "id": "98-xy-p_268261", "name": "Mega Tokyo's Pikachu", "number": "98/XY-P", "rarity": "Promo", "setCode": "PR-1451"}
 ]
}`

// TestMatchProductForeignExpansion pins that the name fallback refuses the
// Cardmarket expansions that are non-English catalogs wearing an English
// set's name. Both shapes of the trap have to hold: "Pokémon Jungle" is
// Japan's Jungle and only the game-name trim stands between it and the
// English set, while "XY Promos" is the datastore's own name for the English
// promos - the set gate cannot tell it apart at all. An expansion naming the
// set the English way stays open. And the XY Promos refusal holds only for a
// product naming no collector number: the datastore's set itself carries
// twelve Japanese promos under their ##/XY-P numbers, and a numbered product
// is asking for one of them.
func TestMatchProductForeignExpansion(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(pokemonDatastore))
	if err != nil {
		t.Fatal(err)
	}

	mkm := &Index{gameID: GamePokemon}
	for _, tt := range []struct {
		expansion, name, number, want string
	}{
		{"Jungle", "Marowak", "39", "39-64_45142_unl"},
		{"Pokémon Jungle", "Marowak", "39", ""},
		{"Magma Gang VS Aqua Gang: Double Crisis", "Marowak", "39", ""},
		{"XY Promos", "Pikachu", "", ""},
		{"XY Promos", "Mega Tokyo's Pikachu", "98", "98-xy-p_268261"},
	} {
		product := MKMProduct{
			Name:          tt.name,
			Number:        tt.number,
			ExpansionName: tt.expansion,
		}
		got := mkm.matchProduct(&product)
		if got != tt.want {
			t.Errorf("matchProduct(%q, %q) = %q, want %q", tt.expansion, tt.name, got, tt.want)
		}
	}
}

// TestPokemonBasicEnergy pins which names go quiet. The whole name is read,
// not a substring of it: a shelf sells the special energies on their own
// account, and one Trainer merely has the word in its title.
func TestPokemonBasicEnergy(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		// The nine kinds, both ways the catalog writes them.
		{"Grass Energy", true},
		{"Fire Energy", true},
		{"Water Energy", true},
		{"Lightning Energy", true},
		{"Psychic Energy", true},
		{"Fighting Energy", true},
		{"Darkness Energy", true},
		{"Metal Energy", true},
		{"Fairy Energy", true},
		{"Basic Grass Energy", true},
		{"Basic Lightning Energy", true},
		// Cardmarket pads some of its names with a trailing space.
		{"Water Energy ", true},
		{"basic water energy", true},
		// Special energies are cards a shelf sells for themselves, and a
		// run that cannot place one has something to say about it.
		{"Rainbow Energy", false},
		{"Jet Energy", false},
		{"Luminous Energy", false},
		{"Double Colorless Energy", false},
		{"Herbal Energy", false},
		// Not an energy at all - a Trainer with the word in its name,
		// which a substring test would have swallowed.
		{"Superior Energy Retrieval", false},
		{"Energy Retrieval", false},
		{"Energy Search", false},
		// Nor is a Pokemon whose name merely opens the same way.
		{"Grass", false},
		{"", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := pokemonBasicEnergy(tt.name); got != tt.want {
				t.Errorf("pokemonBasicEnergy(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestNoPrintingSkipsBasicEnergy pins that the skip is the Pokemon index's
// alone, and that a game sharing the route still refuses out loud.
func TestNoPrintingSkipsBasicEnergy(t *testing.T) {
	for _, tt := range []struct {
		desc   string
		gameID int
		name   string
		want   error
	}{
		{"a Pokemon basic energy goes quiet", GamePokemon, "Water Energy", nil},
		{"its bracketed spelling too", GamePokemon, "Grass Energy [Basic]", nil},
		{"a Pokemon special energy still refuses", GamePokemon, "Rainbow Energy", errNoPrinting},
		{"an ordinary Pokemon card still refuses", GamePokemon, "Pikachu", errNoPrinting},
		{"another game's energy still refuses", GameYuGiOh, "Water Energy", errNoPrinting},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			mkm := &Index{gameID: tt.gameID}
			if got := mkm.noPrinting(&MKMProduct{Name: tt.name}); !errors.Is(got, tt.want) {
				t.Errorf("noPrinting(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
