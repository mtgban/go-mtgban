package magic

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

func TestSplitVariantsKeepsParenthesizedNames(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"Erase (Not the Urza's Legacy One)", []string{"Erase (Not the Urza's Legacy One)"}},
		{"B.F.M. (Big Furry Monster) (Left)", []string{"B.F.M. (Big Furry Monster)", "Left"}},
		{"Dwight, Assistant (to the) King (Foil)", []string{"Dwight, Assistant (to the) King", "Foil"}},
	}
	for _, tt := range tests {
		got := matcher.SplitVariants(tt.in)
		if !slices.Equal(got, tt.want) {
			t.Errorf("SplitVariants(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
