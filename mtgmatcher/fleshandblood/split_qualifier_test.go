package fleshandblood

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// splitQualifierFixture holds the two shapes a feed makes when it takes the
// parenthetical off a name before the lookup: a pitch color the catalog
// files no number beside, and a marvel whose number wears its label.
const splitQualifierFixture = `{
	"game": "fleshandblood",
	"sets": {
		"TH": {"name": "The Hunted", "releaseDate": "2025-06-06"},
		"ROS": {"name": "Rosetta", "releaseDate": "2025-03-28"},
		"FLR": {"name": "Blitz Deck: Rosetta - Florian", "releaseDate": "2025-03-28"}
	},
	"cards": [
		{"id": "hnt199_612679", "name": "To the Point (Red)", "number": "HNT199", "setCode": "TH", "rarity": "Rare", "finish": "Normal", "image": "x"},
		{"id": "hnt200_612680", "name": "To the Point (Yellow)", "number": "HNT200", "setCode": "TH", "rarity": "Rare", "finish": "Normal", "image": "x"},
		{"id": "hnt201_612681", "name": "To the Point (Blue)", "number": "HNT201", "setCode": "TH", "rarity": "Rare", "finish": "Normal", "image": "x"},
		{"id": "ros002-mv_564555_coldfoil", "name": "Florian (Marvel)", "number": "ROS002-MV", "setCode": "ROS", "rarity": "Marvel", "finish": "Cold Foil", "image": "x"},
		{"id": "flr001_577202_rainbowfoil", "name": "Florian", "number": "FLR001", "setCode": "FLR", "rarity": "Rare", "finish": "Rainbow Foil", "image": "x"}
	]
}`

// TestSplitOffQualifier pins that a listing whose qualifier was split into
// the wording answers exactly as the same listing spelled whole does.
//
// The pitch color is part of a Flesh and Blood name, and the marvel label
// names the printing the catalog numbers "ROS002-MV". A feed that writes
// either as a parenthetical has it taken off the name before the lookup,
// which left "To the Point" naming nothing and "Florian" naming the plain
// card of another set.
func TestSplitOffQualifier(t *testing.T) {
	b, err := Load(strings.NewReader(splitQualifierFixture))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		desc  string
		split mtgmatcher.InputCard
		whole mtgmatcher.InputCard
		want  string
	}{
		{
			desc:  "a pitch color the catalog files no number beside goes back on the name",
			split: mtgmatcher.InputCard{Name: "To the Point", Variation: "Red", Edition: "The Hunted"},
			whole: mtgmatcher.InputCard{Name: "To the Point (Red)", Edition: "The Hunted"},
			want:  "hnt199_612679",
		},
		{
			desc:  "and so does each of the colors beside it",
			split: mtgmatcher.InputCard{Name: "To the Point", Variation: "Blue", Edition: "The Hunted"},
			whole: mtgmatcher.InputCard{Name: "To the Point (Blue)", Edition: "The Hunted"},
			want:  "hnt201_612681",
		},
		{
			desc:  "a number written with its label reaches the marvel it names",
			split: mtgmatcher.InputCard{Name: "Florian", Variation: "ROS002-MV Marvel", Edition: "Rosetta"},
			whole: mtgmatcher.InputCard{Name: "Florian (Marvel)", Variation: "ROS002-MV", Edition: "Rosetta"},
			want:  "ros002-mv_564555_coldfoil",
		},
		{
			desc:  "and the plain card of the other set still answers its own listing",
			split: mtgmatcher.InputCard{Name: "Florian", Variation: "FLR001", Edition: "Blitz Deck: Rosetta - Florian"},
			whole: mtgmatcher.InputCard{Name: "Florian", Variation: "FLR001", Edition: "Blitz Deck: Rosetta - Florian"},
			want:  "flr001_577202_rainbowfoil",
		},
		{
			desc:  "a word no printing is named with picks nothing",
			split: mtgmatcher.InputCard{Name: "To the Point", Variation: "Marvel", Edition: "The Hunted"},
			whole: mtgmatcher.InputCard{Name: "To the Point", Variation: "Marvel", Edition: "The Hunted"},
			want:  "",
		},
	} {
		t.Run(test.desc, func(t *testing.T) {
			for label, in := range map[string]mtgmatcher.InputCard{"split": test.split, "whole": test.whole} {
				card := in
				uuid, err := b.Match(&card)
				if test.want == "" {
					if err == nil {
						t.Errorf("%s: Match = %q, want an error", label, uuid)
					}
					continue
				}
				if err != nil {
					t.Errorf("%s: Match = %v, want %q", label, err, test.want)
					continue
				}
				if uuid != test.want {
					t.Errorf("%s: Match = %q, want %q", label, uuid, test.want)
				}
			}
		})
	}
}
