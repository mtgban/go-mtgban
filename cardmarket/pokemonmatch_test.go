package cardmarket

import (
	"errors"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

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
  {"externalLinks": {"tcgPlayerId": 45142}, "finish": "1st Edition", "id": "39-64_45142_1stedition", "name": "Marowak", "number": "39/64", "rarity": "Uncommon", "setCode": "JU"},
  {"externalLinks": {"tcgPlayerId": 45142}, "finish": "Unlimited", "id": "39-64_45142_unlimited", "name": "Marowak", "number": "39/64", "rarity": "Uncommon", "setCode": "JU"},
  {"externalLinks": {"tcgPlayerId": 114004}, "finish": "Holofoil", "id": "xy95_114004_holofoil", "name": "Pikachu", "number": "XY95", "rarity": "Promo", "setCode": "PR-1451"},
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
	installDatastore(t, "pokemon", pokemonDatastore)

	mkm := &Index{gameID: cm.GamePokemon}
	for _, tt := range []struct {
		expansion, name, number, want string
	}{
		{"Jungle", "Marowak", "39", "39-64_45142_unlimited"},
		{"Pokémon Jungle", "Marowak", "39", ""},
		{"Magma Gang VS Aqua Gang: Double Crisis", "Marowak", "39", ""},
		{"XY Promos", "Pikachu", "", ""},
		{"XY Promos", "Mega Tokyo's Pikachu", "98", "98-xy-p_268261"},
	} {
		product := cm.Product{
			Name:          tt.name,
			Number:        tt.number,
			ExpansionName: tt.expansion,
		}
		got, _ := mkm.matchPokemon(&product)
		if got != tt.want {
			t.Errorf("matchPokemon(%q, %q) = %q, want %q", tt.expansion, tt.name, got, tt.want)
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
		{"a Pokemon basic energy goes quiet", cm.GamePokemon, "Water Energy", nil},
		{"its bracketed spelling too", cm.GamePokemon, "Grass Energy [Basic]", nil},
		{"a Pokemon special energy still refuses", cm.GamePokemon, "Rainbow Energy", errNoPrinting},
		{"an ordinary Pokemon card still refuses", cm.GamePokemon, "Pikachu", errNoPrinting},
		{"another game's energy still refuses", cm.GameYuGiOh, "Water Energy", errNoPrinting},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			mkm := &Index{gameID: tt.gameID}
			if got := mkm.noPrinting(&cm.Product{Name: tt.name}); !errors.Is(got, tt.want) {
				t.Errorf("noPrinting(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// letteredDatastore is the published datastore cut down to the two promo
// programmes whose numbers carry a letter, every row copied verbatim from
// it. Field Blower 125a was handed out at league play and Camerupt is an
// alternate art, and Cardmarket shelves each under the set it reprints -
// "Guardians Rising", "XY Black Star Promos" - rather than the programme
// that gave it out.
const letteredDatastore = `{
 "game": "pokemon",
 "sets": {
  "SM02": {"abbreviation": "SM02", "baseSetSize": 145, "name": "SM - Guardians Rising", "releaseDate": "2017-05-05"},
  "PR-1451": {"abbreviation": "PR", "name": "XY Promos", "releaseDate": "2013-12-16", "type": "promo"},
  "PR-1539": {"abbreviation": "PR", "name": "League & Championship Cards", "releaseDate": "2016-06-01"},
  "PR-1938": {"abbreviation": "PR", "name": "Alternate Art Promos", "releaseDate": "2014-08-13", "type": "promo"}
 },
 "cards": [
  {"externalLinks": {"tcgPlayerId": 185137}, "finish": "Reverse Holofoil", "id": "125a-145_185137_reverseholofoil", "name": "Field Blower", "number": "125a", "originalName": "Field Blower - 125a/145 (Pokemon League)", "promoTypes": ["pokemon league"], "rarity": "Promo", "setCode": "PR-1539", "total": "145", "type": "Item", "variant": "Pokemon League"},
  {"externalLinks": {"tcgPlayerId": 148345}, "finish": "Holofoil", "id": "xy198a_148345_holofoil", "name": "M Camerupt EX", "number": "XY198a", "originalName": "M Camerupt EX - XY198a", "rarity": "Promo", "setCode": "PR-1938", "type": "Fire"}
 ]
}`

// TestMatchPokemonLettered pins that a number with a letter hung off it
// reaches both programmes that number their cards that way, and that the
// prefixed spelling reaches them too. Only Alternate Art Promos was tried
// before, so every League & Championship product refused.
func TestMatchPokemonLettered(t *testing.T) {
	installDatastore(t, "pokemon", letteredDatastore)
	mkm := &Index{gameID: cm.GamePokemon}

	for _, tt := range []struct {
		desc      string
		product   cm.Product
		wantID    string
		wantError error
	}{
		{
			// The shelf is the set it reprints, and the row is the
			// league programme's.
			desc:    "a league promo reaches League & Championship Cards",
			product: cm.Product{Name: "Field Blower", Number: "125a", ExpansionName: "SM - Guardians Rising"},
			wantID:  "125a-145_185137_reverseholofoil",
		},
		{
			// The promo shelf writes its programme's prefix onto the
			// number, so the letter test has to see past it.
			desc:    "a prefixed number still reads as lettered",
			product: cm.Product{Name: "M Camerupt EX", Number: "198a", ExpansionName: "XY Black Star Promos"},
			wantID:  "xy198a_148345_holofoil",
		},
		{
			// A plain number names no lettered promo, and the shelf
			// carries no row for it.
			desc:      "a plain number reaches neither programme",
			product:   cm.Product{Name: "Field Blower", Number: "125", ExpansionName: "SM - Guardians Rising"},
			wantError: errNoPrinting,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			product := tt.product
			id, err := mkm.matchPokemon(&product)
			if tt.wantError != nil {
				if !errors.Is(err, tt.wantError) {
					t.Fatalf("matchPokemon = (%q, %v), want error %v", id, err, tt.wantError)
				}
				return
			}
			if err != nil {
				t.Fatalf("matchPokemon returned %v", err)
			}
			if id != tt.wantID {
				t.Errorf("matchPokemon = %q, want %q", id, tt.wantID)
			}
		})
	}
}

// TestPokemonCodeCard pins what a Pokemon shelf sells that is not a card.
func TestPokemonCodeCard(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		{"Code Card - Sword & Shield", true},
		{"Online Code Card", true},
		// A counter the booster box ships with, printed on card stock and
		// shelved beside the singles; no catalog has a row for one.
		{"VSTAR Marker", true},
		// The Pokemon themselves are cards, and their names open the
		// same way.
		{"Arceus VSTAR", false},
		{"Charizard VSTAR", false},
		{"Pikachu", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := pokemonCodeCard(tt.name); got != tt.want {
				t.Errorf("pokemonCodeCard(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}
