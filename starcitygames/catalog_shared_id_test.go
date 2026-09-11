package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestSharedIdentifierPairs pins the printings Star City Games tells apart on
// its shelves but not in its identifiers. Each pair below is two genuinely
// different printings the catalog sends one scryfall_id and one tcgplayer_id
// for, so whichever the matcher answered with priced them both - the record
// keeps the dearer entry. The fixtures are copied from the export verbatim,
// shared identifiers and all, because the shared identifier is the point.
func TestSharedIdentifierPairs(t *testing.T) {
	withMagic(t)

	for _, tt := range []struct {
		desc                string
		p                   CatalogProduct
		wantSet, wantNumber string
	}{
		{
			desc: "the Timeshifts shelf is its own set, not Modern Horizons",
			p: CatalogProduct{
				SKU: "SGL-MTG-MH12-005-ENF", Name: "Ranger-Captain of Eos",
				Game: "Magic: The Gathering", Set: "Modern Horizons", Rarity: "Mythic Rare",
				Finish: "Foil", FinishGroup: "Foil",
				Language: "English", CollectorNumber: "005",
				ScryfallID: "af3928b4-813a-4120-8799-de34235d60ac", TCGPlayerID: "190873",
				ProductType: ProductTypeSingles,
			},
			wantSet: "H1R", wantNumber: "5",
		},
		{
			desc: "and the original keeps its own printing",
			p: CatalogProduct{
				SKU: "SGL-MTG-MH1-21-ENF", Name: "Ranger-Captain of Eos",
				Game: "Magic: The Gathering", Set: "Modern Horizons", Rarity: "Mythic Rare",
				Finish: "Foil", FinishGroup: "Foil",
				Language: "English", CollectorNumber: "021",
				ScryfallID: "af3928b4-813a-4120-8799-de34235d60ac", TCGPlayerID: "190873",
				ProductType: ProductTypeSingles,
			},
			wantSet: "MH1", wantNumber: "21",
		},
		{
			desc: "a World Championship sku names the year's deck",
			p: CatalogProduct{
				SKU: "SGL-MTG-WCHP-04MB_5DN_134-ENN", Name: "Krark-Clan Ironworks",
				Game: "Magic: The Gathering", Set: "World Championships", Rarity: "Uncommon",
				Finish: "Non-foil", FinishGroup: "Non-foil",
				Language: "English", CollectorNumber: "134",
				ScryfallID: "c60174d6-1f9d-4870-b3db-34d6fcb3f6ab", TCGPlayerID: "11808",
				ProductType: ProductTypeSingles,
			},
			wantSet: "WC04", wantNumber: "mb134",
		},
		{
			desc: "not the card the deck reprinted",
			p: CatalogProduct{
				SKU: "SGL-MTG-5DN-134-ENN", Name: "Krark-Clan Ironworks",
				Game: "Magic: The Gathering", Set: "Fifth Dawn", Rarity: "Uncommon",
				Finish: "Non-foil", FinishGroup: "Non-foil",
				Language: "English", CollectorNumber: "134",
				ScryfallID: "c60174d6-1f9d-4870-b3db-34d6fcb3f6ab", TCGPlayerID: "11808",
				ProductType: ProductTypeSingles,
			},
			wantSet: "5DN", wantNumber: "134",
		},
		{
			desc: "a lettered prerelease names the set's own card",
			p: CatalogProduct{
				SKU: "SGL-MTG-PRM-PRE_LTR_402b-ENF", Name: "Delighted Halfling",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				Finish: "Foil", FinishGroup: "Foil",
				Language: "English", CollectorNumber: "402",
				ScryfallID: "e65be588-4209-4ad7-9bdc-d601fdc0e8f6", TCGPlayerID: "501096",
				ProductType: ProductTypeSingles,
			},
			wantSet: "LTR", wantNumber: "402",
		},
		{
			desc: "its unlettered twin is the datestamped promo",
			p: CatalogProduct{
				SKU: "SGL-MTG-PRM-PRE_LTR_402a-ENF", Name: "Delighted Halfling",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				Finish: "Foil", FinishGroup: "Foil",
				Language: "English", CollectorNumber: "402",
				ScryfallID: "e65be588-4209-4ad7-9bdc-d601fdc0e8f6", TCGPlayerID: "501096",
				ProductType: ProductTypeSingles,
			},
			wantSet: "PLTR", wantNumber: "402s",
		},
		{
			desc: "a promo pack is stamped, not embossed",
			p: CatalogProduct{
				SKU: "SGL-MTG-PRM-PP_AFR_087-ENF", Name: "Acererak the Archlich",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				Finish: "Foil", FinishGroup: "Foil",
				Language: "English", CollectorNumber: "087",
				ScryfallID: "621a9aad-403e-412f-ad21-b90bc84bc696", TCGPlayerID: "247201",
				ProductType: ProductTypeSingles,
			},
			wantSet: "PAFR", wantNumber: "87p",
		},
		{
			desc: "the Ampersand beside it is the embossed one",
			p: CatalogProduct{
				SKU: "SGL-MTG-PRM-AMP_AFR_087-ENF", Name: "Acererak the Archlich",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				Finish: "Foil", FinishGroup: "Foil",
				Language: "English", CollectorNumber: "087",
				ScryfallID: "621a9aad-403e-412f-ad21-b90bc84bc696", TCGPlayerID: "247201",
				ProductType: ProductTypeSingles,
			},
			wantSet: "PAFR", wantNumber: "87a",
		},
		{
			desc: "a resale promo is the starred printing",
			p: CatalogProduct{
				SKU: "SGL-MTG-PRM-RESL_2024_118-ENF", Name: "Headless Rider",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				Finish: "Foil", FinishGroup: "Foil",
				Language: "English", CollectorNumber: "118",
				ScryfallID: "dc4dad13-45c1-4b64-a713-a5c003b70afa", TCGPlayerID: "263351",
				ProductType: ProductTypeSingles,
			},
			wantSet: "PVOW", wantNumber: "118\u2605",
		},
		{
			desc: "not the promo pack whose ids it borrows",
			p: CatalogProduct{
				SKU: "SGL-MTG-PRM-PP_VOW_118-ENF", Name: "Headless Rider",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				Finish: "Foil", FinishGroup: "Foil",
				Language: "English", CollectorNumber: "118",
				ScryfallID: "dc4dad13-45c1-4b64-a713-a5c003b70afa", TCGPlayerID: "263351",
				ProductType: ProductTypeSingles,
			},
			wantSet: "PVOW", wantNumber: "118p",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := resolveProduct(GameMagic, tt.p)
			if err != nil {
				t.Fatalf("resolveProduct(%s) = %v", tt.p.SKU, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNumber {
				t.Errorf("%s resolved to %s #%s, want %s #%s",
					tt.p.SKU, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
			}
		})
	}
}
