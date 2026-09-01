package abugames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestForeignWhiteBorder pins the listing naming a white-bordered Fourth
// Edition in a language it never had. The catalog holds 4ED in English and
// white and 4BB in Japanese and black, and nothing else, so this answered
// with the black-bordered card and put its price beside that one's.
func TestForeignWhiteBorder(t *testing.T) {
	card := ABUCard{DisplayTitle: "Sylvan Library (WB)", Edition: "4th Edition", Number: "273",
		Title: "Non-English - Japanese - White Bordered", Language: []string{"Japanese"}}
	in, err := preprocess(&card)
	if err == nil {
		_, err = mtgmatcher.Match(in)
	}
	if err == nil {
		t.Errorf("preprocess(%q) = %v, want a refusal", card.DisplayTitle, in)
	}
}

// TestNumberMarks pins the printings a set marks apart with something other
// than a dagger. Portal drops the reminder text from its second printing and
// files it at 69d, and reading only a dagger left both listings on 69.
func TestNumberMarks(t *testing.T) {
	for _, test := range []struct {
		desc    string
		card    ABUCard
		wantSet string
		wantNum string
	}{
		{"the printing that keeps its reminder text", ABUCard{
			DisplayTitle: "Storm Crow (B - Includes reminder text)", Edition: "Portal", Number: "69"}, "POR", "69d"},
		{"and the one that drops it", ABUCard{
			DisplayTitle: "Storm Crow (A - No reminder text)", Edition: "Portal", Number: "69"}, "POR", "69"},
		{"the flag written without its space", ABUCard{
			DisplayTitle: "Island (261) -FOIL", Edition: "Ravnica Allegiance", Number: "261"}, "RNA", "261"},
		{"the black-bordered Fourth Edition, which the catalog files as Japanese", ABUCard{
			DisplayTitle: "Sylvan Library (BB)", Edition: "4th Edition", Number: "273",
			Title: "Non-English - Japanese - BB", Language: []string{"Japanese"}}, "4BB", "273"},
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
