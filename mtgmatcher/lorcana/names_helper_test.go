package lorcana

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func loadNamesBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	path := os.Getenv("LORCANA_PATH")
	if path == "" {
		t.Skip("LORCANA_PATH not set")
	}
	f, err := datastore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	b, err := Load(f)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
