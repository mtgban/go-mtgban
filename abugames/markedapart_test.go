package abugames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestMarkedApart pins the printing a listing reaches where a set files two
// of them under one number and tells them apart with a mark this storefront
// never carries. The number names both and so names neither, and the wording
// beside it - which is the only thing that knows - went unread.
func TestMarkedApart(t *testing.T) {
	for _, test := range []struct {
		desc    string
		card    ABUCard
		wantSet string
		wantNum string
	}{
		{"the light mana symbol, which only the wording names", ABUCard{
			DisplayTitle: "Erg Raiders (b Light)", Edition: "Arabian Nights", Number: "25"}, "ARN", "25†"},
		{"and the dark one beside it", ABUCard{
			DisplayTitle: "Erg Raiders (a Dark)", Edition: "Arabian Nights", Number: "25"}, "ARN", "25"},
		{"a printing told apart by its flavor text", ABUCard{
			DisplayTitle: "Anaconda (B - No flavor text)", Edition: "Portal", Number: "158"}, "POR", "158†"},
		{"and the one that has it", ABUCard{
			DisplayTitle: "Anaconda (A - Flavor text)", Edition: "Portal", Number: "158"}, "POR", "158"},
		{"a wording naming both printings keeps the number", ABUCard{
			DisplayTitle: "Grizzly Fate (The List)", Edition: "Judgment", Number: "JUD-119"}, "PLST", "JUD-119"},
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
