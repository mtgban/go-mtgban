package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"
)

// TestRiftboundT1Shelf pins the T1 2025 Worlds Champion Collection shelf: its
// Showcase print is the Signature Edition bundle's, and a starred number is
// that bundle's serial-numbered print. The products are the catalog's own.
func TestRiftboundT1Shelf(t *testing.T) {
	b := loadRiftboundBackend(t)
	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}

	exp := cm.Expansion{Name: "T1 2025 Worlds Champion Collection", SetCode: "T1X"}
	for _, tt := range []struct {
		id      int
		product cm.CatalogProduct
		want    string
	}{
		{904070, cm.CatalogProduct{Name: "Ambessa, The Wolf (V.1 - Showcase)", Number: "S001", Rarity: "Showcase", Version: 1}, "pr-716012_foil"},
		{904071, cm.CatalogProduct{Name: "Ambessa, The Wolf (V.2 - Signed Showcase)", Number: "S001*", Rarity: "Signed Showcase", Version: 2}, "pr-716013_foil"},
	} {
		got := mkm.resolveMapped(tt.id, tt.product, exp)
		if got.err != nil || got.cardID != tt.want || got.cardIDFoil != tt.want {
			t.Errorf("%d %q: resolveMapped = (%q, %q, %v), want %q in both slots",
				tt.id, tt.product.Name, got.cardID, got.cardIDFoil, got.err, tt.want)
		}
	}
}
