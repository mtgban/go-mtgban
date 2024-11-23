package starcitygames

import (
	"errors"
	"strconv"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Who / What / When / Where / Why": "Who // What // When // Where // Why",
}

func languageTags(language, edition, variant, number string) (string, string, error) {
	switch language {
	case "Japanese", "ja":
		switch edition {
		case "Chronicles":
			edition = "BCHR"
		case "4th Edition - Black Border":
			variant = strings.TrimSuffix(variant, " BB")
		case "Strixhaven Mystical Archive",
			"Strixhaven Mystical Archive - Foil Etched":
			num, err := strconv.Atoi(strings.TrimLeft(number, "0"))
			if err != nil {
				return "", "", err
			}
			if num < 64 {
				return "", "", errors.New("non-english")
			}
		case "War of the Spark":
			if !strings.Contains(variant, "Alternate Art") {
				return "", "", errors.New("non-english")
			}
		default:
			if variant != "" {
				variant += " "
			}
			variant += "Japanese"
		}
	case "Italian", "it":
		if edition == "Renaissance" {
			edition = "Rinascimento"
		} else {
			if variant != "" {
				variant += " "
			}
			variant += "Italian"
		}
	}
	return edition, variant, nil
}

func preprocess(card *SCGCardVariant, cardEdition, language string, foil bool, cn string) (*mtgmatcher.InputCard, error) {
	// Processing variant first because it gets added on later
	variant := strings.Replace(card.Subtitle, "&amp;", "&", -1)
	variant = strings.Replace(variant, "(", "", -1)
	variant = strings.Replace(variant, ")", "", -1)

	cardName := strings.Replace(card.Name, "&amp;", "&", -1)
	cardName = strings.Replace(cardName, "{", "", -1)
	cardName = strings.Replace(cardName, "}", "", -1)

	vars := mtgmatcher.SplitVariants(cardName)
	cardName = vars[0]
	if len(vars) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(vars[1:], " ")
	}

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	// Check tokens with the same name as certain cards
	isToken := strings.HasPrefix(card.Name, "{") && strings.Contains(card.Name, "}")
	if isToken && !strings.Contains(cardName, "Token") {
		cardName += " Token"
	}

	edition := strings.Replace(cardEdition, "&amp;", "&", -1)
	if strings.HasSuffix(edition, "(Foil)") {
		edition = strings.TrimSuffix(edition, " (Foil)")
		foil = true
	}
	vars = mtgmatcher.SplitVariants(edition)
	edition = vars[0]
	if len(vars) > 1 {
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(vars[1:], " ")
	}

	var err error
	edition, variant, err = languageTags(language, edition, variant, cn)
	if err != nil {
		return nil, err
	}

	// SKU documented as
	// * for singles:
	// SGL-[Brand]-[Set]-[Collector Number]-[Language][Foiling][Condition]
	// * for world champtionship:
	// SGL-[Brand]-WCPH-[Year][Player Initials][Set][Collector Number][Sideboard]-[Language][Foiling][Condition]
	// * for promotional cards:
	// SGL-[Brand]-PRM-[Promo][Set][Collector Number]-[Language][Foiling][Condition]
	//
	// examples
	// * SGL-MTG-PRM-SECRET_SLD_1095-ENN1
	// * SGL-MTG-PRM-PP_MKM_187-ENN
	// * SGL-MTG-PWSB-PCA_115-ENN1

	canProcessSKU := language == "en" || language == "English"
	// We can't use the numbers reported because they match the plain version
	// and the Match search doesn't upgrade these custom tags
	switch {
	case strings.Contains(variant, "Serial"),
		strings.Contains(variant, "Compleat"),
		strings.Contains(variant, "Oversized"):
		canProcessSKU = false
	}

	var number string
	fields := strings.Split(card.Sku, "-")
	if len(fields) > 3 && canProcessSKU {
		setCode := fields[2]
		number = strings.TrimLeft(fields[3], "0")

		switch setCode {
		case "MPS2":
			setCode = "MPS"
		case "MPS3":
			setCode = "MP2"
		case "PWSB":
			setCode = "PLST"
			fields := strings.Split(number, "_")
			if len(fields) == 2 {
				subSetCode := fields[0]
				subNumber := fields[1]

				// Handle MH22 and similar
				_, err := mtgmatcher.GetSet(subSetCode)
				if err != nil && len(subSetCode) > 3 && unicode.IsDigit(rune(setCode[len(setCode)-1])) {
					subSetCode = subSetCode[:len(subSetCode)-1]
				}

				number = subSetCode + "-" + strings.TrimLeft(subNumber, "0")
			} else if len(fields) == 4 {
				if fields[0] == "PRM" {
					number = fields[2] + "-" + strings.TrimLeft(fields[3], "0")
				}
			}
		case "PRM", "PRM3":
			fields := strings.Split(number, "_")

			switch {
			// Decouple Secret Lair
			case len(fields) > 2 && fields[0] == "SECRET":
				setCode = fields[1]
				number = strings.TrimLeft(fields[2], "0")
			// Separate the multiple LTR Prerelease cards
			case strings.HasPrefix(number, "PRE_LTR_"):
				number = strings.TrimPrefix(number, "PRE_LTR_")
				if strings.HasSuffix(number, "a") {
					setCode = "PLTR"
					number = strings.Replace(number, "a", "s", 1)
				} else if strings.HasSuffix(number, "b") {
					setCode = "LTR"
					number = strings.TrimSuffix(number, "b")
				}
			// Prevent edition from mismatching
			case strings.HasPrefix(number, "PP_2023_"):
				edition = "PF23"
				number = strings.TrimLeft(fields[2], "0")
			}
		default:
			if strings.Contains(cardName, "//") && strings.HasSuffix(number, "a") {
				number = strings.TrimSuffix(number, "a")
			}

			// Handle set renames like OTC2 and LTR2
			_, err := mtgmatcher.GetSet(setCode)
			if err != nil && len(setCode) > 3 && unicode.IsDigit(rune(setCode[len(setCode)-1])) {
				setCode = setCode[:len(setCode)-1]
			}
		}

		// Check if we found it and return the id
		out := mtgmatcher.MatchWithNumber(cardName, setCode, number)
		if len(out) == 1 {
			return &mtgmatcher.InputCard{
				Id:        out[0].UUID,
				Foil:      foil,
				Variation: variant,
				Language:  language,
			}, nil
		}
	}

	switch edition {
	case "Promo: General",
		"Promo: General - Alternate Foil":
		switch cardName {
		case "Swiftfoot Boots":
			if variant == "Launch" {
				edition = "PW22"
				variant = ""
			}
		case "Dismember":
			if variant == "Commander Party Phyrexian" {
				edition = "PW22"
			}
		case "Rukh Egg":
			if variant == "10th Anniversary" {
				edition = "P8ED"
			}
		case "Mind Stone":
			if variant == "Bring-a-Friend" {
				edition = "PW21"
				variant = ""
			}
		case "Scavenging Ooze":
			if variant == "Love Your LGS Retro Frame" {
				edition = "PLG21"
			}
		case "Arcane Signet":
			switch variant {
			case "Play Draft Retro Frame":
				edition = "P30M"
				variant = "1P"
			case "Festival":
				edition = "P30M"
				variant = "1F"
			case "Festival Foil Etched":
				edition = "P30M"
				variant = "1F★"
			}
		case "Counterspell":
			switch variant {
			case "Festival Full Art":
				edition = "PF24"
			}
		case "Llanowar Elves":
			switch variant {
			case "Resale Retro Frame":
				edition = "PRES"
			}
		case "Pyromancer's Gauntlet":
			switch variant {
			case "Hasbro Retro Frame":
				edition = "PMEI"
			}
		case "Lathril, Blade of the Elves":
			switch variant {
			case "Resale Foil Etched":
				edition = "PRES"
			}
		case "Rampant Growth":
			if variant == "Release Foil Etched" {
				edition = "PW23"
			}
		case "Commander's Sphere":
			if variant == "Play Draft" {
				edition = "PW24"
			}
		}
	case "Unfinity":
		if strings.Contains(cardName, "Sticker Sheet") {
			edition = "SUNF"
		}
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      foil,
		Language:  language,
	}, nil
}
