package starcitygames

import (
	"errors"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

var cardTable = map[string]string{
	"Transcatation":                   "Transcantation",
	"Who / What / When / Where / Why": "Who // What // When // Where // Why",
}

func convert(fullName, subtitle, edition string) (card *SCGCard, ed string, err error) {
	fields := strings.Split(fullName, " [")
	if len(fields) < 2 || strings.HasPrefix(fullName, "[") {
		err = errors.New("probably a token")
		return
	}

	cardName := fields[0]
	tag := strings.TrimSuffix(fields[1], "]")
	tags := strings.Replace(tag, "-", " ", -1)
	maybeNum := mtgmatcher.ExtractNumber(tags)

	switch edition {
	case "Planechase Planes: 2012 Edition":
		// Unsupported by Scryfall
		if tag == "SGL-MTG-PC22-P1-ENN" {
			cardName = "Token"
		}
	case "Starter 1999":
		// Incorrect number in SCG
		if mtgmatcher.IsBasicLand(cardName) {
			cardName = "Token"
		}
	case "Core Set 2019":
		if strings.Contains(fullName, "M19-GP") {
			edition = "M19 Gift Pack"
		}
	case "Duel Decks: Anthology",
		"World Championship":
		everyTag := strings.Fields(tags)
		if len(everyTag) > 2 {
			edition = everyTag[2]
		}
		switch edition {
		case "WCHP":
			switch tag {
			case "SGL-MTG-WCHP-129-ENN",
				"SGL-MTG-WCHP-72-ENN":
				edition = "WCD 1998"
			case "SGL-MTG-WCHP-143-ENN",
				"SGL-MTG-WCHP-321-ENN",
				"SGL-MTG-WCHP-298-ENN",
				"SGL-MTG-WCHP-139-ENN",
				"SGL-MTG-WCHP-161-ENN",
				"SGL-MTG-WCHP-64-ENN":
				edition = "WCD 1999"
			case "SGL-MTG-WCHP-1-ENN",
				"SGL-MTG-WCHP-322-ENN",
				"SGL-MTG-WCHP-15-ENN",
				"SGL-MTG-WCHP-135-ENN",
				"SGL-MTG-WCHP-125-ENN":
				edition = "WCD 2000"
			case "SGL-MTG-WCHP-47-ENN",
				"SGL-MTG-WCHP-87-ENN":
				edition = "WCD 2001"
			case "SGL-MTG-WCHP-124-ENN":
				edition = "WCD 2002"
			case "SGL-MTG-WCHP-132-ENN",
				"SGL-MTG-WCHP-205-ENN":
				edition = "WCD 2003"
			case "SGL-MTG-WCHP-100-ENN",
				"SGL-MTG-WCHP-281-ENN",
				"SGL-MTG-WCHP-328-ENN",
				"SGL-MTG-WCHP-52-ENN":
				edition = "WCD 2004"
			}
		case "WC96":
			edition = "PTC"
			subtitle = maybeNum
		}
	case "Alliances",
		"Antiquities",
		"Champions of Kamigawa",
		"Chronicles",
		"Commander Anthology Volume II",
		"Fallen Empires",
		"Guilds of Ravnica",
		"Homelands",
		"Ravnica Allegiance",
		"Unglued":
		if subtitle != "" {
			subtitle += " "
		}
		subtitle += maybeNum
	case "Promo Cards":
		switch tag {
		case "SGL-MTG-PRM-00042151-ENN",
			"SGL-MTG-PRM-00024149-ENN",
			"SGL-MTG-PRM-00024296-ENN",
			"SGL-MTG-PRM-00024272-ENN":
			edition = "Magazine Inserts"
		case "SGL-MTG-PRM-HERO_JOU_162-ENF":
			edition = "Journey into Nyx Hero's Path"
		case "SGL-MTG-PRM-NEM_112a-ENF":
			edition = "Starter 2000"

		case "SGL-MTG-PRM-PRE_P3K_115a-ENN":
			subtitle = "Prerelease April"
		case "SGL-MTG-PRM-PRE_P3K_115b-ENN":
			subtitle = "Prerelease July"
		case "SGL-MTG-PRM-10A_8ED_216-ENF":
			subtitle = "Release"
		case "SGL-MTG-PRM-JDG_J10_008-ENF":
			subtitle = "Judge 2010"
		case "SGL-MTG-PRM-PRE_ELD_233-ENF":
			subtitle = "Prerelease ELD"
		case "SGL-MTG-PRM-PRE_XLN_248-ENF":
			subtitle = "Prerelease XLN"
		case "SGL-MTG-PRM-PP_ELD_233-ENN",
			"SGL-MTG-PRM-PP_ELD_233-ENF":
			subtitle = "Promo Pack ELD"
		case "SGL-MTG-PRM-PP_XLN_248-ENN",
			"SGL-MTG-PRM-PP_XLN_248-ENF":
			subtitle = "Promo Pack XLN"
		case "SGL-MTG-PRM-WPNG_2011_070-ENF",
			"SGL-MTG-PRM-WPNG_2011_069-ENF":
			subtitle = "WPN 2011"
		case "SGL-MTG-PRM-WPNG_2010_051-ENF",
			"SGL-MTG-PRM-WPNG_2010_050-ENF":
			subtitle = "WPN 2010"
		case "SGL-MTG-PRM-ARENA_USG_132-ENF":
			subtitle = "Arena 2000"
		case "SGL-MTG-PRM-ARENA_2001_001-ENF":
			subtitle = "Arena 2001 1"
		case "SGL-MTG-PRM-ARENA_2001_011-ENF":
			subtitle = "Arena 2001 11"

		default:
			switch {
			case strings.HasPrefix(tag, "SGL-MTG-PRM-003944"):
				edition = "PHEL"
			case strings.Contains(tag, "SMMR_10"):
				edition = "Summer of Magic"
			case strings.Contains(tag, "CHMP"):
				edition = "Champs and States"
			case strings.Contains(tag, "SDCC13_M14"):
				edition = "PSDC"
			case strings.Contains(tag, "SDCC14_M15"):
				edition = "PS14"
			case strings.Contains(tag, "UGIN"):
				edition = "Ugin's Fate"
			case strings.Contains(tag, "DCILM"):
				edition = "DCI Legend Membership"
			case strings.Contains(tag, "JNRS_E"):
				edition = "Junior Series Europe"
			case strings.Contains(tag, "JSS_"):
				edition = "Junior Super Series"
			case strings.Contains(tag, "RPTQ"):
				edition = "Pro Tour Promos"
			case strings.Contains(tag, "GPX_2018"):
				edition = "Grand Prix Promos"
			case strings.Contains(tag, "GIFT_2017"):
				edition = "2017 Gift Pack"
			case strings.Contains(tag, "-GURU_"):
				edition = "Guru"
			case strings.Contains(tag, "EURO_"):
				edition = "European Land Program"
			case strings.Contains(tag, "APAC_"):
				edition = "Asia Pacific Land Program"
			case strings.Contains(tag, "SSD_2017"):
				edition = "XLN Standard Showdown"
			case strings.Contains(tag, "SSD_2018"):
				edition = "M19 Standard Showdown"
			case strings.Contains(tag, "RVWK_A"),
				strings.Contains(tag, "RVWK_B"):
				edition = "GRN Ravnica Weekend"
				if strings.Contains(tag, "RVWK_B") {
					edition = "RNA Ravnica Weekend"
				}
				tags := strings.Replace(tag, "-", " ", -1)
				fields := strings.Fields(strings.Replace(tags, "_", " ", -1))
				if len(fields) > 4 {
					subtitle = fields[4]
				}

			case strings.Contains(tag, "DFRY"):
				subtitle = "Dragonfury"
			case strings.Contains(tag, "HSCON"):
				subtitle = "HASCON"
			case strings.Contains(tag, "BAB_"):
				subtitle = "Buy-a-Box"
			case strings.Contains(tag, "BUN_"):
				subtitle = "Bundle"
			case strings.Contains(tag, "-DPW_"):
				subtitle = "Duels of the Planeswalakers"
			case strings.Contains(tag, "-MF"):
				subtitle = "Magic Fest"
			case strings.Contains(tag, "-15A_"):
				subtitle = "15th Anniversary"
			case strings.Contains(tag, "-PP_"):
				subtitle = "Promo Pack"
			case strings.Contains(tag, "-PRE_"):
				subtitle = "Prerelease"
			case strings.Contains(tag, "PLYR"):
				subtitle = "Textless"
			case strings.Contains(tag, "2HG_2005"):
				subtitle = "2HG"
			case strings.Contains(tag, "CLASH"):
				subtitle = "Clash Pack"
			case strings.Contains(tag, "WPNG"):
				subtitle = "WPN"
			case strings.Contains(tag, "BOOK"):
				subtitle = "Book"
			case strings.Contains(tag, "WMCQ"):
				subtitle = "WMCQ"
			case strings.Contains(tag, "LNCH"):
				subtitle = "Launch"

			case strings.Contains(tag, "JDG"):
				subtitle = ""
				fields := strings.Split(tag, "_")
				if len(fields) > 1 {
					edition = fields[1]
				}
			case strings.Contains(tag, "SECRET_SLD"):
				tags := strings.Replace(tag, "-", " ", -1)
				tags = strings.Replace(tags, "_", " ", -1)
				subtitle = mtgmatcher.ExtractNumber(tags)
				edition = "Secret Lair"
			case strings.Contains(tag, "FNM_"), strings.Contains(tag, "ARENA"):
				subtitle = "FNM"
				if strings.Contains(tag, "ARENA") {
					subtitle = "Arena"
				}
				tags := strings.Replace(tag, "_", " ", -1)
				year := mtgmatcher.ExtractYear(tags)
				switch {
				case strings.Contains(tags, "USG"):
					year = "1999"
				case strings.Contains(tags, "6ED"),
					strings.Contains(tags, "MMQ"):
					year = "2000"
				}
				if year != "" {
					subtitle += " " + year
				}
			}
		}
	}

	if mtgmatcher.IsBasicLand(cardName) {
		switch edition {
		case "Asia Pacific Land Program":
			var landMap = map[string]string{
				"SGL-MTG-PRM-APAC_RED_001-ENN":   "4",
				"SGL-MTG-PRM-APAC_RED_002-ENN":   "2",
				"SGL-MTG-PRM-APAC_RED_003-ENN":   "5",
				"SGL-MTG-PRM-APAC_RED_004-ENN":   "3",
				"SGL-MTG-PRM-APAC_RED_005-ENN":   "1",
				"SGL-MTG-PRM-APAC_BLUE_001-ENN":  "9",
				"SGL-MTG-PRM-APAC_BLUE_002-ENN":  "7",
				"SGL-MTG-PRM-APAC_BLUE_003-ENN":  "10",
				"SGL-MTG-PRM-APAC_BLUE_004-ENN":  "8",
				"SGL-MTG-PRM-APAC_BLUE_005-ENN":  "6",
				"SGL-MTG-PRM-APAC_CLEAR_001-ENN": "14",
				"SGL-MTG-PRM-APAC_CLEAR_002-ENN": "12",
				"SGL-MTG-PRM-APAC_CLEAR_003-ENN": "15",
				"SGL-MTG-PRM-APAC_CLEAR_004-ENN": "13",
				"SGL-MTG-PRM-APAC_CLEAR_005-ENN": "11",
			}
			subtitle = landMap[tag]

		case "European Land Program":
			var landMap = map[string]string{
				"SGL-MTG-PRM-EURO_BLUE_001-ENN":   "4",
				"SGL-MTG-PRM-EURO_BLUE_002-ENN":   "2",
				"SGL-MTG-PRM-EURO_BLUE_003-ENN":   "5",
				"SGL-MTG-PRM-EURO_BLUE_004-ENN":   "3",
				"SGL-MTG-PRM-EURO_BLUE_005-ENN":   "1",
				"SGL-MTG-PRM-EURO_RED_001-ENN":    "9",
				"SGL-MTG-PRM-EURO_RED_002-ENN":    "7",
				"SGL-MTG-PRM-EURO_RED_003-ENN":    "10",
				"SGL-MTG-PRM-EURO_RED_004-ENN":    "8",
				"SGL-MTG-PRM-EURO_RED_005-ENN":    "6",
				"SGL-MTG-PRM-EURO_PURPLE_001-ENN": "14",
				"SGL-MTG-PRM-EURO_PURPLE_002-ENN": "12",
				"SGL-MTG-PRM-EURO_PURPLE_003-ENN": "15",
				"SGL-MTG-PRM-EURO_PURPLE_004-ENN": "13",
				"SGL-MTG-PRM-EURO_PURPLE_005-ENN": "11",
			}
			subtitle = landMap[tag]

		default:
			fields := strings.Fields(tags)
			if len(fields) > 3 {
				if subtitle != "" {
					subtitle += " "
				}
				subtitle += mtgmatcher.ExtractNumber(fields[3])
			}
		}
	}

	return &SCGCard{
		Name:     cardName,
		Subtitle: subtitle,
		Language: "English",
	}, edition, nil
}

func preprocess(card *SCGCard, edition string) (*mtgmatcher.Card, error) {
	cardName := strings.Replace(card.Name, "&amp;", "&", -1)

	var skipLang bool
	switch card.Language {
	case "English":
	case "Japanese":
		switch edition {
		case "4th Edition BB":
			if mtgmatcher.IsBasicLand(cardName) {
				skipLang = true
			}
		case "War of the Spark":
			if card.Subtitle != "(Alternate Art)" {
				skipLang = true
			}
		case "Ikoria: Lair of Behemoths - Variants":
			switch cardName {
			case "Crystalline Giant",
				"Battra, Dark Destroyer",
				"Mothra's Great Cocoon":
			default:
				skipLang = true
			}
		default:
			skipLang = true
		}
	case "Italian":
		switch edition {
		case "3rd Edition BB":
			if mtgmatcher.IsBasicLand(cardName) {
				skipLang = true
			}
		case "Legends":
		case "Renaissance":
		case "The Dark":
		default:
			skipLang = true
		}
	default:
		skipLang = true
	}
	if skipLang {
		return nil, errors.New("non-english")
	}

	edition = strings.Replace(edition, "&amp;", "&", -1)

	variant := strings.Replace(card.Subtitle, "&amp;", "&", -1)
	variant = strings.Replace(variant, "(", "", -1)
	variant = strings.Replace(variant, ")", "", -1)

	switch {
	case strings.HasPrefix(cardName, "APAC Land"),
		strings.HasPrefix(cardName, "Euro Land"),
		strings.Contains(variant, "Oversized"),
		strings.Contains(edition, "Oversized"),
		mtgmatcher.IsToken(cardName):
		return nil, errors.New("non-single")
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	switch cardName {
	case "Captain Sisay":
		if edition == "Mystery Booster" {
			return nil, errors.New("invalid")
		}
	default:
		if mtgmatcher.IsBasicLand(cardName) {
			if strings.Contains(variant, "APAC") {
				edition = "Asia Pacific Land Program"
			} else if strings.Contains(variant, "Euro") {
				edition = "European Land Program"
			}
		}
	}

	vars := mtgmatcher.SplitVariants(cardName)
	cardName = vars[0]
	if len(vars) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += vars[1]
	}

	switch card.Language {
	case "Japanese", "Italian":
		if variant != "" {
			variant += " "
		}
		variant += card.Language
	}

	switch edition {
	case "3rd Edition BB":
		variant = strings.TrimSuffix(variant, " BB")
	default:
		if strings.HasSuffix(edition, "(Foil)") {
			edition = strings.TrimSuffix(edition, " (Foil)")
		}
		if strings.HasSuffix(edition, "Alternate Frame") {
			edition = strings.TrimSuffix(edition, " - Alternate Frame")

			// Decouple showcase and boderless from this tag
			if strings.Contains(variant, "Alternate Art") {
				set, err := mtgmatcher.GetSet(edition)
				if err != nil {
					return nil, err
				}
				for _, card := range set.Cards {
					if card.Name == cardName {
						if card.HasFrameEffect(mtgjson.FrameEffectShowcase) {
							variant = "Showcase"
							break
						}
						if card.BorderColor == mtgjson.BorderColorBorderless {
							variant = "Borderless"
							break
						}
					}
				}
			}
		}
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      card.Foil,
	}, nil
}
