package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func TestValidatePrintingDescriptors(t *testing.T) {
	realDatastore(t)
	for _, tt := range []struct {
		name, set, number, variation string
		want                         bool
	}{
		{"Counterspell", "PF24", "1", "MagicFest", true},
		{"Counterspell", "PF24", "1", "MagicFest 5", false},
		{"Counterspell", "PF24", "1", "Unknown Treatment", false},
		{"Counterspell", "PF24", "1", "MagicFest Borderless", false},
		{"Counterspell", "PURL", "2", "NYCC 2024", true},
		{"Counterspell", "PURL", "2", "NYCC 2023", false},
		{"Voja, Jaws of the Conclave", "MKM", "432", "Prerelease", true},
		{"Voja, Jaws of the Conclave", "MKM", "432", "Promo Pack", false},
	} {
		t.Run(tt.name+tt.variation, func(t *testing.T) {
			cards := testBackend.MatchInSetNumber(tt.name, tt.set, tt.number)
			if len(cards) != 1 {
				t.Fatalf("fixture needs one printing, got %d", len(cards))
			}
			in := mtgmatcher.InputCard{Name: tt.name, Variation: tt.variation}
			if got := testBackend.ValidatePrinting(in, cards[0].UUID); got != tt.want {
				t.Fatalf("got %t, want %t", got, tt.want)
			}
		})
	}
}
