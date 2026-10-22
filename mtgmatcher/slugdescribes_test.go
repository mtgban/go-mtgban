package mtgmatcher

import (
	"strings"
	"testing"
)

// TestSlugDescribesPunctuation pins wordings carrying punctuation words: a
// run is read straight across one.
func TestSlugDescribesPunctuation(t *testing.T) {
	for _, tt := range []struct {
		wording, slug string
		want          bool
	}{
		{"OP05-119 - Alternate Art", "alternateart", true},
		{"Alternate - Art", "alternateart", true},
		{"(Alternate Art) - (Red Parallel)", "redparallel", true},
		{"Black & White", "blackwhite", true},
		{"Don!! Card (33) (Buggy)", "doncard", true},
		{"Premium Card Collection -Best Selection Vol. 6-", "bestselectionvol6", true},
		{"3-on-3 Cup", "3on3cup", true},
		{"- - -", "alternateart", false},
		{"- Foil -", "foil", true},
		{"- Foil -", "coldfoil", false},
		{"!! & -", "", false},
	} {
		got := SlugDescribes(tt.wording, tt.slug)
		if got != tt.want {
			t.Errorf("SlugDescribes(%q, %q) = %v, want %v", tt.wording, tt.slug, got, tt.want)
		}
	}
}

// BenchmarkSlugDescribesLongPunctuation is the fuzzed shape: an 8 KB wording
// of punctuation words, which no run may walk to the end.
func BenchmarkSlugDescribesLongPunctuation(b *testing.B) {
	wording := strings.Repeat("- ", 4096)
	for b.Loop() {
		SlugDescribes(wording, "alternateart")
	}
}
