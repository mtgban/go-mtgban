package pokemon

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestCardMarks pins the marks an old card wears against the way the catalog
// keeps each. A storefront prints the symbol; the catalog writes the star and
// the prism star into the name it files, and files the delta species beside
// the name as the variant the printing carries. A name published with the
// symbol reached no card at all, and the listing went unpriced - $3,899 of
// Treecko among them.
func TestCardMarks(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc    string
		in      mtgmatcher.InputCard
		wantSet string
		wantNum string
	}{
		{
			desc:    "the delta species reaches its printing beside the number",
			in:      mtgmatcher.InputCard{Name: "Snorlax δ", Edition: "EX Dragon Frontiers", Variation: "10/101"},
			wantSet: "DF", wantNum: "10",
		},
		{
			// The catalog files this mark beside the name rather than in
			// it, so reading it as the variant reaches the printing the
			// number would otherwise have to: dropping it answers with the
			// plain card standing at another number in the same set.
			desc:    "the mark alone reaches the delta printing, with no number to help",
			in:      mtgmatcher.InputCard{Name: "Blastoise δ", Edition: "EX Crystal Guardians"},
			wantSet: "CG", wantNum: "2",
		},
		{
			desc:    "and the plain name reaches the plain printing beside it",
			in:      mtgmatcher.InputCard{Name: "Blastoise", Edition: "EX Crystal Guardians"},
			wantSet: "CG", wantNum: "14",
		},
		{
			desc:    "and the same name without it still reaches the printing",
			in:      mtgmatcher.InputCard{Name: "Snorlax", Edition: "EX Dragon Frontiers", Variation: "10/101"},
			wantSet: "DF", wantNum: "10",
		},
		{
			desc:    "the star is a word where the catalog writes it out",
			in:      mtgmatcher.InputCard{Name: "Treecko ★", Edition: "Team Rocket Returns", Variation: "109/109"},
			wantSet: "RR-1428", wantNum: "109",
		},
		{
			desc:    "the prism star likewise",
			in:      mtgmatcher.InputCard{Name: "Giratina ◇", Edition: "SM - Ultra Prism", Variation: "58/156"},
			wantSet: "SM05", wantNum: "58",
		},
		{
			desc:    "a name carrying no mark is left as it was",
			in:      mtgmatcher.InputCard{Name: "Umbreon", Edition: "EX Delta Species", Variation: "17/113"},
			wantSet: "DS", wantNum: "17",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("Match(%v) = %s|%s, want %s|%s", tt.in, co.SetCode, co.Number, tt.wantSet, tt.wantNum)
			}
		})
	}
}
