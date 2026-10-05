package strikezone

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestSecretLairDrop pins when a bare Secret Lair listing is refused. What
// the store never writes is the drop, and the set files some cards under
// several: a listing saying nothing about which reaches one of them for no
// reason, and every other drop is then priced as that one.
func TestSecretLairDrop(t *testing.T) {
	b := realDatastore(t)

	// The store writes the drop inside the name it publishes, never in the
	// condition column beside it, so that is where these say it.
	for _, tt := range []struct {
		desc, name  string
		wantRefused bool
	}{
		{"a name the set files under three drops", "Path of Ancestry", true},
		{"nor does one whose other drop is printed in another finish", "Windfall", false},
		{"the number it never wrote is what was missing", "Path of Ancestry (0914)", false},
		{"any wording at all names the drop", "Kodama's Reach (2294 Reskin)", false},
		{"a name standing at one drop needs none", "Sliver Hive", false},
		{"nor does one whose other drop has a flavor name", "Dictate of Erebos", false},
		{"unless a flavor name nothing knows was cut off the listing", "Mimir's Ancient Wisdum - Teferi's Ageless Insight", true},
		{"nor does a drop whose other number is its foil twin", "Aether Vial", false},
		{"nor one whose other number is its step-and-compleat", "Plague Sliver", false},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card, err := preprocess(b, tt.name, "Secret Lair", "")
			refused := errors.Is(err, mtgmatcher.ErrUnsupported)
			if refused != tt.wantRefused {
				t.Fatalf("preprocess(%q) refused = %v, want %v (err %v)",
					tt.name, refused, tt.wantRefused, err)
			}
			if !refused && card == nil {
				t.Fatalf("preprocess(%q) returned no card and no refusal", tt.name)
			}
		})
	}
}

// TestSecretLairDropByFinish pins that a name filed under several drops is
// not refused when the finish on sale is printed in only one of them, and
// that the drop it lands on follows the finish.
func TestSecretLairDropByFinish(t *testing.T) {
	b := realDatastore(t)

	// Teferi, Time Raveler has a nonfoil-only drop and a foil-only one.
	drops := map[string]string{}
	for _, notes := range []string{"Normal", "Foil"} {
		card, err := preprocess(b, "Teferi, Time Raveler", "Secret Lair", notes)
		if err != nil {
			t.Fatalf("%s: %v", notes, err)
		}
		if card.Variation == "" {
			t.Fatalf("%s: no drop was named", notes)
		}
		drops[notes] = card.Variation
	}
	if drops["Normal"] == drops["Foil"] {
		t.Errorf("both finishes read as drop %s", drops["Normal"])
	}

	// Path of Ancestry is printed in both finishes in more than one drop.
	for _, notes := range []string{"Normal", "Foil"} {
		_, err := preprocess(b, "Path of Ancestry", "Secret Lair", notes)
		if !errors.Is(err, mtgmatcher.ErrUnsupported) {
			t.Errorf("Path of Ancestry %s: err %v, want a refusal", notes, err)
		}
	}
}
