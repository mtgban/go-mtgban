package lorcana

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestQualifiedNameStaysWhole pins the two catalog names that carry their
// qualifier inside them. Splitting the parenthetical off looked up the bare
// name, so the errata printing was unreachable even spelled out exactly and
// answered with the original standing at the same number.
func TestQualifiedNameStaysWhole(t *testing.T) {
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

	for _, tt := range []struct{ desc, name, number, want string }{
		{"the errata printing answers to its own name",
			"Bucky - Squirrel Squeak Tutor (Errata Version)", "73/204", "m-597095_foil"},
		{"and so does the other one",
			"Elsa - Gloves Off (Errata Version)", "39/204", "m-618771_foil"},
		{"while the bare name still reaches the original",
			"Bucky - Squirrel Squeak Tutor", "73/204", "289_silver"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := mtgmatcher.Match(&mtgmatcher.InputCard{
				Name: tt.name, Edition: "Rise of the Floodborn", Variation: tt.number, Foil: true,
			})
			if err != nil {
				t.Fatalf("Match(%q) = %v", tt.name, err)
			}
			if id != tt.want {
				t.Errorf("Match(%q) = %q, want %q", tt.name, id, tt.want)
			}
		})
	}
}
