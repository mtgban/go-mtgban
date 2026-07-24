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

	"Doric, Nature's Warden // Doric, Owlbear Avenger": "Casal, Lurkwood Pathfinder // Casal, Pathbreaker Owlbear",

	// SCG misspells the back face ("Shidake" vs the correct "Shidako").
	"Orochi Eggwatcher // Shidake, Broodmistress": "Orochi Eggwatcher // Shidako, Broodmistress",
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
	// SCG's "3rd Edition - Black Border" is mtgjson's Foreign Black Border.
	if setCode == "3BB" {
		return "FBB"
	}
	// SCG's DD3 is the third Duel Deck (Divine vs. Demonic) on The List.
	if setCode == "DD3" {
		return "DDD"
	}
	_, err := mtgmatcher.GetSet(setCode)
	if err != nil && len(setCode) > 3 && unicode.IsDigit(rune(setCode[len(setCode)-1])) {
		switch setCode {
		case "4ED2":
			setCode = "4EDALT"
		case "MPS3":
			setCode = "MP2"
		case "CM12", "CMD2", "C132", "C142", "C152", "C162", "C172", "C182", "C192", "C202":
			setCode = "O" + setCode[:3]
		case "UMA2":
			setCode = "PUMA"
		case "MH12", "MH13":
			setCode = "H1R"
		case "MH22", "MH23":
			setCode = "MH2"
		case "MB12":
			setCode = "CMB1"
		case "MB13":
			setCode = "CMB2"
		default:
			setCode = setCode[:len(setCode)-1]
		}
	}
	return setCode
}

// Arena League ran 1996-2006, so this set list is closed.
var arenaLeagueSets = []string{"PARL", "PAL99", "PAL00", "PAL01", "PAL02", "PAL03", "PAL04", "PAL05", "PAL06"}

// arenaLeaguePrinting returns the set code and number of the card's single
// Arena League printing, if exactly one exists across all Arena League sets.
// Basics (printed every year) resolve to more than one and are left alone.
func arenaLeaguePrinting(cardName string) (code, number string, ok bool) {
	count := 0
	for _, set := range arenaLeagueSets {
		for _, c := range mtgmatcher.MatchInSet(cardName, set) {
			code, number = set, c.Number
			count++
		}
	}
	return code, number, count == 1
}

// playPromoPrefixes are the SCG SKU prefixes for Wizards Play Network / store
// event promos (Play, Commander Party, Bring-a-Friend, Open House, Two-Headed
// Giant, Store event, Magic promo, WPN, Draft Weekend, Release). They all map
// to a Play-promo printing regardless of the SKU's year/set, so they're
// resolved by looking the card up rather than parsing the SKU.
var playPromoPrefixes = map[string]bool{
	"PLAY": true, "CP": true, "BAF": true, "OPNH": true, "TWO": true,
	"SSD": true, "MA": true, "WPNP": true, "DRFT": true, "RLS": true,
	"LNY": true, "PRES": true,
}

// isPlayPromoSet reports whether a set code is one of the Wizards Play Network /
// store-promo families: PW* (Wizards Play), PLG* (Love Your LGS), PL<year>
// (Play), PSPL.
func isPlayPromoSet(set string) bool {
	return strings.HasPrefix(set, "PW") || strings.HasPrefix(set, "PLG") ||
		set == "PSPL" ||
		(strings.HasPrefix(set, "PL") && len(set) > 2 && set[2] >= '0' && set[2] <= '9')
}

// isPWYear reports whether a set code is a yearly Wizards Play set (PW<digits>),
// preferred over one-off PW* sets when a card appears in several.
func isPWYear(set string) bool {
	return strings.HasPrefix(set, "PW") && len(set) > 2 && set[2] >= '0' && set[2] <= '9'
}

// playPromoPrinting returns the set code and number of the card's single
// Play-promo printing across the Wizards Play Network families (PW*, PLG*,
// PSPL), if exactly one exists. The SCG SKU's year/set is unreliable (some are
// year-offset, some map to PLG/PSPL), so the unique printing is authoritative;
// cards with none (non-WPN promos) or several are left alone.
func playPromoPrinting(cardName string) (code, number string, ok bool) {
	type printing struct{ code, number string }
	var all, pwYear []printing
	for _, set := range mtgmatcher.GetAllSets() {
		if !isPlayPromoSet(set) {
			continue
		}
		for _, c := range mtgmatcher.MatchInSet(cardName, set) {
			all = append(all, printing{set, c.Number})
			if isPWYear(set) {
				pwYear = append(pwYear, printing{set, c.Number})
			}
		}
	}
	if len(all) == 1 {
		return all[0].code, all[0].number, true
	}
	// When a card appears in several Play sets, prefer its single yearly
	// Wizards Play (PW<year>) printing over one-off PW* sets.
	if len(pwYear) == 1 {
		return pwYear[0].code, pwYear[0].number, true
	}
	return "", "", false
}

// SKU documented as
// * for singles:
// SGL-[Brand]-[Set]-[Collector Number]-[Language][Foiling][Condition]
// * for world champtionship:
// SGL-[Brand]-WCHP-[Year][Player Initials]_[Set]_[Collector Number][Sideboard]-[Language][Foiling][Condition]
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

	// The Italian Legends and The Dark are their own sets rather than
	// foreign-language reprints of the English ones. Only remap when the set is
	// present, otherwise the English printing would be matched instead.
	if language == "IT" {
		var italian string
		switch setCode {
		case "LEG":
			italian = "LEGITA"
		case "DRK":
			italian = "DRKITA"
		}
		if italian != "" {
			if _, err := mtgmatcher.GetSet(italian); err == nil {
				setCode = italian
			}
		}
	}

	switch setCode {
	case "WCHP":
		year := number[:2]
		setCode = "WC" + year
		if year == "96" {
			setCode = "PTC"
		}

		fields := strings.Split(number, "_")
		cards := mtgmatcher.MatchInSet(cardName, setCode)
		if len(cards) == 1 {
			number = cards[0].Number
		} else if len(fields) == 3 {
			initials := strings.ToLower(fields[0][2:])
			subNumber := strings.TrimLeft(fields[2], "0")
			number = initials + subNumber
			// rebuild "sideboard"
			if strings.HasSuffix(number, "s") {
				number += "b"
			}
		}
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
				// MagicFest <year> is the PF<yy> promo set.
				if fields[1] == "MF" && len(fields[2]) == 4 {
					fields[2] = "PF" + fields[2][2:]
				}
				number = fields[2] + "-" + strings.TrimLeft(fields[3], "0")

				// Single promo from Unfinity
				if fields[2] == "UST" {
					setCode = "ULST"
					number = strings.TrimLeft(fields[3], "0")
				}
			}
		}
	case "PRM", "PRM3":
		fields := strings.Split(number, "_")

		switch {
		// Decouple Secret Lair
		case len(fields) > 2 && fields[0] == "SECRET":
			setCode = fields[1]
			number = strings.TrimLeft(fields[2], "0")
			if len(mtgmatcher.MatchWithNumber(cardName, setCode, number)) == 0 &&
				len(mtgmatcher.MatchWithNumber(cardName, "SLP", number)) > 0 {
				setCode = "SLP"
			}
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
		case strings.HasPrefix(number, "SPT_"):
			setCode = "PSPL"
			cards := mtgmatcher.MatchInSet(cardName, setCode)
			if len(cards) == 1 {
				number = cards[0].Number
			}
		case len(fields) > 2 && len(fields[1]) == 4 &&
			(strings.HasPrefix(number, "FEST_") || strings.HasPrefix(number, "CF_")):
			setCode = "PF" + fields[1][2:]

			if cardName == "Counterspell" && setCode == "PF23" {
				setCode = "PF24"
			}

			cards := mtgmatcher.MatchInSet(cardName, setCode)
			if len(cards) == 1 {
				number = cards[0].Number
			}
		case strings.HasPrefix(number, "NYCC24_"):
			setCode = "PURL"
			cards := mtgmatcher.MatchInSet(cardName, setCode)
			if len(cards) == 1 {
				number = cards[0].Number
			}
		case strings.HasPrefix(number, "PT_"):
			setCode = "PPRO"
			cards := mtgmatcher.MatchInSet(cardName, setCode)
			if len(cards) == 1 {
				number = cards[0].Number
			}
		case strings.HasPrefix(number, "LYLGS_"):
			setCode = "PLG" + fields[1][2:]
			if strings.HasPrefix(number, "LYLGS_2021b") {
				setCode = "PLG21"
			}
			cards := mtgmatcher.MatchInSet(cardName, setCode)
			if len(cards) == 1 {
				number = cards[0].Number
			}
		case strings.HasPrefix(number, "BAB_MH"):
			if number == "BAB_MH3_496" {
				setCode = "MH3"
				number = "496"
			} else if number == "BAB_MH1_255" {
				setCode = "MH1"
				number = "255"
			}
		case strings.HasPrefix(number, "SSD_2024"):
			setCode = "PSS4"
			if number == "SSD_2024_005b" {
				setCode = "PCBB"
				number = "5"
			}
		case strings.HasPrefix(number, "CD_") && len(fields) > 2:
			setCode = fields[1]
			number = strings.TrimLeft(fields[2], "0")
		case strings.HasPrefix(number, "MB2_") && len(fields) > 2:
			// Mystery Booster 2 reprints are catalogued in The List (PLST)
			// under their source set, numbered <source-set>-<number> (some
			// carry a letter, e.g. TD0-A80). Only remap when the card actually
			// has such a PLST printing, so an unrelated MB2_ sub-code is left
			// alone rather than blindly forced into PLST.
			prefix := fields[1] + "-"
			num := strings.TrimLeft(fields[2], "0")
			for _, c := range mtgmatcher.MatchInSet(cardName, "PLST") {
				if !strings.HasPrefix(c.Number, prefix) {
					continue
				}
				digits := strings.TrimLeftFunc(strings.TrimPrefix(c.Number, prefix),
					func(r rune) bool { return !unicode.IsDigit(r) })
				if digits == num {
					setCode = "PLST"
					number = c.Number
					break
				}
			}
		case strings.HasPrefix(number, "ARENA_"):
			// Arena League promo: resolve to the card's unique arenaleague
			// printing (the base-set field in the SKU doesn't map to a fixed
			// Arena League year).
			if code, num, ok := arenaLeaguePrinting(cardName); ok {
				setCode = code
				number = num
			} else if len(fields) > 1 {
				// Basic lands appear in every Arena League year, so there's no
				// unique printing; let the matcher's arena handling pick the
				// right year from the base set (its name carries the year hint).
				if base, err := mtgmatcher.GetSet(fields[1]); err == nil {
					arena := mtgmatcher.InputCard{
						Name:      cardName,
						Variation: "Arena " + base.Name,
						Foil:      foil,
						Language:  language,
					}
					if id, err := mtgmatcher.Match(&arena); err == nil {
						return &mtgmatcher.InputCard{Id: id, Foil: foil, Language: language}, nil
					}
				}
			}
		case strings.HasPrefix(number, "EWK_") && len(fields) > 1:
			// Eternal Weekend promo; the PEWK number is prefixed with the year.
			setCode = "PEWK"
			for _, c := range mtgmatcher.MatchInSet(cardName, setCode) {
				if strings.HasPrefix(c.Number, fields[1]) {
					number = c.Number
					break
				}
			}
		case strings.HasPrefix(number, "PRE_") && len(fields) == 3:
			// Prerelease promo. Usually P<SET> #<num>s, but some sets carry the
			// prerelease reprint in the main set instead (e.g. LCI #188), so
			// fall back to <SET> #<num> when the promo set has no such card.
			num := strings.TrimLeft(fields[2], "0")
			if len(mtgmatcher.MatchWithNumber(cardName, "P"+fields[1], num+"s")) > 0 {
				setCode = "P" + fields[1]
				number = num + "s"
			} else {
				setCode = fields[1]
				number = num
			}
		case strings.HasPrefix(number, "SCHP_") && len(fields) == 3:
			// Store Championship promo: SCHP_<year>_<num> -> SCH #<num>.
			setCode = "SCH"
			number = strings.TrimLeft(fields[2], "0")
		case strings.HasPrefix(number, "15A_") && len(fields) == 3:
			// 15th Anniversary promo.
			setCode = "P15A"
			if cards := mtgmatcher.MatchInSet(cardName, setCode); len(cards) == 1 {
				number = cards[0].Number
			}
		case len(fields) > 0 && playPromoPrefixes[fields[0]]:
			// Wizards Play Network / store-event promo: resolve to the card's
			// unique Play-promo printing. When there's no such printing, some of
			// these (e.g. RLS_INR release promos) reference a real set directly.
			if code, num, ok := playPromoPrinting(cardName); ok {
				setCode = code
				number = num
			} else if len(fields) == 3 {
				if _, err := mtgmatcher.GetSet(fields[1]); err == nil {
					setCode = fields[1]
					number = strings.TrimLeft(fields[2], "0")
				}
			}
		case (strings.HasPrefix(number, "BUN_") || strings.HasPrefix(number, "BAB_")) && len(fields) == 3:
			// Bundle / Buy-a-Box promo: P<SET> #<num>.
			setCode = "P" + fields[1]
			number = strings.TrimLeft(fields[2], "0")
		}
	case "PUMA":
		cards := mtgmatcher.MatchInSet(cardName, setCode)
		if len(cards) == 1 {
			number = cards[0].Number
		}
	case "MH2":
		cards := mtgmatcher.MatchWithNumber(cardName, "H2R", number)
		if len(cards) == 1 {
			setCode = "H2R"
			number = cards[0].Number
		}
	default:
		if strings.Contains(cardName, "//") {
			number = strings.TrimSuffix(number, "a")
		}
	}

	backup := mtgmatcher.InputCard{
		Edition:   setCode,
		Variation: number,
		Foil:      foil,
		Language:  language,
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

			// Let's check if there is a duplicated card somewhere, and repeat the check
			out := mtgmatcher.MatchWithNumber(cardName, setCode, number+"★")
			if len(out) == 1 {
				card = out[0]

				if len(card.Finishes) == 1 &&
					(((card.HasFinish(mtgmatcher.FinishFoil) || card.HasFinish(mtgmatcher.FinishEtched)) && !foil) ||
						(card.HasFinish(mtgmatcher.FinishNonfoil) && foil)) {
					return &backup, errors.New("invalid number/foil combination")
				}
			} else {
				return &backup, errors.New("invalid number/foil combination")
			}
		}

		// Force Etched for specific sets that need extra decoupling
		variant := ""
		if strings.Contains(SKU, "-STA-") || strings.Contains(SKU, "-STA2-") ||
			strings.Contains(SKU, "-MH2-") || strings.Contains(SKU, "-MH22-") || strings.Contains(SKU, "-MH23-") ||
			strings.Contains(SKU, "-MH1-") || strings.Contains(SKU, "-MH12-") || strings.Contains(SKU, "-MH13-") {
			isEtched := foil && (strings.Contains(SKU, "-STA2-") || strings.Contains(SKU, "-MH23-") || strings.Contains(SKU, "-MH13-"))

			card.UUID, _ = mtgmatcher.MatchId(card.UUID, foil, isEtched)

			if isEtched {
				variant = "etched"
			}
		}

		return &mtgmatcher.InputCard{
			Id:        card.UUID,
			Variation: variant,
			Foil:      foil,
			Language:  language,
		}, nil
	}
	if len(out) > 1 {
		alias := mtgmatcher.NewAliasingError()
		for _, id := range out {
			alias.Dupes = append(alias.Dupes, id.UUID)
		}
		return &backup, alias
	}
	return &backup, errors.New("not found")
}

func preprocess(hit Hit) (*mtgmatcher.InputCard, error) {
	card := hit.Variants[0]
	edition := hit.SetName
	language := hit.Language
	foil := hit.FinishPricingTypeID == 2
	cn := strings.TrimLeft(hit.CollectorNumber, "0")

	// Processing variant first because it gets added on later
	variant := card.Subtitle
	variant = strings.Replace(variant, "(", "", -1)
	variant = strings.Replace(variant, ")", "", -1)

	cardName := hit.Name
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
	isToken := strings.HasPrefix(hit.Name, "{") && strings.Contains(hit.Name, "}")
	if isToken && !strings.Contains(cardName, "Token") {
		cardName += " Token"
	}

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

	canProcessSKU := true
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
			return out, nil
		}

		// In case SKU processing failed, gather valid info as much as possible.
		// A malformed SKU yields a nil card, so guard before touching it.
		if out != nil {
			if _, err := mtgmatcher.GetSet(out.Edition); err == nil {
				edition = out.Edition
			}
			if _, err := strconv.Atoi(out.Variation); err == nil {
				variant = out.Variation
			}
			foil = out.Foil
			language = out.Language
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
			if variant == "Love Your LGS" || variant == "Love Your LGS Retro" {
				edition = "PLG21"
			}
		case "Arcane Signet":
			switch variant {
			case "Play Draft Retro":
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
			case "Hasbro Retro":
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
		case "Fabled Passage":
			if variant == "Love Your LGS Retro" {
				edition = "PW21"
			}
		case "Lightning Bolt":
			if variant == "MagicFest" {
				edition = "PF19"
				if strings.Contains(card.Sku, "2025") {
					edition = "PF25"
				}
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
	case "Modern Horizons 2":
		if len(mtgmatcher.MatchWithNumber(cardName, "H2R", cn)) == 1 {
			edition = "H2R"
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
