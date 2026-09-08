package yugioh

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
)

// TestProductGroupHoldsOneCard pins that the key a product's printings are
// folded on never gathers two different cards. Nothing downstream can report
// that failure: a key gathering too much yields one card whose FoilUUIDs
// price another card's printing under one of its finishes, and every count
// still adds up.
//
// What it turns on is the order productKey reads the identifiers in. An
// upstream id is not unique the way a product id is: it names a card, and a
// card is sold as several products. 458 Flesh and Blood fabIds and 712
// Pokemon tcgdexIds are carried by more than one product - an extended-art
// printing beside the plain one, a fused card beside the half it fuses - so
// the product id has to answer first. Reversing the two merges 491 pairs in
// Flesh and Blood alone, and nothing but this says so.
//
// The check runs on the entries rather than on the backend, because the fold
// has already happened by the time a backend exists: a product's printings
// share one Card, so comparing what they say about themselves afterwards
// compares a value with itself.
func TestProductGroupHoldsOneCard(t *testing.T) {
	path := os.Getenv("YUGIOH_PATH")
	if path == "" {
		t.Skip("YUGIOH_PATH not set; skipping Yugioh matcher suite")
	}
	f, err := datastore.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	var payload Datastore
	if err := json.NewDecoder(f).Decode(&payload); err != nil {
		t.Fatal(err)
	}

	type identity struct{ name, number, setCode, variant string }
	first := map[string]identity{}
	firstID := map[string]string{}
	for i := range payload.Cards {
		card := &payload.Cards[i]
		key := card.productKey()
		this := identity{card.Name, card.Number, card.SetCode, card.Variant}
		seen, found := first[key]
		if !found {
			first[key], firstID[key] = this, card.ID
			continue
		}
		if seen != this {
			t.Errorf("key %q folds %s (%s %s %s %q) with %s (%s %s %s %q), a different card",
				key, firstID[key], seen.name, seen.setCode, seen.number, seen.variant,
				card.ID, this.name, this.setCode, this.number, this.variant)
		}
	}
	if len(first) == 0 {
		t.Fatal("the datastore grouped no products at all")
	}
}
