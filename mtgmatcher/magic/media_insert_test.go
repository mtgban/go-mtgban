package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestMediaInsertReprints pins both printings of every card in
// mediaInsertOriginals: the Japanese original a storefront names by its
// number or its language, and the English reprint it names by neither.
//
// The reprint is the half worth pinning. Its number used to be written into
// the rule, and by the time anyone looked three of the five had moved - Shock,
// Duress and Voltaic Key each pointed at a number the set does not hold, so
// every English listing of them answered "unknown variant".
func TestMediaInsertReprints(t *testing.T) {
	for _, tt := range []struct {
		name              string
		marker            string
		japanese, english string
	}{
		{"Diabolic Edict", "31", "2019-2", "2024-5"},
		{"Shock", "32", "2019-3", "2024-9"},
		{"Duress", "34", "2019-6", "2024-10"},
		{"Voltaic Key", "35", "2020-1", "2024-11"},
		{"Dark Ritual", "38", "2020-4", "2025-8"},
		// From here on no storefront writes a number of its own, so the
		// language is all that tells the two printings apart. Counterspell
		// and Disenchant carry a case of their own elsewhere for unrelated
		// promos, which is why the rule is not written as a case.
		{"Crop Rotation", "42", "2020-7", "2025-9"},
		{"Counterspell", "44", "2021-1", "2025-15"},
		{"Bone Shredder", "45", "2021-2", "2025-16"},
		{"Disenchant", "", "2022-1", "2025-17"},
		{"Wild Growth", "", "2022-2", "2026-2"},
		{"Frantic Search", "", "2022-4", "2026-20"},
		{"Worn Powerstone", "", "2023-2", "2026-21"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assertMediaInsert(t, tt.name, tt.marker, tt.japanese, tt.english)
		})
	}

	// The volumes whose English reprint has not landed: the number a
	// listing writes still reaches the Japanese printing, and so does one
	// writing nothing, there being no other.
	for _, tt := range []struct {
		name, marker, japanese string
	}{
		{"Avalanche Riders", "55", "2023-5"},
		{"Culling the Weak", "57", "2023-8"},
		{"Snuff Out", "1", "2024-1"},
		{"Gush", "1", "2024-4"},
		{"Ancestral Mask", "1", "2025-2"},
		{"Wrath of God", "1", "2025-3"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			assertMediaInsert(t, tt.name, tt.marker, tt.japanese, tt.japanese)
		})
	}

	// Duress alone was also a comic insert, which is a different set.
	assertNumber(t, mtgmatcher.InputCard{
		Name: "Duress", Edition: "IDW Comics", Variation: "IDW Comics 2014",
	}, "PIDW", "17")
}

// assertMediaInsert walks the listing shapes a storefront writes for one of
// these: the reprint by wording alone, and the original by its language, by
// the language written into the product name, and by its own number.
func assertMediaInsert(t *testing.T, name, marker, japanese, english string) {
	t.Helper()
	assertNumber(t, mtgmatcher.InputCard{
		Name: name, Edition: "Media Promos", Variation: "Graphic Novel Insert",
	}, "PMEI", english)
	assertNumber(t, mtgmatcher.InputCard{
		Name: name, Edition: "Media Promos", Variation: "JP Graphic Novel Insert",
	}, "PMEI", japanese)
	assertNumber(t, mtgmatcher.InputCard{
		Name: name, Edition: "Media Promos", Variation: "Magazine Insert",
		Language: "Japanese",
	}, "PMEI", japanese)
	if marker != "" {
		assertNumber(t, mtgmatcher.InputCard{
			Name: name, Edition: "Media Promos", Variation: "Magazine Insert " + marker,
		}, "PMEI", japanese)
	}
}

func assertNumber(t *testing.T, in mtgmatcher.InputCard, setCode, number string) {
	t.Helper()
	card := in
	id, err := testBackend.Match(&card)
	if err != nil {
		t.Errorf("Match(%v) = %v, want %s %s", in, err, setCode, number)
		return
	}
	co, found := testBackend.UUIDs[id]
	if !found {
		t.Errorf("Match(%v) = %s, which is not a card", in, id)
		return
	}
	if co.SetCode != setCode || co.Number != number {
		t.Errorf("Match(%v) = %s %s, want %s %s", in, co.SetCode, co.Number, setCode, number)
	}
}
