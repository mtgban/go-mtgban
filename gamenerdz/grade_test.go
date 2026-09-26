package gamenerdz

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

func TestGrade(t *testing.T) {
	tests := []struct {
		displayName string
		cond        mtgban.Condition
	}{
		{"Kilnmouth Dragon (LGN-104) - Legions Foil(MP)", mtgban.MP},
		{"Timber Wolves - Revised Edition (MP)", mtgban.MP},
		{"Defiler of Vigor - Dominaria United (LP)", mtgban.SP},
		{"Adarkar Wastes (DMU-243) - Dominaria United Foil (LP)", mtgban.SP},
		{"Ancestral Recall (LEA-048) - Alpha (HP)", mtgban.HP},
		{"Black Lotus (LEA-233) - Alpha (D)", mtgban.PO},
		{"Charizard 4/102 - Base Holofoil (DMG)", mtgban.PO},
		{"Kilnmouth Dragon (LGN-104) - Legions Foil", mtgban.NM},
		{"Avatar of Hope (PRE-003) - Prophecy Promos Foil", mtgban.NM},
		// Bracketed capitals that are not a grade are a name, and a name
		// says nothing about condition.
		{"Richard Garfield, Ph.D. (LIST-017) - The List (UNF)", mtgban.NM},
	}
	for _, tt := range tests {
		cond := grade(tt.displayName)
		if cond != tt.cond {
			t.Errorf("%q: got %q; want %q", tt.displayName, cond, tt.cond)
		}
	}
}
