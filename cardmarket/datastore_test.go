package cardmarket

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// datastoreBackend reads a datastore written inline as the game's.
func datastoreBackend(t *testing.T, game, doc string) *mtgmatcher.Backend {
	t.Helper()
	b, err := mtgmatcher.Open(game, strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	return b
}
