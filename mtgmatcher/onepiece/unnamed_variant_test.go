package onepiece

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestWantsUnnamedVariant pins the wordings that say a listing is not the
// base printing without saying which one it is. The catalog labels the same
// printing "Parallel" on one card and "Alternate Art" on another, so the word
// cannot be matched against a label - it is read as a refusal of the base.
func TestWantsUnnamedVariant(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		variation string
		want      bool
	}{
		{"the storefront's own word for it", "EB01-013 Parallel", true},
		{"the catalog's word, spelled by a storefront", "OP02-030 Alternate Art", true},
		{"the short spelling", "OP05-007 alt art", true},
		{"however it is cased", "EB01-013 PARALLEL", true},
		{"a bare number says nothing of the sort", "EB01-013", false},
		{"nor does a number alone", "OP02-030", false},
		{"nor an unrelated wording", "OP16 Release Event", false},
		{"nor an empty variation", "", false},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := mtgmatcher.InputCard{Variation: tt.variation}
			if got := wantsUnnamedVariant(&in); got != tt.want {
				t.Errorf("wantsUnnamedVariant(%q) = %v, want %v", tt.variation, got, tt.want)
			}
		})
	}
}
