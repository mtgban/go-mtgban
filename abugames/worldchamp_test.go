package abugames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestWorldChampSideboard pins the deck a World Championship listing reaches
// when the player named held the card in their sideboard alone. The listing
// says only who played it, and reading that silence as "not the sideboard"
// put Leon Lindback's City of Brass on Eric Tam's copy.
func TestWorldChampSideboard(t *testing.T) {
	for _, test := range []struct {
		desc    string
		title   string
		wantNum string
	}{
		{"a player who held it only in the sideboard", "City of Brass (Leon Lindback - 1996)", "ll112sb"},
		{"one who held it in the deck", "City of Brass (Eric Tam - 1996)", "et112"},
		{"and another", "City of Brass (George Baxter - 1996)", "gb112"},
		{"and a third", "City of Brass (Shawn Regnier - 1996)", "shr112"},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := ABUCard{DisplayTitle: test.title, Edition: "Pro Tour Collector Set", Number: "60"}
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
			if co.SetCode != "PTC" || co.Number != test.wantNum {
				t.Errorf("Match(%q) = %s|%s, want PTC|%s", in, co.SetCode, co.Number, test.wantNum)
			}
		})
	}
}
