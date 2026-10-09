package strikezone

import "testing"

func TestShelfName(t *testing.T) {
	for _, tt := range []struct{ h1, want string }{
		{"Singles Alpha Buy Lists", "Alpha"},
		{"Singles Tarkir: Dragonstorm\t Buy Lists", "Tarkir: Dragonstorm"},
		{"Lorcana Singles\u00a0Disney Lorcana Promo Cards Buy Lists", "Disney Lorcana Promo Cards"},
		{"Lorcana Singles Disney Lorcana Promo Cards", "Disney Lorcana Promo Cards"},
	} {
		got := shelfName(tt.h1)
		if got != tt.want {
			t.Errorf("shelfName(%q) = %q, want %q", tt.h1, got, tt.want)
		}
	}
}

// TestBasicLandArt pins that a bare basic land on the Alpha and Beta shelves
// reads as the first art, and that the lands the store marks B or C are left
// to the matcher.
func TestBasicLandArt(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct{ name, edition, wantVariation string }{
		{"Forest", "Beta", "A"},
		{"Plains", "Alpha", "A"},
		{"Forest B", "Beta", ""},
		{"Island Sanctuary", "Alpha", ""},
	} {
		card, err := preprocess(b, tt.name, tt.edition, "")
		if err != nil {
			t.Fatal(err)
		}
		if card.Variation != tt.wantVariation {
			t.Errorf("%s on %s: variation %q, want %q", tt.name, tt.edition, card.Variation, tt.wantVariation)
		}
	}
}

// TestShelfCards pins the cards a shelf holds that belong to another set, or
// that a shelf names a promo type for.
func TestShelfCards(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct{ name, edition, notes, wantSet, wantNumber string }{
		{"Chrome Mox (Borderless)", "Aetherdrift", "Normal", "SPG", ""},
		{"Sol Ring (C21)", "Secret Lair Commander: Heads I Win", "Near Mint Normal English", "PLST", ""},
		{"Serra Angel (Retro Frame)", "Promos: 30th Anniversary Promos", "Near Mint Foil English", "P30H", ""},
		{"Go for the Throat", "Promos: Standard Showdown", "Near Mint Foil English", "PCBB", "3"},
		{"Vito, Thorn of the Dusk Rose", "Promos: Media", "Near Mint Foil English", "PM21", "127★"},
		{"Swords to Plowshares", "Promos: Media", "Near Mint Foil English", "PMEI", "2026-4"},
		{"Gandalf, Friend of the Shire", "Promos: Play", "Near Mint Foil English", "PF23", "1"},
	} {
		card, err := preprocess(b, tt.name, tt.edition, tt.notes)
		if err != nil {
			t.Fatal(err)
		}
		id, err := b.Match(card)
		if err != nil {
			t.Fatalf("%s on %s: %v", tt.name, tt.edition, err)
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.SetCode != tt.wantSet || (tt.wantNumber != "" && co.Number != tt.wantNumber) {
			t.Errorf("%s on %s landed on %s #%s, want %s #%s", tt.name, tt.edition, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
		}
	}
}
