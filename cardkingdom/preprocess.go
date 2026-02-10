package cardkingdom

import (
	"errors"
	"strings"

	"github.com/mtgban/go-cardkingdom"
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

	// Lightning Bolt
	"F19-001": "PF19-001",

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
	"MF19-001": "MPF19-1",
	"MZNR-091": "MKHC-91",

	// Naya Sojourners
	"PM10-028": "DCI-29",
	// Mitotic Slime
	"PM11-185": "DCI-53",

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

	// Flusterstorm BaB
	"MH1-255P":  "MH1-255",
	"PMH3-0496": "MH3-496",

	// Glimpse, the Unthinkable
	"MB2-0355": "MB2-594",

	// Spider-man Play Promos
	"PPSPM-0002B":  "PW25-10",
	"FPPSPM-0002B": "PW25-10",
	"PPSPM-0005":   "PW25-12",
	"FPPSPM-0005":  "PW25-12",
	"PPSPM-0003B":  "PW25-13",
	"FPPSPM-0003B": "PW25-13",

	// Some Avatar Eternal cards got merged in foil/nonfoil,
	// but they actually have different numbers
	"TLE-0210": "TLE-265",
	"TLE-0211": "TLE-266",
	"TLE-0212": "TLE-267",
	"TLE-0214": "TLE-268",
	"TLE-0215": "TLE-269",
	"TLE-0217": "TLE-270",
	"TLE-0218": "TLE-273",
	"TLE-0219": "TLE-274",
	"TLE-0220": "TLE-275",
	"TLE-0221": "TLE-276",
	"TLE-0234": "TLE-277",
	"TLE-0235": "TLE-278",
	"TLE-0236": "TLE-279",
	"TLE-0238": "TLE-280",
	"TLE-0239": "TLE-281",
	"TLE-0240": "TLE-282",
	"TLE-0241": "TLE-283",
	"TLE-0244": "TLE-285",
	"TLE-0245": "TLE-286",
	"TLE-0246": "TLE-287",
	"TLE-0247": "TLE-288",
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

func Preprocess(card cardkingdom.Product) (*mtgmatcher.InputCard, error) {
	foilVariant := strings.Contains(card.Variation, "Foil") && !strings.Contains(card.Variation, "Non")
	isFoil := card.IsFoil || foilVariant
	isEtched := strings.Contains(card.Variation, "Etched")

	// Retrieve setCode and number
	sku := card.Sku
	fields := strings.Split(sku, "-")
	if len(fields) < 2 {
		return nil, errors.New("unsupported SKU format")
	}
	setCode := fields[0]

	// Strip the initial F from set codes that do not exist
	if isFoil && strings.HasPrefix(sku, "F") && setCodeExists(setCode[1:]) {
		sku = sku[1:]
	}
	// Same for Etched and E
	if isEtched && strings.HasPrefix(sku, "E") && setCodeExists(setCode[1:]) {
		sku = sku[1:]
	}
	// ccccombo (EF is for emblem foils)
	if isFoil && isEtched && strings.HasPrefix(sku, "FE") && setCodeExists(setCode[2:]) {
		sku = sku[2:]
	}

	// Custom replacements
	fixup, found := skuFixupTable[sku]
	if found {
		sku = fixup
	}

	// Update the fields if needed
	fields = strings.Split(sku, "-")
	setCode = fields[0]

	number := strings.Join(fields[1:], "")
	number = strings.TrimLeft(number, "0")
	number = strings.TrimRight(number, "JP")
	number = strings.TrimRight(number, "IT")

	edition := setCode
	variation := strings.ToLower(number)

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
		if strings.HasPrefix(variation, "sr") {
			variation = strings.Replace(variation, "sr", "shr", 1)
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
		case strings.Contains(variation, "Symbol"):
			maybeNum := setCode + "-" + strings.TrimLeft(number, "0")
			if len(mtgmatcher.MatchInSetNumber(card.Name, "PLST", maybeNum)) == 1 {
				edition = "PLST"
				variation = maybeNum
			}
		case strings.Contains(variation, "Ugin's Fate"):
			edition = "UGIN"
		case strings.Contains(setCode, "DFT") && strings.Contains(card.Name, "Raceway"):
			edition = "DFT"
			variation += " Bundle"
		case variation == "Commander's Bundle Promo":
			edition = strings.TrimPrefix(setCode, "P")
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
	case "Avatar: The Last Airbender Eternal-Legal":
		// Look up the sku again, and restore the original one if foil
		_, found := skuFixupTable[strings.TrimPrefix(card.Sku, "F")]
		if found && isFoil {
			fields = strings.Split(card.Sku, "-")
			variation = strings.TrimLeft(fields[1], "0")
		}
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

	return &mtgmatcher.InputCard{
		Name:      card.Name,
		Edition:   edition,
		Variation: variation,
		Foil:      isFoil,
	}, nil
}
