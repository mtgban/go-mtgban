package abugames

import (
	"testing"
)

// TestFoilFlag pins the flag this storefront writes both with its space and
// without. Reading only the spaced form priced a foil as a nonfoil, and put
// its price beside the nonfoil's on one uuid.
func TestFoilFlag(t *testing.T) {
	b := realDatastore(t)
	for _, test := range []struct {
		desc  string
		title string
		want  bool
	}{
		{"the flag written without its space", "Island (261) -FOIL", true},
		{"and with it", "Island (261) - FOIL", true},
		{"a card that names no finish", "Island (261)", false},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := ABUCard{DisplayTitle: test.title, Edition: "Ravnica Allegiance", Number: "261"}
			in, err := preprocess(b, &card)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", test.title, err)
			}
			if in.Foil != test.want {
				t.Errorf("preprocess(%q).Foil = %v, want %v", test.title, in.Foil, test.want)
			}
			if _, err := b.Match(in); err != nil {
				t.Errorf("Match(%q) = %v", in, err)
			}
		})
	}
}

// TestFoilOfUnfoiled pins which printings may carry the storefront's FOIL
// flag: a foil or an etched one, and not one that has neither finish.
func TestFoilOfUnfoiled(t *testing.T) {
	b := realDatastore(t)
	for _, test := range []struct {
		desc    string
		title   string
		landed  string
		edition string
		number  string
		skipped bool // the FOIL listing is discarded
	}{
		{"an etched printing", "Anguished Unmaking (ETCHED) - FOIL", "Anguished Unmaking (ETCHED) - FOIL", "Double Masters 2022", "469", false},
		{"a foil printing", "Island (261) - FOIL", "Island (261) - FOIL", "Ravnica Allegiance", "261", false},
		{"a printing with no foil", "Black Lotus - FOIL", "Black Lotus", "Limited Edition Alpha", "", true},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := ABUCard{DisplayTitle: test.landed, Edition: test.edition, Number: test.number}
			in, err := preprocess(b, &card)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", test.landed, err)
			}
			id, err := matchCard(b, &card, in)
			if err != nil {
				t.Fatalf("matchCard(%q) = %v", test.landed, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			got := foilOfUnfoiled(test.title, co)
			if got != test.skipped {
				t.Errorf("foilOfUnfoiled(%q) on %s %s = %v, want %v", test.title, co.SetCode, co.Number, got, test.skipped)
			}
		})
	}
}
