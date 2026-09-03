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
	if len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("ALLPRINTINGS5_PATH not set; skipping the Secret Lair suite")
	}

	// The store writes the drop inside the name it publishes, never in the
	// condition column beside it, so that is where these say it.
	for _, tt := range []struct {
		desc, name  string
		wantRefused bool
	}{
		{"a name the set files under three drops", "Path of Ancestry", true},
		{"and one it files under two", "Kodama's Reach", true},
		{"the number it never wrote is what was missing", "Path of Ancestry (0914)", false},
		{"any wording at all names the drop", "Kodama's Reach (2294 Reskin)", false},
		{"a name standing at one drop needs none", "Sliver Hive", false},
		{"nor does a drop whose other number is its foil twin", "Aether Vial", false},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card, err := preprocess(tt.name, "Secret Lair", "")
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
