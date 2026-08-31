package magiccorner

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestBuylistNumber pins the number a buylist listing states outright. The
// store publishes one name for a card and prices its treatments separately,
// so the name alone puts every one of them on the plain printing: the three
// Aang listings below are bought at three prices and answered with one card.
func TestBuylistNumber(t *testing.T) {
	if len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("Need ALLPRINTINGS5_PATH variable set to run this test")
	}

	const name = "Aang, at the Crossroads // Aang, Destined Savior"
	tests := []struct {
		desc    string
		id      string
		number  int
		wantSet string
		wantNum string
	}{
		{"the plain printing", "TLA0203", 203, "TLA", "203"},
		{"the borderless one beside it", "TLA0304", 304, "TLA", "304"},
		{"and the third treatment", "TLA0346", 346, "TLA", "346"},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			card, err := preprocessBL(name, "Avatar: The Last Airbender", test.id, test.number)
			if err != nil {
				t.Fatal(err)
			}
			id, err := mtgmatcher.Match(card)
			if err != nil {
				t.Fatalf("Match(%q) = %v", card, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != test.wantSet || co.Number != test.wantNum {
				t.Errorf("Match(%q) = %s|%s, want %s|%s", card, co.SetCode, co.Number, test.wantSet, test.wantNum)
			}
		})
	}
}
