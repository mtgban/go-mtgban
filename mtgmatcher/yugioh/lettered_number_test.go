package yugioh

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// letteredNumberFixture mirrors Legendary Decks II, whose cards are numbered
// by the deck they came in - ENJ26 beside ENK22 - and the Championship Prize
// Cards, whose prize sheet numbers open with a P. Cardtrader publishes those
// tails bare, as "J26" and "P03". The Secrets of Eternity pair is the silent
// case: SECE-ENS03 shares its digits with SECE-EN013's neighbour, so a
// wording naming no number is answered by whichever printing the tiering
// reaches first.
const letteredNumberFixture = `{
	"game": "yugioh",
	"sets": {
		"LDK2": {"name": "Legendary Decks II", "releaseDate": "2016-10-07"},
		"25YC": {"name": "Championship Prize Cards 2025", "releaseDate": "2025-01-01"},
		"SECE": {"name": "Secrets of Eternity", "releaseDate": "2015-01-16"}
	},
	"cards": [
		{"id": "ldk2-enj26_123593_unl", "name": "Polymerization", "number": "LDK2-ENJ26", "setCode": "LDK2", "rarity": "Common", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 123593}},
		{"id": "ldk2-enk22_123546_unl", "name": "Polymerization", "number": "LDK2-ENK22", "setCode": "LDK2", "rarity": "Common", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 123546}},
		{"id": "25yc-enp03_637084_lim", "name": "Blackwing - Gale the Whirlwind", "number": "25YC-ENP03", "setCode": "25YC", "rarity": "Rare", "finish": "Limited", "image": "x", "externalLinks": {"tcgPlayerId": 637084}},
		{"id": "sece-en013_100013_1e", "name": "Infernoid Antra", "number": "SECE-EN013", "setCode": "SECE", "rarity": "Common", "finish": "1st Edition", "image": "x", "externalLinks": {"tcgPlayerId": 100013}},
		{"id": "sece-ens03_100003_1e", "name": "Infernoid Antra", "number": "SECE-ENS03", "setCode": "SECE", "rarity": "Super Rare", "finish": "1st Edition", "image": "x", "externalLinks": {"tcgPlayerId": 100003}}
	]
}`

// TestLetterLedNumberRead pins that a bare letter-led collector number is
// read as a number. Left unread, the deck letter never reaches the filter:
// every printing of the name in the set survives, and the listing either
// aliases or is answered with a same-named stranger.
func TestLetterLedNumberRead(t *testing.T) {
	b, err := Load(strings.NewReader(letteredNumberFixture))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{
			desc: "the deck letter picks its own printing",
			in: mtgmatcher.InputCard{Name: "Polymerization", Variation: "J26 Common",
				Edition: "Legendary Decks II"},
			want: "ldk2-enj26_123593_unl",
		},
		{
			desc: "and so does the other deck's",
			in: mtgmatcher.InputCard{Name: "Polymerization", Variation: "K22 Common",
				Edition: "Legendary Decks II"},
			want: "ldk2-enk22_123546_unl",
		},
		{
			desc: "a prize sheet number reads the same way, tail and all",
			in: mtgmatcher.InputCard{Name: "Blackwing - Gale the Whirlwind",
				Variation: "P03 Rare | YCS Stamp", Edition: "Championship Prize Cards 2025"},
			want: "25yc-enp03_637084_lim",
		},
		{
			desc: "the rarity sheet's number is no longer answered by its neighbour",
			in: mtgmatcher.InputCard{Name: "Infernoid Antra", Variation: "S03 Super Rare",
				Edition: "Secrets of Eternity"},
			want: "sece-ens03_100003_1e",
		},
		{
			desc: "a full number keeps its precedence over any letter-led field",
			in: mtgmatcher.InputCard{Name: "Polymerization", Variation: "LDK2-ENK22 Common",
				Edition: "Legendary Decks II"},
			want: "ldk2-enk22_123546_unl",
		},
		{
			desc: "and so does a plain digit run",
			in: mtgmatcher.InputCard{Name: "Infernoid Antra", Variation: "013 Common",
				Edition: "Secrets of Eternity"},
			want: "sece-en013_100013_1e",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%q, %q) = %v", tt.in.Name, tt.in.Variation, err)
			}
			if id != tt.want {
				t.Errorf("Match(%q, %q) = %q, want %q", tt.in.Name, tt.in.Variation, id, tt.want)
			}
		})
	}
}

// TestLabelWordIsNotANumber pins the shape guard: a variation word is only
// read as a letter-led number when it is one letter followed by digits, so
// the label words and years a wording is otherwise made of stay wording.
func TestLabelWordIsNotANumber(t *testing.T) {
	for _, tt := range []struct{ variation, want string }{
		{"J26 Common", "J26"},
		{"P03 Rare | YCS Stamp", "P03"},
		{"Common", ""},
		{"2012 Pre-registration", ""},
		{"Secret Rare", ""},
		{"AB12 Common", ""},
	} {
		if got := extractNumber(tt.variation); got != tt.want {
			t.Errorf("extractNumber(%q) = %q, want %q", tt.variation, got, tt.want)
		}
	}
}
