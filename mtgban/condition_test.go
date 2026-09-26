package mtgban

import (
	"errors"
	"testing"
)

// TestParseCondition covers every word stores share for a grade, a few case
// and space variants, and the words a scraper must map itself because their
// grade depends on the store.
func TestParseCondition(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want Condition
	}{
		{"nm", NM}, {"near mint", NM}, {"mint", NM}, {"nm-mint", NM}, {"nm/m", NM},
		{"NEAR MINT", NM}, {" Near Mint ", NM},

		{"sp", SP}, {"lp", SP}, {"ex", SP}, {"very fine", SP},
		{"slightly played", SP}, {"lightly played", SP}, {"light play", SP},

		{"mp", MP}, {"gd", MP}, {"moderately played", MP}, {"moderate play", MP},
		{"medium play", MP}, {"used", MP}, {"played", MP}, {"fine", MP},

		{"hp", HP}, {"heavily played", HP}, {"heavy play", HP},

		{"po", PO}, {"d", PO}, {"dmg", PO}, {"damaged", PO}, {"poor", PO},
	} {
		got, err := ParseCondition(tt.in)
		if err != nil {
			t.Errorf("ParseCondition(%q) returned error: %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseCondition(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}

	for _, in := range []string{"", "EX+", "GEM-MT"} {
		_, err := ParseCondition(in)
		if !errors.Is(err, ErrInvalidCondition) {
			t.Errorf("ParseCondition(%q) error = %v, want ErrInvalidCondition", in, err)
		}
	}
}
