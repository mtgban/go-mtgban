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

// TestEtchedListingIsNotFoil pins that a "(Foil Etched)" listing asks for the
// etched finish alone, in both the retail and the buylist notes, so a foil
// printing such as the Galaxy foil Sol Ring never claims it.
func TestEtchedListingIsNotFoil(t *testing.T) {
	b := realDatastore(t)

	tests := []struct {
		name, notes, number string
	}{
		{"Sol Ring (Retro Frame) (Foil Etched)", "Foil", "1074"},
		{"Plains (Tokyo Lands) (Foil Etched)", "Near Mint Foil English", "46"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			card, err := preprocess(b, tt.name, "Secret Lair", tt.notes)
			if err != nil {
				t.Fatal(err)
			}
			if card.Foil {
				t.Errorf("etched listing kept Foil: %+v", card)
			}
			id, err := b.Match(card)
			if err != nil {
				t.Fatal(err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != "SLD" || co.Number != tt.number || !co.Etched {
				t.Errorf("got %s #%s etched=%v, want SLD #%s etched", co.SetCode, co.Number, co.Etched, tt.number)
			}
		})
	}
}
