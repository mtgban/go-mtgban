package abugames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestVariantLetter pins the printing a listing reaches when a set prints one
// card several times over and this storefront tells them apart by a letter.
// The letter is the whole identity - the words beside it are the storefront's
// own name for the art - and the collector number, appended last, buried it.
func TestVariantLetter(t *testing.T) {
	for _, test := range []struct {
		desc    string
		card    ABUCard
		wantSet string
		wantNum string
	}{
		{"a letter the catalog spells the same way", ABUCard{
			DisplayTitle: "Knight of the Kitchen Sink (c Pro: Loose Lips)", Edition: "Unstable", Number: "12"}, "UST", "12c"},
		{"the last of six", ABUCard{
			DisplayTitle: "Everythingamajig (f Scry)", Edition: "Unstable", Number: "147"}, "UST", "147f"},
		{"a letter the storefront writes with a dash", ABUCard{
			DisplayTitle: "Secret Base (e Crossbreed Labs)", Edition: "Unstable", Number: "165"}, "UST", "165e"},
		{"a set whose own numbering the letters were scrambled against", ABUCard{
			DisplayTitle: "Strip Mine (c No Horizon)", Edition: "Antiquities", Number: "82"}, "ATQ", "82c"},
		{"and its sibling", ABUCard{
			DisplayTitle: "Strip Mine (a Even)", Edition: "Antiquities", Number: "82"}, "ATQ", "82a"},
		{"a card the catalog files under a dagger", ABUCard{
			DisplayTitle: "Sudden Setback (b - Black Bottle)", Edition: "Murders at Karlov Manor", Number: "72"}, "MKM", "72†"},
		{"the plain half of that pair", ABUCard{
			DisplayTitle: "Sudden Setback (a - Green Bottle)", Edition: "Murders at Karlov Manor", Number: "72"}, "MKM", "72"},
		{"a set where the words are the identity keeps them", ABUCard{
			DisplayTitle: "Urza's Mine (d Tower)", Edition: "Chronicles", Number: "114d",
			Title: "Non-English - Japanese", Language: []string{"Japanese"}}, "BCHR", "114d"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := test.card
			in, err := preprocess(&card)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", card.DisplayTitle, err)
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

// TestVariantLetterCase pins that a basic land's capital letter and this
// lower-case one never answer for each other, and that a word opening a
// variation is not read as one.
func TestVariantLetterCase(t *testing.T) {
	for _, v := range []string{"a Dark", "b - Green Lamp", "f Scry"} {
		if variantLetter(v) == "" {
			t.Errorf("variantLetter(%q) = %q, want a letter", v, variantLetter(v))
		}
	}
	for _, v := range []string{"", "A Waterfall", "Alternate Art", "Prerelease", "b", "buy-a-box"} {
		if got := variantLetter(v); got != "" {
			t.Errorf("variantLetter(%q) = %q, want empty", v, got)
		}
	}
}
