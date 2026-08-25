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
	"MF19-001":  "MPF19-1",
	"MZNR-091":  "MKHC-91",
	"MTAFR-001": "PLST-TAFR-1",

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

	// Maximum effort
	"SLD-IFIYW-01":   "SLD-IFIYW-1",
	"SLD-IFIYW-02":   "SLD-IFIYW-2",
	"SLD-IFIYW-03":   "SLD-IFIYW-3",
	"SLD-IFIYW-04":   "SLD-IFIYW-4",
	"SLD-IFIYW-05":   "SLD-IFIYW-5",
	"CFSLD-IFIYW-06": "SLD-IFIYW-6",
	"CFSLD-IFIYW-07": "SLD-IFIYW-7",
	"CFSLD-IFIYW-08": "SLD-IFIYW-8",
	"CFSLD-IFIYW-09": "SLD-IFIYW-9",
	"CFSLD-IFIYW-10": "SLD-IFIYW-10",
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

// The wrappings a sku puts on the code of the set a token was filed with,
// the treatment first and the token itself last
var tokenSetPrefixes = []string{"F", "T", "FT", "SF", "RF", "CF"}

// unindexedTokenSheet reports whether a sku names a sheet of tokens no set in
// the datastore stands for, as the Jumpstart theme cards do: the set the
// sheet came with is carried, the sheet itself never was, so no row of it can
// match and none is worth reporting.
func unindexedTokenSheet(sku string) bool {
	fields := strings.Split(sku, "-")
	if len(fields) < 2 || setCodeExists(fields[0]) {
		return false
	}
	for _, prefix := range tokenSetPrefixes {
		trimmed := strings.TrimPrefix(fields[0], prefix)
		if trimmed != fields[0] && setCodeExists(trimmed) {
			return true
		}
	}
	return false
}

// resolveEmblem answers the datastore's spelling and number for an emblem row,
// and empty strings for anything else or for a planeswalker the named set does
// not pin to exactly one emblem.
func resolveEmblem(edition, cardName, cardVariation string) (string, string) {
	face := cardName
	for _, separator := range []string{" // ", " - "} {
		before, _, found := strings.Cut(face, separator)
		if found {
			face = before
		}
	}

	hint := cardVariation
	inner, found := strings.CutPrefix(face, "Emblem (")
	if found {
		hint = strings.TrimSuffix(inner, ")")
	} else if face != "Emblem" {
		return "", ""
	}
	if hint == "" {
		return "", ""
	}

	set, err := mtgmatcher.GetSet(edition)
	if err != nil {
		return "", ""
	}

	var name, number string
	var matches int
	for _, token := range set.Tokens {
		if !strings.HasSuffix(token.Name, " Emblem") ||
			!strings.Contains(strings.ToLower(token.Name), strings.ToLower(hint)) {
			continue
		}
		name, number = token.Name, token.Number
		matches++
	}
	if matches != 1 {
		return "", ""
	}
	return name, number
}

// Preprocess turns a feed entry into the card description the matcher takes,
// reporting an error for the entries that are not cards.
func Preprocess(card cardkingdom.Product) (*mtgmatcher.InputCard, error) {
	foilVariant := strings.Contains(card.Variation, "Foil") && !strings.Contains(card.Variation, "Non")
	isFoil := card.IsFoil || foilVariant
	isEtched := strings.Contains(card.Variation, "Etched")

	// Retrieve setCode and number
	sku := card.SKU
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
		_, found := skuFixupTable[strings.TrimPrefix(card.SKU, "F")]
		if found && isFoil {
			fields = strings.Split(card.SKU, "-")
			variation = strings.TrimLeft(fields[1], "0")
		}
	case "Secret Lair":
		// Restore the correct hyphen set for this given drop
		if strings.Contains(number, "IFIYW") {
			variation = strings.Join(fields[1:], "-")
		}
	}

	// CK writes an emblem as the bare word plus the planeswalker, either
	// parenthesized in the name or left in the variation, while the
	// datastore spells the whole planeswalker name into the token name
	if name, number := resolveEmblem(edition, card.Name, card.Variation); name != "" {
		card.Name = name
		variation = number
	}

	// Preserve any remaining tag
	for _, tag := range preserveTags {
		if strings.Contains(card.Variation, tag) && !strings.Contains(variation, tag) {
			variation += " " + tag
		}
	}

	// Drop one side of dfc tokens, without doubling the suffix when the
	// kept face already carries it, and leaving alone the split cards a
	// T-prefixed set code sweeps in: the set carries those under both faces
	if (strings.Contains(card.Name, " // ") || strings.Contains(card.Name, " - ")) &&
		(strings.Contains(card.Name, "Token") || strings.HasPrefix(setCode, "T") || strings.HasPrefix(setCode, "FT")) &&
		len(mtgmatcher.MatchInSetNumber(card.Name, setCode, number)) == 0 {
		if strings.Contains(card.Name, " // ") {
			card.Name = strings.Split(card.Name, " // ")[0]
		} else {
			card.Name = strings.Split(card.Name, " - ")[0]
		}
		if !strings.HasSuffix(card.Name, "Token") {
			card.Name += " Token"
		}
	}
	// Tokens are filed under their own set code when the datastore carries
	// one, while a code it does not carry names the set the tokens are
	// filed with once its treatment and token wrappings are stripped
	if (strings.Contains(card.Name, "Token") || strings.Contains(card.Name, "Bounty")) &&
		!setCodeExists(setCode) {
		for _, prefix := range tokenSetPrefixes {
			trimmed := strings.TrimPrefix(setCode, prefix)
			if trimmed != setCode && setCodeExists(trimmed) {
				edition = trimmed
				break
			}
		}
	}

	// The treatment a token sku wraps its set code in promises a finish the
	// sheet was never sold in, and stripping the wrapping to reach the sheet
	// drops that promise: the row would land on the plain printing and be
	// priced as it, right beside the plain row that belongs there
	printing := tokenPrinting(edition, number)
	if isFoil && printing != nil &&
		!printing.HasFinish("foil") && !printing.HasFinish("etched") {
		return nil, mtgmatcher.ErrUnsupported
	}

	return &mtgmatcher.InputCard{
		Name:      card.Name,
		Edition:   edition,
		Variation: variation,
		Foil:      isFoil,
	}, nil
}

// tokenPrinting answers the printing a token sheet files at a number, asking
// the sheet rather than the name because a token only carries the Token
// suffix when a real card answers to the same name.
func tokenPrinting(code, number string) *mtgmatcher.Card {
	set, err := mtgmatcher.GetSet(code)
	if err != nil || set.Type != "token" {
		return nil
	}
	for i, printing := range set.Cards {
		if printing.Number == number {
			return &set.Cards[i]
		}
	}
	return nil
}

func preprocessGraded(title string) (*mtgmatcher.InputCard, error) {
	if strings.Contains(title, "Multiverse Mystery Slab") {
		return nil, mtgmatcher.ErrUnsupported
	}

	vars := mtgmatcher.SplitVariants(title)
	if len(vars) != 2 {
		return nil, errors.New("unsupported format")
	}

	cardName := vars[0]
	edition := strings.Replace(vars[1], "- ", "", -1)
	variant := ""

	// Remove serialized number tags
	if strings.Contains(cardName, "/") {
		fields := strings.Fields(cardName)
		for i := range fields {
			if strings.Contains(fields[i], "/") {
				fields[i] = ""
			}
		}
		cardName = strings.Join(fields, " ")
		cardName = strings.Replace(cardName, "  ", " ", -1)
	}

	for _, score := range supportedScores {
		before, after, found := strings.Cut(edition, score)
		if !found {
			continue
		}
		edition = strings.TrimSpace(before)
		variant = strings.TrimSpace(after)
		break
	}

	isFoil := strings.Contains(variant, "Foil") || strings.Contains(edition, "Foil")
	edition = strings.TrimSuffix(edition, " Foil")
	variant = strings.TrimSuffix(variant, " Foil")

	// Hack to remove 9.5-style scores
	variant = strings.Replace(variant, ".", "", -1)
	num := mtgmatcher.ExtractNumber(variant)
	if num != "" {
		variant = strings.Replace(variant, num, "", -1)
	}
	variant = strings.TrimSpace(variant)

	if strings.Contains(edition, "Final Fantasy") {
		if variant != "" {
			variant += " "
		}
		variant += edition
		if strings.HasPrefix(edition, "Final Fantasy Through the Ages") {
			edition = "FCA"
		}
	}

	// Move tags to the appropriate field to help edition matching
	for _, tag := range []string{
		"Borderless", "Extended Art", "Serialized", "Textured", "Japan Showcase", "Raised", "Halo",
		"Breaking News Showcase", "Breaking New", "Showcase Magnified", "Godzilla Series", "Etched",
		"Showcase", // needs to be last
	} {
		if strings.HasSuffix(edition, tag) {
			edition = strings.TrimSuffix(edition, " "+tag)
			if variant != "" {
				variant += " "
			}
			variant += tag
		}
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Edition:   edition,
		Variation: variant,
		Foil:      isFoil,
	}, nil
}

var supportedScores = []string{
	"PSA", "BGS", "CGC",
}

var gradeMap = map[string]map[string]string{
	"PSA": {
		"10": "NM",
		"9":  "NM",
		"8":  "NM",
		"7":  "NM",
		"6":  "SP",
		"5":  "SP",
		"4":  "MP",
		"3":  "MP",
		"2":  "HP",
		"1":  "HP",
	},
	"BGS": {
		"10": "NM",
		"9":  "NM",
		"8":  "SP",
		"7":  "SP",
		"6":  "MP",
		"5":  "MP",
		"4":  "MP",
		"3":  "HP",
		"2":  "HP",
		"1":  "PO",
	},
	"CGC": {
		"Pristine":          "NM",
		"Pristine 10":       "NM",
		"10":                "NM",
		"9":                 "NM",
		"8":                 "NM",
		"7":                 "SP",
		"6":                 "SP",
		"5":                 "MP",
		"4":                 "MP",
		"3":                 "HP",
		"2":                 "HP",
		"1":                 "PO",
		"Authentic Altered": "PO",
	},
}

func parseGradedCondition(title string) string {
	var grade string
	var score string
	for _, score = range supportedScores {
		_, after, found := strings.Cut(title, score+" ")
		if !found {
			continue
		}
		grade = after
		break
	}

	if grade == "" {
		return ""
	}

	grade = strings.Split(grade, ")")[0]
	grade = strings.Split(grade, ".")[0]
	grade = strings.TrimSuffix(grade, " Quad ++")
	grade = strings.TrimSuffix(grade, " Quad++")

	// A grade the table does not cover has no condition, and must not fall
	// through to the zero value: an empty condition is silently promoted to
	// NM when the entry is added
	condition, found := gradeMap[score][grade]
	if !found {
		return ""
	}

	return condition
}
