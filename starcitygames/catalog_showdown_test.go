package starcitygames

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestShowdownAndStoreChampionship pins the two shelves where an event's own
// promos were being priced as somebody else's printing. Fixtures are copied
// from the export verbatim, shared identifiers and all.
func TestShowdownAndStoreChampionship(t *testing.T) {
	withMagic(t)

	for _, tt := range []struct {
		desc                string
		p                   CatalogProduct
		wantSet, wantNumber string
		wantRefused         bool
	}{
		{
			desc: "a 2024 Standard Showdown land is that year's event set",
			p: CatalogProduct{
				SKU: "SGL-MTG-PRM-SSD_2024_001-ENF", Name: "Plains",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				Finish: "Foil", FinishGroup: "Foil",
				Language: "English", CollectorNumber: "001",
				ScryfallID: "36da00e3-3ef6-4ad5-a53d-e71cfdafc1e6", TCGPlayerID: "193283",
				ProductType: ProductTypeSingles,
			},
			wantSet: "PSS4", wantNumber: "1", wantRefused: false,
		},
		{
			desc: "not the 2019 promo pack whose ids it borrows",
			p: CatalogProduct{
				SKU: "SGL-MTG-PRM-PP_2019_001-ENF", Name: "Plains",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				Finish: "Foil", FinishGroup: "Foil",
				Language: "English", CollectorNumber: "001",
				ScryfallID: "36da00e3-3ef6-4ad5-a53d-e71cfdafc1e6", TCGPlayerID: "193283",
				ProductType: ProductTypeSingles,
			},
			wantSet: "PPP1", wantNumber: "1", wantRefused: false,
		},
		{
			desc: "the 2017 event keeps its own set",
			p: CatalogProduct{
				SKU: "SGL-MTG-PRM-SSD_2017_001-ENF", Name: "Plains",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				Finish: "Foil", FinishGroup: "Foil",
				Language: "English", CollectorNumber: "SSD_2017_001",
				ScryfallID: "35bad4e4-d961-49cd-90da-aa4ed79d9ebb", TCGPlayerID: "146690",
				ProductType: ProductTypeSingles,
			},
			wantSet: "PSS2", wantNumber: "1", wantRefused: false,
		},
		{
			desc: "a Store Championship promo is the foil it was struck as",
			p: CatalogProduct{
				SKU: "SGL-MTG-PRM-SCHP_2025_041-ENF", Name: "City of Brass",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				Finish: "Foil", FinishGroup: "Foil",
				Language: "English", CollectorNumber: "041",
				ScryfallID: "5317801d-8ad0-466a-9cd6-f97f16cc7c06", TCGPlayerID: "641612",
				ProductType: ProductTypeSingles,
			},
			wantSet: "SCH", wantNumber: "41", wantRefused: false,
		},
		{
			desc: "and the shop's unfoiled listing of one is not a card",
			p: CatalogProduct{
				SKU: "SGL-MTG-PRM-SCHP_2025_041-ENN", Name: "City of Brass",
				Game: "Magic: The Gathering", Set: "Promo", Rarity: "Promo",
				Finish: "Non-foil", FinishGroup: "Non-foil",
				Language: "English", CollectorNumber: "041",
				ScryfallID: "", TCGPlayerID: "",
				ProductType: ProductTypeSingles,
			},
			wantSet: "", wantNumber: "", wantRefused: true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := resolveProduct(GameMagic, tt.p)
			if tt.wantRefused {
				if !errors.Is(err, mtgmatcher.ErrUnsupported) {
					t.Fatalf("resolveProduct(%s) = (%q, %v), want ErrUnsupported", tt.p.SKU, id, err)
				}
				return
			}
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
