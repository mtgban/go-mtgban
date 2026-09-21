package coolstuffinc

import (
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var numFixes = map[string]string{
	"GolgariSignetCM2":                "CM2191",
	"GolgariSignetCM2v2":              "CM2192",
	"TempleoftheFalse_God271":         "CM2271",
	"TempleoftheFalseGod":             "CM2272",
	"SolemnSimulacrum":                "CM2218",
	"SolemnSimulacrum__219v2":         "CM2219",
	"Aura_Shards":                     "PLSTCMD-182",
	"aurashardslist2":                 "PLSTINV-233",
	"NissaWhoShakestheWorld518v2":     "SLD518",
	"SorcerousSpyglassv2":             "PXLN248p",
	"253486Signed_gold":               "WC97JK1",
	"one420eleshnornmotherofmachines": "ONE420",
	"295044":                          "SLD51",
	"wild0042":                        "SLD42",
	"Sol381656":                       "SLD1512",
	"TDM0300a":                        "TDM300",
	"LTR0425":                         "LTR425",
	"LeylineoftheVoidv2":              "PM20107p",
	"386443":                          "POTJ149p",
	"Ugin001":                         "M211",
}

var variantTable = map[string]string{
	"Jeff A Menges":                                    "Jeff A. Menges",
	"Jeff a Menges":                                    "Jeff A. Menges",
	"San Diego Comic-Con Promo M15":                    "SDCC 2014",
	"San Diego Comic-Con Promo M14":                    "SDCC 2013",
	"EURO Land White Cliffs of Dover Ben Thompson art": "EURO White Cliffs of Dover",
	"EURO Land Danish Island Ben Thompson art":         "EURO Land Danish Island",
	"Eighth Edition Prerelease Promo":                  "Release Promo",
	"Release 27 Promo":                                 "Release",
	"2/2 Power and Toughness":                          "misprint",
	"Big Furry Monster Left Side":                      "28",
	"Big Furry Monster Right Side":                     "29",
}

var nameTable = map[string]string{
	"Yennet, Cryptic Sovereign":              "Yennett, Cryptic Sovereign",
	"Invasion of Moag // Bloomweaver Dryads": "Invasion of Moag // Bloomwielder Dryads",
	"Bene Supremo":                           "Greater Good",
	"Ambitious Farmhand // Seasoned Cather":  "Ambitious Farmhand // Seasoned Cathar",
	"Maalfield Twins":                        "Maalfeld Twins",
	"Environmental Studies":                  "Environmental Sciences",
	"Proficient Pryodancer":                  "Proficient Pyrodancer",
	"Leotau Grizalho":                        "Grizzled Leotau",
	"____ ____ ____ Trespasser":              "_____ _____ _____ Trespasser",
	"Garravoraz":                             "Vorstclaw",
	"Gepanzerter Wasserwanderer":             "Plated Seastrider",
	"Camminatore di Phyrexia":                "Phyrexian Walker",
	"Odric, Lunarch Marshall":                "Odric, Lunarch Marshal",
}

func preprocess(b *mtgmatcher.Backend, cardName, edition, variant, imgURL string) (*mtgmatcher.InputCard, error) {
	imgName := strings.TrimSuffix(path.Base(imgURL), filepath.Ext(imgURL))
	fixup, found := numFixes[imgName]
	if found {
		imgName = fixup
	}

	variant = cleanVariant(variant)
	vars, found := variantTable[variant]
	if found {
		variant = vars
	}

	if strings.Contains(cardName, "Signed") && strings.Contains(cardName, "by") {
		cuts := mtgmatcher.Cut(cardName, "Signed")
		cardName = cuts[0]
	}

	variants := mtgmatcher.SplitVariants(cardName)
	if len(variants) > 1 {
		cardName = variants[0]
		if variant != "" {
			variant += " "
		}
		variant += strings.Join(variants[1:], " ")
	}

	isFoil := false
	if strings.Contains(cardName, "FOIL") {
		cardName = strings.Replace(cardName, " FOIL", "", 1)
		isFoil = true
	}

	if strings.HasSuffix(cardName, "Promo") {
		cuts := mtgmatcher.Cut(cardName, "Promo")
		cardName = cuts[0]
	}
	cardName = strings.TrimSpace(cardName)

	fixup, found = nameTable[cardName]
	if found {
		cardName = fixup
	}

	cardName, edition, variant, err := magicShelfFixups(cardName, edition, variant)
	if err != nil {
		return nil, err
	}

	// Skip tokens with the same names as cards
	if strings.Contains(variant, "Emblem") && !b.IsToken(cardName) {
		return nil, mtgmatcher.ErrUnsupported
	}

	if len(imgName) > 4 {
		for i := range 2 {
			maybeSet := strings.ToUpper(imgName[:i+3])
			maybeNum := strings.TrimLeft(imgName[i+3:], "_0")
			if len(b.MatchInSetNumber(cardName, maybeSet, maybeNum)) == 1 {
				return &mtgmatcher.InputCard{
					Name:      cardName,
					Variation: maybeNum,
					Edition:   maybeSet,
					Foil:      isFoil,
				}, nil
			}
		}
		// A letter can stand between the set code and the number, marking
		// the treatment: "TMCS0032" is the surge foil of TMC 32, where
		// "TMC0093" is the pixel art one filed at its own number. Neither
		// length above reads it - three characters leave "S0032", which is
		// no number, and four leave "TMCS", which is no set - so the two
		// listings both answer with 93 and the cheaper of them is priced as
		// the dearer. The letters are dropped and the digits behind them
		// asked for, which the set and the number together still have to
		// agree on. The promo pack is left alone: there the letters name a
		// set of its own rather than a treatment, and "BIGUPP0006" is the
		// promo pack printing, not the sixth card of The Big Score.
		maybeSet := strings.ToUpper(imgName[:3])
		maybeNum := strings.TrimLeft(imgName[3:], "_0")
		trimmed := strings.TrimLeft(maybeNum, letters)
		if trimmed != maybeNum && edition != "Universal Promo Pack" {
			maybeNum = strings.TrimLeft(trimmed, "_0")
			if len(b.MatchInSetNumber(cardName, maybeSet, maybeNum)) == 1 {
				return &mtgmatcher.InputCard{
					Name:      cardName,
					Variation: maybeNum,
					Edition:   maybeSet,
					Foil:      isFoil,
				}, nil
			}
		}
	}

	var language string
	switch edition {
	// The shelf mixes every foreign black-bordered print run under one
	// name; the language written in the note or the bracket is the only
	// thing that says which one. Only Italian (FBB) and Japanese (4BB)
	// have a row in the datastore - the rest were never captured as their
	// own set, so a German/French/Spanish/Chinese/Korean listing has
	// nothing to resolve against.
	case "Black Bordered (foreign)":
		switch {
		case strings.Contains(variant, "Italian"):
			edition = "FBB"
			language = "Italian"
		case strings.Contains(variant, "Japanese"):
			edition = "4BB"
			language = "Japanese"
		default:
			return nil, mtgmatcher.ErrUnsupported
		}

	case "Promo":
		// Black Lotus - Ultra Pro Puzzle - Eight of 9
		if strings.Contains(cardName, "Ultra Pro Puzzle") {
			return nil, mtgmatcher.ErrUnsupported
		}

		switch variant {
		case "Junior Super Series Promo",
			"Junior Super Series Promo Carl Critchlow art":
			edition = "PSUS"
			variant = ""
		default:
			possibleEd, possibleVar := card2promo(cardName, variant)
			if variant != possibleVar {
				variant = possibleVar
			}
			if possibleEd != "" {
				edition = possibleEd
			}
		}

	case "Prerelease Promo":
		variant = strings.Replace(variant, "Core 21", "Core Set 2021", 1)
		if variant == "Ixalan Prerelease Promo" {
			variant = "Prerelease Ixalan"
		}

	case "Universal Promo Pack":
		if strings.HasPrefix(imgName, "UPP") && len(imgName) > 6 {
			maybeSet := strings.ToUpper(imgName[3:6])
			if maybeSet != "" {
				variant = maybeSet
			}
		}

	case "Deckmasters":
		variant = strings.TrimSpace(strings.Split(variant, "Deckmaster")[0])

	case "Unfinity":
		variant = strings.Replace(variant, ",", "/", -1)

	case "Conspiracy: Take the Crown":
		if cardName == "Kaya, Ghost Assassin" && variant == "Alternate Art Foil" {
			variant = "222"
		}
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      isFoil,
		Language:  language,
	}, nil
}

func cleanVariant(variant string) string {
	if strings.Contains(variant, "Picture") {
		variant = strings.Replace(variant, "Picture 1", "", 1)
		variant = strings.Replace(variant, "Picture 2", "", 1)
		variant = strings.Replace(variant, "Picture 3", "", 1)
		variant = strings.Replace(variant, "Picture 4", "", 1)
	}
	if strings.Contains(variant, "Artist") {
		variant = strings.Replace(variant, "Artist ", "", 1)
	}
	variant = strings.Replace(variant, "(", "", -1)
	variant = strings.Replace(variant, ")", "", -1)
	variant = strings.Replace(variant, ",", "", -1)
	variant = strings.Replace(variant, ".", "", -1)
	variant = strings.Replace(variant, "- ", "", -1)
	variant = strings.Replace(variant, "  ", " ", -1)
	variant = strings.Replace(variant, "\r\n", " ", -1)
	variant = strings.Replace(variant, "\n", " ", -1)
	variant = strings.Replace(variant, "  ", " ", -1)
	return strings.TrimSpace(variant)
}

var preserveTags = []string{
	"etched",
	"retro frame",
	"step-and-compleat",
	"serialized",
}

func card2promo(cardName, variant string) (string, string) {
	var edition string
	ogVariant := variant

	switch {
	case strings.Contains(variant, "30th Anniversary") && !strings.Contains(variant, "History Promos"):
		if strings.Contains(variant, "Japanese 30th Anniversary Celebration Tokyo Promo") {
			return "P30T", ""
		}
		return "P30A", ""
	case strings.Contains(variant, "Cowboy Bebop"):
		return "PCBB", ""
	case strings.Contains(variant, "PWCS"):
		return "PWCS", ""
	case strings.Contains(variant, "Magic Spotlight"):
		return "PSPL", ""
	case strings.Contains(variant, "Friday Night Magic Promo"):
		variant = "FNM"
	case strings.Contains(variant, "Japan Standard Cup 2025 Promo"):
		return "PJSC", ""
	case strings.Contains(variant, "MKM Standard Showdown"):
		return "MKM Standard Showdown", ""
	}

	switch cardName {
	case "Demonic Tutor":
		if variant == "Daarken Judge Rewards Promo" {
			variant = "Judge 2008"
		} else if variant == "Anna Steinbauer Judge Promo" {
			variant = "Judge 2020"
		}
	case "Vampiric Tutor":
		if variant == "Judge Rewards Promo Old Border" {
			variant = "Judge 2000"
		} else if variant == "Judge Rewards Promo New Border" {
			variant = "Judge 2018"
		}
	case "Wasteland":
		if variant == "Judge Rewards Promo Carl Critchlow art" {
			variant = "Judge 2010"
		} else if variant == "Judge Rewards Promo Steve Belledin art" {
			variant = "Judge 2015"
		}
	case "Vindicate":
		if variant == "Judge Rewards Promo Mark Zug art" {
			variant = "Judge 2007"
		} else if variant == "Judge Rewards Promo Karla Ortiz art" {
			variant = "Judge 2013"
		}
	case "Fling":
		// Only one of the two is present
		if variant == "Gateway Promo Wizards Play Network Daren Bader art" {
			variant = "DCI"
		}
	case "Sylvan Ranger":
		if variant == "Judge Rewards Promo Mark Zug art" {
			variant = "WPN"
		}
	case "Goblin Warchief":
		if ogVariant == "Friday Night Magic Promo Old Border" {
			return "F06", "5"
		} else if ogVariant == "Friday Night Magic Promo New Border" {
			return "F16", "5"
		}
	case "Cabal Therapy":
		if strings.HasPrefix(variant, "Gold-bordered") {
			variant = "2003"
		}
	case "Rishadan Port":
		if strings.HasPrefix(variant, "Gold-bordered") {
			variant = "2000"
		}
	case "Hangarback Walker":
		edition = "Love your LGS"
		variant = "2"
	case "Chord of Calling":
		edition = "Double Masters"
		variant = "Release"
	case "Wrath of God":
		if variant == "Player Rewards Promo textless" {
			edition = "P07"
			variant = "1"
		} else {
			edition = "Double Masters"
			variant = "Release"
		}
	case "Conjurer's Closet":
		edition = "PW21"
		variant = "6"
	case "Cryptic Command":
		if variant == "Qualifier Promo" {
			edition = "PPRO"
			variant = "2020-1"
		}
	case "Dauntless Dourbark":
		edition = "DCI"
		variant = "12"
	case "Eye of Ugin":
		edition = "J20"
		variant = "10"
	case "Serra Avatar":
		if variant == "Junior Super Series Promo Dermot Power art" {
			edition = "PSUS"
			variant = "2"
		}
	case "Steward of Valeron":
		edition = "PURL"
		variant = "1"
	case "Llanowar Elves":
		switch ogVariant {
		case "Friday Night Magic Promo":
			edition = "FNM"
			variant = "11"
		case "Open House Promo":
			edition = "PDOM"
			variant = "168"
		case "Retro Frame Tin Promo":
			edition = "PDMU"
			variant = "Retro Frame"
		}
	case "Masked Vandal":
		edition = "KHM"
		variant = "405"
	case "Reliquary Tower":
		if variant == "Textless Commander Promo" {
			edition = "PF23"
			variant = "3"
		}
	case "Gideon, Ally of Zendikar":
		if variant == "2016 San Diego Comic Con Zombie Promo BFZ" {
			edition = "PS16"
			variant = "29"
		} else if variant == "Regional Championship Qualifiers 2022" {
			edition = "PRCQ"
			variant = "1"
		}
	case "Snapcaster Mage":
		if variant == "2016 Regional PTQ Promo" {
			edition = "PPRO"
		} else if variant == "Regional Championship Qualifier 2023" {
			edition = "PRCQ"
		}
	case "Avacyn's Pilgrim":
		if variant == "2025 Festival Promo" {
			edition = "PF25"
		}
	case "Counterspell":
		if variant == "Festival # 0001" {
			edition = "PF24"
		}

	case "Aven Mindcensor", "Dig Through Time", "Goblin Guide", "Scavenging Ooze":
		if variant == "Love Your Local Game Store Promo" {
			return "PLG21", ""
		}
	case "Bolas's Citadel":
		if variant == "Draft Weekend Promo" {
			edition = "PWAR"
			variant = "79"
		} else if variant == "Love Your Local Game Store Promo" {
			edition = "PLG21"
			variant = "3"
		}
	case "Nicol Bolas",
		"Earthquake",
		"Serra Angel":
		if variant == "Japanese Magic x Duel Masters Promo" {
			return "PMDA", ""
		}
	case "Sol Ring":
		if variant == "Commander Promo" {
			edition = "PF19"
		}
	case "Sakura-Tribe Elder":
		if variant == "Textless Victor Adame Minguez art" {
			edition = "PLG24"
		}
	case "Ephemerate":
		if variant == "Japanese Summer Vacation 2022 Promo" {
			edition = "PSVC"
		}
	case "Dragon's Hoard":
		if variant == "Tarkir: Dragonstorm Magic Academy Promo" {
			return "PW25", "1p"
		}
	case "Lightning Bolt":
		switch variant {
		case "MagicFest Promo textless":
			return "PF19", "1"
		case "Future Sight Frame 2025 MagicCon Atlanta Promo":
			return "PF25", "13"
		}
	case "Ugin, the Spirit Dragon":
		if variant == "Retro Frame 2025 MagicCon Las Vegas Promo" {
			return "PF25", "6"
		}
	case "Cloud, Midgar Mercenary":
		switch variant {
		case "0001 FFVII Commander Deck Game Edition Promo":
			return "PMEI", "2025-21"
		case "English Language Pro Tour Final Fantasy Promo":
			return "PPRO", "2025-1"
		case "Japanese Language Magic Spotlight: Final Fantasy Promo":
			return "PSPL", "4"
		}
	case "Ultimate Green Goblin":
		return "PW25", "12"
	case "J. Jonah Jameson":
		return "PF25", "17"
	case "Katara, the Fearless":
		return "PURL", "2025-3"
	case "Portable Hole":
		return "AFR", "398"
	}
	return edition, variant
}

// PreprocessBuylist is Preprocess for the buylist feed, which describes a card
// differently from the sale catalog.
func PreprocessBuylist(b *mtgmatcher.Backend, card CSIPriceEntry) (*mtgmatcher.InputCard, error) {
	// A two-sided token sheet prints one physical card for a pairing the
	// ordinary per-card pipeline below was never built for: a name like
	// "Soldier (Token) // Beast (Token)" carries no external id to route
	// through the way ManaPool's scryfall_id does, so it needs its own
	// anchor before that pipeline mangles it, and refuses outright rather
	// than fall into whatever single-face guess the rest of this function
	// would otherwise make of one half of the name.
	if strings.Contains(card.Name, "(Token)") && strings.Contains(card.Name, " // ") {
		return preprocessTokenPairBuylist(b, card)
	}

	num := strings.TrimLeft(card.Number, "0")
	cleanVar := cleanVariant(card.Notes)
	edition := card.ItemSet
	isFoil := card.IsFoil == 1
	cardName := card.Name

	if mtgmatcher.Contains(cardName, "signed by") {
		return nil, mtgmatcher.ErrUnsupported
	}

	variant := num
	if variant == "" {
		variant = cleanVar
	}

	fixName := mtgmatcher.SplitVariants(cardName)
	var altVariant string
	if len(fixName) > 1 {
		altVariant = strings.Join(fixName[1:], " ")
		if variant != "" {
			variant += " "
		}
		variant += altVariant
	}
	cardName = fixName[0]

	fixup, found := nameTable[cardName]
	if found {
		cardName = fixup
	}

	vars, found := variantTable[variant]
	if found {
		variant = vars
		cleanVar = vars
	}
	vars, found = variantTable[cleanVar]
	if found {
		variant = vars
		cleanVar = vars
	}

	cardName, edition, variant, err := magicShelfFixups(cardName, edition, variant)
	if err != nil {
		return nil, err
	}

	// Skip tokens with the same names as cards
	if strings.Contains(variant, "Emblem") && !b.IsToken(cardName) {
		return nil, mtgmatcher.ErrUnsupported
	}

	switch edition {
	case "Coldsnap Theme Deck":
		if mtgmatcher.IsBasicLand(cardName) {
			return nil, mtgmatcher.ErrUnsupported
		}
	case "Zendikar", "Battle for Zendikar", "Oath of the Gatewatch":
		// Strip the extra letter from the name
		if mtgmatcher.IsBasicLand(cardName) {
			cardName = strings.Fields(cardName)[0]
		}
	case "Unstable":
		variant = cleanVar
	case "Mystery Booster Reprints":
		// Keep the cards that strictly need extra information
		switch cardName {
		case "Aerial Responder",
			"Command Tower",
			"Laboratory Maniac",
			"Everythingamajig",
			"Ineffable Blessing":
			variant = cleanVar
		case "Forest":
			if variant == "291" {
				variant = "292"
			}
		}
	case "Secret Lair":
		variant = cleanVar

		if num != "" {
			variant = num + " " + cleanVar
			// CSI's notes often just repeat the collector number followed by the
			// product/deck name ("2406 - Goblin Storm Commander Deck"), which is
			// noise that doubles the number and trips the variant/token filters.
			// When the notes lead with the collector number, trust it alone.
			if strings.HasPrefix(cleanVar, num) {
				variant = num
			}
		}
	case "Deckmasters":
		variant = strings.TrimSpace(strings.Split(variant, "Deckmaster")[0])
	case "Duel Decks: Anthology":
		if num != "" {
			variant = num + " " + cleanVar
		}
	case "Conspiracy: Take the Crown":
		if cardName == "Kaya, Ghost Assassin" && strings.Contains(variant, "Alternate") {
			variant = "222"
		}
	case "D&D Ampersand":
		edition = "PAFR"
		variant = "Ampersand"
	case "Promo":
		variant = cleanVar
		switch variant {
		case "Ravnica Weekend Promo":
			edition = variant
			variant = num
		case "Stained Glass Art":
			edition = "SLD"
			variant = num
		case "Junior Super Series Promo",
			"Junior Super Series Promo Carl Critchlow art":
			edition = "PSUS"
			variant = ""
		default:
			possibleEd, possibleVar := card2promo(cardName, variant)
			if variant != possibleVar {
				variant = possibleVar
			}
			if possibleEd != "" {
				edition = possibleEd
			}
		}
	case "Unfinity":
		if strings.Contains(variant, ",") {
			variant = strings.Replace(altVariant, ",", "/", -1)
		}
	}

	// Add previously removed/ignored tags
	for _, tag := range preserveTags {
		if strings.Contains(strings.ToLower(cleanVar), tag) && !strings.Contains(strings.ToLower(variant), tag) {
			if variant != "" {
				variant += " "
			}
			variant += tag
		}
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Variation: variant,
		Edition:   edition,
		Foil:      isFoil,
	}, nil
}

// preprocessTokenPairBuylist resolves a two-sided token buylist row to the
// combined entity mtgmatcher/magic derives for it, anchored by the row's
// own set code and collector number - CSI publishes both (Code and Number)
// alongside the storefront's edition wording, unlike ManaPool's own single
// scryfall_id shape. A row this precise with no combined entity on file
// (no mtgjson tokenProducts record at all - not every real pairing has
// one) is refused rather than left for the rest of PreprocessBuylist to
// mistake for a single face.
func preprocessTokenPairBuylist(b *mtgmatcher.Backend, card CSIPriceEntry) (*mtgmatcher.InputCard, error) {
	isFoil := card.IsFoil == 1

	tokenSet := magic.EditionTokenSetCode(b, card.ItemSet)
	if tokenSet == "" && card.Code != "" {
		tokenSet = magic.SetTokenSetCode(b, card.Code)
	}
	if tokenSet == "" {
		tokenSet = card.Code
	}
	if tokenSet != "" {
		for _, number := range csiTokenPairNumbers(card.Number) {
			if uuid := magic.MatchNativeTokenPair(b, tokenSet, number, card.Name); uuid != "" {
				if id, err := b.MatchID(uuid, isFoil); err == nil {
					return &mtgmatcher.InputCard{ID: id}, nil
				}
			}
			if tcgID := magic.MatchTokenPairingBySetNumber(b, tokenSet, number, card.Name, isFoil); tcgID != "" {
				if id, err := b.MatchID(tcgID, isFoil); err == nil {
					return &mtgmatcher.InputCard{ID: id}, nil
				}
			}
		}
	}

	if pairID := magic.MatchTokenPairingByNamesAndEdition(b, card.Name, card.ItemSet, isFoil); pairID != "" {
		if id, err := b.MatchID(pairID, isFoil); err == nil {
			return &mtgmatcher.InputCard{ID: id}, nil
		}
	}

	return nil, mtgmatcher.ErrUnsupported
}

// csiTokenPairNumbers returns the collector numbers worth anchoring a two-
// sided token's first face against, in the order CSI's own Number field
// gives them: a compound "020/021" one per face, trusting which half names
// which face at face value since which one is first is not fixed across
// editions; a bare "003" (a boxed set's own singles-shaped numbering) tried
// as-is.
func csiTokenPairNumbers(number string) []string {
	before, after, found := strings.Cut(number, "/")
	if !found {
		if n := strings.TrimLeft(strings.TrimSpace(number), "0"); n != "" {
			return []string{n}
		}
		return nil
	}
	var out []string
	for _, n := range []string{before, after} {
		if n := strings.TrimLeft(strings.TrimSpace(n), "0"); n != "" {
			out = append(out, n)
		}
	}
	return out
}

// magicShelfFixups reads the shapes this storefront gives a few classes of
// listing before the matcher sees them: an emblem named "Emblem" with the
// planeswalker in the notes, where the catalog names it after the
// planeswalker; the duel decks headed in the singular; the blank cards a
// few sets ship, which are not cards; a note saying either of two numbers
// may be received, of which the first is the one asked for; and the T the
// storefront hangs on a token's number.
func magicShelfFixups(cardName, edition, variant string) (string, string, string, error) {
	if cardName == "Emblem" && variant != "" {
		if strings.Contains(variant, "Token") || strings.Contains(variant, "//") {
			return "", "", "", mtgmatcher.ErrUnsupported
		}
		// The buylist writes the token number ahead of the planeswalker
		// ("7 The Capitoline Triad", "8/8 Tamiyo, the Moon Sage").
		cardName, variant = emblemNumber.ReplaceAllString(variant, "")+" Emblem", ""
	}
	if strings.HasPrefix(cardName, "Blank Card") {
		return "", "", "", mtgmatcher.ErrUnsupported
	}
	if strings.HasPrefix(edition, "Duel Deck:") {
		edition = "Duel Decks:" + strings.TrimPrefix(edition, "Duel Deck:")
	}
	if m := eitherNumber.FindStringSubmatch(variant); m != nil {
		variant = strings.TrimLeft(m[1], "0")
	}
	if m := tokenNumber.FindStringSubmatch(variant); m != nil {
		variant = m[1] + " Token"
	}
	return cardName, edition, variant, nil
}

var (
	eitherNumber = regexp.MustCompile(`(?i)either card number (\d+) or \d+`)
	tokenNumber  = regexp.MustCompile(`^(\d+)T Token$`)
	emblemNumber = regexp.MustCompile(`^\d+(?:/\d+)?[A-Za-z]? `)
)
