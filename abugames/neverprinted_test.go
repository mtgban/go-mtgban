package abugames

import (
	"errors"
	"testing"
)

// TestNeverPrinted pins the listings naming a card the catalog does not hold.
// Each was answered with the nearest printing wearing another identity - a
// tutorial card, or the same card reprinted out of a different set - and
// carried the storefront's price there, beside the price already on it.
func TestNeverPrinted(t *testing.T) {
	b := realDatastore(t)
	for _, test := range []struct {
		desc string
		card ABUCard
	}{
		{"a nonfoil of a card its set sells only in foil as a bundle promo", ABUCard{
			DisplayTitle: "Momo, Friendly Flier (394)", Edition: "Avatar: The Last Airbender", Number: "394"}},
		{"and one sold only in foil as neon ink", ABUCard{
			DisplayTitle: "Fire Lord Zuko (360)", Edition: "Avatar: The Last Airbender", Number: "360"}},
		{"a reprint's nonfoil, where that set reprinted it in foil", ABUCard{
			DisplayTitle: "Entreat the Angels (The List)", Edition: "Avacyn Restored", Number: "20"}},
	} {
		t.Run(test.desc, func(t *testing.T) {
			card := test.card
			in, err := preprocess(b, &card)
			if err == nil {
				_, err = b.Match(in)
			}
			if err == nil {
				t.Errorf("preprocess(%q) = %v, want a refusal", card.DisplayTitle, in)
				return
			}
			if !errors.Is(err, errUnprintedFinish) {
				t.Logf("%q refused as %v", card.DisplayTitle, err)
			}
		})
	}
}

// TestEternalNonfoil walks every eternalNonfoil row: the number it is keyed
// on must be the card's foil-only printing, and a nonfoil listing of it must
// land on the alternative nonfoil of the same name.
func TestEternalNonfoil(t *testing.T) {
	b := realDatastore(t)
	set, err := b.GetSet("TLE")
	if err != nil {
		t.Fatal(err)
	}
	type eternalCase struct {
		name, title, number, want string
	}
	var tests []eternalCase
	for i := range set.Cards {
		card := &set.Cards[i]
		want, found := eternalNonfoil[card.Number]
		if !found {
			continue
		}
		if card.HasFinish(finishNonfoil) {
			t.Errorf("TLE %s %s sells in nonfoil, its row is never read", card.Number, card.Name)
		}
		tests = append(tests, eternalCase{card.Name, card.Name + " (" + card.Number + ")", card.Number, want})
	}
	if len(tests) != len(eternalNonfoil) {
		t.Errorf("found %d of the %d eternalNonfoil numbers in TLE", len(tests), len(eternalNonfoil))
	}
	// The storefront files this one under its tutorial card's number.
	tests = append(tests, eternalCase{"Sledding Otter-Penguin", "Sledding Otter-Penguin (218)", "208", "273"})

	for _, test := range tests {
		t.Run(test.title, func(t *testing.T) {
			card := ABUCard{DisplayTitle: test.title, Edition: "Avatar: The Last Airbender Eternal", Number: test.number}
			in, err := preprocess(b, &card)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", test.title, err)
			}
			id, err := b.Match(in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", in, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != "TLE" || co.Number != test.want || co.Foil || co.Name != test.name || !co.IsAlternative {
				t.Errorf("Match(%v) = %s|%s %s foil=%v alternative=%v, want TLE|%s %s nonfoil alternative",
					in, co.SetCode, co.Number, co.Name, co.Foil, co.IsAlternative, test.want, test.name)
			}
		})
	}
}
