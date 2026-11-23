package magic

import (
	"errors"
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
