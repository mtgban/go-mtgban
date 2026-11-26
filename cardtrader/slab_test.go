package cardtrader

import (
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// The wordings are ones listings carry, some cut short, but for the last group and
// the digits of the certificate numbers.
func TestCTSlab(t *testing.T) {
	for _, tt := range []struct {
		description string
		grader      string
		score       float64
		label       string
		grade       mtgban.Condition
	}{
		// The tool's bracketed opening
		{"[PSA 10.0 Gem-Mint] PSA 10", "PSA", 10, "", mtgban.NM},
		{"[PSA 1.0]", "PSA", 1, "", mtgban.HP},
		{"[CGC 10.0 Pristine] CGC 10 Pristine", "CGC", 10, "Pristine", mtgban.NM},
		{"[CGC 10.0 - Reverse-Stamped-Holo] CGC 10 Stamped", "CGC", 10, "", mtgban.NM},
		{"[BGS 10.0 Pristine] BGS 12345678", "BGS", 10, "Pristine", mtgban.NM},
		{"[PSA] GEM MT 10", "PSA", 10, "", mtgban.NM},
		{"[US Edition - PSA 8.0]", "PSA", 8, "", mtgban.NM},
		{"[PSA 10.0] # PSA 10 GEM MINT", "PSA", 10, "", mtgban.NM},
		// A price in the opening is not the score
		{"[PSA 145] PSA 9", "PSA", 9, "", mtgban.NM},
		{"[PSA 49.99] Psa 7", "PSA", 7, "", mtgban.NM},

		// Written by hand, the score after the grader
		{"psa9", "PSA", 9, "", mtgban.NM},
		{"\nBeckett BGS 9,5", "BGS", 9.5, "", mtgban.NM},
		{"Cgc 7’5", "CGC", 7.5, "", mtgban.SP},
		{"CGC grade 9.5", "CGC", 9.5, "", mtgban.NM},
		{"GRADED CGC UNIVERSAL GRADE 9", "CGC", 9, "", mtgban.NM},
		{"GRADATO BGS VOTO 8", "BGS", 8, "", mtgban.SP},
		{"becket 9,5 ", "BGS", 9.5, "", mtgban.NM},
		{"bgs bekett 10", "BGS", 10, "", mtgban.NM},
		{"BECKETT BLACK LABEL 10 ", "BGS", 10, "Black Label", mtgban.NM},
		{"cgc prestine 10", "CGC", 10, "Pristine", mtgban.NM},
		{"bgs 9.9 - 9.5, 9.5, 9.5, 9", "BGS", 9.5, "", mtgban.NM},
		{"Graded card: PSA 9. Original photo", "PSA", 9, "", mtgban.NM},
		{"CGC 6..POP serie5", "CGC", 6, "", mtgban.SP},
		{"VINTAGE PSA 6 EXCELLENT-NEAR MINT 1ST EDITION", "PSA", 6, "", mtgban.SP},
		{"RARE 1ST EDITION VINTAGE PSA 9 MINT", "PSA", 9, "", mtgban.NM},
		{"MINT PSA 10 123456789", "PSA", 10, "", mtgban.NM},
		// The plus, quad and basic marks of the subgrades
		{"BGS 8.5 + (centering 9.5)", "BGS", 8.5, "", mtgban.SP},
		{"BGS 9.5++ (9.5, 9.5, 10, 10)", "BGS", 9.5, "", mtgban.NM},
		{"BGS 9.5Q++", "BGS", 9.5, "", mtgban.NM},
		{"BGS 9.5b+!!", "BGS", 9.5, "", mtgban.NM},
		{"BGS 9quad++ ( 9/9.5/10/9)", "BGS", 9, "", mtgban.NM},

		// The score ahead of the grader
		{"\t\nGradata 9 PSA", "PSA", 9, "", mtgban.NM},
		{"graded 8 by Becket", "BGS", 8, "", mtgban.SP},
		{"9.5 bgs", "BGS", 9.5, "", mtgban.NM},
		{"9.5 GEM MINT BECKETT", "BGS", 9.5, "", mtgban.NM},

		// No score to read, the grader kept; TAG has no grades to fold by
		{"grade 9 PSA non-retro foil", "PSA", 0, "grade 9 PSA non-retro foil", ""},
		{"CGC Certified", "CGC", 0, "CGC Certified", ""},
		{"[BGS]", "BGS", 0, "BGS", ""},
		{"PSA NM CERT 123456789", "PSA", 0, "PSA NM CERT 123456789", ""},
		{"PSA 10123456789", "PSA", 0, "PSA 10123456789", ""},
		{"AUTHENTIC ALTERED BGS 0012345678", "BGS", 0, "AUTHENTIC ALTERED BGS 0012345678", ""},
		{"GRADED CGC (undergraded) EXC-", "CGC", 0, "GRADED CGC (undergraded) EXC-", ""},
		{"CGC1O PRISTINE", "CGC", 0, "CGC1O PRISTINE", ""},
		{"[TAG 9.0]", "TAG", 0, "TAG 9.0", ""},
		// A grader outside the supported ones, whoever it compares itself to
		{"[GRAAD 9.5]", "", 0, "GRAAD 9.5", ""},
		{"GRAAD 10, PSA 10 equivalent", "", 0, "GRAAD 10, PSA 10 equivalent", ""},
		{"Graad 9.5 for PSA ", "", 0, "Graad 9.5 for PSA", ""},
		{"[AiGrad 8.5] BGS 8", "", 0, "AiGrad 8.5", ""},
		{"", "", 0, "Graded", ""},

		// Wordings no listing in the capture carries
		{"PSA 1st Edition 9", "PSA", 0, "PSA 1st Edition 9", ""},
		{"PSA EX-MT 6", "PSA", 6, "", mtgban.SP},
		{"PSA NM-MT+ 8.5", "PSA", 8.5, "", mtgban.NM},
		{"PSA #12345678", "PSA", 0, "PSA #12345678", ""},
		{"PSA 9 #12345678", "PSA", 9, "", mtgban.NM},
		{"PSA Authentic", "PSA", 0, "PSA Authentic", ""},
		{"CGC Pristine", "CGC", 0, "CGC Pristine", ""},
		{"BGS 9.5gem", "BGS", 0, "BGS 9.5gem", ""},
		{"graded, with price tag", "", 0, "graded, with price tag", ""},
		{"2 PSA cards", "PSA", 0, "2 PSA cards", ""},
		{"graded 2 PSA cards", "PSA", 0, "graded 2 PSA cards", ""},
		{"Graded 9 PSA, see photo", "PSA", 9, "", mtgban.NM},
		{"Graded 9 by PSA see photo", "PSA", 9, "", mtgban.NM},
		{"price tag 5 PSA 9", "PSA", 9, "", mtgban.NM},
		{"tag 9", "TAG", 0, "tag 9", ""},
		{"PSA PR 1", "PSA", 1, "", mtgban.HP},
		{"PSA POOR 1", "PSA", 1, "", mtgban.HP},
		{"PSA FAIR 1.5", "PSA", 1.5, "", mtgban.HP},
		{"PSA FR 1.5", "PSA", 1.5, "", mtgban.HP},
		{"PSA GOOD 2", "PSA", 2, "", mtgban.HP},
		{"psa grade: 9", "PSA", 9, "", mtgban.NM},
	} {
		slab, grade, unread := ctSlab(tt.description)
		want := mtgban.Slab{Grader: tt.grader, Score: tt.score, Label: tt.label}
		if *slab != want {
			t.Errorf("%q: got %+v, want %+v", tt.description, *slab, want)
		}
		wantGrade, wantUnread := tt.grade, ""
		if wantGrade == "" {
			wantGrade, wantUnread = mtgban.NM, tt.label
		}
		if grade != wantGrade || unread != wantUnread {
			t.Errorf("%q: got %s, %q; want %s, %q", tt.description, grade, unread, wantGrade, wantUnread)
		}
	}
}

// TestInfoForScraper pins each sub-seller's published name and shorthand:
// a graded seller is named for its tier, and GradedMode marks it.
func TestInfoForScraper(t *testing.T) {
	ct := &Market{backend: &matcher.Backend{Game: matcher.GameMagic}}
	for _, tt := range []struct {
		name, wantName, wantShorthand string
		graded                        bool
	}{
		{"Card Trader", "Card Trader", "CT", false},
		{"Card Trader Zero", "Card Trader Zero", "CT0", false},
		{"Card Trader 1DR", "Card Trader 1DR", "CT1DR", false},
		{"Card Trader Graded", "Card Trader", "CTGraded", true},
		{"Card Trader Zero Graded", "Card Trader Zero", "CT0Graded", true},
		{"Card Trader 1DR Graded", "Card Trader 1DR", "CT1DRGraded", true},
	} {
		info := ct.InfoForScraper(tt.name)
		if info.Name != tt.wantName || info.Shorthand != tt.wantShorthand || info.GradedMode != tt.graded {
			t.Errorf("InfoForScraper(%q) = %q %q graded=%t, want %q %q graded=%t", tt.name,
				info.Name, info.Shorthand, info.GradedMode, tt.wantName, tt.wantShorthand, tt.graded)
		}
	}
}
