package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestResolveFleshAndBloodMarvelTwins pins the two shelves where SCG's own
// catalog fields send two different printings to the same match: a plain
// rarity landing on its Marvel twin because nothing steered it away, and a
// renamed foreign printing landing on the English card whose name it
// borrows. Fixtures are copied from the export verbatim.
func TestResolveFleshAndBloodMarvelTwins(t *testing.T) {
	withGameDatastore(t, "fleshandblood", "FLESHANDBLOOD_PATH")

	for _, tt := range []struct {
		desc                  string
		p                     CatalogProduct
		wantNumber            string
		wantMarvel, wantMaori bool
	}{
		{
			desc: "a plain rarity is that plain printing",
			p: CatalogProduct{
				SKU: "SGL-FAB-UPR-042b_043b-ENC", Name: "Aether Ashwing // Ash",
				Game: "Flesh and Blood", Set: "Uprising", Rarity: "Token",
				Finish: "Cold Foil", FinishGroup: "Alt Foil",
				Language: "English", CollectorNumber: "042b",
				ProductType: ProductTypeSingles,
			},
			wantNumber: "UPR043", wantMarvel: false, wantMaori: false,
		},
		{
			desc: "not the Marvel twin at the same number",
			p: CatalogProduct{
				SKU: "SGL-FAB-UPR2-042_043-ENC", Name: "Aether Ashwing // Ash",
				Game: "Flesh and Blood", Set: "Uprising", Rarity: "Marvel",
				Finish: "Cold Foil", FinishGroup: "Alt Foil",
				Language: "English", CollectorNumber: "042",
				ProductType: ProductTypeSingles,
			},
			wantNumber: "UPR043", wantMarvel: true, wantMaori: false,
		},
		{
			desc: "the English printing is the name it sells under",
			p: CatalogProduct{
				SKU: "SGL-FAB-SUP2-009-ENC", Name: "Pleiades, Superstar",
				Game: "Flesh and Blood", Set: "Super Slam", Rarity: "Marvel",
				Finish: "Cold Foil", FinishGroup: "Alt Foil",
				Language: "English", CollectorNumber: "009",
				ProductType: ProductTypeSingles,
			},
			wantNumber: "SUP009", wantMarvel: true, wantMaori: false,
		},
		{
			desc: "the Maori one is not, though the sku sends the same name",
			p: CatalogProduct{
				SKU: "SGL-FAB-SUP2-009b-ENC", Name: "Pleiades, Superstar",
				Game: "Flesh and Blood", Set: "Super Slam", Rarity: "Marvel",
				Finish: "Cold Foil", FinishGroup: "Alt Foil",
				Language: "English", CollectorNumber: "009",
				ProductType: ProductTypeSingles,
			},
			wantNumber: "SUP009", wantMarvel: true, wantMaori: true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := resolveProduct(GameFleshAndBlood, tt.p)
			if err != nil {
				t.Fatalf("resolveProduct(%s) = %v", tt.p.SKU, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.Number != tt.wantNumber {
				t.Errorf("%s resolved to number %s, want %s", tt.p.SKU, co.Number, tt.wantNumber)
			}
			if got := co.HasPromoType("marvel"); got != tt.wantMarvel {
				t.Errorf("%s resolved to marvel=%v, want %v", tt.p.SKU, got, tt.wantMarvel)
			}
			if got := co.HasPromoType("pleiadessuperstar"); got != tt.wantMaori {
				t.Errorf("%s resolved to the Maori printing=%v, want %v (name=%q)", tt.p.SKU, got, tt.wantMaori, co.Name)
			}
		})
	}
}
