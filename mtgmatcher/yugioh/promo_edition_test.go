package yugioh

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// catchAllFixture mirrors the shape Cool Stuff Inc's buylist has: one card
// printed in three sets it files under a single "Promo" bucket, plus a fourth
// set whose own name carries the bucket's word. That fourth set is what makes
// the bucket harmful rather than merely useless - Match's edition heuristics
// land on it, so every listing in the bucket is answered with its printing,
// and a listing spelling its collector number in full is answered with
// nothing at all.
//
// The collector numbers, rarities and print run are the catalog's; the set
// names are the fixture's own, since only the codes the numbers open with are
// load-bearing here.
const catchAllFixture = `{
	"game": "yugioh",
	"sets": {
		"LART": {"name": "Legendary Collection Art Cards", "releaseDate": "2021-09-24"},
		"UBP1": {"name": "Ultimate Bonds Premium Pack", "releaseDate": "2018-05-04"},
		"TN23": {"name": "2023 Tin Quarter Century Edition", "releaseDate": "2023-09-15"},
		"DL09": {"name": "Duelist League Promo", "releaseDate": "2010-09-01", "type": "promo"},
		"LOB":  {"name": "Legend of Blue Eyes White Dragon", "releaseDate": "2002-03-08"},
		"LOB2": {"name": "Legend of Blue Eyes White Dragon 25th Anniversary", "releaseDate": "2023-02-24"}
	},
	"cards": [
		{"id": "lart-en004_158235_lim", "name": "Exodia the Forbidden One", "number": "LART-EN004", "setCode": "LART", "rarity": "Ultra Rare", "finish": "Limited", "image": "x", "externalLinks": {"tcgPlayerId": 158235}},
		{"id": "ubp1-en005_26541_lim", "name": "Exodia the Forbidden One", "number": "UBP1-EN005", "setCode": "UBP1", "rarity": "Secret Rare", "finish": "Limited", "image": "x", "externalLinks": {"tcgPlayerId": 26541}},
		{"id": "tn23-en002_514575_lim", "name": "Exodia the Forbidden One", "number": "TN23-EN002", "setCode": "TN23", "rarity": "Quarter Century Secret Rare", "finish": "Limited", "image": "x", "externalLinks": {"tcgPlayerId": 514575}},
		{"id": "dl09-en001_1_unl", "name": "Exodia the Forbidden One", "number": "DL09-EN001", "setCode": "DL09", "rarity": "Rare", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 1}},
		{"id": "lob-001_21792_1e", "name": "Blue-Eyes White Dragon", "number": "LOB-001", "setCode": "LOB", "rarity": "Ultra Rare", "finish": "1st Edition", "image": "x", "externalLinks": {"tcgPlayerId": 21792}},
		{"id": "lob-001_21793_unl", "name": "Blue-Eyes White Dragon", "number": "LOB-001", "setCode": "LOB2", "rarity": "Ultra Rare", "finish": "Unlimited", "image": "x", "externalLinks": {"tcgPlayerId": 21793}}
	]
}`

func catchAllBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	b, err := Load(strings.NewReader(catchAllFixture))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestNumberSetAnswersCatchAllEdition pins the redirection: a bucket naming no
// set is answered by the set code the collector number opens with, dashed or
// run together, and only when that set prints the card at that number.
func TestNumberSetAnswersCatchAllEdition(t *testing.T) {
	b := catchAllBackend(t)

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
	}{
		{
			desc: "the undashed number names its own set",
			in:   mtgmatcher.InputCard{Name: "Exodia the Forbidden One", Variation: "LARTEN004", Edition: "Promo"},
			want: "lart-en004_158235_lim",
		},
		{
			// The three rows differ in nothing but this number, and it is what
			// keeps a $170 Secret Rare buy price off the $22 Ultra Rare.
			desc: "the secret rare reaches its own set",
			in:   mtgmatcher.InputCard{Name: "Exodia the Forbidden One", Variation: "UBP1EN005", Edition: "Promo"},
			want: "ubp1-en005_26541_lim",
		},
		{
			desc: "the quarter century rare reaches its own set",
			in:   mtgmatcher.InputCard{Name: "Exodia the Forbidden One", Variation: "TN23EN002", Edition: "Promo"},
			want: "tn23-en002_514575_lim",
		},
		{
			desc: "the dashed spelling of the same number reads the same",
			in:   mtgmatcher.InputCard{Name: "Exodia the Forbidden One", Variation: "UBP1-EN005", Edition: "Promo"},
			want: "ubp1-en005_26541_lim",
		},
		{
			desc: "the rarity the note spells rides along unharmed",
			in:   mtgmatcher.InputCard{Name: "Exodia the Forbidden One", Variation: "TN23EN002 Quarter Century Rare", Edition: "Promo"},
			want: "tn23-en002_514575_lim",
		},
		{
			// The number is the storefront's own claim, and one pointing at a
			// set the card was never printed in says nothing about where the
			// listing belongs: the bucket keeps whatever it was reaching.
			desc: "a number naming a set without the card leaves the edition alone",
			in:   mtgmatcher.InputCard{Name: "Exodia the Forbidden One", Variation: "LOB001", Edition: "Promo"},
			want: "dl09-en001_1_unl",
		},
		{
			desc: "an edition that names a set is never overridden",
			in:   mtgmatcher.InputCard{Name: "Exodia the Forbidden One", Variation: "UBP1EN005", Edition: "Duelist League Promo"},
			want: "dl09-en001_1_unl",
		},
		{
			desc: "a decorated edition that names a set is not overridden either",
			in:   mtgmatcher.InputCard{Name: "Exodia the Forbidden One", Variation: "UBP1EN005", Edition: "Yu-Gi-Oh! Duelist League Promo Singles"},
			want: "dl09-en001_1_unl",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			in := test.in
			uuid, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match = %v, want %q", err, test.want)
			}
			if uuid != test.want {
				co, _ := b.GetUUID(uuid)
				t.Errorf("Match = %q (%v), want %q", uuid, co, test.want)
			}
		})
	}
}

// TestNumberSetNeedsABucket pins the other half of the contract, which the
// golden corpus states twice: a listing naming no edition at all keeps
// aliasing over every set the number reaches. A bucket is an edition a
// storefront wrote and no set answers, and there is nothing to correct in a
// listing that claimed nothing - Yu-Gi-Oh reprints one card under one exact
// code across sibling sets, so a number read as a set would hand back one of
// them where the whole pool is the honest answer.
func TestNumberSetNeedsABucket(t *testing.T) {
	b := catchAllBackend(t)

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
	}{
		{
			desc: "no edition at all keeps the whole pool",
			in:   mtgmatcher.InputCard{Name: "Blue-Eyes White Dragon", Variation: "LOB-001"},
		},
		{
			desc: "and so does one spelled with nothing but punctuation",
			in:   mtgmatcher.InputCard{Name: "Blue-Eyes White Dragon", Variation: "LOB-001", Edition: " - "},
		},
		{
			// The twins are reissued under their original's numbers, so the
			// code the number opens with names a set without telling which of
			// the two the listing is: a bucket that could have said did not.
			desc: "a bucket cannot pick between print-run twins either",
			in:   mtgmatcher.InputCard{Name: "Blue-Eyes White Dragon", Variation: "LOB-001", Edition: "Promo"},
		},
		{
			desc: "nor when the storefront drops the dash",
			in:   mtgmatcher.InputCard{Name: "Blue-Eyes White Dragon", Variation: "LOB001", Edition: "Promo"},
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			in := test.in
			uuid, err := b.Match(&in)
			if err == nil {
				co, _ := b.GetUUID(uuid)
				t.Fatalf("Match = %q (%v), want an aliasing error", uuid, co)
			}
			var alias *mtgmatcher.AliasingError
			if !errors.As(err, &alias) {
				t.Fatalf("Match = %v, want an aliasing error", err)
			}
		})
	}
}

// TestFieldSet pins what a variation field has to look like to name a set: a
// code with a collector number's tail behind it, and the longest such code
// when several open the field.
func TestFieldSet(t *testing.T) {
	b := catchAllBackend(t)

	tests := []struct {
		desc  string
		field string
		want  string
	}{
		{"the dashed code names its set", "UBP1-EN005", "UBP1"},
		{"so does the run-together spelling", "UBP1EN005", "UBP1"},
		{"the language infix is optional", "LART004", "LART"},
		{"a misprint tail rides behind the digits", "LOB-001K", "LOB"},
		{"a code no set carries names nothing", "ZZZZ-EN001", ""},
		{"a bare number names nothing", "004", ""},
		{"neither does a word", "Secret", ""},
		{"nor a code with no number behind it", "UBP1-EN", ""},
		{"nor a code whose tail is not a number", "LART004X09", ""},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			var got string
			if set := fieldSet(b, test.field); set != nil {
				got = set.Code
			}
			if got != test.want {
				t.Errorf("fieldSet(%q) = %q, want %q", test.field, got, test.want)
			}
		})
	}
}
