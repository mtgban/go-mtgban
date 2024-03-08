package cardkingdom

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// This table contains all SKUs that contain incorrect codes or codes that could
// be mistaken for edition codes (thus misdirecting the matcher) or that contain
// incorrect numbers. Sometimes both.
var skuFixupTable = map[string]string{
	// Mishra's Factory
	"ATQ-080A": "ATQ-080C",
	"ATQ-080B": "ATQ-080A",
	"ATQ-080C": "ATQ-080B",
	"ATQ-080D": "ATQ-080D",

	// Extremely Slow Zombie
	"UST-054C":  "UST-54A",
	"UST-054A":  "UST-54B",
	"UST-054D":  "UST-54C",
	"UST-054B":  "UST-54D",
	"FUST-054C": "UST-54A",
	"FUST-054A": "UST-54B",
	"FUST-054D": "UST-54C",
	"FUST-054B": "UST-54D",

	// Some of the lands from the first Arena set
	"PAL96-001": "PARL-001",
	"PAL96-003": "PARL-003",
	"PAL96-004": "PARL-004",

	// Sets containing launch promos
	"PJMP-496B": "JMP-496",
	"P2XM-383":  "2XM-383",
	"P2XM-384":  "2XM-384",
	"P2X2-578":  "2X2-578",
	"P2X2-579":  "2X2-579",
	"P40K-181":  "40K-181",
	"PUNF-538":  "UNF-538",

	// Jaya Ballard, Task Mage
	"MPS-001A": "PRES-001A",

	// Mind Stone
	"FPLG21-001": "PW21-005",

	// Path of Ancestry
	"PF21-001":  "PLG21-C3",
	"FPF21-001": "PLG21-C3",

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
	// Liliana's Specter
	"PM11-104": "PDCI-52",
	// Mitotic Slime
	"PM11-185": "PDCI-53",
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
			strings.Contains(variation, "Premier"),
			strings.Contains(variation, "Commander Promo"),
			strings.Contains(variation, "Festival"),
			strings.Contains(variation, "MPS"):
			variation = number
		case edition == "PW22":
			variation = ""
		}
	case "World Championships":
		edition = setCode
		variation = number

		// Duplicate sku
		switch sku {
		case "PTC-ET015SB":
			if card.Name == "Circle of Protection: Red" {
				variation = "et17sb"
			}
		case "PTC-MJ364":
			if strings.Contains(card.Variation, "Michael Loconto") {
				variation = "ml364"
			}
		case "PTC-MJ365":
			if strings.Contains(card.Variation, "Michael Loconto") {
				variation = "ml365"
			}
		case "PTC-MJ366":
			if strings.Contains(card.Variation, "Michael Loconto") {
				variation = "ml366"
			}
		case "WC99-ML347b":
			if strings.Contains(card.Variation, "TMP - A") {
				variation = "ml347a"
			}
		case "WC02-CR335":
			if strings.Contains(card.Variation, "Sim Han How") {
				variation = "shh335"
			}
		default:
			if strings.HasPrefix(variation, "sr") {
				variation = strings.Replace(variation, "sr", "shr", 1)
			}
		}
	case "Alpha", "Beta", "Unlimited", "3rd Edition", "4th Edition",
		"Antiquities", "Fallen Empires", "Alliances", "Homelands",
		"Zendikar", "Battle for Zendikar", "Oath of the Gatewatch",
		"Unstable", "Unglued", "Unfinity", "Portal II",
		"The Lord of the Rings: Tales of Middle-earth",
		"The Lord of the Rings: Tales of Middle-earth Commander Decks",
		"The Lord of the Rings: Tales of Middle-earth Variants",
		"Ravnica Remastered", "Ravnica Remastered Variants",
		"Murders at Karlov Manor", "Murders at Karlov Manor Variants":
		variation = number

	case "Secret Lair":
		// The SLP cards need to be recognized differently
		// So do the the Step-and-Compleat cards
		if !strings.Contains(variation, "Prize") && !strings.HasPrefix(sku, "SAC") {
			variation = number
		}
		// Override variation due to the SLD thick cards using the same non-thick number
		if strings.Contains(card.Variation, "Display") {
			variation = "Thick Display"
		}
	case "Innistrad: Midnight Hunt Variants", "Innistrad: Crimson Vow Variants":
		if variation == "Eternal Night" {
			variation = "Showcase"
		}
	case "Mystery Booster/The List":
		switch setCode {
		case "ULST":
			edition = setCode
			variation = number
		case "MCMB1", "CMB1":
		default:
			if variation != "" {
				variation = setCode[1:] + "-" + strings.TrimLeft(number, "0")
			}
		}
	}

	// Preserve Etched property in case variation became overwritten with the number
	if strings.Contains(card.Variation, "Etched") && !strings.Contains(variation, "Etched") {
		variation += " Etched"
	}

	// Drop one side of dfc tokens
	if (strings.Contains(card.Name, " // ") || strings.Contains(card.Name, " - ")) &&
		(strings.Contains(card.Name, "Token") || (len(setCode) > 3 && setCode[0] == 'T')) {
		if strings.Contains(card.Name, " // ") {
			card.Name = strings.Split(card.Name, " // ")[0] + " Token"
		} else {
			card.Name = strings.Split(card.Name, " - ")[0] + " Token"
		}
	}
	// Use number for tokens
	if strings.Contains(card.Name, "Token") {
		variation = number

		if edition == "Duel Decks: Garruk Vs. Liliana" && card.Name == "Beast Token" {
			variation = "T" + number
		}

		// Quiet exit for duplicated tokens from this set
		if setCode == "TC16" && (strings.HasSuffix(number, "a") || strings.HasSuffix(number, "b")) {
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
