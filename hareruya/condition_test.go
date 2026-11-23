package hareruya

import (
	"fmt"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// TestHaCondition pins that a slab grades by its score and carries the slab
// for the graded seller, and that one it cannot read is published as NM and
// logged. The labels are the storefront's own.
func TestHaCondition(t *testing.T) {
	for _, tt := range []struct {
		label string
		want  mtgban.Condition
		slab  *mtgban.Slab
	}{
		{"NM", mtgban.NM, nil},
		{"EX", mtgban.SP, nil},
		{"PSA10", mtgban.NM, &mtgban.Slab{Grader: "PSA", Score: 10}},
		{"PSA5", mtgban.SP, &mtgban.Slab{Grader: "PSA", Score: 5}},
		{"BGS9.5", mtgban.NM, &mtgban.Slab{Grader: "BGS", Score: 9.5}},
		{"BGS8.5", mtgban.SP, &mtgban.Slab{Grader: "BGS", Score: 8.5}},
		{"BGS6", mtgban.MP, &mtgban.Slab{Grader: "BGS", Score: 6}},
		{"CGC7.5", mtgban.SP, &mtgban.Slab{Grader: "CGC", Score: 7.5}},
		{"PSA 10.0", mtgban.NM, &mtgban.Slab{Grader: "PSA", Score: 10}},
		{"PSA11", mtgban.NM, &mtgban.Slab{Grader: "PSA", Label: "PSA11"}},
		{"BGS10.5", mtgban.NM, &mtgban.Slab{Grader: "BGS", Score: 10}},
		{"CGC8.25", mtgban.NM, &mtgban.Slab{Grader: "CGC", Label: "CGC8.25"}},
		{"CGC Pristine 10", mtgban.NM, &mtgban.Slab{Grader: "CGC", Score: 10, Label: "Pristine"}},
		{"CGC Pristine", mtgban.NM, &mtgban.Slab{Grader: "CGC", Score: 10, Label: "Pristine"}},
		{"BGS Black Label 10", mtgban.NM, &mtgban.Slab{Grader: "BGS", Score: 10, Label: "Black Label"}},
		{"BGS Pristine", mtgban.NM, &mtgban.Slab{Grader: "BGS", Label: "BGS Pristine"}},
		{"TAG10", mtgban.NM, &mtgban.Slab{Grader: "TAG", Label: "TAG10"}},
		{"SGC 9", mtgban.NM, &mtgban.Slab{Label: "SGC 9"}},
		{"ARS10", mtgban.NM, &mtgban.Slab{Label: "ARS10"}},
		{"PSA", mtgban.NM, &mtgban.Slab{Grader: "PSA", Label: "PSA"}},
	} {
		var logged []string
		ha := &Hareruya{logCallback: func(format string, a ...any) {
			logged = append(logged, fmt.Sprintf(format, a...))
		}}
		got, slab, err := ha.haCondition(tt.label)
		if err != nil || got != tt.want || !sameSlab(slab, tt.slab) {
			t.Errorf("haCondition(%q) = %q, %+v, %v; want %q, %+v", tt.label, got, slab, err, tt.want, tt.slab)
		}
		var want []string
		if tt.slab != nil && tt.slab.Label == tt.label {
			want = []string{"[HA] " + fmt.Sprintf(mtgban.UnreadSlabFormat, tt.label)}
		}
		if !slices.Equal(logged, want) {
			t.Errorf("haCondition(%q) logged %q, want %q", tt.label, logged, want)
		}
	}

	for _, label := range []string{"MP Signed", "Poor Ink", "Poor Water", "BGS9.5 ※ケースに欠けあり"} {
		if !reAltered.MatchString(label) {
			t.Errorf("%q is a damaged or altered copy, not a grade", label)
		}
	}
}

func sameSlab(a, b *mtgban.Slab) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
