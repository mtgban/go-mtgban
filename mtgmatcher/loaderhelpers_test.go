package mtgmatcher

import (
	"slices"
	"testing"
)

// TestPlainOrdinal pins the reduction the games numbering their cards
// behind a set code share: the ordinal, unpadded, and nothing for a number
// that carries none.
func TestPlainOrdinal(t *testing.T) {
	for _, tt := range []struct{ number, want string }{
		{"OP01-007", "7"},
		{"OP01-001a", "1"},
		{"WTR007", "7"},
		{"YS13-ENV07", "7"},
		{"GD01-001", "1"},
		{"P-041", "41"},
		{"000", "0"},
		{"LEADER", ""},
		{"DON", ""},
		{"", ""},
	} {
		if got := PlainOrdinal(tt.number); got != tt.want {
			t.Errorf("PlainOrdinal(%q) = %q, want %q", tt.number, got, tt.want)
		}
	}
	for _, tt := range []struct{ number, want string }{
		{"007", "7"}, {"0", "0"}, {"00", "0"}, {"", ""}, {"12", "12"},
	} {
		if got := CanonicalTail(tt.number); got != tt.want {
			t.Errorf("CanonicalTail(%q) = %q, want %q", tt.number, got, tt.want)
		}
	}
}

// TestProductKeyOf pins that a printing folds onto the product id it
// publishes and stands for itself where it publishes none.
func TestProductKeyOf(t *testing.T) {
	if got := ProductKeyOf(map[string]string{"tcgplayerProductId": "531486"}, "p-041_531486"); got != "531486" {
		t.Errorf("ProductKeyOf with an id = %q, want the id", got)
	}
	if got := ProductKeyOf(nil, "op01-003_ct277276"); got != "op01-003_ct277276" {
		t.Errorf("ProductKeyOf without an id = %q, want the uuid", got)
	}
	for _, tt := range []struct {
		color string
		want  []string
	}{
		{"", nil},
		{"Red", []string{"Red"}},
		{"Red/Green", []string{"Red", "Green"}},
		{"Red; Green", []string{"Red", "Green"}},
	} {
		if got := SplitColors(tt.color); !slices.Equal(got, tt.want) {
			t.Errorf("SplitColors(%q) = %v, want %v", tt.color, got, tt.want)
		}
	}
}
