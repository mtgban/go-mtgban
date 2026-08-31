package onepiece

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastoretest"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func loadNamesBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	path := os.Getenv("ONEPIECE_PATH")
	if path == "" {
		t.Skip("ONEPIECE_PATH not set")
	}
	f, err := datastoretest.Open(path)
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
