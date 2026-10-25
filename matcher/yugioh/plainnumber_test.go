package yugioh

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
		{"LOB-EN001", "1"},
		{"POTD-EN001", "1"},
		{"PSV-088", "88"},
		{"MP24-EN002", "2"},
		{"25LP-EN000", "0"},
		{"DL10-EN100", "100"},
		// The tail is not always a language: SE, SP and TK are print runs
		// of their own, and the ordinal behind them is still the ordinal.
		{"ABPF-ENSE1", "1"},
		// A letter behind the ordinal names the printing, not the number.
		{"BLAR-EN10K", "10"},
		{"", ""},
	} {
		got := Rules{}.PlainNumber(tt.number)
		if got != tt.want {
			t.Errorf("PlainNumber(%q) = %q, want %q", tt.number, got, tt.want)
		}
	}
}
