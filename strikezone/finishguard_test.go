package strikezone

import "testing"

// TestFinishPrinted pins the three shapes the guard tells apart: a genuine
// mismatch with an alternate finish elsewhere in the same set (drop), a
// mismatch with a same-shelf sibling already pricing the other finish
// (drop), and a single-printing card with neither (keep, since output()
// already clamps it correctly and dropping would lose its only price).
func TestFinishPrinted(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		desc           string
		name, set      string
		foil           bool
		hasFoilSibling bool
		want           bool
	}{
		{
			desc: "same-set alternate finish is a genuine mismatch",
			name: "Chasm Skulker", set: "FIC", foil: true, want: false,
		},
		{
			desc: "a shelf sibling pricing the other finish is a genuine mismatch",
			name: "Gandalf, Friend of the Shire", set: "PLTR", foil: false, hasFoilSibling: true, want: false,
		},
		{
			desc: "no alternate and no sibling keeps the only printing",
			name: "Ramos, Dragon Engine", set: "C17", foil: false, want: true,
		},
		{
			desc: "a foil tag on a nonfoil-only printing keeps it too",
			name: "Iconic Shield", set: "MSC", foil: true, want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			cards := b.MatchInSet(tt.name, tt.set)
			if len(cards) == 0 {
				t.Fatalf("no %q in %s", tt.name, tt.set)
			}
			co, err := b.GetUUID(cards[0].UUID)
			if err != nil {
				t.Fatal(err)
			}
			if got := finishPrinted(b, co, tt.foil, "", tt.hasFoilSibling); got != tt.want {
				t.Errorf("finishPrinted(%s) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

// TestFromCuteToBrutePathway pins that this shelf's pathways redirect to
// their PLST reprint instead of the foil-only Secret Lair Ultimate printing
// the bare name would otherwise collide with.
func TestFromCuteToBrutePathway(t *testing.T) {
	b := realDatastore(t)

	card, err := preprocess(b, "Brightclimb Pathway", "Secret Lair Commander: From Cute to Brute", "")
	if err != nil {
		t.Fatal(err)
	}
	id, err := b.Match(card)
	if err != nil {
		t.Fatal(err)
	}
	co, err := b.GetUUID(id)
	if err != nil {
		t.Fatal(err)
	}
	if co.SetCode != "PLST" || co.Number != "ZNR-259" {
		t.Errorf("got %s #%s, want PLST #ZNR-259", co.SetCode, co.Number)
	}
}
