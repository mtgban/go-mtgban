package magic

import (
	"errors"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

// TestLanguageRoutesEdition pins that the language a listing is in, not only
// its wording, routes a plain edition to the set printed in that language, and
// that a card the set never printed is refused rather than guessed.
func TestLanguageRoutesEdition(t *testing.T) {
	realDatastore(t)
	for _, tt := range []struct {
		name, edition, language, wantSet string
	}{
		{"Sylvan Library", "Legends", "Italian", "LEGITA"},
		{"Season of the Witch", "The Dark", "Italian", "DRKITA"},
		{"Azure Drake", "Chronicles", "Japanese", "BCHR"},
		{"Junún Efreet", "Renaissance", "Italian", "RIN"},
		{"Badlands", "Foreign Black Border", "Italian", "FBB"},
		{"Badlands", "Foreign Black Border", "French", "FBBFR"},
		{"Badlands", "Foreign Black Border", "German", "FBBDE"},
		{"Counterspell", "Fourth Edition Foreign Black Border", "Japanese", "4BB"},
		{"Counterspell", "Fourth Edition Foreign Black Border", "Spanish", "4BBES"},
		{"Counterspell", "Fourth Edition Foreign Black Border", "Portuguese", "4BBPT"},
		{"Counterspell", "Fourth Edition Foreign Black Border", "Korean", "4BBKO"},
		{"Counterspell", "Fourth Edition Foreign Black Border", "Chinese Traditional", "4BBZHT"},
		{"Angry Mob", "Renaissance", "French", "REN"},
		{"Angry Mob", "Renaissance", "German", "RENDE"},
		{"Badlands", "Revised Edition", "Italian", "FWB"},
		{"Badlands", "Revised Edition", "French", "FWBFR"},
		{"Badlands", "Revised Edition", "German", "FWBDE"},
		{"Badlands", "Foreign White Border", "Italian", "FWB"},
		{"Badlands", "Foreign White Border", "French", "FWBFR"},
		{"Badlands", "Foreign White Border", "", "FWB"},
		{"Badlands", "German Revised FWB", "", "FWBDE"},
		{"Badlands", "Revised Edition", "", "3ED"},
		{"Badlands", "Revised Edition", "Spanish", ""},
		{"Badlands", "Foreign White Border", "English", ""},
		{"Badlands", "Fourth Edition (Foreign White Border)", "Italian", ""},
		// mtgjson has no German FWB of it, and its German FBB is another set's
		{"Drudge Skeletons", "Foreign White Border", "German", ""},
		// An edition naming no set says nothing of the border
		{"Badlands", "Some Unheard Edition", "German", ""},
		{"Badlands", "Some Unheard Edition", "Italian", ""},
		{"Winter Orb", "Some Unheard Edition", "Japanese", ""},
		{"Angry Mob", "Some Unheard Edition", "French", ""},
		{"Badlands", "Fourth Edition", "Italian", ""},
		// FBB was never printed in Spanish
		{"Badlands", "Foreign Black Border", "Spanish", ""},
		// Rinascimento reprinted no Legends or The Dark card
		{"Alabaster Potion", "Renaissance", "Italian", ""},
		// A white-bordered Japanese Fourth Edition exists, and is not carried
		{"Alabaster Potion", "Fourth Edition", "Japanese", ""},
	} {
		in := &matcher.InputCard{Name: tt.name, Edition: tt.edition, Language: tt.language}
		id, err := testBackend.Match(in)
		if tt.wantSet == "" {
			if !errors.Is(err, matcher.ErrUnsupported) {
				t.Errorf("%s from %s in %s = %q %v, want unsupported", tt.name, tt.edition, tt.language, id, err)
			}
			continue
		}
		if err != nil {
			t.Errorf("%s from %s in %s: %v", tt.name, tt.edition, tt.language, err)
			continue
		}
		co, err := testBackend.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.SetCode != tt.wantSet {
			t.Errorf("%s from %s in %s landed on %s, want %s", tt.name, tt.edition, tt.language, co.SetCode, tt.wantSet)
		}
	}
}

// A card printed in one of a set's languages alone is held by that language's
// set, beside the cards printed in all of them.
func TestLanguageSetHoldsItsOwnCards(t *testing.T) {
	realDatastore(t)
	const wald = "d44cd8fc-9f0c-533b-a2d2-83b30b2130a0"
	for _, tt := range []struct {
		number, edition, language, want string
	}{
		{"306a", "Foreign Black Border", "German", wald},
		{"306a", "Foreign Black Border (German)", "", wald},
		{"306", "Foreign Black Border", "German", "0fd7280a-90be-539d-ae3f-db6c93ecb549"},
		{"306", "Foreign Black Border", "Italian", "54dcb147-244e-5220-85a0-9f9deaeefc77"},
		{"306a", "Foreign Black Border", "French", ""},
		{"306a", "Foreign Black Border", "Italian", ""},
	} {
		in := &matcher.InputCard{Name: "Forest", Edition: tt.edition, Variation: tt.number, Language: tt.language}
		id, err := testBackend.Match(in)
		if id != tt.want || (err != nil) != (tt.want == "") {
			t.Errorf("Forest #%s from %s in %q = %q %v, want %q", tt.number, tt.edition, tt.language, id, err, tt.want)
		}
	}

	// A vendor's Scryfall id names it too
	id, err := testBackend.Match(&matcher.InputCard{ID: "38382bb9-3b8e-4819-9998-55dbf1ab0fd7", Language: "German"})
	if err != nil || id != wald {
		t.Errorf("German Forest #306a by its Scryfall id = %q %v", id, err)
	}
	co, err := testBackend.GetUUID(wald)
	if err != nil {
		t.Fatal(err)
	}
	if co.SetCode != "FBBDE" || co.Edition != "Foreign Black Border (German)" || co.Language != "German" ||
		co.PrintedName != "Wald" || !slices.Contains(co.Printings, "FBBDE") {
		t.Errorf("Forest #306a is %s %q in %s, %q", co.SetCode, co.Edition, co.Language, co.PrintedName)
	}
	for _, ls := range languageSets {
		for _, card := range testBackend.Sets[ls.base].Cards {
			if card.Language == langs[ls.tag] {
				t.Errorf("%s #%s %s is in %s, not %s", ls.base, card.Number, card.Language, ls.base, ls.code)
			}
		}
	}
}
