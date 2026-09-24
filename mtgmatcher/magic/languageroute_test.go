package magic

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
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
		// Rinascimento reprinted no Legends or The Dark card
		{"Alabaster Potion", "Renaissance", "Italian", ""},
		// A white-bordered Japanese Fourth Edition exists, and is not carried
		{"Alabaster Potion", "Fourth Edition", "Japanese", ""},
	} {
		in := &mtgmatcher.InputCard{Name: tt.name, Edition: tt.edition, Language: tt.language}
		id, err := testBackend.Match(in)
		if tt.wantSet == "" {
			if !errors.Is(err, mtgmatcher.ErrUnsupported) {
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
