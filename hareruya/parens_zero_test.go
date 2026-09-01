package hareruya

import "testing"

// TestSplitParensKeepsAZeroPower pins that the unpadding a collector number
// needs is not applied to a power and toughness. The two are told apart
// already - the slash in "0/4" is not the slash in a promo code's two
// spellings - but the strip ran over both, and "/4" names no printing.
func TestSplitParensKeepsAZeroPower(t *testing.T) {
	for _, tt := range []struct {
		title string
		want  string
	}{
		{"《Card》(0/4)", "0/4"},
		{"《Card》(0/1)", "0/1"},
		{"《Card》(1/1)", "1/1"},
		{"《Card》(10/100)", "10/100"},
		// A collector number still loses its padding.
		{"《Card》(007)", "7"},
		{"《Card》(0100)", "100"},
		// And a two-spelling promo code still keeps its first side.
		{"《Card》(PRM-001/ABC)", "PRM-001"},
	} {
		if got, _, _ := splitParens(tt.title); got != tt.want {
			t.Errorf("splitParens(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}

// TestSplitParensReadsTheTreatment pins which trailing group is read as the
// treatment a promo line is named by. The number is stated before the card
// name and the treatment after it, and only a wording the table answers
// counts: a group holding the card's colour is not one, and the prerelease
// is the promo a set numbers among its own cards.
func TestSplitParensReadsTheTreatment(t *testing.T) {
	for _, tt := range []struct {
		title string
		want  string
	}{
		{"【EN】(203)《Scute Swarm》(リセールプロモ)[ZNR-P] 緑", "リセールプロモ"},
		{"【EN】(127)《Vito》(リセールプロモ)[M21-P] 黒", "リセールプロモ"},
		// The prerelease is read from the title further down instead.
		{"【EN】(432)《Voja》(プレリリース)[MKM-P] 金", ""},
		// A colour is not a treatment.
		{"【EN】(041)《City of Brass》(緑)[SCH] 土地", ""},
		// Nor is a group the table has no answer for.
		{"【EN】(001)《Card》(nonsense)[SET] 青", ""},
		// A treatment with no number before it is the number itself.
		{"【EN】《Vexing Shusher》(発売記念)[SHM-P] 金", ""},
	} {
		if _, _, got := splitParens(tt.title); got != tt.want {
			t.Errorf("splitParens(%q) treatment = %q, want %q", tt.title, got, tt.want)
		}
	}
}
