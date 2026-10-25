package fleshandblood

import "testing"

// TestPaddedNumberMatches pins the padding rule. The catalog writes three
// numbers a digit wider than the printing wears them - "JDG0077" for JDG077 -
// and a storefront writes whichever it was handed, so both have to reach the
// same printing and neither may reach its neighbours.
func TestPaddedNumberMatches(t *testing.T) {
	for _, tt := range []struct {
		input, full string
		want        bool
	}{
		{"JDG0077", "JDG0077", true},
		{"JDG077", "JDG0077", true},
		{"HER0160", "HER160", true},
		{"HER156", "HER0156", true},
		// Padding is all that may be folded away: a different number is
		// still a different printing.
		{"JDG0077", "JDG024", false},
		{"HER0160", "HER035", false},
		{"HER0156", "HER0160", false},
		// A run of zeros inside the number is not padding.
		{"WTR040", "WTR40", true},
		{"WTR104", "WTR14", false},
	} {
		if got := numberMatches(tt.input, tt.full); got != tt.want {
			t.Errorf("numberMatches(%q, %q) = %v, want %v", tt.input, tt.full, got, tt.want)
		}
	}
}

// TestPaddedNumberExtracted pins that a padded number is read as a number at
// all: left unread, the match falls back on the name and answers with some
// other printing of it.
func TestPaddedNumberExtracted(t *testing.T) {
	for _, tt := range []struct{ variation, want string }{
		{"JDG0077", "JDG0077"},
		{"HER0160", "HER0160"},
		{"HER156", "HER156"},
		{"WTR215", "WTR215"},
		{"WTR040 // WTR039", "WTR040 // WTR039"},
	} {
		if got := extractNumber(tt.variation); got != tt.want {
			t.Errorf("extractNumber(%q) = %q, want %q", tt.variation, got, tt.want)
		}
	}
}
