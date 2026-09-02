package hareruya

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestRetailPromoLine pins the retail listings whose set tag says only that
// the printing is the set's promos rather than the set itself. Two shapes
// reach a printing the set does not hold: a wording naming which promo, and
// the tag on its own, which is all the storefront says for the Standard
// Showdown lands. Four of those five are not stocked today, so nothing but
// this exercises them.
func TestRetailPromoLine(t *testing.T) {
	if len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("ALLPRINTINGS5_PATH not set; skipping the promo line suite")
	}

	for _, tt := range []struct {
		desc, jp, en, card, foil string
		wantSet, wantNumber      string
	}{
		{
			desc: "the tag alone names the Standard Showdown packs",
			jp:   "《梢の眺望/Canopy Vista》[BFZ-P] 土地",
			en:   "《Canopy Vista》[Other Promos]",
			card: "Canopy Vista", foil: "0",
			wantSet: "PSS1", wantNumber: "234",
		},
		{
			desc: "and so it does for the four beside it that are not stocked",
			jp:   "《燃えがらの林間地/Cinder Glade》[BFZ-P] 土地",
			en:   "《Cinder Glade》[Other Promos]",
			card: "Cinder Glade", foil: "0",
			wantSet: "PSS1", wantNumber: "235",
		},
		{
			desc: "prairie stream",
			jp:   "《大草原の川/Prairie Stream》[BFZ-P] 土地",
			en:   "《Prairie Stream》[Other Promos]",
			card: "Prairie Stream", foil: "0",
			wantSet: "PSS1", wantNumber: "241",
		},
		{
			desc: "smoldering marsh",
			jp:   "《燻る湿地/Smoldering Marsh》[BFZ-P] 土地",
			en:   "《Smoldering Marsh》[Other Promos]",
			card: "Smoldering Marsh", foil: "0",
			wantSet: "PSS1", wantNumber: "247",
		},
		{
			desc: "sunken hollow",
			jp:   "《窪み渓谷/Sunken Hollow》[BFZ-P] 土地",
			en:   "《Sunken Hollow》[Other Promos]",
			card: "Sunken Hollow", foil: "0",
			wantSet: "PSS1", wantNumber: "249",
		},
		{
			desc: "the plain card the shop sells beside them keeps its set",
			jp:   "【Foil】《梢の眺望/Canopy Vista》[BFZ] 土地R",
			en:   "【Foil】《Canopy Vista》[BFZ]",
			card: "Canopy Vista", foil: "1",
			wantSet: "BFZ", wantNumber: "234",
		},
		{
			desc: "a gift box promo is filed in the set's promos, not the set",
			jp:   "【Foil】《鎌豹/Scythe Leopard》(ギフトボックス)[BFZ-P] 緑U",
			en:   "【Foil】《Scythe Leopard》[Gift Box]",
			card: "Scythe Leopard", foil: "1",
			wantSet: "PBFZ", wantNumber: "188",
		},
		{
			desc: "dreg mangler",
			jp:   "【Foil】《屑肉の刻み獣/Dreg Mangler》(ギフトボックス)[RTR-P] 金U",
			en:   "【Foil】《Dreg Mangler》[Gift Box]",
			card: "Dreg Mangler", foil: "1",
			wantSet: "PRTR", wantNumber: "158",
		},
		{
			desc: "sultai charm",
			jp:   "【Foil】《スゥルタイの魔除け/Sultai Charm》(ギフトボックス)[KTK-P] 金U",
			en:   "【Foil】《Sultai Charm》[Gift Box]",
			card: "Sultai Charm", foil: "1",
			wantSet: "PKTK", wantNumber: "204",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			theCard, err := Preprocess(Product{
				ProductName: tt.jp, ProductNameEN: tt.en,
				CardName: tt.card, FoilFlag: tt.foil,
			})
			if err != nil {
				t.Fatalf("Preprocess(%q) = %v", tt.jp, err)
			}
			cardID, err := mtgmatcher.Match(theCard)
			if err != nil {
				t.Fatalf("Match(%q) = %v", theCard, err)
			}
			co, err := mtgmatcher.GetUUID(cardID)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNumber {
				t.Errorf("%q landed on %s #%s, want %s #%s",
					tt.jp, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
			}
		})
	}
}
