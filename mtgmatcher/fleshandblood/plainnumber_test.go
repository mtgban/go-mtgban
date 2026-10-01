package fleshandblood

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
		{"1HP085", "85"},
		{"1HP001", "1"},
		{"HER0156", "156"},
		{"WTR160", "160"},
		{"", ""},
	} {
		got := Rules{}.PlainNumber(tt.number)
		if got != tt.want {
			t.Errorf("PlainNumber(%q) = %q, want %q", tt.number, got, tt.want)
		}
	}
}
