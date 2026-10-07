package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgban"
)

// TestEmitPricesFoilOnlyProduct pins that a product sold only in foil, here
// LTR Nazgul #725's V.8, prices no plain column: its Low is its foil Low. A
// printing with no foil of its own keeps that Low.
func TestEmitPricesFoilOnlyProduct(t *testing.T) {
	b := realDatastore(t)
	const plain, foil = "6b193b04-ca18-5b08-ad0f-fc5c08e3a65f", "6b193b04-ca18-5b08-ad0f-fc5c08e3a65f_f"
	for _, tc := range []struct {
		cardIDFoil       string
		wantPlain, wantF int
	}{
		{foil, 0, 2},
		{"", 1, 0},
	} {
		mkm := &Index{inventory: mtgban.InventoryRecord{}, exchangeRate: 1}
		mkm.gameID = cm.GameMagic
		mkm.resolver.backend = b
		mkm.priceGuide = map[int]cm.PriceGuide{
			738289: {IDProduct: 738289, LowPrice: 198.99, FoilLowPrice: 198.99, FoilTrendPrice: 245.63},
		}
		channel := make(chan responseChan, 8)
		err := mkm.emitPrices(channel, &cm.Product{IDProduct: 738289}, plain, tc.cardIDFoil, false)
		if err != nil {
			t.Fatal(err)
		}
		close(channel)
		got := map[string]int{}
		for result := range channel {
			got[result.cardID]++
		}
		if got[plain] != tc.wantPlain || got[foil] != tc.wantF {
			t.Errorf("foil %q: priced %d plain and %d foil columns, want %d and %d", tc.cardIDFoil, got[plain], got[foil], tc.wantPlain, tc.wantF)
		}
	}
}
