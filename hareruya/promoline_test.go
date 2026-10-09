package hareruya

import (
	"testing"
)

// TestRetailPromoLine pins the retail listings whose set tag says only that
// the printing is the set's promos rather than the set itself. Two shapes
// reach a printing the set does not hold: a wording naming which promo, and
// the tag on its own, which is all the storefront says for the Standard
// Showdown lands. Four of those five are not stocked today, so nothing but
// this exercises them.
func TestRetailPromoLine(t *testing.T) {
	for _, tt := range []struct {
		desc, jp, en, card, foil, image string
		wantSet, wantNumber             string
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
		{
			desc: "a deck card names the player it belonged to",
			jp:   "【金枠】《神の怒り/Wrath of God》[PT96] 白 Michael Loconto",
			en:   "【Gold Frame】《Wrath of God》[PT96] Michael Loconto",
			card: "【Gold Frame】Wrath of God", foil: "0",
			wantSet: "PTC", wantNumber: "ml58",
		},
		{
			desc: "and the storefront's spelling of a player is corrected",
			jp:   "【金枠】《闇への追放/Dark Banishing》[PT96] 黒 Leon Linback",
			en:   "【Gold Frame】《Dark Banishing》[PT96] Leon Linback",
			card: "【Gold Frame】Dark Banishing", foil: "0",
			wantSet: "PTC", wantNumber: "ll119",
		},
		{
			desc: "a deck's copies of a card are told apart by the art in the image",
			jp:   "【金枠】《森/Forest》[PT96] 土地(C) Preston Poulter",
			en:   "【Gold Frame】《Forest》[PT96](C) Preston Poulter",
			card: "【Gold Frame】Forest", foil: "0",
			image:   "https://files.hareruyamtg.com/img/goods/L/PTC/pp0377.jpg",
			wantSet: "PTC", wantNumber: "pp377",
		},
		{
			desc: "and 2001 images letter the first art where the catalog leaves it bare",
			jp:   "【金枠】《山/Mountain》[WC01] 土地 Jan Tomcani(343)INV",
			en:   "【Gold Frame】《Mountain》[WC01] Jan Tomcani(343)INV",
			card: "【Gold Frame】Mountain", foil: "0",
			image:   "https://files.hareruyamtg.com/img/goods/L/WC/2001/jt0343b.jpg",
			wantSet: "WC01", wantNumber: "jt343a",
		},
		{
			desc: "a serial numbered copy is the serialized printing",
			jp:   "【ダブルレインボウ・Foil】(401)■旧枠■《神無き祭殿/Godless Shrine》(シリアル入り)[RVR] 土地R",
			en:   "【DR・Foil】(401)■RetroF■《Godless Shrine》(serial number)[RVR]",
			card: "Godless Shrine", foil: "1",
			wantSet: "RVR", wantNumber: "401z",
		},
		{
			desc: "a convention promo leaves the set it was drawn from",
			jp:   "【Foil】《紅蓮の達人チャンドラ/Chandra, Pyromaster》(SDCC2014)[M15-P] 赤R",
			en:   "【Foil】《Chandra, Pyromaster》[SDCC]",
			card: "Chandra, Pyromaster", foil: "1",
			wantSet: "PS14", wantNumber: "134",
		},
		{
			desc: "a basic land's art letter is its place among the set's printings",
			jp:   "《沼/Swamp》(B)[DKM] 土地",
			en:   " 《Swamp》[DKM]  B",
			card: "Swamp", foil: "0",
			wantSet: "DKM", wantNumber: "43",
		},
		{
			desc: "and so it is where the letter follows the card name",
			jp:   "《島/Island》A（Light Blue） Mark Poole[IE]",
			en:   "《Island》A（Light Blue） Mark Poole[IE]",
			card: "Island", foil: "0",
			wantSet: "CEI", wantNumber: "291",
		},
		{
			desc: "or where it follows the set tag",
			jp:   "《森/Forest》[Summer Magic]B 土地",
			en:   "《Forest》[Summer Magic]B",
			card: "Forest", foil: "0",
			wantSet: "SUM", wantNumber: "305",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			b := withMagic(t)
			theCard, err := Preprocess(b, Product{
				ProductName: tt.jp, ProductNameEN: tt.en,
				CardName: tt.card, FoilFlag: tt.foil, ImageURL: tt.image,
			})
			if err != nil {
				t.Fatalf("Preprocess(%q) = %v", tt.jp, err)
			}
			cardID, err := b.Match(theCard)
			if err != nil {
				t.Fatalf("Match(%q) = %v", theCard, err)
			}
			co, err := b.GetUUID(cardID)
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
