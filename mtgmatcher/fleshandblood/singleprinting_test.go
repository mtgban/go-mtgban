package fleshandblood

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// singlePrintingFixture holds rows from the published datastore: a hero
// sold in one printing whose TCGplayer finish label disagrees with the
// storefronts' (Professor Teklovossen is Normal to TCGplayer, Rainbow Foil
// everywhere else), and a product sold in two finishes at a number that is
// otherwise alone in the backend (Hyper Driver (Red) DYN110, Normal and
// Cold Foil).
const singlePrintingFixture = `{"data": {
	"game": "fleshandblood",
	"sets": {
		"TCC": {"name": "Round the Table: TCCxLSS", "releaseDate": "2023-09-29"},
		"DYN": {"name": "Dynasty", "releaseDate": "2022-11-11"},
		"ARA": {"name": "Blitz Deck: Rosetta - Aurora", "releaseDate": "2024-09-20"},
		"BDOA": {"name": "Blitz Deck: Outsiders - Arakni", "releaseDate": "2023-03-24"}
	},
	"cards": [
		{"artist": "Simon Dominic", "externalLinks": {"fabId": "TCC001", "tcgPlayerId": 517456}, "fabId": "TCC001", "finish": "Normal", "id": "tcc001_517456", "image": "https://tcgplayer-cdn.tcgplayer.com/product/517456_400w.jpg", "name": "Professor Teklovossen", "number": "TCC001", "rarity": "Majestic", "setCode": "TCC"},
		{"artist": "Alexander Mokhov", "color": "Red", "externalLinks": {"fabId": "DYN110", "tcgPlayerId": 453390}, "fabId": "DYN110", "finish": "Normal", "id": "dyn110_453390", "image": "https://tcgplayer-cdn.tcgplayer.com/product/453390_400w.jpg", "name": "Hyper Driver (Red)", "number": "DYN110", "rarity": "Common", "setCode": "DYN"},
		{"artist": "Alexander Mokhov", "color": "Red", "externalLinks": {"fabId": "DYN110", "tcgPlayerId": 453390}, "fabId": "DYN110", "finish": "Cold Foil", "id": "dyn110_453390_coldfoil", "image": "https://tcgplayer-cdn.tcgplayer.com/product/453390_400w.jpg", "name": "Hyper Driver (Red)", "number": "DYN110", "rarity": "Common", "setCode": "DYN"},
		{"artist": "Asur Misoa", "externalLinks": {"fabId": "AUA001", "tcgPlayerId": 577174}, "fabId": "AUA001", "finish": "Rainbow Foil", "id": "ara001_577174_rainbowfoil", "image": "https://tcgplayer-cdn.tcgplayer.com/product/577174_400w.jpg", "name": "Aurora", "number": "ARA001", "rarity": "Rare", "setCode": "ARA"},
		{"artist": "Tomasz Jedruszek", "externalLinks": {"fabId": "ARA001", "tcgPlayerId": 489116}, "fabId": "ARA001", "finish": "Normal", "id": "ara001_489116", "image": "https://tcgplayer-cdn.tcgplayer.com/product/489116_400w.jpg", "name": "Arakni, Solitary Confinement", "number": "ARA001", "rarity": "Common", "setCode": "BDOA"}
	]
}}`

// TestSinglePrintingRainbowFinish pins the single-printing fallback: a hero
// sold in one printing lands on it whatever finish the storefront names.
func TestSinglePrintingRainbowFinish(t *testing.T) {
	b, err := Load(strings.NewReader(singlePrintingFixture))
	if err != nil {
		t.Fatal(err)
	}
	in := mtgmatcher.InputCard{Name: "Professor Teklovossen", Variation: "TCC001", Finish: "Rainbow Foil", Foil: true}
	got, err := b.Match(&in)
	if err != nil {
		t.Fatalf("Match(Professor Teklovossen, TCC001, Rainbow Foil) = %v", err)
	}
	if want := "tcc001_517456"; got != want {
		t.Errorf("Match(Professor Teklovossen, TCC001, Rainbow Foil) = %q, want %q", got, want)
	}
}

// TestMultiPrintingRainbowFinishRefused pins the fallback's guard: Hyper
// Driver (Red) DYN110 sells in two finishes, so a Rainbow Foil neither one
// wears still refuses, even though DYN110 is otherwise alone in the
// backend.
func TestMultiPrintingRainbowFinishRefused(t *testing.T) {
	b, err := Load(strings.NewReader(singlePrintingFixture))
	if err != nil {
		t.Fatal(err)
	}
	in := mtgmatcher.InputCard{Name: "Hyper Driver (Red)", Variation: "DYN110", Finish: "Rainbow Foil", Foil: true}
	if got, err := b.Match(&in); err == nil {
		t.Errorf("Match(Hyper Driver (Red), DYN110, Rainbow Foil) = %q, want a refusal", got)
	}
}

// TestSharedNumberWrongFinishRefused pins the fallback's other guard: Aurora
// ARA001 sells in one printing, but Arakni, Solitary Confinement is ARA001
// too, so a finish Aurora never wore does not land on it.
func TestSharedNumberWrongFinishRefused(t *testing.T) {
	b, err := Load(strings.NewReader(singlePrintingFixture))
	if err != nil {
		t.Fatal(err)
	}
	in := mtgmatcher.InputCard{Name: "Aurora", Variation: "ARA001", Finish: "Cold Foil", Foil: true}
	if got, err := b.Match(&in); err == nil {
		t.Errorf("Match(Aurora, ARA001, Cold Foil) = %q, want a refusal", got)
	}
}
