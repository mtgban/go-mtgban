package gamenerdz

import "testing"

func TestGrade(t *testing.T) {
	tests := []struct {
		displayName string
		cond        string
	}{
		{"Kilnmouth Dragon (LGN-104) - Legions Foil(MP)", "MP"},
		{"Timber Wolves - Revised Edition (MP)", "MP"},
		{"Defiler of Vigor - Dominaria United (LP)", "SP"},
		{"Adarkar Wastes (DMU-243) - Dominaria United Foil (LP)", "SP"},
		{"Ancestral Recall (LEA-048) - Alpha (HP)", "HP"},
		{"Black Lotus (LEA-233) - Alpha (D)", "PO"},
		{"Kilnmouth Dragon (LGN-104) - Legions Foil", "NM"},
		{"Avatar of Hope (PRE-003) - Prophecy Promos Foil", "NM"},
		// Bracketed capitals that are not a grade are a name, and a name
		// says nothing about condition.
		{"Richard Garfield, Ph.D. (LIST-017) - The List (UNF)", "NM"},
	}
	for _, tt := range tests {
		cond := grade(tt.displayName)
		if cond != tt.cond {
			t.Errorf("%q: got %q; want %q", tt.displayName, cond, tt.cond)
		}
	}
}
