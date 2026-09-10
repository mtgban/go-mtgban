package fleshandblood

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// finishTwinsFixture holds the products the catalog sells twice at one
// number, once per finish: the cold and rainbow foil "Will of Arcana", The
// Hunted's art card plain and in cold foil, and the Mastery Pack Uzuri,
// whose plain copy the catalog files at the marvel rarity beside the marvel
// itself. Ash and the Invoke are the pairs a label already tells apart.
// Every row is the datastore's.
const finishTwinsFixture = `{
	"game": "fleshandblood",
	"sets": {
		"ROS": {"name": "Rosetta", "releaseDate": "2024-09-27"},
		"TH": {"name": "The Hunted", "releaseDate": "2025-02-14"},
		"MPA": {"name": "Mastery Pack Assassin", "releaseDate": "2026-06-05"},
		"UPR": {"name": "Uprising", "releaseDate": "2022-06-24"}
	},
	"cards": [
		{"externalLinks": {"fabId": "ROS000", "tcgPlayerId": 577711}, "fabId": "ROS000", "finish": "Cold Foil", "id": "ros000_577711_coldfoil", "image": "x", "name": "Will of Arcana", "number": "ROS000", "rarity": "Fabled", "setCode": "ROS", "variant": "Cold Foil"},
		{"externalLinks": {"fabId": "ROS000", "tcgPlayerId": 578820}, "fabId": "ROS000", "finish": "Rainbow Foil", "id": "ros000_578820_rainbowfoil", "image": "x", "name": "Will of Arcana", "number": "ROS000", "rarity": "Fabled", "setCode": "ROS", "variant": "Rainbow Foil"},
		{"externalLinks": {"tcgPlayerId": 618220}, "finish": "Normal", "id": "618220", "image": "x", "name": "The Hunted Art Card", "rarity": "Token", "setCode": "TH"},
		{"externalLinks": {"tcgPlayerId": 618221}, "finish": "Cold Foil", "id": "618221_coldfoil", "image": "x", "name": "The Hunted Art Card", "rarity": "Token", "setCode": "TH", "variant": "Cold Foil"},
		{"externalLinks": {"tcgPlayerId": 711375}, "finish": "Normal", "id": "mpa004_711375", "image": "x", "name": "Uzuri", "number": "MPA004", "rarity": "Marvel", "setCode": "MPA"},
		{"externalLinks": {"tcgPlayerId": 711376}, "finish": "Cold Foil", "id": "mpa004_711376_coldfoil", "image": "x", "name": "Uzuri", "number": "MPA004", "promoTypes": ["marvel"], "rarity": "Basic", "setCode": "MPA", "variant": "Marvel"},
		{"externalLinks": {"tcgPlayerId": 275748}, "finish": "Cold Foil", "id": "upr043_275748_coldfoil", "image": "x", "name": "Ash // Aether Ashwing", "number": "UPR043", "rarity": "Common", "setCode": "UPR", "variant": "Cold Foil"},
		{"externalLinks": {"tcgPlayerId": 275749}, "finish": "Cold Foil", "id": "upr043_275749_coldfoil", "image": "x", "name": "Ash // Aether Ashwing", "number": "UPR043", "rarity": "Marvel", "setCode": "UPR", "variant": "Marvel"},
		{"externalLinks": {"tcgPlayerId": 274330}, "finish": "Normal", "id": "upr008_274330", "image": "x", "name": "Invoke Dominia // Dominia", "number": "UPR008", "rarity": "Majestic", "setCode": "UPR"},
		{"externalLinks": {"tcgPlayerId": 274331}, "finish": "Cold Foil", "id": "upr008_274331_coldfoil", "image": "x", "name": "Invoke Dominia // Dominia", "number": "UPR008", "rarity": "Marvel", "setCode": "UPR", "variant": "Marvel"},
		{"externalLinks": {"fabId": "OMN203", "tcgPlayerId": 682858}, "fabId": "OMN203", "finish": "Cold Foil", "id": "omn203_682858_coldfoil", "image": "x", "name": "Lightning Flow", "number": "OMN203", "rarity": "Basic", "setCode": "UPR", "variant": "C", "watermark": "c"},
		{"externalLinks": {"fabId": "OMN203", "tcgPlayerId": 682884}, "fabId": "OMN203", "finish": "Cold Foil", "id": "omn203_682884_coldfoil", "image": "x", "name": "Lightning Flow", "number": "OMN203", "rarity": "Basic", "setCode": "UPR", "variant": "A", "watermark": "a"}
	]
}`

// TestFinishTwins pins that a product's finish twins are told apart by the
// finish: the one the wording names, else the one the flag says. Each pair
// aliased, which drops both listings, once the loader stopped labelling a
// printing with the finish its variant restates.
func TestFinishTwins(t *testing.T) {
	b, err := Load(strings.NewReader(finishTwinsFixture))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc  string
		in    mtgmatcher.InputCard
		want  string
		alias bool
	}{
		{
			desc: "the treatment the wording names picks its twin",
			in:   mtgmatcher.InputCard{Name: "Will of Arcana", Variation: "ROS000 Cold Foil", Edition: "Rosetta"},
			want: "ros000_577711_coldfoil",
		},
		{
			desc: "and the other treatment the other",
			in:   mtgmatcher.InputCard{Name: "Will of Arcana", Variation: "ROS000 Rainbow Foil", Edition: "Rosetta"},
			want: "ros000_578820_rainbowfoil",
		},
		{
			desc:  "two foil twins and no treatment named is still a tie",
			in:    mtgmatcher.InputCard{Name: "Will of Arcana", Variation: "ROS000", Edition: "Rosetta", Foil: true},
			alias: true,
		},
		{
			desc: "a wording naming no finish prices the plain copy",
			in:   mtgmatcher.InputCard{Name: "The Hunted Art Card", Edition: "The Hunted"},
			want: "618220",
		},
		{
			desc: "and the flag alone reaches the foil one",
			in:   mtgmatcher.InputCard{Name: "The Hunted Art Card", Edition: "The Hunted", Foil: true},
			want: "618221_coldfoil",
		},
		{
			desc: "the treatment in the wording reaches it too",
			in:   mtgmatcher.InputCard{Name: "The Hunted Art Card", Variation: "Cold Foil", Edition: "The Hunted"},
			want: "618221_coldfoil",
		},
		{
			// The catalog files the plain Uzuri at the marvel rarity, so
			// both twins wear the label and the marvel's cold foil is
			// what tells them apart.
			desc: "a marvel names the cold foil it is printed in",
			in:   mtgmatcher.InputCard{Name: "Uzuri (Marvel)", Variation: "MPA004", Edition: "Mastery Pack Assassin"},
			want: "mpa004_711376_coldfoil",
		},
		{
			desc: "and the plain copy answers the plain wording",
			in:   mtgmatcher.InputCard{Name: "Uzuri", Variation: "MPA004", Edition: "Mastery Pack Assassin"},
			want: "mpa004_711375",
		},
		{
			desc: "a label the wording describes still outranks the finish",
			in:   mtgmatcher.InputCard{Name: "Ash // Aether Ashwing (Marvel)", Variation: "UPR043 Cold Foil", Foil: true},
			want: "upr043_275749_coldfoil",
		},
		{
			desc: "and the plain cold foil answers the treatment alone",
			in:   mtgmatcher.InputCard{Name: "Ash // Aether Ashwing", Variation: "UPR043 Cold Foil", Foil: true},
			want: "upr043_275748_coldfoil",
		},
		{
			desc: "a marvel beside a plain copy of another rarity is its label's",
			in:   mtgmatcher.InputCard{Name: "Invoke Dominia // Dominia (Marvel)", Variation: "UPR008", Edition: "Uprising"},
			want: "upr008_274331_coldfoil",
		},
		{
			desc: "the artwork letter still picks among cold foils",
			in:   mtgmatcher.InputCard{Name: "Lightning Flow (A)", Variation: "OMN203 Cold Foil", Foil: true},
			want: "omn203_682884_coldfoil",
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			got, err := b.Match(&in)
			if tt.alias {
				var alias *mtgmatcher.AliasingError
				if !errors.As(err, &alias) {
					t.Fatalf("Match(%v) = %q, %v, want an aliasing error", tt.in, got, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("Match(%v): %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("Match(%v) = %s, want %s", tt.in, got, tt.want)
			}
		})
	}
}
