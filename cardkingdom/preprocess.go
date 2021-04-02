package cardkingdom

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

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

	"PAL96-001": "PARL-001",
	"PAL96-003": "PARL-003",
	"PAL96-004": "PARL-004",

	"PUMA-062A": "PDTK-061A",

	"FDOM-269P": "PDOM-001P",
	"MPS-001A":  "PRES-001A",

	"PJMP-496B": "JMP-496",
	"P2XM-383":  "2XM-383",
	"P2XM-384":  "2XM-384",
}

func Preprocess(card CKCard) (*mtgmatcher.Card, error) {
	if mtgmatcher.IsToken(card.Name) ||
		strings.Contains(card.Variation, "Misprint") ||
		strings.Contains(card.Variation, "Oversized") ||
		strings.Contains(card.Edition, "Art Series") ||
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
				setCode = "Prerelease"
			}
		}
		if len(setCode) == 4 && strings.HasPrefix(setCode, "T") {
			return nil, fmt.Errorf("unknown sku code %s", setCode)
		}
		if len(setCode) == 4 && (mtgmatcher.Contains(card.Variation, "buyabox") || mtgmatcher.Contains(card.Variation, "bundle")) {
			set, err := mtgmatcher.GetSet(setCode[1:])
			if err != nil {
				return nil, fmt.Errorf("unknown sku code %s", setCode)
			}
			setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)
			if mtgmatcher.Contains(card.Variation, "buyabox") && setDate.After(mtgmatcher.BuyABoxNotUniqueDate) {
				setCode = setCode[1:]
			} else if mtgmatcher.Contains(card.Variation, "bundle") && setDate.After(mtgmatcher.BuyABoxInExpansionSetsDate) {
				setCode = setCode[1:]
			}
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

	variation := card.Variation
	edition := card.Edition
	switch edition {
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
		"Unstable", "Unglued", "Portal II", "Strixhaven Mystical Archive":
		variation = number
	}

	return &mtgmatcher.Card{
		Name:      card.Name,
		Edition:   edition,
		Variation: variation,
		Foil:      isFoil,
	}, nil
}
