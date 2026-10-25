package cardgamecorner

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func TestOwnersBySKU(t *testing.T) {
	editions := []Edition{
		{ID: 1091, Name: "DCI Promo"},
		{ID: 3105, Name: "Adventures in the Forgotten Realms"},
		{ID: promoEditionID, Name: "Promo"},
	}
	listed := []listing{
		// The catch-all edition loses to the shelf the card belongs on
		{editions[2], Card{ID: 1, SKU: "p2024DCISLP023", Name: "Goblin Guide", OrigName: "Guida dei Goblin", Extra: "/goblin-guide.jpg"}},
		{editions[0], Card{ID: 2, SKU: "p2024DCISLP023", Name: "Goblin Guide", OrigName: "Guida dei Goblin", Extra: "/goblin-guide.jpg"}},

		// A repeat with the Italian name in the English field loses
		{editions[1], Card{ID: 3, SKU: "AFR0205", Name: "Spoglie della Caccia", OrigName: "Spoglie della Caccia", Extra: "/spoils.jpg"}},
		{editions[1], Card{ID: 4, SKU: "AFR0205", Name: "Spoils of the Hunt", OrigName: "Spoglie della Caccia", Extra: "/spoils.jpg"}},

		// So does one without an image, whatever its id
		{editions[1], Card{ID: 5, SKU: "DSC0033", Name: "Experimental Lab", OrigName: "Laboratorio", Extra: "/noimage.png"}},
		{editions[1], Card{ID: 6, SKU: "DSC0033", Name: "Experimental Lab", OrigName: "Laboratorio", Extra: "/lab.jpg"}},

		// Between equals, the lower id
		{editions[1], Card{ID: 8, SKU: "DFT0212", Name: "Loot", OrigName: "Loot", Extra: "/loot.jpg"}},
		{editions[1], Card{ID: 7, SKU: "DFT0212", Name: "Loot", OrigName: "Loot", Extra: "/loot.jpg"}},
	}

	owners := ownersBySKU(listed, editions)
	for sku, want := range map[string]int{
		"p2024DCISLP023": 2,
		"AFR0205":        4,
		"DSC0033":        6,
		"DFT0212":        7,
	} {
		got := owners[sku].card.ID
		if got != want {
			t.Errorf("%s: got listing %d, want %d", sku, got, want)
		}
	}
	if len(owners) != 4 {
		t.Errorf("got %d owners, want 4", len(owners))
	}
}

func TestWantedLanguage(t *testing.T) {
	tests := []struct {
		game     mtgmatcher.Game
		language string
		edition  string
		want     bool
	}{
		{mtgmatcher.GameMagic, "EN", "Zendikar Rising", true},
		{mtgmatcher.GameMagic, "DE", "Zendikar Rising", false},
		{mtgmatcher.GameYuGiOh, "EN", "Legend of Blue Eyes White Dragon", true},
		{mtgmatcher.GameYuGiOh, "US", "Legend of Blue Eyes White Dragon", true},
		{mtgmatcher.GameYuGiOh, "IT", "Legend of Blue Eyes White Dragon", false},
		{mtgmatcher.GameMagic, "US", "Zendikar Rising", false},
		{mtgmatcher.GameMagic, "IT", "Revised EU FBB", true},
		{mtgmatcher.GameMagic, "IT", "Zendikar Rising", false},
		{mtgmatcher.GameMagic, "JP", "War of the Spark: Japanese Alternate-Art Planeswalkers", true},
		{mtgmatcher.GameMagic, "JP", "War of the Spark", false},
	}
	for _, test := range tests {
		got := wantedLanguage(test.game, test.language, test.edition)
		if got != test.want {
			t.Errorf("wantedLanguage(%s, %s, %q) = %t, want %t", test.game, test.language, test.edition, got, test.want)
		}
	}
}
