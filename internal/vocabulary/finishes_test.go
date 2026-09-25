package vocabulary

import (
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPublishedFinishesHaveARow reports a finish a datastore publishes that
// mtgmatcher.Finishes has no row for: it loads under its own name, but it is
// taken for a foil and has no run or label until someone adds the row.
func TestPublishedFinishesHaveARow(t *testing.T) {
	var played int
	for _, game := range GameNames() {
		path := PathOf(game)
		if path == "" {
			continue
		}
		b, err := datastore.Read(game, path)
		if err != nil {
			t.Fatal(err)
		}
		played++
		missing := map[string]int{}
		for _, co := range b.UUIDs {
			if co.Sealed {
				continue
			}
			if _, found := mtgmatcher.FinishOf(co.Finish); !found {
				missing[co.Finish]++
			}
		}
		for finish, count := range missing {
			t.Errorf("%s: %d printings are sold in %q, which mtgmatcher.Finishes has no row for", game, count, finish)
		}
	}
	if played == 0 {
		t.Skip("no game datastore is named in this run")
	}
}
