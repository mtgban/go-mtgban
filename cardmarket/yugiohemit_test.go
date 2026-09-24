package cardmarket

import (
	"slices"
	"testing"

	cm "github.com/mtgban/go-cardmarket"
)

// TestEmitYugiohColumns pins that a Yu-Gi-Oh product's own columns land on
// the printing it resolved to, whichever run that is, and that the guide's
// trend-foil lands nowhere: it is not the first edition's price. Reading the
// run as a foil sent every product down the second pair alone, which priced
// the unlimited printing from trend-foil and everything else from nothing.
func TestEmitYugiohColumns(t *testing.T) {
	b := datastoreBackend(t, "yugioh", yugiohShelfDatastore)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	mkm.exchangeRate = 1
	mkm.priceGuide = map[int]cm.PriceGuide{
		1: {IDProduct: 1, LowPrice: 1, TrendPrice: 2, FoilTrendPrice: 3},
		3: {IDProduct: 3, LowPrice: 7, TrendPrice: 8, FoilTrendPrice: 9},
		4: {IDProduct: 4, LowPrice: 10, TrendPrice: 11},
	}
	products := []cm.Product{
		// Unlimited, with a first edition beside it
		{IDProduct: 1, Name: "Tri-Horned Dragon (V.1 - Secret Rare)", Number: "000", ExpansionName: "Legend of Blue Eyes White Dragon"},
		// Printed in a first edition and nothing else
		{IDProduct: 3, Name: "Topologic Bomber Dragon", Number: "065", ExpansionName: "2018 Mega-Tin Mega Pack"},
		// Printed unlimited and nothing else
		{IDProduct: 4, Name: "Tri-Horned Dragon (V.4 - Secret Rare)", Number: "EN000", ExpansionName: "Legend of Blue Eyes White Dragon"},
	}
	channel := make(chan responseChan, 16)
	for i := range products {
		if err := mkm.processProduct(channel, &products[i]); err != nil {
			t.Fatalf("processProduct(%q): %v", products[i].Name, err)
		}
	}
	close(channel)

	got := map[string][]float64{}
	for result := range channel {
		got[result.cardID] = append(got[result.cardID], result.entry.Price)
	}
	want := map[string][]float64{
		"lob-000_22538_unlimited": {1, 2},
		"mp18-en065_1":            {7, 8},
		"lob-en000_1":             {10, 11},
	}
	for uuid, prices := range want {
		if !slices.Equal(got[uuid], prices) {
			t.Errorf("%s priced %v, want %v", uuid, got[uuid], prices)
		}
	}
	for uuid := range got {
		if _, expected := want[uuid]; !expected {
			t.Errorf("%s was priced %v and should not have been", uuid, got[uuid])
		}
	}
}
