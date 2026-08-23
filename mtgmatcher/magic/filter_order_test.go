package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// A card printed once per World Championship deck leaves Match several
// equally correct answers, and it keeps the first. FilterCards gathers
// those by ranging a map, so without an order imposed at the end the
// same listing resolves to a different physical card from one call to
// the next, and every inventory diff churns forever.
func TestFilterCardsOrderIsStable(t *testing.T) {
	if len(testBackend.GetUUIDs()) == 0 {
		t.Skip("mtgmatcher datastore not loaded")
	}

	for _, name := range []string{"City of Brass", "Llanowar Elves", "Wasteland"} {
		first, err := testBackend.Match(&mtgmatcher.InputCard{Name: name, Edition: "World Championship Decks"})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for i := range 200 {
			got, err := testBackend.Match(&mtgmatcher.InputCard{Name: name, Edition: "World Championship Decks"})
			if err != nil {
				t.Fatalf("%s: call %d: %v", name, i, err)
			}
			if got != first {
				t.Fatalf("%s: call %d returned %s, first call returned %s", name, i, got, first)
			}
		}
	}
}
