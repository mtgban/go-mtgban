package starcitygames

import (
	"errors"
	"fmt"
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

// Special sets like collectors have an extra number as suffix
// Handle set renames like OTC2 and LTR2
func fixupSetCode(setCode string) string {
	_, err := mtgmatcher.GetSet(setCode)
	if err != nil && len(setCode) > 3 && unicode.IsDigit(rune(setCode[len(setCode)-1])) {
		switch setCode {
		case "4ED2":
			setCode = "4EDALT"
		case "MPS3":
			setCode = "MP2"
		default:
			setCode = setCode[:len(setCode)-1]
		}
	}
	return setCode
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
func ProcessSKU(cardName, SKU string) (*mtgmatcher.InputCard, error) {
	fields := strings.Split(SKU, "-")
	if len(fields) < 5 || len(fields[4]) < 3 {
		return nil, fmt.Errorf("Malformed SKU: %s", SKU)
	}

	setCode := fixupSetCode(fields[2])
	number := strings.TrimLeft(fields[3], "0")
	language := fields[4][:2]
	foil := fields[4][2] != 'N'

	switch setCode {
	case "PWSB":
		setCode = "PLST"
		fields := strings.Split(number, "_")
		if len(fields) == 2 {
			subSetCode := fixupSetCode(fields[0])
			subNumber := fields[1]

			number = subSetCode + "-" + strings.TrimLeft(subNumber, "0")
		} else if len(fields) == 4 {
			if fields[0] == "PRM" {
				// Fix promo set code not being tagged as promo
				if fields[1] == "GMDY" && !strings.HasPrefix(fields[2], "P") {
					fields[2] = "P" + fields[2]
				}
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
			setCode = "PF23"
			number = strings.TrimLeft(fields[2], "0")
		}
	case "MB13":
		setCode = "MB2"
		cards := mtgmatcher.MatchInSet(cardName, setCode)
		if len(cards) == 1 {
			number = cards[0].Number
		}
	default:
		if strings.Contains(cardName, "//") {
			number = strings.TrimSuffix(number, "a")
		}
	}

	// Check if we found it and return the id
	out := mtgmatcher.MatchWithNumber(cardName, setCode, number)
	if len(out) == 1 {
		card := out[0]
		// If there's a single finish make sure the number+finish combination is correct
		// Otherwise let it be processed upstream
		if len(card.Finishes) == 1 &&
			(((card.HasFinish(mtgmatcher.FinishFoil) || card.HasFinish(mtgmatcher.FinishEtched)) && !foil) ||
				(card.HasFinish(mtgmatcher.FinishNonfoil) && foil)) {
			return nil, errors.New("invalid number/foil combination")
		}
		return &mtgmatcher.InputCard{
			Id:       out[0].UUID,
			Foil:     foil,
			Language: language,
		}, nil
	}
	if len(out) > 1 {
		alias := mtgmatcher.NewAliasingError()
		for _, id := range out {
			alias.Dupes = append(alias.Dupes, id.UUID)
		}
		return nil, alias
	}
	return nil, errors.New("not found")
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

	canProcessSKU := language == "en" || language == "English"
	// We can't use the numbers reported because they match the plain version
	// and the Match search doesn't upgrade these custom tags
	switch {
	case strings.Contains(variant, "Serial"),
		strings.Contains(variant, "Compleat"),
		strings.Contains(variant, "Oversized"):
		canProcessSKU = false
	}

	if canProcessSKU {
		out, err := ProcessSKU(cardName, card.Sku)
		if err == nil {
			// We need to attach this field to take into account promotions
			// like STA (nonfoil/foil/etched) with the same collector number
			out.Variation = variant
			return out, nil
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
			case "Marvel NYCC 2024 Borderless":
				edition = "PURL"
			}
		case "Pyromancer's Gauntlet":
			switch variant {
			case "Hasbro Retro Frame":
				edition = "PMEI"
			}
		case "Rampant Growth":
			if variant == "Release Foil Etched" {
				edition = "PW23"
			}
		case "Commander's Sphere":
			if variant == "Play Draft" {
				edition = "PW24"
			}
		case "Sakura-Tribe Elder":
			if variant == "Love Your LGS Textless" {
				edition = "PLG24"
			}
		case "Wastes":
			if variant == "Magic Academy Full Art" {
				edition = "PLG25"
			}
		case "Avacyn's Pilgrim":
			if variant == "Festival Full Art" {
				edition = "PF25"
			}
		case "Ponder", "The First Sliver":
			if variant == "Festival" {
				edition = "PLG25"
			}
		}
	case "Unfinity":
		if strings.Contains(cardName, "Sticker Sheet") {
			edition = "SUNF"
		}
	case "Promo: Date Stamped",
		"Promo: Planeswalker Stamped":
		if cardName == "Mirror Room" {
			variant = strings.Replace(variant, "Fractured Room", "", 1)
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
