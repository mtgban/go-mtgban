package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
)

// TestCatalogFabMarvel pins the tier the catalog spends a field of its own on.
// A marvel is a separate product sharing its card's number and treatment, so
// the sku, the set, the collector number and the finish are the same words for
// both, and the rarity is the only thing that tells them apart. Dropping it
// did not lose the listing, it moved the money: Star City Games asks $99.99
// for the Dynasty marvel of Construct Nitro Mechanoid, and that price was
// being quoted against the ordinary cold foil beside it.
func TestCatalogFabMarvel(t *testing.T) {
	withGameDatastore(t, "FLESHANDBLOOD_PATH", fleshandblood.Load)

	product := func(rarity string) CatalogProduct {
		return CatalogProduct{
			SKU: "SGL-FAB-DYN2-092-ENC", Name: "Construct Nitro Mechanoid // Nitro Mechanoid",
			Game: "Flesh and Blood", Set: "Dynasty", ProductType: ProductTypeSingles,
			CollectorNumber: "092", Finish: "Cold Foil", FinishGroup: "Alt Foil",
			Language: "English", Rarity: rarity,
		}
	}

	for _, tt := range []struct {
		desc       string
		rarity     string
		wantRarity string
	}{
		{"the rarity names the marvel", "Marvel", "Marvel"},
		{"and a tier that describes rather than names leaves the printing alone", "Majestic", "Majestic"},
		{"as does a catalog that says nothing", "", "Majestic"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := resolveProduct(GameFleshAndBlood, product(tt.rarity))
			if err != nil {
				t.Fatalf("resolveProduct(%q) = %v", tt.rarity, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.Rarity != tt.wantRarity {
				t.Errorf("resolveProduct(%q) = %s (%s), want rarity %s",
					tt.rarity, id, co.Rarity, tt.wantRarity)
			}
		})
	}
}
