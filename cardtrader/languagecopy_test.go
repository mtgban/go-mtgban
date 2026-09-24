package cardtrader

import "testing"

// TestProcessProductsMatchesEachListingAlone pins that a listing is matched on
// its own copy of the blueprint's card: the Italian listing routes the edition
// to Legends Italian, and the English one after it must not inherit that.
func TestProcessProductsMatchesEachListingAlone(t *testing.T) {
	b := realDatastore(t)

	bp := &Blueprint{
		ID:          37010,
		Name:        "Sylvan Library",
		GameID:      GameMagic,
		CategoryID:  CategoryMagicSingles,
		ScryfallID:  "f486df00-7c4a-4ff0-bb0b-c8b5432ac742",
		TCGplayerID: 4035,
	}
	bp.Expansion.Name = "Legends"
	bp.Properties.Number = "207"

	ct := &Market{
		backend:     b,
		gameID:      GameMagic,
		blueprints:  map[int]*Blueprint{bp.ID: bp},
		logCallback: func(string, ...any) {},
	}

	newProduct := func(id int, lang string) Product {
		var p Product
		p.ID = id
		p.BlueprintID = bp.ID
		p.Quantity = 1
		p.Price.Cents = 100
		p.Price.Currency = "USD"
		p.Properties.Condition = "Near Mint"
		p.Properties.Number = bp.Properties.Number
		p.Properties.MTGLanguage = lang
		return p
	}

	ch := make(chan resultChan, 2)
	ct.processProducts(ch, bp.ID, []Product{newProduct(1, "it"), newProduct(2, "en")})
	close(ch)

	want := map[string]string{"1": "LEGITA", "2": "LEG"}
	for result := range ch {
		co, err := b.GetUUID(result.cardID)
		if err != nil {
			t.Fatal(err)
		}
		listing := result.invEntry.InstanceID
		if co.SetCode != want[listing] {
			t.Errorf("listing %s landed on %s, want %s", listing, co.SetCode, want[listing])
		}
		delete(want, listing)
	}
	for listing, set := range want {
		t.Errorf("listing %s did not land, want %s", listing, set)
	}
}
