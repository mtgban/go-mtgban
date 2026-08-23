package cardmarket

import (
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
