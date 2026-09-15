package mtgmatcher

import (
	"io"
	"strings"
	"testing"
)

// Open builds the sealed index, so a loader that never called SortSealed
// still answers ResolveSealed off one index rather than rebuilding it per
// call.
func TestOpenBuildsTheSealedIndex(t *testing.T) {
	// No game package can be registered from here - every one of them imports
	// this package - so Open is given a fake, for the length of this test
	// only: the registry is a package variable with no unregister hook, and
	// leaving a game in it that no datastore backs would answer
	// RegisteredGames for the whole binary.
	saved := registeredGames
	t.Cleanup(func() { registeredGames = saved })
	RegisterGame("opentest", func(io.Reader) (*Backend, error) {
		return &Backend{}, nil
	})

	b, err := Open("opentest", strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if b.Game != "opentest" {
		t.Errorf("Game = %q", b.Game)
	}
	if b.sealedIdx == nil {
		t.Fatal("the sealed index was not built")
	}
	if got := b.sealedLookup(); got != b.sealedIdx {
		t.Error("sealedLookup rebuilt the index it already had")
	}
}
