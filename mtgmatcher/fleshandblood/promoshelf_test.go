package fleshandblood

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// promoShelfFixture is the published datastore cut down to the promos these
// tests turn on, every row copied verbatim from it: the older promo set
// holding Dash I/O at HER089 and Bloodrot Pox at FAB133, the Hero programme's
// own set holding Dash I/O at HER156, the Local Game Store programme's
// holding Bloodrot Pox at LGS125, and the FAB programme's set, which is what
// the bare word "Promos" names.
const promoShelfFixture = `{
	"game": "fleshandblood",
	"sets": {
		"FAB": {"name": "Promos", "releaseDate": ""},
		"HER": {"name": "Hero Card Promos", "releaseDate": ""},
		"LGS": {"name": "Local Game Store Promos", "releaseDate": ""},
		"PR": {"name": "Flesh and Blood: Promo Cards", "releaseDate": "2019-10-11", "type": "promo"},
		"EVO": {"name": "Bright Lights", "releaseDate": "2023-10-13"}
	},
	"cards": [
		{"externalLinks": {"fabId": "FAB305"}, "fabId": "FAB305", "finish": "Cold Foil", "id": "fab305_cold", "name": "Imperial Seal of Command", "number": "FAB305", "rarity": "Promo", "setCode": "FAB"},
		{"externalLinks": {"fabId": "HER156"}, "fabId": "HER156", "finish": "Rainbow Foil", "id": "her156_rainbow", "name": "Dash I/O", "number": "HER156", "rarity": "Promo", "setCode": "HER"},
		{"externalLinks": {"fabId": "LGS125"}, "fabId": "LGS125", "finish": "Cold Foil", "id": "lgs125_cold", "name": "Bloodrot Pox", "number": "LGS125", "rarity": "Promo", "setCode": "LGS"},
		{"externalLinks": {"fabId": "HER089", "tcgPlayerId": 518587}, "fabId": "HER089", "finish": "Cold Foil", "id": "her089_518587_cold", "name": "Dash I/O", "number": "HER089", "rarity": "Promo", "setCode": "PR"},
		{"externalLinks": {"fabId": "FAB133", "tcgPlayerId": 497121}, "fabId": "FAB133", "finish": "Rainbow Foil", "id": "fab133_497121_rainbow", "name": "Bloodrot Pox", "number": "FAB133", "rarity": "Promo", "setCode": "PR"},
		{"externalLinks": {"fabId": "FAB172", "tcgPlayerId": 537819}, "fabId": "FAB172", "finish": "Rainbow Foil", "id": "fab172_537819_rainbow", "name": "Meganetic Protocol", "number": "FAB172", "rarity": "Promo", "setCode": "PR"},
		{"externalLinks": {"fabId": "EVO001", "tcgPlayerId": 518564}, "fabId": "EVO001", "finish": "Normal", "id": "evo001_518564", "name": "Dash I/O", "number": "EVO001", "rarity": "Legendary", "setCode": "EVO"}
	]
}`

// TestPromoShelf pins that a promo listing lands on the set holding its
// number whatever promo shelf the storefront named: the programme's own set
// where it holds the number, the older set where only it does, and by name
// alone when the number carries no programme.
func TestPromoShelf(t *testing.T) {
	b, err := Load(strings.NewReader(promoShelfFixture))
	if err != nil {
		t.Fatal(err)
	}
	mtgmatcher.SetGlobalDatastore(b)
	for _, tt := range []struct {
		name, edition, number, want string
	}{
		{"Dash I/O", "Promos", "HER089", "her089_518587_cold"},
		{"Dash I/O", "Flesh and Blood Promos", "HER156", "her156_rainbow"},
		{"Dash I/O", "Promotional Cards", "HER089", "her089_518587_cold"},
		{"Bloodrot Pox", "Promos", "FAB133", "fab133_497121_rainbow"},
		{"Bloodrot Pox", "Flesh and Blood: Promo Cards", "LGS125", "lgs125_cold"},
		{"Imperial Seal of Command", "Flesh and Blood Promos", "FAB305", "fab305_cold"},
		{"Meganetic Protocol", "Promos", "172", "fab172_537819_rainbow"},
		// A set the storefront names outright is not a promo shelf
		{"Dash I/O", "Bright Lights", "EVO001", "evo001_518564"},
	} {
		in := mtgmatcher.InputCard{Name: tt.name, Edition: tt.edition, Variation: tt.number, Foil: true}
		got, err := mtgmatcher.Match(&in)
		if err != nil {
			t.Errorf("Match(%q, %q, %q) = %v", tt.name, tt.edition, tt.number, err)
			continue
		}
		if got != tt.want {
			t.Errorf("Match(%q, %q, %q) = %q, want %q", tt.name, tt.edition, tt.number, got, tt.want)
		}
	}
}
