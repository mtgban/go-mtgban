package palworld

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPlainNumberMatchesLoader pins the rules to the loader. PlainNumber is
// what folds a number a person typed, and OriginalNumber is what it is
// compared against, so the two spelling a number differently finds nothing
// and raises nothing - the failure a caller reads as "no such card".
func TestPlainNumberMatchesLoader(t *testing.T) {
	path := os.Getenv("PALWORLD_PATH")
	if path == "" {
		t.Skip("Need PALWORLD_PATH set to run this test")
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
	mtgmatcher.SetGlobalDatastore(b)

	var seen int
	for _, code := range mtgmatcher.GetAllSets() {
		set, err := mtgmatcher.GetSet(code)
		if err != nil {
			continue
		}
		for _, card := range set.Cards {
			plain := Rules{}.PlainNumber(card.Number)
			if plain != card.OriginalNumber {
				t.Errorf("%s %q: PlainNumber is %q, OriginalNumber is %q",
					code, card.Number, plain, card.OriginalNumber)
			}
			// Folding a number already plain has nothing left to do.
			again := Rules{}.PlainNumber(plain)
			if again != plain {
				t.Errorf("%s %q: folding %q again gives %q", code, card.Number, plain, again)
			}
			seen++
		}
	}
	if seen == 0 {
		t.Fatal("no cards to check")
	}
}
