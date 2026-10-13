package cardtrader

import (
	"maps"
	"testing"
)

// TestProcessProductsKeepsCartFields pins what a processed Magic listing
// carries for a cart to find it again on Card Trader's own pages: its price
// in Card Trader's own currency, and its properties, the language and the
// foil flag, empty when nonfoil.
func TestProcessProductsKeepsCartFields(t *testing.T) {
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

	var p Product
	p.ID = 1
	p.BlueprintID = bp.ID
	p.Quantity = 1
	p.Price.Cents = 1450
	p.Price.Currency = "USD"
	p.Properties.Condition = "Near Mint"
	p.Properties.Number = bp.Properties.Number
	p.Properties.MTGLanguage = "en"
	p.User.Name = "seller"
	p.User.CountryCode = "IT"

	ch := make(chan resultChan, 1)
	ct.processProducts(ch, bp.ID, []Product{p})
	close(ch)

	result, ok := <-ch
	if !ok {
		t.Fatal("the listing did not land")
	}
	want := map[string]string{
		"SubSellerName": "seller",
		"SubSellerGeo":  "IT",
		"PriceCents":    "1450",
		"PriceCurrency": "USD",
		"Properties":    "mtg_foil=&mtg_language=en",
	}
	if !maps.Equal(result.invEntry.CustomFields, want) {
		t.Errorf("fields are %v, want %v", result.invEntry.CustomFields, want)
	}
}

// TestListingFields pins each game's properties: its language, and the
// finish flags or treatment it names, a false flag empty, none where the
// blueprint is the printing.
func TestListingFields(t *testing.T) {
	var p Product
	p.Price.Cents = 999
	p.Price.Currency = "EUR"
	p.User.Name = "seller"
	p.User.CountryCode = "DE"
	p.Properties.MTGLanguage = "jp"
	p.Properties.MTGFoil = true
	p.Properties.LorcanaLanguage = "zh-CN"
	p.Properties.RiftboundLanguage = "en"
	p.Properties.RiftboundFoil = true
	p.Properties.OnePieceLanguage = "jp"
	p.Properties.YuGiOhLanguage = "de"
	p.Properties.PalworldLanguage = "en"
	p.Properties.FabLanguage = "en"
	p.Properties.FabFoilNew = "Rainbow Foil"
	p.Properties.FirstEdition = true
	p.Properties.PokemonLanguage = "en"
	p.Properties.GundamLanguage = "en"

	got := listingFields(GameMagic, p)
	want := map[string]string{"SubSellerName": "seller", "SubSellerGeo": "DE", "PriceCents": "999", "PriceCurrency": "EUR", "Properties": "mtg_foil=true&mtg_language=jp"}
	if !maps.Equal(got, want) {
		t.Errorf("Magic fields are %v, want %v", got, want)
	}

	for _, tc := range []struct {
		name   string
		gameID int
		want   string
	}{
		{"Lorcana", GameLorcana, "lorcana_foil=&lorcana_language=zh-CN"},
		{"Riftbound", GameRiftbound, "riftbound_foil=true&riftbound_language=en"},
		{"One Piece", GameOnePiece, "onepiece_foil=&onepiece_language=jp"},
		{"Yu-Gi-Oh", GameYuGiOh, "first_edition=true&yugioh_language=de"},
		{"Flesh and Blood", GameFleshAndBlood, "fab_foil_new=Rainbow+Foil&fab_language=en&first_edition=true"},
		{"Pokemon", GamePokemon, "first_edition=true&pokemon_language=en&pokemon_reverse="},
		{"Gundam", GameGundam, "gundam_language=en"},
		{"Palworld", GamePalworld, "palworld_language=en"},
	} {
		props := listingFields(tc.gameID, p)["Properties"]
		if props != tc.want {
			t.Errorf("%s properties are %q, want %q", tc.name, props, tc.want)
		}
	}
}
