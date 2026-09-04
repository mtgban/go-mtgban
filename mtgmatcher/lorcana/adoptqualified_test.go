package lorcana

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestAdoptQualifiedName pins when the bare name is swapped for the catalog's
// decorated one. The qualifier is the whole of the licence: Cool Stuff Inc
// sells both Buckys under one name and tells them apart in a note, so a note
// naming the errata reaches the errata and a note silent about it keeps the
// original standing at the same number.
func TestAdoptQualifiedName(t *testing.T) {
	path := os.Getenv("LORCANA_PATH")
	if path == "" {
		t.Skip("Need LORCANA_PATH set to run this test")
	}
	reader, err := datastore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	backend, err := Load(reader)
	if err != nil {
		t.Fatal(err)
	}
	mtgmatcher.SetGlobalDatastore(backend)

	for _, tt := range []struct{ desc, name, variation, want string }{
		{"a note naming the errata reaches the errata",
			"Bucky - Squirrel Squeak Tutor", "73/204, 3-Cost Errata, Foil No Ward", "m-597095_foil"},
		{"a note silent about it keeps the original",
			"Bucky - Squirrel Squeak Tutor", "73/204, 2-Cost w/ Ward", "289_silver"},
		{"the other errata reads the same way",
			"Elsa - Gloves Off", "39/204, Errata", "m-618771_foil"},
		{"and is left alone unnamed",
			"Elsa - Gloves Off", "39/204", "255_silver"},
		{"the decorated name still answers for itself",
			"Bucky - Squirrel Squeak Tutor (Errata Version)", "73/204", "m-597095_foil"},
		// The qualifier's category noun alone must not license the swap:
		// a note about the original says the word version just as well
		{"a note about the original saying version keeps the original",
			"Bucky - Squirrel Squeak Tutor", "73/204, Original Version w/ Ward", "289_silver"},
		{"nor can a longer word smuggle the noun in",
			"Bucky - Squirrel Squeak Tutor", "73/204, Conversion w/ Ward", "289_silver"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := mtgmatcher.Match(&mtgmatcher.InputCard{
				Name: tt.name, Edition: "Rise of the Floodborn", Variation: tt.variation, Foil: true,
			})
			if err != nil {
				t.Fatalf("Match(%q, %q) = %v", tt.name, tt.variation, err)
			}
			if id != tt.want {
				t.Errorf("Match(%q, %q) = %q, want %q", tt.name, tt.variation, id, tt.want)
			}
		})
	}

	// The number is what names the set and the printing; without one the
	// qualifier licenses nothing, whatever else the wording says.
	t.Run("a wording with no number licenses nothing", func(t *testing.T) {
		in := &mtgmatcher.InputCard{
			Name: "Bucky - Squirrel Squeak Tutor", Edition: "Rise of the Floodborn", Variation: "Errata",
		}
		adoptQualifiedName(backend, in)
		if in.Name != "Bucky - Squirrel Squeak Tutor" {
			t.Errorf("adoptQualifiedName left name %q, want it unchanged", in.Name)
		}
	})
}
