package onepiece

import (
	"maps"
	"strings"
	"testing"
)

// TestSlugRunsPunctuation pins runs across punctuation words: read straight
// across one in the middle, and begun only by a word that spells something.
func TestSlugRunsPunctuation(t *testing.T) {
	for _, tt := range []struct {
		wording, slug string
		runs          map[int]int
		at            int
	}{
		{"OP05-119 - Alternate Art", "alternateart", map[int]int{2: 3}, 2},
		{"Alternate - Art", "alternateart", map[int]int{0: 2}, 0},
		{"- - Alternate Art - Alternate Art", "alternateart", map[int]int{2: 3, 5: 6}, 5},
		{"(Alternate Art) - (Red Parallel)", "parallel", map[int]int{4: 4}, 4},
		{"3rd Anniversary Tournament - (Leader)", "anniversarytournament", map[int]int{1: 2}, 1},
		{"- - -", "parallel", map[int]int{}, -1},
		{"- - -", "", map[int]int{}, -1},
	} {
		words := strings.Fields(tt.wording)
		runs := slugRuns(words, tt.slug)
		if !maps.Equal(runs, tt.runs) {
			t.Errorf("slugRuns(%q, %q) = %v, want %v", tt.wording, tt.slug, runs, tt.runs)
		}
		at := slugSaidAt(tt.wording, tt.slug)
		if at != tt.at {
			t.Errorf("slugSaidAt(%q, %q) = %d, want %d", tt.wording, tt.slug, at, tt.at)
		}
	}
}
