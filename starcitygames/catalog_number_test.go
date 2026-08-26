package starcitygames

import (
	"slices"
	"testing"
)

// TestFabNumbers pins the collector numbers a Flesh and Blood sku is read as.
// The set segment is the datastore's own number prefix, and dropping it is what
// made every deck reprint alias: "005" is every set's fifth card, so a Bravo
// Hero Deck single competed with the Tales of Aria one, and the loser of that
// tie was decided by nothing at all. The bare number stays last so the listings
// no prefix reaches keep resolving exactly as they did.
func TestFabNumbers(t *testing.T) {
	for _, tt := range []struct {
		name string
		sku  string
		want []string
	}{
		{"plain number takes the set segment", "SGL-FAB-BVO-005-ENN",
			[]string{"BVO005", "005"}},
		{"1st edition marker is trimmed after the code it decorates", "SGL-FAB-ARC1-036-ENR",
			[]string{"ARC1036", "ARC036", "036"}},
		{"unlimited marker likewise", "SGL-FAB-ARCU-036-ENN",
			[]string{"ARCU036", "ARC036", "036"}},
		{"two-digit marker is trimmed whole", "SGL-FAB-EVR12-021-ENN",
			[]string{"EVR12021", "EVR021", "021"}},
		{"a code that merely ends in a marker letter keeps it", "SGL-FAB-KSU-001-ENN",
			[]string{"KSU001", "001"}},
		{"a code that merely ends in a marker letter, three letters long", "SGL-FAB-UZU-004-ENN",
			[]string{"UZU004", "004"}},
		{"a number segment carrying its own code ignores the shelf it sits on",
			"SGL-FAB-PRM-HER_022-ENR", []string{"HER022", "HER 022"}},
		{"a fused pair joins the way the datastore spells it", "SGL-FAB-ELE1-111_032-ENN",
			[]string{"ELE1111 // ELE1032", "ELE111 // ELE032", "111 032"}},
		{"a promo fused pair takes its own code", "SGL-FAB-PRM-LGS_127_128-ENC",
			[]string{"LGS127 // LGS128", "LGS 127 128"}},
		{"the variant letter rides along", "SGL-FAB-MON1-155b-ENC",
			[]string{"MON1155b", "MON155b", "155b"}},
		{"a lettered part stays the wording it is", "SGL-FAB-AGB-019_CC-ENN",
			[]string{"AGB019 CC", "019 CC"}},
		// A sku short of a number segment has nothing to prefix, and one
		// whose number is all letters names no number at all.
		{"a sealed sku has no number to build", "SLD-FAB-BBX-WTR-EN-UNL",
			[]string{"WTR"}},
		{"a truncated sku falls back to nothing", "SGL-FAB-BVO",
			[]string{""}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := fabNumbers(tt.sku); !slices.Equal(got, tt.want) {
				t.Errorf("fabNumbers(%q) = %q, want %q", tt.sku, got, tt.want)
			}
		})
	}
}

// TestFabSingleNumbered pins what tells a product named by both its faces
// apart from a genuine double-sided card: the sku's number segment. One
// number is a printing plus the token on its back, a pair is two printings.
func TestFabSingleNumbered(t *testing.T) {
	for _, tt := range []struct {
		name string
		sku  string
		want bool
	}{
		{"a plain number is one number", "SGL-FAB-NUU-026-ENN", true},
		{"a number behind its own code is still one", "SGL-FAB-PRM-FAB_233-ENR", true},
		{"the variant letter does not make a second", "SGL-FAB-MON1-155b-ENC", true},
		{"a lettered part is wording, not a number", "SGL-FAB-AGB-019_CC-ENN", true},
		{"a fused pair holds two", "SGL-FAB-OMN-048_047-ENN", false},
		{"a lettered number beside a second one holds two", "SGL-FAB-MST-158c_047-ENN", false},
		{"a coded pair holds two", "SGL-FAB-PRM-LGS_127_128-ENC", false},
		{"an all-letters segment names no number", "SGL-FAB-AMX-H01-ENN", false},
		{"a truncated sku names none either", "SGL-FAB-BVO", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := fabSingleNumbered(tt.sku)
			if got != tt.want {
				t.Errorf("fabSingleNumbered(%q) = %v, want %v", tt.sku, got, tt.want)
			}
		})
	}
}
