package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestCatalogDoubleRainbowPrinting pins the printing a double rainbow product
// reaches. Star City Games sells the serialized printing under a finish name
// of its own but sends the plain printing's scryfall id beside it, so the two
// met on one uuid and a $900 buylist row stood next to a $50 one.
//
// The id still decides wherever it names a double rainbow printing itself,
// which is every one of these products but this one.
func TestCatalogDoubleRainbowPrinting(t *testing.T) {
	withMagic(t)

	for _, tt := range []struct {
		desc    string
		product CatalogProduct
		want    string
	}{
		{
			desc: "the finish reaches the serialized printing the id misses",
			product: CatalogProduct{
				SKU: "SGL-MTG-RVR4-313-ENF", ScryfallID: "631a15c2-0a6d-4859-91e9-a08a7e756054",
				Name: "Cyclonic Rift", Set: "Ravnica Remastered", CollectorNumber: "313",
				Finish: "Double Rainbow Foil", Language: "English",
			},
			want: "313z",
		},
		{
			desc: "the plain foil beside it stays where it was",
			product: CatalogProduct{
				SKU: "SGL-MTG-RVR3-313-ENF", ScryfallID: "631a15c2-0a6d-4859-91e9-a08a7e756054",
				Name: "Cyclonic Rift", Set: "Ravnica Remastered", CollectorNumber: "313",
				Finish: "Foil", Language: "English",
			},
			want: "313",
		},
		{
			// An id that already names the double rainbow answers for itself,
			// and reading the finish must not move it.
			desc: "an id that names the treatment is left alone",
			product: CatalogProduct{
				SKU: "SGL-MTG-BRR3-080-ENF", ScryfallID: "70398b01-0301-4a1c-8768-2f189f12577c",
				Name: "Gilded Lotus", Set: "The Brothers' War Retro Artifacts",
				CollectorNumber: "080", Finish: "Double Rainbow Foil", Language: "English",
			},
			want: "80z",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := resolveProduct(GameMagic, tt.product)
			if err != nil {
				t.Fatalf("resolveProduct(%s) = %v", tt.product.SKU, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.Number != tt.want {
				t.Errorf("resolveProduct(%s) = #%s, want #%s", tt.product.SKU, co.Number, tt.want)
			}
			if tt.want != "313" && !co.HasPromoType(magic.PromoTypeDoubleRainbow) {
				t.Errorf("resolveProduct(%s) = %v, want a double rainbow", tt.product.SKU, co.PromoTypes)
			}
		})
	}
}
