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
		{904070, cm.CatalogProduct{Name: "Ambessa, The Wolf (V.1 - Showcase)", Number: "S001", Rarity: "Showcase", Version: 1}, "716012"},
		{904071, cm.CatalogProduct{Name: "Ambessa, The Wolf (V.2 - Signed Showcase)", Number: "S001*", Rarity: "Signed Showcase", Version: 2}, "716013"},
	} {
		got := mkm.resolveMapped(tt.id, tt.product, exp)
		if got.err != nil || got.cardID != got.cardIDFoil {
			t.Errorf("%d %q: resolveMapped = (%q, %q, %v), want one printing in both slots",
				tt.id, tt.product.Name, got.cardID, got.cardIDFoil, got.err)
			continue
		}
		co, err := b.GetUUID(got.cardID)
		if err != nil {
			t.Fatal(err)
		}
		if co.Identifiers["tcgplayerProductId"] != tt.want {
			t.Errorf("%d %q: landed on %s, TCGplayer product %q, want %q",
				tt.id, tt.product.Name, got.cardID, co.Identifiers["tcgplayerProductId"], tt.want)
		}
	}
}

// TestRiftboundPromoRunes pins that a promo shelf's runes keep to their own
// set, where TCGplayer files them: each "<Set>: Promos" R06b lands on that
// set's R06b, and a champion's lettered art on the shelf stays unplaced.
func TestRiftboundPromoRunes(t *testing.T) {
	b := loadRiftboundBackend(t)
	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}

	for _, tt := range []struct {
		id      int
		shelf   string
		product cm.CatalogProduct
		want    string
	}{
		{898192, "Vendetta: Promos", cm.CatalogProduct{Name: "Order Rune", Number: "R06b", Rarity: "Showcase"}, "709750"},
		{874112, "Spiritforged: Promos", cm.CatalogProduct{Name: "Order Rune", Number: "R06b", Rarity: "Showcase"}, "680432"},
		{917613, "Radiance: Promos", cm.CatalogProduct{Name: "Order Rune", Number: "R06b", Rarity: "Showcase"}, "732645"},
		{900504, "Origins: Promos", cm.CatalogProduct{Name: "Yasuo, Remorseful", Number: "076a", Rarity: "Showcase"}, ""},
	} {
		got := mkm.resolveMapped(tt.id, tt.product, cm.Expansion{Name: tt.shelf})
		if tt.want == "" {
			if got.cardID != "" {
				t.Errorf("%d %q on %s: landed on %s, want no printing", tt.id, tt.product.Name, tt.shelf, got.cardID)
			}
			continue
		}
		if got.err != nil || got.cardID == "" {
			t.Errorf("%d %q on %s: resolveMapped = (%q, %v), want a printing", tt.id, tt.product.Name, tt.shelf, got.cardID, got.err)
			continue
		}
		co, err := b.GetUUID(got.cardID)
		if err != nil {
			t.Fatal(err)
		}
		if co.Identifiers["tcgplayerProductId"] != tt.want {
			t.Errorf("%d %q on %s: landed on %s, TCGplayer product %q, want %q",
				tt.id, tt.product.Name, tt.shelf, got.cardID, co.Identifiers["tcgplayerProductId"], tt.want)
		}
	}
}
