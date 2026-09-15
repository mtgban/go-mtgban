package mtgmatcher

import (
	"io"
	"strings"
	"testing"
)

func init() {
	RegisterGame("opentest", func(io.Reader) (*Backend, error) {
		return &Backend{}, nil
	})
}

// Open stamps the game and builds the sealed index, so a loader that never
// called SortSealed still answers ResolveSealed off one index rather than
// rebuilding it per call.
func TestOpenStampsTheGameAndBuildsTheSealedIndex(t *testing.T) {
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
