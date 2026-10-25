package onepiece

import (
	"testing"
)

// TestPlainNumber pins the shorthand this game's numbers reduce to. The whole
// number stays on Card.Number and is what names a printing; PlainNumber is
// the ordinal a person types, which is also what a storefront publishes.
func TestPlainNumber(t *testing.T) {
	for _, tt := range []struct {
		number, want string
	}{
		{"OP01-007", "7"},
		{"OP01-120", "120"},
		{"EB01-001", "1"},
		{"ST01-100", "100"},
		{"P-001", "1"},
		{"OP09-051", "51"},
		// DON is a word, not an ordinal, so it reduces to no ordinal.
		{"DON", ""},
		{"", ""},
	} {
		got := Rules{}.PlainNumber(tt.number)
		if got != tt.want {
			t.Errorf("PlainNumber(%q) = %q, want %q", tt.number, got, tt.want)
		}
	}
}
