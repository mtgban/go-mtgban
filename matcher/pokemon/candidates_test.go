package pokemon

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

func TestCandidateSetsKeepsPromoShelf(t *testing.T) {
	b := &matcher.Backend{Sets: map[string]*matcher.Set{
		"PLS": {Name: "Plasma Storm"}, "PR": {Name: "Miscellaneous Promos"},
	}}
	for _, tt := range []struct {
		edition string
		want    []string
	}{
		{"Plasma Storm Promos", []string{"PR"}},
		{"Plasma Storm", []string{"PLS"}},
	} {
		in := &matcher.InputCard{Name: "Moltres EX", Edition: tt.edition, Variation: "014/135 Holo Promo"}
		got := (Rules{}).CandidateSets(b, in, []string{"PLS", "PR"})
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: candidates = %v, want %v", tt.edition, got, tt.want)
		}
	}
}
