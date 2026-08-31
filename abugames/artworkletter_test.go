package abugames

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestArtworkLetter pins the letter a basic land's title names its artwork
// by. The storefront prices each artwork apart and gives them all one
// collector number, so the number buried the letter and every artwork of a
// land answered with the same printing.
func TestArtworkLetter(t *testing.T) {
	path := os.Getenv("ALLPRINTINGS5_PATH")
	if path == "" {
		t.Skip("Need ALLPRINTINGS5_PATH variable set to run this test")
	}
	if err := datastore.Load(path); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		desc    string
		title   string
		edition string
		number  string
		wantSet string
		wantNum string
	}{
		{"the letter alone, as three sets write it", "Swamp (C)", "Ice Age", "373", "ICE", "375"},
		{"and its sibling", "Swamp (A)", "Ice Age", "373", "ICE", "373"},
		{"a letter with the store's own name for the art", "Island (A) - Waterfall", "Portal", "200", "POR", "200"},
		{"the same land, another artwork", "Island (D) - Arches", "Portal", "200", "POR", "203"},
		{"a name the store writes without a dash", "Island (D) Sunset", "Mirage", "338", "MIR", "338"},
		{"and the artwork beside it", "Island (A) Palm Tree", "Mirage", "338", "MIR", "335"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := ABUCard{DisplayTitle: test.title, SimpleTitle: "", Edition: test.edition, Number: test.number}
			in, err := preprocess(&card)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", test.title, err)
			}
			id, err := mtgmatcher.Match(in)
			if err != nil {
				t.Fatalf("Match(%q) = %v", in, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != test.wantSet || co.Number != test.wantNum {
				t.Errorf("Match(%q) = %s|%s, want %s|%s", in, co.SetCode, co.Number, test.wantSet, test.wantNum)
			}
		})
	}
}

// TestArtworkLetterOnlyLands pins that nothing but a basic land is read this
// way: a bare letter on any other card is as likely to open a word the
// wording spends on something else.
func TestArtworkLetterOnlyLands(t *testing.T) {
	if got := artworkLetter("Storm Crow", "A Waterfall"); got != "" {
		t.Errorf("artworkLetter on a non-land = %q, want empty", got)
	}
	if got := artworkLetter("Swamp", "C"); got != "C" {
		t.Errorf("artworkLetter on a land = %q, want %q", got, "C")
	}
	if got := artworkLetter("Swamp", "Prerelease"); got != "" {
		t.Errorf("artworkLetter on a word = %q, want empty", got)
	}
}
