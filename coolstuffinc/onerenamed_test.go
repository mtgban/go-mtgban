package coolstuffinc

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
)

// TestOnePieceRenamedTreatment pins what the rename may and may not reach.
// The guard that carries it is the set wearing one label throughout: every
// set holding a real Full Art holds Parallel beside it, so none of the 75
// reaches this at all. The label check behind it is defensive, for a set that
// someday wears Full Art alone.
func TestOnePieceRenamedTreatment(t *testing.T) {
	path := os.Getenv("ONEPIECE_PATH")
	if path == "" {
		t.Skip("Need ONEPIECE_PATH set to run this test")
	}
	reader, err := datastore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	backend, err := onepiece.Load(reader)
	if err != nil {
		t.Fatal(err)
	}
	mtgmatcher.SetGlobalDatastore(backend)

	for _, tt := range []struct {
		desc, id, name, want string
	}{
		{"a word the set does not use reaches the one printing it can mean",
			"st21-003_615568_foil", "Sanji - 003 (Full Art)", "st21-003_615569_foil"},
		{"a set wearing several labels does not say which one is meant",
			"st29-002_671548", "Nami - 002 (Full Art)", ""},
		{"a listing naming nothing is left alone",
			"st21-003_615568_foil", "Sanji - 003", ""},
		{"and so is one already on the printing it names",
			"st21-003_615569_foil", "Sanji - 003 (Full Art)", ""},
		{"a word the catalog never uses reaches nothing",
			"st21-003_615568_foil", "Sanji - 003 (Shiny Chrome)", ""},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := onePieceRenamedTreatment(tt.id, tt.name); got != tt.want {
				t.Errorf("onePieceRenamedTreatment(%q, %q) = %q, want %q", tt.id, tt.name, got, tt.want)
			}
		})
	}
}
