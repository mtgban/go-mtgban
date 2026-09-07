package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestResolveProductEtched covers the finish only the catalog's finish name
// carries. A sku spells an etched printing "-ENF" exactly as it spells a plain
// foil, so where the printing holds [etched nonfoil] and nothing else, an
// etched product would resolve to the non-foil twin the shop's own "-ENN" sku
// already owns, and the two would fight over it. The products are the ones the
// catalog sends, ids and all, since which path resolves them is the point.
func TestResolveProductEtched(t *testing.T) {
	withMagic(t)

	for _, tt := range []struct {
		product              CatalogProduct
		wantSet, wantNum     string
		wantEtched, wantFoil bool
	}{
		{
			// No id, so the wording path is the only one that runs.
			CatalogProduct{
				SKU: "SGL-MTG-PRM3-SECRET_SLD_1053-ENF", Name: "Talisman of Dominance",
				Game: "Magic: The Gathering", Set: "Secret Lair Drop", Rarity: "Promo",
				Finish: "Foil Etched", FinishGroup: "Alt Foil", Language: "English",
				CollectorNumber: "1053", ProductType: ProductTypeSingles,
			}, "SLD", "1053", true, false,
		},
		{
			CatalogProduct{
				SKU: "SGL-MTG-PRM-SECRET_SLD_1053-ENN", Name: "Talisman of Dominance",
				Game: "Magic: The Gathering", Set: "Secret Lair Drop", Rarity: "Promo",
				ScryfallID: "8b5713b7-f2a6-41c5-a485-058b7393570b", TCGPlayerID: "283295",
				Finish: "Non-foil", FinishGroup: "Non-foil", Language: "English",
				CollectorNumber: "1053", ProductType: ProductTypeSingles,
			}, "SLD", "1053", false, false,
		},
		{
			CatalogProduct{
				SKU: "SGL-MTG-PRM3-FEST_2022_001-ENF", Name: "Arcane Signet",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				Finish: "Foil Etched", FinishGroup: "Alt Foil", Language: "English",
				CollectorNumber: "FEST_2022_001", ProductType: ProductTypeSingles,
			}, "P30M", "1F★", true, false,
		},
		{
			CatalogProduct{
				SKU: "SGL-MTG-PRM-FEST_2022_001-ENF", Name: "Arcane Signet",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				ScryfallID: "e96d398f-7393-4b8e-9972-2d7c394d23ad", TCGPlayerID: "482666",
				Finish: "Foil", FinishGroup: "Foil", Language: "English",
				CollectorNumber: "001F", ProductType: ProductTypeSingles,
			}, "P30M", "1F", false, true,
		},
	} {
		t.Run(tt.product.SKU, func(t *testing.T) {
			id, err := resolveProductID(GameMagic, tt.product)
			if err != nil {
				t.Fatalf("resolveProductID: %v", err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID: %v", err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("got %s #%s, want %s #%s", co.SetCode, co.Number, tt.wantSet, tt.wantNum)
			}
			if co.Etched != tt.wantEtched || co.Foil != tt.wantFoil {
				t.Errorf("got etched=%v foil=%v, want etched=%v foil=%v",
					co.Etched, co.Foil, tt.wantEtched, tt.wantFoil)
			}
		})
	}
}
