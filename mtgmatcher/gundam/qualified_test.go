package gundam

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// qualifiedFixture holds a number sold plain and under an event label, and
// a number sold in two rarities, every row the datastore's.
const qualifiedFixture = `{
	"game": "gundam",
	"sets": {
		"GCG-PR": {"name": "Gundam Promotional Cards", "releaseDate": "2025-03-01", "type": "promo"},
		"GD01-b": {"name": "Edition Beta", "releaseDate": "2025-02-06"}
	},
	"cards": [
		{"color": "Red", "externalLinks": {"tcgPlayerId": 666557}, "finish": "Holofoil", "id": "gd02-091_666557_holofoil", "image": "x", "name": "Haman Karn", "number": "GD02-091", "promoTypes": ["newtypechallenge"], "rarity": "Rare", "setCode": "GCG-PR", "type": "Pilot", "variant": "Newtype Challenge 2025 Mission 3"},
		{"color": "Red", "externalLinks": {"tcgPlayerId": 666560}, "finish": "Holofoil", "id": "gd02-091_666560_holofoil", "image": "x", "name": "Haman Karn", "number": "GD02-091", "rarity": "Rare", "setCode": "GCG-PR", "type": "Pilot"},
		{"color": "Blue", "externalLinks": {"tcgPlayerId": 616528}, "finish": "Holofoil", "id": "st01-001_616528_holofoil", "image": "x", "name": "Gundam", "number": "ST01-001", "rarity": "Legend Rare", "setCode": "GD01-b", "type": "Unit"},
		{"color": "Blue", "externalLinks": {"tcgPlayerId": 616531}, "finish": "Holofoil", "id": "st01-001_616531_holofoil", "image": "x", "name": "Gundam", "number": "ST01-001", "rarity": "LR+", "setCode": "GD01-b", "type": "Unit"}
	]
}`

// TestQualifiedNameReachesItsPrinting pins that a storefront writing the
// catalog's qualifier into the name reaches the printing it qualifies: the
// spelling is in the name hashes and never canonical, and the branch that
// read it back through CanonicalNames emptied the name instead.
func TestQualifiedNameReachesItsPrinting(t *testing.T) {
	b, err := Load(strings.NewReader(qualifiedFixture))
	if err != nil {
		t.Fatal(err)
	}
	for _, variation := range []string{"", "GD02-091"} {
		in := mtgmatcher.InputCard{Name: "Haman Karn Newtype Challenge 2025 Mission 3", Variation: variation}
		got, err := b.Match(&in)
		if err != nil || got != "gd02-091_666557_holofoil" {
			t.Errorf("Match(%q, %q) = %q, %v; want the Newtype Challenge printing", in.Name, variation, got, err)
		}
	}
	in := mtgmatcher.InputCard{Name: "Haman Karn", Variation: "GD02-091"}
	got, err := b.Match(&in)
	if err != nil || got != "gd02-091_666560_holofoil" {
		t.Errorf("Match(%q) = %q, %v; want the plain printing", "Haman Karn", got, err)
	}
}
