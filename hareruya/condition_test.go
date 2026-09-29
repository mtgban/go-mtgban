package hareruya

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// TestHaCondition pins that a slab grades by its score and is marked for the
// graded seller. The labels are the storefront's own.
func TestHaCondition(t *testing.T) {
	for _, tt := range []struct {
		label  string
		want   mtgban.Condition
		graded bool
	}{
		{"NM", mtgban.NM, false},
		{"EX", mtgban.SP, false},
		{"PSA10", mtgban.NM, true},
		{"PSA5", mtgban.SP, true},
		{"BGS9.5", mtgban.NM, true},
		{"BGS8.5", mtgban.SP, true},
		{"BGS6", mtgban.MP, true},
		{"CGC7.5", mtgban.SP, true},
	} {
		got, graded, err := haCondition(tt.label)
		if err != nil || got != tt.want || graded != tt.graded {
			t.Errorf("haCondition(%q) = %q, %v, %v; want %q, %v", tt.label, got, graded, err, tt.want, tt.graded)
		}
	}

	_, _, err := haCondition("PSA11")
	if err == nil {
		t.Errorf("haCondition(%q) took a score no grader gives", "PSA11")
	}
}
