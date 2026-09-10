package starcitygames

import (
	"testing"
)

// TestCatalogWordingBeatsTwinID pins the printings whose catalog ids are
// their plain twin's. The id generator keys on the collector number, so a
// printing sharing its number with a plain sibling wears the sibling's
// Scryfall and TCGplayer ids - the serialized Cyclonic Rift carries the
// plain 313's pair, and every Ampersand promo its promo-pack sibling's.
// The record's own wording - the finish naming the treatment, the sku slug
// naming the promo - is what still says which printing is meant, so an id
// the wording contradicts must fall through to the sku-driven path, while
// an id the wording agrees with stays authoritative. Every product here is
// copied verbatim from the catalog export.
func TestCatalogWordingBeatsTwinID(t *testing.T) {
	withMagic(t)

	for _, tt := range []struct {
		desc string
		p    CatalogProduct
		want string
	}{
		{
			desc: "a serialized printing wearing the plain twin's ids",
			p: CatalogProduct{SKU: "SGL-MTG-RVR4-313-ENF", Name: "Cyclonic Rift",
				Game: "Magic: The Gathering", ProductType: ProductTypeSingles,
				Set: "Ravnica Remastered", Finish: "Double Rainbow Foil",
				FinishGroup: "Alt Foil", CollectorNumber: "313", Language: "English",
				ScryfallID: "631a15c2-0a6d-4859-91e9-a08a7e756054", TCGPlayerID: "530801"},
			want: "ff706d32-ee22-5e83-bd31-223ea1d853e7",
		},
		{
			desc: "the plain twin itself keeps its id",
			p: CatalogProduct{SKU: "SGL-MTG-RVR3-313-ENF", Name: "Cyclonic Rift",
				Game: "Magic: The Gathering", ProductType: ProductTypeSingles,
				Set: "Ravnica Remastered", Finish: "Foil", FinishGroup: "Foil",
				CollectorNumber: "313", Language: "English",
				ScryfallID: "631a15c2-0a6d-4859-91e9-a08a7e756054", TCGPlayerID: "530801"},
			want: "fb65c72a-2eb4-5dd4-8ba9-7f06b6654205_f",
		},
		{
			desc: "a double rainbow printing that was never serialized keeps its own id",
			p: CatalogProduct{SKU: "SGL-MTG-PRM-BAB_2024_001-ENF", Name: "Sol Ring",
				Game: "Magic: The Gathering", ProductType: ProductTypeSingles,
				Set: "Promo", Finish: "Double Rainbow Foil", FinishGroup: "Alt Foil",
				CollectorNumber: "001", Language: "English",
				ScryfallID: "120e2b4b-afc7-4bf0-a09f-568e08f6bd8f", TCGPlayerID: "594545"},
			want: "4f2c5045-ee0c-5e16-bd37-76d782e052e3",
		},
		{
			desc: "a rainbowfoil serialized headliner the finish still calls double rainbow",
			p: CatalogProduct{SKU: "SGL-MTG-SOS3-306-ENA", Name: "Emeritus of Ideation // Ancestral Recall",
				Game: "Magic: The Gathering", ProductType: ProductTypeSingles,
				Set: "Secrets of Strixhaven", Finish: "Double Rainbow Foil",
				FinishGroup: "Alt Foil", CollectorNumber: "306", Language: "English",
				ScryfallID: "ef371352-ec8f-4da4-9085-67195068fb79", TCGPlayerID: "689798"},
			want: "66843b36-7479-586e-aec8-59d1fddfe3a1",
		},
		{
			desc: "an Ampersand promo wearing the promo-pack sibling's ids",
			p: CatalogProduct{SKU: "SGL-MTG-PRM-AMP_AFR_004-ENF", Name: "The Book of Exalted Deeds",
				Game: "Magic: The Gathering", ProductType: ProductTypeSingles,
				Set: "Promo", Finish: "Foil", FinishGroup: "Foil",
				CollectorNumber: "004", Language: "English",
				ScryfallID: "b0477339-1a1f-45f3-a9ba-704de1a788cc", TCGPlayerID: "244202"},
			want: "1cdf2ed5-fc69-519d-a46f-21bbd5d93318",
		},
		{
			desc: "an Ampersand promo of a card with no promo-pack sibling",
			p: CatalogProduct{SKU: "SGL-MTG-PRM-AMP_AFR_015-ENF", Name: "Flumph",
				Game: "Magic: The Gathering", ProductType: ProductTypeSingles,
				Set: "Promo", Finish: "Foil", FinishGroup: "Foil",
				CollectorNumber: "015", Language: "English"},
			want: "589b2875-1693-50ec-9f2f-504ef54ce965",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := resolveProduct(GameMagic, tt.p)
			if err != nil {
				t.Fatalf("resolveProduct(%s) = %v", tt.p.SKU, err)
			}
			if id != tt.want {
				t.Errorf("resolveProduct(%s) = %s, want %s", tt.p.SKU, id, tt.want)
			}
		})
	}
}
