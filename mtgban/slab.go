package mtgban

import (
	"math"
	"slices"
	"strconv"
)

// Slab is a grading service's label on one copy of a card, as the store
// lists it. Grader and Score are empty where the store does not give them;
// an entry with no Slab is a raw copy. See docs/adr/0006-graded-slabs.md.
type Slab struct {
	// Grader is one of the services in graders.
	Grader string `json:"grader,omitempty"`

	// Score is from 1 to 10, in half steps, or zero when the store gives
	// none.
	Score float64 `json:"score,omitempty"`

	// Label is the rest of what the label says, kept as the store prints it:
	// "Pristine", "Black Label", or the whole wording of a slab the scraper
	// could not read.
	Label string `json:"label,omitempty"`
}

// graders are the grading services a Slab may name.
var graders = []string{"PSA", "BGS", "CGC", "TAG"}

// UnreadSlabFormat is the line a scraper logs, with the store's wording, for
// a slab it publishes as NM because it could not read it.
const UnreadSlabFormat = "unsupported slab %q, published as NM"

// UnreadSlab is the slab of a copy whose label a scraper could not read,
// keeping the store's wording, to be published as NM and logged with
// UnreadSlabFormat. A grader outside the supported ones is left out.
func UnreadSlab(grader, wording string) *Slab {
	if !slices.Contains(graders, grader) {
		grader = ""
	}
	return &Slab{Grader: grader, Label: wording}
}

// Fold reads a slab as a condition, for the singles view of it: PSA 9 is NM.
// A half point reads as the whole point below it; a slab with no score
// folds only by a label slabLabels names for the grader. It reports false for
// anything else, which the scraper publishes as an UnreadSlab.
func (s Slab) Fold() (Condition, bool) {
	key := s.Label
	if s.Score > 0 {
		key = strconv.Itoa(int(s.Score))
	}
	return slabGrade(s.Grader, key)
}

// sameSlab reports whether two entries carry the same slab, or are both raw.
func sameSlab(a, b *Slab) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// valid reports whether the slab names a supported grader, or none, and a
// score in range, or none.
func (s Slab) valid() bool {
	if s.Grader != "" && !slices.Contains(graders, s.Grader) {
		return false
	}
	if s.Score == 0 {
		return true
	}
	half := s.Score * 2
	return s.Score >= 1 && s.Score <= 10 && half == math.Trunc(half)
}
