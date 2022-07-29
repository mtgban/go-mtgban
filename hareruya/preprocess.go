package hareruya

import (
	"errors"
	"strings"
	"unicode"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var titleTable = map[string]string{
	"【EN】【Foil】《Kytheon, Hero of Akros》/【Foil】《Gideon, Battle-Forged》[SDCC](2015)":           "【EN】【Foil】《アクロスの英雄、キテオン/Kytheon, Hero of Akros》[SDCC](2015)",
	"【EN】【Foil】《Jace, Vryn's Prodigy》/【Foil】《Jace, Telepath Unbound》[SDCC](2015)":            "【EN】【Foil】《ヴリンの神童、ジェイス/Jace, Vryn's Prodigy》[SDCC](2015)",
	"【EN】【Foil】《Liliana, Heretical Healer》/【Foil】《Liliana, Defiant Necromancer》[SDCC](2015)": "【EN】【Foil】《異端の癒し手、リリアナ/Liliana, Heretical Healer》[SDCC](2015) 黒",
	"【EN】【Foil】《Chandra, Fire of Kaladesh》/【Foil】《Chandra, Roaring Flame》[SDCC](2015)":       "【EN】【Foil】《カラデシュの火、チャンドラ/Chandra, Fire of Kaladesh》[SDCC](2015) 赤",
	"【EN】【Foil】《Nissa, Vastwood Seer》/【Foil】《Nissa, Sage Animist》[SDCC](2015)":               "【EN】【Foil】《巨森の予見者、ニッサ/Nissa, Vastwood Seer》[SDCC](2015) 緑",
}

var cardTable = map[string]string{
	"S?ance":   "Seance",
	"Ragrfire": "Ragefire",
}

var missingTable = map[string]string{
	"Odric, Master Tactician":   "E01",
	"Fire // Ice":               "APC",
	"Sheoldred, Whispering One": "PNPH",
	"Wurmcoil Engine":           "PSOM",
	"Richard Garfield, Ph.D.":   "UNH",
	"Booster Tutor":             "UNH",
	"Blast from the Past":       "UNH",
	"Old Fogey":                 "UNH",
	"Ink-Eyes, Servant of Oni":  "PBOK",
}

var promoTable = map[string]string{
	"Jace Beleren":      "PBOOK",
	"Liliana Vess":      "PDP10",
	"Phyrexian Rager":   "PMEI",
	"Warmonger":         "PMEI",
	"Mana Crypt":        "PHPR",
	"Underworld Dreams": "P2HG",

	"Garruk Wildspeaker": "PDTP",
	"Grave Titan":        "PDP12",
	"Primordial Hydra":   "PDP13",

	"Balduvian Horde":             "PWOR",
	"Flooded Strand":              "PNAT",
	"Hall of Triump":              "PJOU",
	"Char":                        "P15A",
	"Reliquary Tower":             "PLG20",
	"Sword of Dungeons & Dragons": "H17",
	"Steward of Valeron":          "PURL",
	"Knight Exemplar":             "PRES",
	"Incinerate":                  "PLGM",
	"Crystalline Sliver":          "F03",
	"Reya Dawnbringer":            "P10E",

	"Naya Sojourners":              "PM10",
	"Emeria Angel":                 "PZEN",
	"Valakut, the Molten Pinnacle": "PZEN",
	"Kalastria Highborn":           "PWWK",
	"Black Sun's Zenith":           "PMBS",
	"Deathless Angel":              "PROE",
	"Pia and Kiran Nalaar":         "PORI",
	"Deeproot Champion":            "PXLN",
	"Unclaimed Territory":          "PXLN",
	"Walk the Plank":               "PXLN",
	"Earl of Squirrel":             "PUST",
	"Ghalta, Primal Hunger":        "PRIX",
	"Steel Leaf Champion":          "PDOM",
	"Llanowar Elves":               "PDOM",
	"Zahid, Djinn of the Lamp":     "PDOM",
	"Demon of Catastrophes":        "PM19",
	"Death Baron":                  "PM19",
	"Rakdos Firewheeler":           "PRNA",
	"Firemind's Research":          "PGRN",
	"Anguished Unmaking":           "PSOI",
	"Negate":                       "PM20",
	"Disfigure":                    "PM20",
	"Thrashing Brontodon":          "PM20",
	"Flame Sweep":                  "PM20",
	"Corpse Knight":                "PM20",
}

var editionTable = map[string]string{
	"AvN": "DDH",
	"BvC": "DDQ",
	"EvI": "DDU",
	"EvK": "DDO",
	"EvT": "DDF",
	"HvM": "DDL",
	"IvG": "DDJ",
	"JvV": "DDM",
	"KvD": "DDG",
	"MvG": "DDT",
	"MvM": "DDS",
	"NvO": "DDR",
	"PvC": "DDE",
	"SvC": "DDN",
	"SvT": "DDK",
	"VvK": "DDI",
	"ZvE": "DDP",

	"DD3・DvD": "DVD",
	"DD3・EvG": "EVG",
	"DD3・GvL": "GVL",
	"DD3・JvC": "JVC",

	"CE": "CED",
	"IE": "CEI",

	"10ED":  "10E",
	"ANN":   "E01",
	"CMA":   "CM1",
	"CMA17": "CMA",
	"DoP":   "DPA",
	"FAL":   "PD2",
	"GRB":   "PD3",
	"MED14": "MD1",
	"P12":   "PC2",
	"PCS":   "HOP",
	"PO2":   "P02",
	"UBT":   "PUMA",

	"FNM":    "Friday Night Magic",
	"2XM-BT": "Double Masters: Extras",

	"MED-GRN": "MED",
	"MED-RNA": "MED",
	"MED-WAR": "MED",

	"Old Arena": "PARL",
	"ONS Arena": "PAL03",
	"MRD Arena": "PAL04",
	"CHK Arena": "PAL05",
	"RAV Arena": "PAL06",

	"CSP構築済":             "CST",
	"CSP Theme Deck":     "CST",
	"World Championship": "PWOR",
	"Basic Set Promos":   "G18",

	"対戦キット":   "Clash Pack",
	"GPプロモ":   "PGPX",
	"アリーナ":    "Arena",
	"BOXプロモ":  "Box Promo",
	"褒賞プログラム": "Reward Program",
	"ゲートウェイ":  "Gateway",
	"ウギンの運命":  "Ugin's Fate",

	"アンパサンド・カード": "PAFR",
	"旧正月プロモ":     "Lunar New Year",
	"旧枠プロモ":      "Retro Frame",
}

func preprocess(title string) (*mtgmatcher.Card, error) {
	// For the hardest cases
	lut, found := titleTable[title]
	if found {
		title = lut
	}

	// Trim language tag
	if !strings.HasPrefix(title, "【EN】") {
		return nil, errors.New("non-english")
	}
	title = strings.TrimPrefix(title, "【EN】")
	title = strings.TrimSpace(title)

	// Like for 4th ed
	if strings.HasPrefix(title, "【Alternate】") ||
		strings.HasPrefix(title, "【アルターネイト版】") {
		return nil, errors.New("unsupported")
	}

	// Trim foil tag
	isFoil := false
	if strings.HasPrefix(title, "【Foil】") {
		isFoil = true
		title = strings.TrimPrefix(title, "【Foil】")
	} else if strings.HasPrefix(title, "【Non-Foil】") {
		title = strings.TrimPrefix(title, "【Non-Foil】")
	}
	title = strings.TrimSpace(title)

	// Parenthesis variant can be anywhere, in the middle of the title
	// or at the end, like here, trim it
	variant := ""
	if strings.HasSuffix(title, ")") {
		if !strings.Contains(title, "Hazmat Suit") &&
			!strings.Contains(title, "B.F.M") &&
			!strings.Contains(title, "Erase") {
			idx := strings.LastIndex(title, "(")
			if idx > 0 {
				variant = title[idx+1:]
				variant = strings.TrimSuffix(variant, ")")
				title = title[:idx]
			}
		}
	}

	// Prefix for special cards
	for _, symbol := range []string{"■", "◆"} {
		if strings.Contains(title, symbol) {
			fields := strings.Split(title, symbol)
			if len(fields) > 2 {
				title = fields[2]
				if variant != "" {
					variant += " "
				}
				variant += fields[1]
			}
		}
	}

	// Double faced cards (+ handle typo)
	if strings.Contains(title, "》/ 《") {
		title = strings.Replace(title, "》/ 《", "》/《", 1)
	}
	if strings.Contains(title, "》/《") {
		title = strings.Replace(title, "》/《", " // ", 1)
	}

	// Separate name from edition (which may contain some variants)
	fields := strings.Split(title, "》")
	cardName := fields[0]

	// In case there is anything *before* the real name, like for
	// `【EN】075《Teferi, Master of Time》[M21]`
	if !strings.HasPrefix(cardName, "《") {
		subfields := strings.Split(cardName, "《")
		if subfields[0] != "" {
			if variant != "" {
				variant += " "
			}
			variant += subfields[0]
		}
		if len(subfields) > 1 {
			cardName = subfields[1]
		}
	} else {
		cardName = strings.TrimPrefix(cardName, "《")
	}
	cardName = strings.TrimSpace(cardName)

	lut, found = cardTable[cardName]
	if found {
		cardName = lut
	}

	edition := ""
	if len(fields) > 1 {
		edition = fields[1]
		subfields := strings.Split(edition, "[")

		if subfields[0] != "" {
			if variant != "" {
				variant += " "
			}
			variant += subfields[0]
			variant = strings.Replace(variant, "(", " ", -1)
			variant = strings.Replace(variant, ")", " ", -1)
			variant = strings.Replace(variant, "  ", " ", -1)
		}

		// Split again, remove anything past the edition, except years
		// `【EN】《Order of Leitbur》Man [FEM] B`
		// `【EN】【Foil】《Demonic Tutor》[Judge Foil] 2020ver`
		if len(subfields) > 1 {
			subsubfields := strings.Split(subfields[1], "]")
			edition = subsubfields[0]
			if len(subsubfields) > 1 {
				maybeYear := subsubfields[1]
				if strings.HasSuffix(maybeYear, "ver") {
					maybeYear = strings.TrimSuffix(maybeYear, "ver")
				} else if strings.HasSuffix(maybeYear, "年版") {
					maybeYear = strings.TrimSuffix(maybeYear, "年版")
				}
				maybeYear = mtgmatcher.ExtractYear(maybeYear)
				if variant != "" {
					variant += " "
				}
				variant += maybeYear

				// Need to handle stuff like :(
				// `【EN】【Foil】《Island》[Other Event anniversary](Ravnica Weekend Dimir) A01/010`
				if mtgmatcher.IsBasicLand(cardName) {
					if variant != "" {
						variant += " "
					}
					variant += subsubfields[1]
					// I don't like unicode no more :(
					variant = strings.Replace(variant, "）", " ", -1)
					variant = strings.Replace(variant, "（", " ", -1)
				}
			}
		}
	}

	// Due to cards like
	// `【EN】《Bruna, the Fading Light》/《Brisela, Voice of Nightmares (Bottom)》[EMN]`
	fields = mtgmatcher.SplitVariants(cardName)
	cardName = fields[0]
	if len(fields) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += fields[1]
	}

	// Buylist mode and other random cards
	// `【EN】《蒸気打ちの親分/Steamflogger Boss》[UST]`
	if strings.Contains(cardName, "/") {
		for _, runeValue := range cardName {
			// Only do this replacement if there is Japanese text, otherwise,
			// assume a typo and replace the slash with the expected char
			if unicode.In(runeValue, unicode.Han, unicode.Hiragana, unicode.Katakana) {
				subfields := strings.Split(cardName, "/")
				if len(subfields) > 1 {
					cardName = subfields[1]
				}
			} else if !strings.Contains(cardName, "//") {
				cardName = strings.Replace(cardName, "/", "+", 1)
			}
			break
		}
	}

	//【EN】《Boom+Bust》[PLC]
	if strings.Contains(cardName, "+") {
		cardName = strings.Replace(cardName, "+", " // ", 1)
	}

	if edition == "エラーカード" || // misprint
		strings.Contains(cardName, "トークン") { // token
		return nil, errors.New("non single")
	}

	switch edition {
	case "":
		// `【EN】 《EURO1 《Plains》》[Euro Lands]`
		if strings.HasPrefix(cardName, "EURO") {
			fields := strings.Split(cardName, "《")
			variant = fields[0]
			if len(fields) > 1 {
				cardName = fields[1]
			}
			edition = "PELP"
		} else {
			edition = missingTable[cardName]
			if edition == "" {
				return nil, errors.New("non single")
			}
		}
	case "Test print":
		return nil, errors.New("unsupported")
	case "SDCC":
		// `【EN】【Foil】《紅蓮の達人チャンドラ/Chandra, Pyromaster》(SDCC2013)[SDCC] 赤`
		variant = strings.Replace(variant, "SDCC", "SDCC ", 1)
	case "無":
		variant = strings.Replace(variant, "プレリリース", "Prerelease", 1)
	case " 青R", " 黒U", "黒", " 赤R", " 緑R", "茶":
		edition = missingTable[cardName]
	case "ジャッジ褒賞",
		"Judge Foil":
		// `【EN】【Foil】■2018Ver.■《Vampiric Tutor》 [Judge Foil]`
		variant = strings.Replace(variant, "Ver.", "", 1)
		variant = strings.Replace(variant, "年版", "", 1)
		variant = strings.Replace(variant, "年度版 金", "", 1)
		variant = strings.Replace(variant, "年度版\u3000金", "", 1)
		edition = "Judge Foil"

		switch cardName {
		case "Vindicate":
			if variant == "" {
				variant = "2013"
			}
		}
	case "Arena Foil Land":
		if variant == "ICE" || variant == "β" {
			edition = "PAL01"
		} else if isFoil {
			edition = "PAL00"
		} else { // !!
			edition = "PAL99"
		}
	case "その他イベント記念系", "Other Event anniversary",
		"ゲームデー", "Game Day Promos",
		"メディア系プロモ", "Media Promo",
		"その他プロモ", "Other Promos",
		"PRM",
		"Intro Pack",
		"基本セット系プロモ",
		"発売記念プロモ":
		ed, found := promoTable[cardName]
		if found {
			edition = ed
		} else if cardName == "Fraternal Exaltation" || cardName == "Splendid Genesis" {
			edition = "Special Occasion"
		} else {
			for _, tag := range []string{
				"PIDW", "PI13", "PI14", "PPRO", "WMC", "PSUS", "PCMP", "PSS1",
			} {
				if len(mtgmatcher.MatchInSet(cardName, tag)) == 1 {
					edition = tag
					break
				}
			}
		}
	case "MPS":
		if len(mtgmatcher.MatchInSet(cardName, "MPS")) == 1 {
			edition = "MPS"
		} else if len(mtgmatcher.MatchInSet(cardName, "MP2")) == 1 {
			edition = "MP2"
		}
	case "FEM":
		variant = strings.TrimPrefix(variant, "Illust.")
		if strings.HasPrefix(variant, "Tom") {
			variant = strings.Replace(variant, "W?nerstrand", "Wänerstrand", 1)
		}
	case "CHR", "ATQ":
		if strings.Contains(variant, "\u3000") {
			// Strip away the first side which is useless
			fields := strings.Split(variant, "\u3000")
			if len(fields) > 1 {
				variant = fields[1]
			}
		}
	case "2XM":
		if mtgmatcher.IsBasicLand(cardName) {
			variant = "unglued"
		}
	case "The List":
		if variant == "magic fest版" {
			variant = "magicfest"
		}
	case "Mystery Booster Playtest Cards":
		variant = strings.Replace(variant, "Emblem", "Symbol", 1)
	default:
		ed, found := editionTable[edition]
		if found {
			edition = ed
		} else if strings.HasPrefix(edition, "FtV:") {
			edition = strings.Replace(edition, "FtV:", "From the Vault: ", 1)
		} else if strings.HasSuffix(edition, "-PRE") {
			edition = "P" + strings.TrimSuffix(edition, "-PRE")
			variant = strings.Replace(variant, "Prereleace", "Prerelease", 1)
			variant = strings.Replace(variant, "プレリリース", "Prerelease", 1)
		} else if strings.HasPrefix(edition, "Retro Frame Promos") {
			if cardName == "Fabled Passage" {
				edition = "PW21"
			} else {
				edition = "PLG21"
			}
		} else if strings.Contains(edition, "Silver screen") {
			edition = "DBL"
		} else if strings.HasSuffix(edition, "-BF") {
			fields := strings.Split(edition, "-")
			set, err := mtgmatcher.GetSet(fields[0])
			if err == nil {
				edition = set.Name + ": Extras"
			}

			variant = strings.Replace(variant, "拡張アート", "Extended Art", 1)
			variant = strings.Replace(variant, "ショーケース", "Showcase", 1)
			variant = strings.Replace(variant, "ボーダーレス", "Borderless", 1)
			variant = strings.TrimSpace(variant)

			switch fields[0] {
			case "CMR":
				if variant == "Alternate Frame" {
					variant = "Extended Art"
				}
			case "IKO":
				switch cardName {
				case "Lukka, Coppercoat Outcast",
					"Narset of the Ancient Way",
					"Vivien, Monsters' Advocate":
					variant = "Borderless"
				default:
					if strings.Contains(cardName, " // ") {
						fields = strings.Split(cardName, " // ")
						if len(fields) > 1 {
							cardName = fields[1]
							if variant != "" {
								variant += " "
							}
							variant += "Godzilla"
						}
					}
				}
			case "M21":
				if variant == "Alternate Frame" || variant == "Extended Art" {
					switch cardName {
					case "Basri Ket",
						"Chandra, Heart of Fire",
						"Containment Priest",
						"Cultivate",
						"Garruk, Unleashed",
						"Grim Tutor",
						"Liliana, Waker of the Dead",
						"Massacre Wurm",
						"Scavenging Ooze",
						"Solemn Simulacrum",
						"Teferi, Master of Time",
						"Ugin, the Spirit Dragon":
						variant = "Borderless"
					default:
						variant = "Extended Art"
					}
				}
			}
		} else if strings.Contains(edition, "-") {
			edition = strings.Split(edition, "-")[0]
		}
	}

	variant = strings.TrimSpace(variant)

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      isFoil,
	}, nil
}
