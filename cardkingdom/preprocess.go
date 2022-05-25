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

	"FPLGS-001":  "PLG20-001",
	"FPLG21-001": "PW21-005",

	"WC97-JS097":    "WC97-JS242",
	"WC97-PM037":    "WC97-PM037B",
	"WC98-RB330":    "WC98-RB330SB",
	"WC01-AB078":    "WC01-AB078SB",
	"WC02-SHH266":   "WC02-SHH266SB",
	"WC02-CR057SBA": "WC02-CR057SB",
	"WC03-WE062":    "WC03-WE062SB",

	// Yellow Hidetsugu
	"PNEO-432": "NEO-432",
}

func Preprocess(card CKCard) (*mtgmatcher.Card, error) {
	// Non-foil cards of this set do not exist
	if card.Variation == "MagicFest Non-Foil - 2020" {
		return nil, errors.New("doesn't exist")
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
		number = strings.TrimRight(number, "jp")

		if len(setCode) > 3 && strings.HasPrefix(setCode, "F") {
			// Some Arena foil are using this custom code that confuses the matcher
			// Just not set for them, and rely on the Variation field as is
			if setCode != "FUSG" && setCode != "F6ED" {
				setCode = setCode[1:]
			}

			// Foil-Etched sets
			if len(setCode) > 3 && setCode[0] == 'E' {
				setCode = setCode[1:]
			}

			// Prerelease cards in foreign language get mixed up in the normal set
			if number == "666" {
				setCode = "Prerelease"
			}
		}
		if len(setCode) == 4 && (mtgmatcher.Contains(card.Variation, "buyabox") || mtgmatcher.Contains(card.Variation, "bundle")) {
			set, err := mtgmatcher.GetSet(setCode[1:])
			if err != nil {
				return nil, fmt.Errorf("unknown edition code %s", setCode)
			}
			setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
			if err != nil {
				return nil, fmt.Errorf("unknown set date %s", err.Error())
			}
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

		// Full rename due to differen set codes
		if strings.HasPrefix(setCode, "PWP2") {
			setCode = strings.Replace(setCode, "PWP2", "PW2", 1)
		}
	}

	variation := card.Variation
	edition := card.Edition
	switch edition {
	case "Promotional":
		edition = setCode
		switch {
		case strings.Contains(variation, "APAC"),
			strings.Contains(variation, "Euro"),
			strings.Contains(variation, "League"),
			strings.Contains(variation, "MPS"):
			variation = number
		case edition == "PW22":
			variation = ""
		}
	case "World Championships":
		edition = setCode
		variation = number

		// Duplicate sku
		if sku == "PTC-ET015SB" && card.Name == "Circle of Protection: Red" {
			variation = "et17sb"
		}
	case "Alpha", "Beta", "Unlimited", "3rd Edition", "4th Edition",
		"Antiquities", "Fallen Empires", "Alliances", "Homelands",
		"Zendikar", "Battle for Zendikar", "Oath of the Gatewatch",
		"Unstable", "Unglued", "Portal II", "Secret Lair":
		variation = number
	}

	// Preserve Etched property in case variation became overwritten with the number
	if strings.Contains(card.Variation, "Etched") && !strings.Contains(variation, "Etched") {
		variation += " Etched"
	}

	// Drop one side of dfc tokens
	if strings.Contains(card.Name, " // ") &&
		(strings.Contains(card.Name, "Token") || (len(setCode) > 3 && setCode[0] == 'T')) {
		card.Name = strings.Split(card.Name, " // ")[0] + " Token"
	}

	return &mtgmatcher.Card{
		Name:      card.Name,
		Edition:   edition,
		Variation: variation,
		Foil:      isFoil,
	}, nil
}
