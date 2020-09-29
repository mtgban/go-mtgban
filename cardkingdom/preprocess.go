package cardkingdom

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	"Surgeon Commander": "Surgeon ~General~ Commander",

	// Numbers for these are derived elsewhere
	"BFM Left":  "B.F.M.",
	"BFM Right": "B.F.M.",

	"The Ultimate Nightmare of WotC Customer Service": "The Ultimate Nightmare of Wizards of the Coast® Customer Service",
	"Our Market Research":                             "Our Market Research Shows That Players Like Really Long Card Names So We Made this Card to Have the Absolute Longest Card Name Ever Elemental",
}

// This table contains all SKUs that contain incorrect codes or codes that could
// be mistaken for edition codes (thus misdirecting the matcher) or that contain
// incorrect numbers. Sometimes both.
var skuFixupTable = map[string]string{
	"013A":     "FEM-013A",
	"FEM-070B": "FEM-030B",

	"ATQ-080A": "ATQ-080C",
	"ATQ-080B": "ATQ-080A",
	"ATQ-080C": "ATQ-080B",
	"ATQ-080D": "ATQ-080D",

	"F18-006": "PGRN-006",
	"F18-029": "PM19-029",
	"F18-054": "PGRN-054",
	"F18-060": "PDOM-060",
	"F18-081": "PDOM-081",
	"F18-110": "PM19-110",
	"F18-180": "PM19-180",
	"F18-204": "PDOM-204",
	"F18-206": "PGRN-206",
	"F19-107": "PRNA-107",
	"F19-171": "PWAR-171",
	"F19-178": "PRNA-178",
	"F19-192": "PRNA-192",
	"F19-193": "PWAR-193",

	"PAL96-001": "PARL-001",
	"PAL96-003": "PARL-003",
	"PAL96-004": "PARL-004",

	"PUMA-050A": "PKTK-036A",
	"PUMA-062A": "PDTK-061A",
	"PUMA-117A": "PFRF-087A",

	"FDOM-269P": "PDOM-001P",
	"MPS-001A":  "PRES-001A",
	"PRED-001":  "PDRC-001",
	"PTHB-352":  "THB-352",

	"PM21-392":  "M21-392",
	"PJMP-496B": "JMP-496",
	"P2XM-383":  "2XM-383",
	"P2XM-384":  "2XM-384",
}

func Preprocess(card CKCard) (*mtgmatcher.Card, error) {
	if strings.Contains(card.Name, "Token") ||
		strings.Contains(card.Name, "Emblem") ||
		strings.Contains(card.Name, "Checklist") ||
		strings.Contains(card.Name, "DFC Helper") ||
		strings.Contains(card.Variation, "Misprint") ||
		strings.Contains(card.Variation, "Oversized") ||
		card.Name == "Blank Card" ||
		card.Edition == "Art Series" ||
		card.Variation == "MagicFest Non-Foil - 2020" ||
		card.SKU == "OVERSIZ" {
		return nil, errors.New("skipping")
	}

	setCode := ""
	number := ""
	isFoil, _ := strconv.ParseBool(card.IsFoil)

	sku := card.SKU
	fixup, found := skuFixupTable[sku]
	if found {
		sku = fixup
	}

	fields := strings.Split(sku, "-")
	if len(fields) > 1 {
		setCode = fields[0]
		number = strings.Join(fields[1:], "")
		number = strings.TrimLeft(number, "0")
		number = strings.ToLower(number)

		if len(setCode) > 3 && strings.HasPrefix(setCode, "F") {
			// Some Arena foil are using this custom code that confuses the matcher
			// Just not set for them, and rely on the Variation field as is
			if setCode != "FUSG" && setCode != "F6ED" {
				setCode = setCode[1:]
			}

			// Prerelease cards in foreign language get mixed up in the normal set
			if number == "666" {
				setCode = "PPRE"
			}
		}
		if len(setCode) == 4 && strings.HasPrefix(setCode, "T") {
			return nil, fmt.Errorf("unknown sku code %s", setCode)
		}

		if (card.Variation == "Game Day Extended Art" ||
			card.Variation == "Game Day Extended" ||
			card.Variation == "Gameday Extended Art" ||
			card.Variation == "Game Day Promo") && strings.HasSuffix(number, "p") {
			if len(setCode) == 3 {
				setCode = "P" + setCode
			}
			number = number[:len(number)-1]
		}
		if setCode == "OPHOP" {
			setCode = "PHOP"
		}
	}

	cardName := card.Name
	name, found := cardTable[cardName]
	if found {
		cardName = name
	}

	variation := card.Variation
	edition := card.Edition
	switch edition {
	case "Duel Decks: Anthology":
		variation = strings.Replace(variation, " - Foil", "", 1)
		variation = strings.Replace(variation, " vs ", " vs. ", 1)
		fields := strings.Fields(variation)
		variation = number
		edition = "Duel Decks Anthology: " + strings.Join(fields[:3], " ")
	case "Promotional",
		"World Championships":
		edition = setCode
		switch {
		case strings.Contains(variation, "APAC"),
			strings.Contains(variation, "Euro"),
			strings.Contains(variation, "MPS"):
			variation = number
		}
	case "Alpha", "Beta", "Unlimited", "3rd Edition", "4th Edition",
		"Antiquities", "Fallen Empires", "Alliances", "Homelands",
		"Zendikar", "Battle for Zendikar", "Oath of the Gatewatch",
		"Unstable", "Unglued":
		variation = number
	}

	return &mtgmatcher.Card{
		Name:      cardName,
		Edition:   edition,
		Variation: variation,
		Foil:      isFoil,
	}, nil
}
