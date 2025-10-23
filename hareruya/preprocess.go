package hareruya

import (
	"errors"
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var reParens = regexp.MustCompile(`\(([^)]+)\)`)
var reBrackets = regexp.MustCompile(`\[([^\]]+)\]`)
var reSquares = regexp.MustCompile(`■([^■]+)■`)
var reJapanese = regexp.MustCompile(`[\p{Hiragana}\p{Katakana}\p{Han}]`)

var reCardName = regexp.MustCompile(`《([^》]+)》`)
var reThick = regexp.MustCompile(`【([^】]+)】`)

func Preprocess(product Product) (*mtgmatcher.InputCard, error) {
	if strings.Contains(product.ProductNameEN, "Wyvern back") ||
		strings.Contains(product.ProductNameEN, "Orversized") ||
		strings.Contains(product.ProductNameEN, "Oversized") ||
		strings.Contains(product.ProductNameEN, "Error Card") ||
		strings.Contains(product.ProductNameEN, "Error card") ||
		strings.Contains(product.ProductNameEN, "H19") ||
		strings.Contains(product.ProductNameEN, "Test Print") ||
		strings.Contains(product.ProductName, "Ultra Pro Puzzle") ||
		strings.Contains(strings.ToLower(product.CardName), "test print") {
		return nil, mtgmatcher.ErrUnsupported
	}

	cardName := product.CardName
	fixup, found := cardTable[cardName]
	if found {
		cardName = fixup
	}

	foil := product.FoilFlag == "1"
	var edition string
	var variant string
	var number string

	// Usually there is more information the JPN product line, but sometimes
	// we need to look at the English version too
	match := reBrackets.FindStringSubmatch(product.ProductName)
	if len(match) > 1 {
		edition = match[1]
		// Use the English information if present
		if reJapanese.MatchString(edition) {
			match = reBrackets.FindStringSubmatch(product.ProductNameEN)
			if len(match) > 1 {
				edition = match[1]
			}
		}
		edition = strings.Split(edition, "-")[0]
	}

	// Variant is always found in the English line
	match = reSquares.FindStringSubmatch(product.ProductNameEN)
	if len(match) > 1 {
		variant = match[1]
	}

	// The number is only found in the JPN line
	match = reParens.FindStringSubmatch(product.ProductName)
	if len(match) > 1 {
		number = match[1]
		number = strings.Split(number, "/")[0]
		number = strings.TrimLeft(number, "0")
	}
	if number != "" {
		if variant != "" {
			variant += " "
		}
		variant += number
	}

	fixup, found = editionTable[edition]
	if found {
		edition = fixup
	}

	switch edition {
	case "Judge Foil":
		if mtgmatcher.IsBasicLand(cardName) && strings.Contains(product.ProductNameEN, "Jacinto") {
			edition = "P23"
			variant = ""
		}
	case "IE", "CE":
		cardName = strings.TrimPrefix(cardName, "【International Edition】")
		cardName = strings.TrimPrefix(cardName, "【Collector's Edition】")

		variants := mtgmatcher.SplitVariants(product.ProductNameEN)
		if len(variants) > 1 {
			variant = variants[1]
		}
	case "SLD":
		if strings.Contains(product.ProductNameEN, "SLD Commander Deck") {
			edition = "PLST"
		}
	default:
		if strings.Contains(edition, "P Stamped_") {
			edition = "Promo Pack"
			fields := strings.Split(edition, "_")
			if len(fields) > 1 {
				edition += " " + fields[1]
			}
		} else if strings.Contains(product.ProductNameEN, "Prerelease") {
			edition += " Prerelease"
		}

		variant = strings.Replace(variant, "RetroF ", "Retro Frame ", 1)
		variant = strings.Replace(variant, "No Emblem", "No Symbol", 1)
		cardName = strings.TrimPrefix(cardName, "【Gold Frame】")
	}

	override, found := promoMap[edition][cardName][variant]
	if found {
		edition = override.Edition
		variant = override.Variant
	}

	language := ""
	if product.Language == "1" {
		language = "Japanese"
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      foil,
		Language:  language,
	}, nil
}

// process titles like
// 【EN】【Foil】(168)《武器製造/Weapons Manufacturing》[EOE] 赤R
// 【EN】【Foil】(086)■プレリリース■《虚空間渡り/Weftwalking》[EOE] 青R
func preprocess(title string) (*mtgmatcher.InputCard, error) {
	if strings.Contains(title, "Ultra Pro Puzzle") {
		return nil, mtgmatcher.ErrUnsupported
	}

	var cardName string
	var edition string
	var variant string
	var number string
	var foil bool

	title = strings.TrimPrefix(title, "【EN】")
	title = strings.Replace(title, "(Bottom)", "", -1)
	title = strings.Replace(title, "(Big Furry Monster)", "", -1)
	title = strings.Replace(title, "SDCC", "SDCC ", -1)
	title = strings.Replace(title, "No Emblem", "No Symbol", -1)

	// /Weapons Manufacturing
	matches := reCardName.FindStringSubmatch(title)
	if len(matches) > 1 {
		cardName = matches[1]
	}

	if strings.Contains(cardName, "/") {
		fields := strings.Split(cardName, "/")
		cardName = ""
		for _, field := range fields {
			if reJapanese.MatchString(field) {
				continue
			}
			cardName = field
		}
	}
	if cardName == "" {
		return nil, errors.New("invalid title format")
	}
	if reJapanese.MatchString(cardName) {
		return nil, mtgmatcher.ErrUnsupported
	}

	// [EOE]
	matches = reBrackets.FindStringSubmatch(title)
	if len(matches) > 1 {
		edition = matches[1]
		edition = strings.Split(edition, "-")[0]
	}

	// (168)
	matches = reParens.FindStringSubmatch(title)
	if len(matches) > 1 {
		number = matches[1]
		number = strings.Split(number, "/")[0]
		number = strings.TrimLeft(number, "0")
	}

	// ■プレリリース■
	matches = reSquares.FindStringSubmatch(title)
	if len(matches) > 1 {
		variant = matches[1]
		variant = strings.TrimSpace(variant)
	}

	//【Foil】/【エッチング・Foil】
	matches = reThick.FindStringSubmatch(title)
	if len(matches) > 1 {
		foil = strings.Contains(matches[1], "Foil")
		if matches[1] == "エッチング・Foil" {
			if variant != "" {
				variant += " "
			}
			variant += "Etched Foil"
		}
	}

	if number != "" {
		if variant != "" {
			variant += " "
		}
		variant += number
	}

	fixup, found := editionTable[edition]
	if found {
		edition = fixup
	}
	fixup, found = editionTable[variant]
	if found {
		variant = fixup
	}

	fixup, found = cardTable[cardName]
	if found {
		cardName = fixup
	}

	if strings.Contains(edition, "Pスタンプ_") ||
		strings.Contains(edition, "P Stamped_") ||
		strings.Contains(variant, "プロモスタンプ付") {
		edition = "Promo Pack"
		fields := strings.Split(edition, "_")
		if len(fields) > 1 {
			edition += " " + fields[1]
		}
	}

	//variant = strings.Replace(variant, "RetroF ", "Retro Frame ", 1)
	//cardName = strings.TrimPrefix(cardName, "【Gold Frame】")

	override, found := promoMap[edition][cardName][variant]
	if found {
		edition = override.Edition
		variant = override.Variant
	}

	if strings.Contains(edition, "WC9") || strings.Contains(edition, "WC0") || edition == "PT96" {
		fields := strings.Fields(title)
		var i int
		for i = len(fields) - 1; i == 0; i-- {
			if reJapanese.MatchString(fields[i]) {
				break
			}
		}

		if i > 0 {
			if variant != "" {
				variant += " "
			}
			variant += strings.Join(fields[i-1:], " ")
		}
	} else if strings.Contains(variant, "P30H") {
		edition = variant
	} else if strings.Contains(title, "プレリリース") {
		variant += " Prerelease"
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      foil,
	}, nil
}

var cardTable = map[string]string{
	"Chicken ? la King":                  "Chicken à la King",
	"Adorable | KittenAdorable | Kitten": "Adorable Kitten",
	"Tyrannosaurs Rex":                   "Tyrannosaurus Rex",
}

var editionTable = map[string]string{
	"2007年版ジャッジ褒賞":        "2007 Judge Rewards",
	"2010年版ジャッジ褒賞":        "2010 Judge Rewards",
	"2013年版ジャッジ褒賞":        "2013 Judge Rewards",
	"2015年版ジャッジ褒賞":        "2015 Judge Rewards",
	"2018年版ジャッジ褒賞":        "2018 Judge Rewards",
	"2020年版":              "2020 Edition",
	"30周年記念":              "30th Anniversary",
	"BOOKプロモ":             "Book Promo",
	"BOXプロモ":              "Box Promo",
	"CSP構築済み":             "CST",
	"DCIマーク":              "DCI Promo",
	"Etched Foil 30周年プロモ": "P30M etched frame",
	"GPプロモ":               "Grand Prix Promos",
	"MCQプロモ":              "MCQ Promo",
	"Nationalプロモ":         "National Promos",
	"PWシンボル付き再版":          "Mystery Booster/The List",
	"RPTQプロモ":             "RPTQ Promos",
	"URL入りイベントプロモ":        "PURL",
	"WMCQプロモ":             "WMCQ Promo",
	"その他プロモ":              "Other Promos",
	"アリーナ":                "Arena",
	"アルターネイト版":            "Alternate",
	"ウギンの運命":              "Ugin's Fate",
	"エッチング・Foil":          "Etched Foil",
	"エラーカード":              "Misprint",
	"ゲートウェイ":              "Gateway",
	"ゲームデー":               "Game Day",
	"コマンドフェスト":            "Command Fest",
	"ジャッジ褒賞":              "Judge Rewards",
	"スポットライトシリーズプロモ":      "Spotlight Series Promo",
	"テキストボックスレス ゲームデー":    "PCMP",
	"テキストレス Magic Fest":   "Textless Magic Fest",
	"テキストレス 褒賞プログラム":      "Textless Player Rewards",
	"テキストレス":              "Textless",
	"テストプリント":             "Test Print",
	"ヒストリープロモ":            " 30th Anniversary",
	"ファイレクシア語 その他プロモ":     "Phyrexian Other Promos",
	"ファイレクシア語 ジャッジ褒賞":     "Phyrexian Judge Reward",
	"フルアート 1":             "Full Art 1",
	"フルアート 2":             "Full Art 2",
	"フルアート コマンドフェスト":      "Fullart CommandFest",
	"プレリリース":              "Prerelease",
	"プロツアープロモ":            "Pro Tour Promos",
	"ボーダーレス Premier Play": "Borderless Premier Play",
	"ボーダーレス その他イベント記念":    "Borderless Other Event Commemoration",
	"ボーダーレス その他イベント記念系":   "Borderless Other Event",
	"ボーダーレス スポットライトシリーズプロモ": "Borderless Spotlight Series Promo",
	"ボーダーレス マーベル・レジェンドプロモ":  "LMAR",
	"ボーダーレス 褒賞プロモ":          "Borderless Player Rewards",
	"ボーダーレス":                "Borderless",
	"マジックリーグ":               "Year of the Tiger 2022",
	"メディア系プロモ":              "Media Promo",
	"リセールプロモ":               "Resale Promo",
	"大判カード":                 "Oversize",
	"対戦キット":                 "Clash Pack",
	"巳年プロモ":                 "Year of the Snake 2023",
	"拡張アート MagicConプロモ":     "Extended Art MagicCon Promo",
	"拡張アート その他プロモ":          "Extended Art Other Promos",
	"新枠 2008年版ジャッジ褒賞":       "Mordern Frame 2008 Judge Rewards",
	"旧枠 2000年版ジャッジ褒賞":       "Retro Frame 2000 Judge Rewards",
	"旧枠 その他プロモ":             "Retro Frame Other Promos",
	"旧枠 ヒストリープロモ":           "Retro Frame 30th Anniversary",
	"旧枠 褒賞プログラム":            "Old Frame Rewards Program",
	"旧枠":                    "Retro Frame",
	"褒賞プログラム":               "Rewards Program",
	"辰年プロモ":                 "Year of the Dragon 2024",

	"Secret Lair Showdown": "SLP",
	"Retro Frame Promos":   "PLG21",
	"30th Promo":           "P30A",
	"POS Reward Promo":     "PW24",
	"CMA":                  "CM1",

	"FNM": "Friday Night Magic",
}

var promoMap = map[string]map[string]map[string]struct {
	Edition string
	Variant string
}{
	"Other event promo": {
		"Swords to Plowshares": {
			"Borderless その他イベント記念系": {
				Edition: "PF25",
				Variant: "12",
			},
		},
	},
	"Other Event Promo": {
		"Swiftfoot Boots": {
			"卯年プロモ": {
				Edition: "PL23",
				Variant: "4",
			},
		},
	},
	"Other Event anniversary": {
		"Sol Ring": {
			"旧枠プロモ": {
				Edition: "PFDN",
				Variant: "1",
			},
		},
		"Vengevine": {
			"WMCQプロモ": {
				Edition: "WMC",
				Variant: "2013",
			},
		},
		"Sakura-Tribe Elder": {
			"E06": {
				Edition: "PJSE",
				Variant: "1E06",
			},
		},
		"Soltari Priest": {
			"E07": {
				Edition: "PJSE",
				Variant: "1E07",
			},
		},
		"Glorious Anthem": {
			"U08": {
				Edition: "PJSE",
				Variant: "1E08",
			},
		},
		"Steward of Valeron": {
			"URL入りイベントプロモ": {
				Edition: "PURL",
				Variant: "1",
			},
		},
		"Cryptic Command": {
			"MCQプロモ": {
				Edition: "PPRO",
				Variant: "2020-1",
			},
		},
		"Reya Dawnbringer": {
			"": {
				Edition: "P10E",
				Variant: "35",
			},
		},
		"Earl of Squirrel": {
			"": {
				Edition: "PUST",
				Variant: "108",
			},
		},
	},
	"Other Promos": {
		"Serra the Benevolent": {
			"Retro Frame その他プロモ": {
				Edition: "PF25",
				Variant: "1",
			},
		},
		"Ugin, the Spirit Dragon": {
			"Retro Frame その他プロモ": {
				Edition: "PF25",
				Variant: "6",
			},
		},
		"Ponder": {
			"その他プロモ": {
				Edition: "PF25",
				Variant: "2",
			},
		},
		"Sliver Hive": {
			"Retro Frame その他プロモ": {
				Edition: "PF25",
				Variant: "7",
			},
		},
		"Sakura-Tribe Elder": {
			"": {
				Edition: "PLG24",
				Variant: "1",
			},
		},
	},
	"PB・Draft Promos": {
		"Arcane Signet": {
			"Retro Frame PBドラフトプロモ": {
				Edition: "P30M",
				Variant: "1P",
			},
		},
		"Commander's Sphere": {
			"PBドラフトプロモ": {
				Edition: "PW24",
				Variant: "8",
			},
		},
		"Chaos Warp": {
			"PBドラフトプロモ": {
				Edition: "PW24",
				Variant: "7",
			},
		},
	},
	"Standard Showdown Promo": {
		"Monstrous Rage": {
			"Retro Frame Standard Showdown": {
				Edition: "PW25",
				Variant: "9",
			},
		},
	},
	"Showdown": {
		"Go for the Throat": {
			"Borderless ショーダウン": {
				Edition: "PCBB",
				Variant: "3",
			},
		},
	},
	"Spotlight Series Promo": {
		"Cloud, Midgar Mercenary": {
			"Borderless スポットライトシリーズプロモ": {
				Edition: "PPRO",
				Variant: "2025-1",
			},
		},
		"Terror of the Peaks": {
			"スポットライトシリーズプロモ": {
				Edition: "PSPL",
				Variant: "1",
			},
		},
	},
	"Commander Play": {
		"Palladium Myr": {
			"Retro Frame Commander Play": {
				Edition: "PW25",
				Variant: "6",
			},
		},
	},
	"Premier Play": {
		"Tifa Lockhart": {
			"Borderless Premier Play": {
				Edition: "PF25",
				Variant: "9",
			},
		},
	},
	"Magic Academy": {
		"Trinket Mage": {
			"Retro Frame Magic Academy": {
				Edition: "PW25",
				Variant: "8",
			},
		},
	},
	"MagicCon Promo": {
		"Sokka, Bold Boomeranger": {
			"Extended Art MagicConプロモ": {
				Edition: "PURL",
				Variant: "2025-4",
			},
		},
		"J. Jonah Jameson": {
			"Extended Art MagicConプロモ": {
				Edition: "PSPM",
				Variant: "3a",
			},
		},
	},
	"MagicFest": {
		"Lightning Bolt": {
			"": {
				Edition: "PF19",
				Variant: "1",
			},
		},
	},
	"Judge Foil": {
		"Demonic Tutor": {
			"2020年版": {
				Edition: "J20",
				Variant: "4",
			},
			"2008年版ジャッジ褒賞": {
				Edition: "G08",
				Variant: "3",
			},
		},
		"Vindicate": {
			"2013年版ジャッジ褒賞": {
				Edition: "G13",
				Variant: "7",
			},
		},
		"Wasteland": {
			"2015Ver. 2015年版ジャッジ褒賞": {
				Edition: "J15",
				Variant: "8",
			},
		},
	},
	"Game Day Promos": {
		"Mutavault": {
			"ゲームデー": {
				Edition: "PCMP",
				Variant: "12",
			},
		},
		"Serra Avenger": {
			"ゲームデー": {
				Edition: "PCMP",
				Variant: "6",
			},
		},
	},
	"P30A": {
		"Arcane Signet": {
			"30周年プロモ": {
				Edition: "P30M",
				Variant: "1F",
			},
		},
	},
	"SLP": {
		"Lightning Bolt": {
			"": {
				Edition: "SLP",
				Variant: "37",
			},
		},
	},
	"": {
		"Celestine Reef": {
			"その他プロモ": {
				Edition: "DCI",
				Variant: "42",
			},
		},
	},
	"Misprint": {
		"Laquatus's Champion": {
			"印刷ミス": {
				Edition: "PTOR",
				Variant: "67†a",
			},
		},
	},
}
