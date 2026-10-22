package magiccorner

import "testing"

func TestOwnersBySKU(t *testing.T) {
	editions := []MCEdition{
		{ID: 1091, Name: "DCI Promo"},
		{ID: 3105, Name: "Adventures in the Forgotten Realms"},
		{ID: mcPromoEditionID, Name: "Promo"},
	}
	listed := []listing{
		// The catch-all edition loses to the shelf the card belongs on
		{editions[2], MCCard{ID: 1, SKU: "p2024DCISLP023", Name: "Goblin Guide", OrigName: "Guida dei Goblin", Extra: "/goblin-guide.jpg"}},
		{editions[0], MCCard{ID: 2, SKU: "p2024DCISLP023", Name: "Goblin Guide", OrigName: "Guida dei Goblin", Extra: "/goblin-guide.jpg"}},

		// A repeat with the Italian name in the English field loses
		{editions[1], MCCard{ID: 3, SKU: "AFR0205", Name: "Spoglie della Caccia", OrigName: "Spoglie della Caccia", Extra: "/spoils.jpg"}},
		{editions[1], MCCard{ID: 4, SKU: "AFR0205", Name: "Spoils of the Hunt", OrigName: "Spoglie della Caccia", Extra: "/spoils.jpg"}},

		// So does one without an image, whatever its id
		{editions[1], MCCard{ID: 5, SKU: "DSC0033", Name: "Experimental Lab", OrigName: "Laboratorio", Extra: "/noimage.png"}},
		{editions[1], MCCard{ID: 6, SKU: "DSC0033", Name: "Experimental Lab", OrigName: "Laboratorio", Extra: "/lab.jpg"}},

		// Between equals, the lower id
		{editions[1], MCCard{ID: 8, SKU: "DFT0212", Name: "Loot", OrigName: "Loot", Extra: "/loot.jpg"}},
		{editions[1], MCCard{ID: 7, SKU: "DFT0212", Name: "Loot", OrigName: "Loot", Extra: "/loot.jpg"}},
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
