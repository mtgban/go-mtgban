package cardtrader

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
)

// TestFoilPrintingID pins which listings the foil flag may move. CardTrader
// files a blueprint's foil listings under the plain printing's id, so the
// flag has to reach the foil sibling; an etched listing raises the very same
// flag, and answering it there would file the etched price under the foil
// printing. The ids are drawn from the datastore so the test holds across
// its releases.
func TestFoilPrintingID(t *testing.T) {
	b := realDatastore(t)

	// Strixhaven Mystical Archive sells one printing in all three finishes,
	// which is what lets the flag cross between two of them.
	plain := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, "235648")
	if plain == "" {
		t.Fatal("datastore carries no TCGplayer id 235648")
	}
	foil, err := b.MatchID(plain, true)
	if err != nil {
		t.Fatal(err)
	}
	etched, err := b.MatchID(plain, false, true)
	if err != nil {
		t.Fatal(err)
	}

	// A derived token pairing's own combined name is deliberately excluded
	// from every name index (mtgmatcher/magic/tokenpairs.go), so the plain
	// HasFoilPrinting path below can never find one's foil sibling by name -
	// resolved instead by the pairing's own tcgplayerProductId plus the
	// foil flag. AFR's dungeon-card pairing sells in foil; a pairing where
	// one face never did (Goblin // Giant Teddy Bear, TC21) is refused
	// rather than silently kept at its nonfoil price.
	pairedFoilCapable := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, "244297")
	if pairedFoilCapable == "" {
		t.Fatal("datastore carries no derived pairing for TCGplayer id 244297")
	}
	pairedFoilCapableFoil, err := b.MatchID(pairedFoilCapable, true)
	if err != nil {
		t.Fatal(err)
	}
	pairedNonfoilOnly := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, "209922")
	if pairedNonfoilOnly == "" {
		t.Fatal("datastore carries no derived pairing for TCGplayer id 209922")
	}

	// Card Trader spells this one "Lord of Ulvenwald", which no printing
	// carries, so only the resolved card's own name finds its foil.
	dfc := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, "247917")
	if dfc == "" {
		t.Fatal("datastore carries no TCGplayer id 247917")
	}
	dfcFoil, err := b.MatchID(dfc, true)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc   string
		cardID string
		want   string
	}{
		{"plain printing reaches its foil", plain, foil},
		{"a double-faced printing reaches its foil", dfc, dfcFoil},
		{"foil printing stays put", foil, foil},
		{"etched printing stays put", etched, etched},
		{"unknown id is left alone", "not-a-uuid", "not-a-uuid"},
		{"a derived pairing sold in foil reaches its own foil sibling", pairedFoilCapable, pairedFoilCapableFoil},
		{"a derived pairing never sold in foil is refused, not kept nonfoil", pairedNonfoilOnly, ""},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			got := foilPrintingID(b, test.cardID)
			if got != test.want {
				t.Errorf("got %q, want %q", got, test.want)
			}
		})
	}
}

// wotcRunDatastore is two WotC holos as published, each sold in a 1st Edition
// and an Unlimited printing of its one TCGplayer product.
const wotcRunDatastore = `{"meta": {"date": "2026-10-23", "version": "1"}, "data": {
  "game": "pokemon",
  "properties": {"rarity": ["Secret Rare", "Holo Rare"], "type": ["Grass", "Lightning"]},
  "sets": {
    "G1": {"abbreviation": "G1", "baseSetSize": 132, "name": "Gym Heroes", "releaseDate": "2000-10-14"},
    "TR": {"abbreviation": "TR", "baseSetSize": 82, "name": "Team Rocket", "releaseDate": "2000-04-24"}
  },
  "cards": [
    {"externalLinks": {"tcgPlayerId": 84647, "tcgdexId": "base5-83"}, "finish": "1st Edition Holofoil", "id": "83-82_84647_1steditionholofoil", "name": "Dark Raichu", "number": "83", "rarity": "Secret Rare", "setCode": "TR", "total": "82", "type": "Lightning", "types": ["Lightning"]},
    {"externalLinks": {"tcgPlayerId": 84647, "tcgdexId": "base5-83"}, "finish": "Unlimited Holofoil", "id": "83-82_84647_unlimitedholofoil", "name": "Dark Raichu", "number": "83", "rarity": "Secret Rare", "setCode": "TR", "total": "82", "type": "Lightning", "types": ["Lightning"]},
    {"externalLinks": {"tcgPlayerId": 88788, "tcgdexId": "gym1-13"}, "finish": "1st Edition Holofoil", "id": "013-132_88788_1steditionholofoil", "name": "Rocket's Scyther", "number": "013", "rarity": "Holo Rare", "setCode": "G1", "total": "132", "type": "Grass", "types": ["Grass"]},
    {"externalLinks": {"tcgPlayerId": 88788, "tcgdexId": "gym1-13"}, "finish": "Unlimited Holofoil", "id": "013-132_88788_unlimitedholofoil", "name": "Rocket's Scyther", "number": "013", "rarity": "Holo Rare", "setCode": "G1", "total": "132", "type": "Grass", "types": ["Grass"]}
  ]
}}`

// TestListingID pins that a Pokemon listing whose named finish the product
// is not sold in keeps the print run it states instead of the foil flag's
// first printing: a reverse holo of a card with no reverse printing is its
// unlimited holo, and a first-edition listing on a version that never says
// holo is the 1st Edition holo.
func TestListingID(t *testing.T) {
	b, err := mtgmatcher.Open("pokemon", strings.NewReader(wotcRunDatastore))
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		desc, id, finish string
		foil, first      bool
		want             string
	}{
		{"reverse flag, no reverse printing", "88788", "Reverse Holofoil", true, false, "013-132_88788_unlimitedholofoil"},
		{"reverse flag on a first-edition listing", "88788", "Reverse Holofoil", true, true, "013-132_88788_1steditionholofoil"},
		{"first edition named without the holo", "84647", "1st Edition", false, true, "83-82_84647_1steditionholofoil"},
		{"first edition holo named outright", "88788", "1st Edition Holofoil", false, true, "013-132_88788_1steditionholofoil"},
		{"no finish named keeps the flag", "84647", "", true, false, "83-82_84647_1steditionholofoil"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card := &mtgmatcher.InputCard{Finish: tt.finish, Foil: tt.foil}
			var p Product
			p.Properties.FirstEdition = tt.first
			got := listingID(b, GamePokemon, tt.id, card, p)
			if got != tt.want {
				t.Errorf("listingID(%s, %q) = %q, want %q", tt.id, tt.finish, got, tt.want)
			}
		})
	}
}
