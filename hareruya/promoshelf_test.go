package hareruya

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPromoShelf pins the retail promo shelf's newer wordings to the
// printings they name: the Wizards Play Network and Standard Showdown
// promos of 2025 and 2026, filed by year; a judge foil told apart by the
// year in its title; and a prerelease card the set numbers among its own,
// which every set since Murders at Karlov Manor does.
func TestPromoShelf(t *testing.T) {
	if len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("ALLPRINTINGS5_PATH not set; skipping the promo shelf suite")
	}

	for _, tt := range []struct {
		desc, jp, en, card, foil string
		wantSet, wantNumber      string
	}{
		{
			desc: "a Final Fantasy Standard Showdown promo",
			jp:   "【Foil】■ボーダーレス■《SeeDの傭兵、スコール/Squall, SeeD Mercenary》(スタンダード・ショーダウン)[流星マーク] 金",
			en:   "【Foil】■Borderless■《Squall, SeeD Mercenary》[Showdown Promo]",
			card: "Squall, SeeD Mercenary", foil: "1",
			wantSet: "PSS5", wantNumber: "2",
		},
		{
			desc: "a 2026 commander event promo in the retro frame",
			jp:   "■旧枠■《彼方地のエルフ/Farhaven Elf》(コマンダーイベントプロモ)[流星マーク] 緑",
			en:   "■RetroF■《Farhaven Elf》[Commander Event Promo]",
			card: "Farhaven Elf", foil: "0",
			wantSet: "PW26", wantNumber: "2",
		},
		{
			desc: "a 2025 commander event promo in full art",
			jp:   "【Foil】■フルアート■《統率の塔/Command Tower》(コマンダーイベントプロモ)[流星マーク] 土地",
			en:   "【Foil】■Full-Art■《Command Tower》[Commander Event Promo]",
			card: "Command Tower", foil: "1",
			wantSet: "PW25", wantNumber: "17",
		},
		{
			desc: "a judge foil told apart by the year in its title",
			jp:   "【Foil】《吸血の教示者/Vampiric Tutor》(2018年版ジャッジ褒賞)[流星マーク] 黒",
			en:   "【Foil】■2018Ver.■《Vampiric Tutor》 [Judge Foil]",
			card: "Vampiric Tutor", foil: "1",
			wantSet: "J18", wantNumber: "2",
		},
		{
			desc: "a prerelease card the set numbers among its own",
			jp:   "【Foil】(431)《法の行使者、トミク/Tomik, Wielder of Law》(プレリリース)[MKM-P] 金",
			en:   "【Foil】(431)《Tomik, Wielder of Law》(Prerelease)[MKM-P]",
			card: "Tomik, Wielder of Law", foil: "1",
			wantSet: "MKM", wantNumber: "431",
		},
		{
			desc: "a prerelease card of a set that files them on its promo line",
			jp:   "【Foil】《Water Gun Balloon Game》(プレリリース)[UNF-P] 茶",
			en:   "【Foil】《Water Gun Balloon Game》[Prerelease]",
			card: "Water Gun Balloon Game", foil: "1",
			wantSet: "UNF", wantNumber: "538",
		},
		{
			desc: "the first Godzilla print, before the name was changed",
			jp:   "【Foil】(373a)■ゴジラ■《虚空の侵略者、スペースゴジラ/Spacegodzilla, Void Invader》[IKO-BF] 黒U",
			en:   "【Foil】■First edition■《Spacegodzilla, Void Invader》/《Void Beckoner》[IKO-BF]",
			card: "Void Beckoner", foil: "1",
			wantSet: "IKO", wantNumber: "373",
		},
		{
			desc: "a game day promo filed by its set's tag",
			jp:   "■テキストボックスレス■《傲慢な完全者/Imperious Perfect》(ゲームデー)[LRW-P] 緑U",
			en:   "《Imperious Perfect》[Game Day Promos]",
			card: "Imperious Perfect", foil: "0",
			wantSet: "PCMP", wantNumber: "9",
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
				t.Errorf("Match(%q) = %s %s, want %s %s", theCard, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
			}
		})
	}
}

// TestBuylistPromoShelf pins the buylist's spellings of the same shelf,
// which names it by its shooting-star mark and the card by the program
// that handed it out, and the Pool Party drop's dazzle foil, told from the
// plain foil only by the marker in the foil's place.
func TestBuylistPromoShelf(t *testing.T) {
	if len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("ALLPRINTINGS5_PATH not set; skipping the buylist promo shelf suite")
	}
	for _, tt := range []struct {
		title, wantSet, wantNumber string
	}{
		{"【EN】【Foil】《止められぬ斬鬼/Unstoppable Slasher》(ジャパンスタンダードカッププロモ)[流星マーク] 黒", "PJSC", "2026-2"},
		{"【EN】【Foil】■ボーダーレス■《ゾンビ使い/Zombie Master》(褒賞プロモ)[流星マーク] 黒", "PW24", "3"},
		{"【EN】【Foil】■拡張アート■《恐れを知らぬ者、カタラ/Katara, the Fearless》(MagicConプロモ)[流星マーク] 金", "PURL", "2025-3"},
		{"【EN】《紅蓮破/Pyroblast》[流星マーク] 赤", "PW23", "8"},
		{"【EN】【Pool Party・Foil】(SCD-288)《太陽の指輪/Sol Ring》[SLD] 茶R", "SLD", "IFIYW-10"},
		{"【EN】(SCD-288)《太陽の指輪/Sol Ring》[SLD] 茶R", "SLD", "IFIYW-5"},
		{"【EN】【Foil】(2062)■ボーダーレス■《Chancla relámpagos》//《稲妻のすね当て/Lightning Greaves》[SLD] 茶", "SLD", "2062★"},
		{"【EN】(2062)■ボーダーレス■《Chancla relámpagos》//《稲妻のすね当て/Lightning Greaves》[SLD] 茶", "SLD", "2062"},
	} {
		t.Run(tt.title, func(t *testing.T) {
			theCard, err := preprocess(tt.title)
			if err != nil {
				t.Fatalf("preprocess(%q) = %v", tt.title, err)
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
				t.Errorf("Match(%q) = %s %s, want %s %s", theCard, co.SetCode, co.Number, tt.wantSet, tt.wantNumber)
			}
		})
	}
}
