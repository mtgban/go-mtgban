package cardkingdom

import (
	"errors"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// This table contains all SKUs that contain incorrect codes or codes that could
// be mistaken for edition codes (thus misdirecting the matcher) or that contain
// incorrect numbers. Sometimes both.
var skuFixupTable = map[string]string{
	// Some of the lands from the first Arena set
	"PAL96-001": "PARL-001",
	"PAL96-003": "PARL-003",
	"PAL96-004": "PARL-004",

	// Jaya Ballard, Task Mage
	"MPS-001A": "PRES-001A",

	// Jace Beleren book promo
	"PBOK-001": "PBOOK-001",

	// Mind Stone
	"PLG21-001": "PW21-005",

	// Path of Ancestry
	"PF21-001": "PLG21-C3",

	// Yellow Hidetsugu
	"PNEO-432": "NEO-432",

	// Random WCD cards
	"WC97-JS097":    "WC97-JS242",
	"WC97-PM037":    "WC97-PM037B",
	"WC98-343":      "WC98-BR343",
	"WC98-344":      "WC98-BR344",
	"WC98-345":      "WC98-BR345",
	"WC98-346":      "WC98-BR346",
	"WC98-RB330":    "WC98-RB330SB",
	"WC01-AB078":    "WC01-AB078SB",
	"WC02-SHH266":   "WC02-SHH266SB",
	"WC02-CR057SBA": "WC02-CR057SB",
	"WC02-SSH335":   "WC02-SHH335",
	"WC02-RL336":    "WC02-RL336A",
	"WC02-RL336A":   "WC02-RL336B",
	"WC02-CR337":    "WC02-CR337A",
	"WC02-CR337A":   "WC02-CR337B",
	"WC02-RL337":    "WC02-RL337A",
	"WC02-RL337A":   "WC02-RL337B",
	"WC03-WE062":    "WC03-WE062SB",

	// Planeshift Altx Art
	"PPLS-074": "PLS-074★",
	"PPLS-107": "PLS-107★",
	"PPLS-133": "PLS-133★",

	// Duplicated ULST cards
	"FMUST-147A": "ULST-55",
	"FMUST-147F": "ULST-56",
	"FMUST-113A": "ULST-38",
	"FMUST-113C": "ULST-37",

	// Wrong PLST codes
	"MF19-001": "MPF19-001",
	"MZNR-091": "MKHC-091",

	// Naya Sojourners
	"PM10-028": "PDCI-29",
	// Mitotic Slime
	"PM11-185": "PDCI-53",

	// Duel Decks Beast Token
	"TDDD-001": "TDDD-T1",
	"TDDD-002": "TDDD-T2",
	"TDDD-003": "TDDD-T3",

	// M20 Promo Pack lands
	"PRM-001P": "PPP1-1",
	"PRM-002P": "PPP1-2",
	"PRM-003P": "PPP1-3",
	"PRM-004P": "PPP1-4",
	"PRM-005P": "PPP1-5",

	// Crucible of Words promo
	"PWOR19-001": "PWOR-2019",

	// Greated Good
	"SLD-1693-IT": "SLD-1693",
}

// List of tags that need to be preserved in one way or another
var preserveTags = []string{
	"Display",
	"Etched",
	"Japanese",
	"JPN",
}

func setCodeExists(code string) bool {
	_, err := mtgmatcher.GetSet(code)
	return err == nil
}

func Preprocess(card CKCard) (*mtgmatcher.Card, error) {
	isFoil, _ := strconv.ParseBool(card.IsFoil)

	sku := card.SKU
	if sku == "" {
		return nil, errors.New("unsupported SKU format")
	}

	// Strip the initial F from set codes that do not exist
	if isFoil && strings.HasPrefix(sku, "F") && setCodeExists(strings.Split(sku, "-")[0][1:]) {
		sku = sku[1:]
	}

	// Custom replacements
	fixup, found := skuFixupTable[sku]
	if found {
		sku = fixup
	}

	// Retrieve setCode and number
	fields := strings.Split(sku, "-")
	if len(fields) < 2 {
		return nil, errors.New("unsupported SKU format")
	}
	setCode := fields[0]
	number := strings.Join(fields[1:], "")
	number = strings.TrimLeft(number, "0")
	number = strings.ToLower(number)
	number = strings.TrimRight(number, "jp")

	edition := setCode
	variation := number

	// Validate if setCode exists, if not preserve info from the card
	if !setCodeExists(setCode) {
		if (len(setCode) > 3 && setCodeExists(setCode[len(setCode)-3:])) ||
			(len(setCode) > 4 && setCodeExists(setCode[len(setCode)-4:])) {
			edition = card.Edition
			variation += " " + card.Variation
		}
	}

	switch card.Edition {
	case "World Championships":
		if strings.HasPrefix(number, "sr") {
			variation = strings.Replace(number, "sr", "shr", 1)
		}
	case "Deckmaster",
		"Collectors Ed",
		"Collectors Ed Intl":
		variation = card.Variation
	case "Promo Pack":
		variation = card.Variation
		edition = card.Edition
	case "Promotional":
		variation = card.Variation
		switch {
		case strings.Contains(variation, "APAC"),
			strings.Contains(variation, "Euro"):
			variation = number
		case strings.Contains(variation, "Arena"),
			strings.Contains(variation, "Game Day"),
			strings.Contains(variation, "Gameday"):
			edition = card.Edition
		case strings.Contains(variation, "Resale"):
			edition = "PRES"
			if len(mtgmatcher.MatchInSet(card.Name, "PMEI")) > 0 {
				edition = "PMEI"
				variation = ""
			}
		}
	case "Mystery Booster/The List":
		edition = card.Edition
		switch setCode {
		case "CMB1":
			variation = card.Variation
		// Code modified from original SKU
		case "ULST":
			edition = setCode
			variation = number
		default:
			variation = setCode[1:] + "-" + strings.TrimLeft(number, "0")
		}
	case "Streets of New Capenna Variants":
		if card.Name == "Gala Greeters" {
			variation = card.Variation
		}
	case "Ultimate Box Topper":
		edition = "PUMA"
	}

	// Preserve any remaining tag
	for _, tag := range preserveTags {
		if strings.Contains(card.Variation, tag) && !strings.Contains(variation, tag) {
			variation += " " + tag
		}
	}

	// Drop one side of dfc tokens
	if (strings.Contains(card.Name, " // ") || strings.Contains(card.Name, " - ")) &&
		(strings.Contains(card.Name, "Token") || strings.HasPrefix(setCode, "T") || strings.HasPrefix(setCode, "FT")) {
		if strings.Contains(card.Name, " // ") {
			card.Name = strings.Split(card.Name, " // ")[0] + " Token"
		} else {
			card.Name = strings.Split(card.Name, " - ")[0] + " Token"
		}
	}
	// Use number for tokens
	if strings.Contains(card.Name, "Token") || strings.Contains(card.Name, "Bounty") {
		// Quiet exit for duplicated tokens from this set
		if len(mtgmatcher.MatchInSetNumber(card.Name, setCode, number)) == 0 {
			return nil, mtgmatcher.ErrUnsupported
		}
	}

	return &mtgmatcher.Card{
		Name:      card.Name,
		Edition:   edition,
		Variation: variation,
		Foil:      isFoil,
	}, nil
}
