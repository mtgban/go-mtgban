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
		if got, _ := splitParens(tt.title); got != tt.want {
			t.Errorf("splitParens(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}
