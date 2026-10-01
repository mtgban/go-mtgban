package gundam

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
		{"GD01-001", "1"},
		{"GD01-100", "100"},
		{"ST01-011", "11"},
		{"T-025", "25"},
	} {
		got := Rules{}.PlainNumber(tt.number)
		if got != tt.want {
			t.Errorf("PlainNumber(%q) = %q, want %q", tt.number, got, tt.want)
		}
	}
}
