package hareruya

import (
	"testing"
)

// TestTitleTreatments pins the treatments the storefront announces in the
// group the plain finish otherwise occupies. The treated printing shares its
// set tag and collector number with the plain one, so a marker read as a
// bare "Foil" prices the treatment as the plain card - and these are the
// printings the shop pays the most for. Each pair below is the same card
// bought twice, once with the marker and once without.
func TestTitleTreatments(t *testing.T) {
	for _, tt := range []struct {
		desc, title, wantSet, wantNumber string
	}{
		{
			desc:    "the surge foil is its own printing of the set it reprints",
			title:   "【EN】【サージ・Foil】(166)《久遠なる栄光の笏/Sceptre of Eternal Glory》[40K-SF] 茶R",
			wantSet: "40K", wantNumber: "166★",
		},
		{
			desc:    "and the same card bought plain stays the plain printing",
			title:   "【EN】【Foil】(166)《久遠なる栄光の笏/Sceptre of Eternal Glory》[40K] 茶R",
			wantSet: "40K", wantNumber: "166",
		},
		{
			desc:    "the step-and-compleat foil carries the marked number",
			title:   "【EN】【S&C・Foil】(681)■ボーダーレス■《影生まれの使徒/Shadowborn Apostle》[SLD] 黒",
			wantSet: "SLD", wantNumber: "681Φ",
		},
		{
			desc:    "and the plain Secret Lair printing keeps the bare number",
			title:   "【EN】【Foil】(681)■ボーダーレス■《影生まれの使徒/Shadowborn Apostle》[SLD] 黒",
			wantSet: "SLD", wantNumber: "681",
		},
		{
			desc:    "the etched foil the table already named still resolves",
			title:   "【EN】【エッチング・Foil】(1072)■旧枠■《オパールのモックス/Mox Opal》[SLD] 茶R",
			wantSet: "SLD", wantNumber: "1072",
		},
		{
			// The marker travels with the number and the frame, so the
			// variant is never equal to it on a real listing: every
			// serialized printing but the excepted Sol Ring read as the
			// plain card it is numbered beside.
			desc:    "the double rainbow foil is the serialized printing",
			title:   "買取：【ダブルレインボウ・Foil】(381)■ボーダーレス■《再誕世界、エムラクール/Emrakul, the World Anew》[MH3-BF] 無R",
			wantSet: "MH3", wantNumber: "381z",
		},
		{
			desc:    "and the same card bought plain stays the borderless one",
			title:   "買取：【Foil】(381)■ボーダーレス■《再誕世界、エムラクール/Emrakul, the World Anew》[MH3-BF] 無R",
			wantSet: "MH3", wantNumber: "381",
		},
		{
			// The catalog files the six sizes of this card as 82a to 82f
			// and names them by the pair the title states, so keeping only
			// the half in front of the slash answered all three with one.
			desc:    "the power and toughness names the variant it belongs to",
			title:   "買取：【Foil】《Garbage Elemental》(3/1)[UST] 赤U",
			wantSet: "UST", wantNumber: "82b",
		},
		{
			desc:    "and the one bought beside it is its own printing",
			title:   "買取：【Foil】《Garbage Elemental》(3/3)[UST] 赤U",
			wantSet: "UST", wantNumber: "82d",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			b := withMagic(t)
			in, err := preprocess(b, tt.title)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", tt.title, err)
			}
			id, err := b.Match(in)
			if err != nil {
				t.Fatalf("Match(%q) = %v", in, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNumber {
				t.Errorf("Match(%q) = %s|%s, want %s|%s", tt.title, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
			}
		})
	}
}

// TestRetailTreatments pins the retail line reading the same markers the
// buylist does: the finish group on the Japanese title, and the stamp the
// English one squares. Without them the listing lands on the printing the
// treated one is numbered beside.
func TestRetailTreatments(t *testing.T) {
	for _, tt := range []struct {
		desc                 string
		product              Product
		wantSet, wantNumber  string
		wantFoil, wantEtched bool
	}{
		{
			desc: "the etched foil is its own finish of the card",
			product: Product{
				ProductName:   "【エッチング・Foil】(048)《豊穣な収穫/Abundant Harvest》[STA-BF] 緑R",
				ProductNameEN: "【Foil Etched】《Abundant Harvest》[STA]",
				CardName:      "Abundant Harvest", FoilFlag: "1", Language: "2",
			},
			wantSet: "STA", wantNumber: "48", wantEtched: true,
		},
		{
			desc: "and the plain foil bought beside it stays the foil",
			product: Product{
				ProductName:   "【Foil】(048)《豊穣な収穫/Abundant Harvest》[STA] 緑R",
				ProductNameEN: "【Foil】《Abundant Harvest》[STA]",
				CardName:      "Abundant Harvest", FoilFlag: "1", Language: "2",
			},
			wantSet: "STA", wantNumber: "48", wantFoil: true,
		},
		{
			desc: "the surge foil is its own printing of the set it reprints",
			product: Product{
				ProductName:   "【サージ・Foil】(286)《オパールの宮殿/Opal Palace》[40K-SF] 土地C",
				ProductNameEN: "【SurgeFoil】《Opal Palace》[40K]",
				CardName:      "Opal Palace", FoilFlag: "1", Language: "2",
			},
			wantSet: "40K", wantNumber: "286★", wantFoil: true,
		},
		{
			desc: "the stamped copy is the promo pack's, not the set's",
			product: Product{
				ProductName:   "【Foil】(123)■プロモスタンプ付■《嵐鱗の末裔/Stormscale Scion》[TDM] 赤R",
				ProductNameEN: "【Foil】(123)■Promo Stamped■《Stormscale Scion》[TDM]",
				CardName:      "Stormscale Scion", FoilFlag: "1", Language: "2",
			},
			wantSet: "PTDM", wantNumber: "123p", wantFoil: true,
		},
		{
			desc: "the Secret Lair dazzle foil is the Pool Party printing",
			product: Product{
				ProductName:   "【Pool Party・Foil】(2XM-080)《命取りの論争/Deadly Dispute》[SLD] 黒R",
				ProductNameEN: "【Pool Party・Foil】(2XM-080)《Deadly Dispute》[SLD]",
				CardName:      "Deadly Dispute", FoilFlag: "1", Language: "2",
			},
			wantSet: "SLD", wantNumber: "IFIYW-6", wantFoil: true,
		},
		{
			desc: "the prerelease square is read off the Japanese title",
			product: Product{
				ProductName:   "【Foil】■プレリリース■《砂塵破/Duneblast》[KTK-PRE] 金R",
				ProductNameEN: "【Foil】◆Prereleace◆《Duneblast》[KTK-PRE]",
				CardName:      "Duneblast", FoilFlag: "1", Language: "2",
			},
			wantSet: "PKTK", wantNumber: "174s", wantFoil: true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			b := withMagic(t)
			in, err := Preprocess(b, tt.product)
			if err != nil {
				t.Fatalf("Preprocess(%q) = %v", tt.product.ProductName, err)
			}
			id, err := b.Match(in)
			if err != nil {
				t.Fatalf("Match(%q) = %v", in, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNumber || co.Foil != tt.wantFoil || co.Etched != tt.wantEtched {
				t.Errorf("Match(%q) = %s|%s foil=%t etched=%t, want %s|%s foil=%t etched=%t", in, co.SetCode, co.Number, co.Foil, co.Etched,
					tt.wantSet, tt.wantNumber, tt.wantFoil, tt.wantEtched)
			}
		})
	}
}
