package abugames

import (
	"errors"
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var cardTable = map[string]string{
	// Typos
	"Bogart Brute":                        "Boggart Brute",
	"Deathgazer Cockatrice":               "Deathgaze Cockatrice",
	"Elminate":                            "Eliminate",
	"Fireblade Artist Ravnica Allegiance": "Fireblade Artist",
	"Mindblade Rendor":                    "Mindblade Render",
	"Neglected Hierloom":                  "Neglected Heirloom",
	"Rathi Berserker":                     "Aerathi Berserker",
	"Smelt and Herd and Saw":              "Smelt // Herd // Saw",
	"Soulmemder":                          "Soulmender",
	"Svagthos, the Restless Tomb":         "Svogthos, the Restless Tomb",
	"Trial and Error":                     "Trial // Error",
	"Simic Signat":                        "Simic Signet",
	"Specmen 73":                          "Specimen 73",
	"Zilortha, Strength Incarnated":       "Zilortha, Strength Incarnate",
	"Makindi Siderunner":                  "Makindi Sliderunner",
	"Anti-Venom, Horrifying Hero":         "Anti-Venom, Horrifying Healer",

	"The Emperor of Palamecia /The Lord Master of Hell": "The Emperor of Palamecia",

	// Funny cards
	"No Name":                         "_____",
	"Absolute Longest Card Name Ever": mtgmatcher.LongestCardEver,
}

var promoTags = []string{
	"Alternate Art Duelist",
	"Arena",
	"Book",
	"Buy-a-Box",
	"Convention",
	"Draft Weekend",
	"FNM",
	"Into Pack",
	"Judge",
	"Launch",
	"OPEN HOUSE FULL ART",
	"Open House",
	"Planeswalker Weekend",
	"Prerelease",
	"Preview",
	"Promo",
	"Release",
	"SDCC",
	"Store Championship",
	"TopDeck Magazine",
}

// errForeignListing marks a listing in a language the catalog never printed
// the card in, which has no printing of its own to be priced against.
var errForeignListing = errors.New("foreign listing")

// boosterFun is the mark the catalog gives every frame a set prints beside
// its plain one - borderless, showcase, extended art, retro. A drop marker
// like sldbonus is not one of them: it says where a card came from, not what
// frame it wears.
const boosterFun = "boosterfun"

// finishNamed reports a printing sold in a finish this storefront would spell
// FOIL, which is the flag its own wording carries.
func finishNamed(co *mtgmatcher.CardObject) bool {
	return co.Foil || co.Etched
}

// resolved answers the printing a description names, and nil when it names
// none. It matches a copy, so the matcher's own edits stay in the probe.
func resolved(name, edition, variation, language string, foil bool) *mtgmatcher.CardObject {
	probe := mtgmatcher.InputCard{Name: name, Edition: edition, Variation: variation, Language: language, Foil: foil}
	id, err := mtgmatcher.Match(&probe)
	if err != nil {
		return nil
	}
	co, err := mtgmatcher.GetUUID(id)
	if err != nil {
		return nil
	}
	return co
}

// variantLetterRe matches the letter this storefront opens a variation with
// when a set prints one card several times over - "(a Dark)", "(b Spring)",
// "(a - Green Bottle)" - and the words it hangs off it.
var variantLetterRe = regexp.MustCompile(`^([a-f])(?: -)? [A-Z(]`)

// variantLetter reads that letter. The storefront spells a basic land's
// artwork in capitals and everything else in lower case, so the two never
// answer for each other.
func variantLetter(variation string) string {
	match := variantLetterRe.FindStringSubmatch(variation)
	if match == nil {
		return ""
	}
	return match[1]
}

// artworkLetterRe matches the letter a basic land's title opens its variation
// with, and the words the storefront hangs off it.
var artworkLetterRe = regexp.MustCompile(`^([A-F])\b`)

// artworkLetter answers the letter a basic land listing names its artwork by,
// and "" for everything else. Only a basic land is read this way: the letter
// is what the catalog files those printings under, and on any other card a
// bare letter is as likely to be the start of a word the wording spends on
// something else.
func artworkLetter(cardName, variation string) string {
	if !mtgmatcher.IsBasicLand(cardName) {
		return ""
	}
	match := artworkLetterRe.FindStringSubmatch(variation)
	if match == nil {
		return ""
	}
	return match[1]
}

func preprocess(card *ABUCard) (*mtgmatcher.InputCard, error) {
	lang := ""
	if len(card.Language) > 0 {
		lang = card.Language[0]
	}

	// Non-Singles magic cards
	switch card.Layout {
	case "Scheme", "Plane", "Phenomenon":
		return nil, errors.New("non-single card")
	}

	// Non-existing cards
	switch card.DisplayTitle {
	case "Silent Submersible (Promo Pack)",
		"Silent Submersible (Promo Pack) - FOIL",
		"Hymn to Tourach (B - Mark Justice - 1996)",
		"Skyclave Shade (Extended Art)",
		"Mountain (6th Edition 343 - Mark Le Pine - 1999)":
		return nil, errors.New("untracked card")
	}
	switch card.ID {
	case "1604919", "1604921", "1604922", // Living Twister
		"1604802", "1604801", "1604799": // Commence the Endgame
		return nil, errors.New("duplicated card")
	}

	isFoil := strings.Contains(strings.ToLower(card.DisplayTitle), " foil") ||
		strings.Contains(strings.ToLower(card.DisplayTitle), " - fol") // SS3 Pyroblast

	edition := card.Edition
	title := card.DisplayTitle
	if title == "" {
		title = card.SimpleTitle
	}

	// Split by -, rebuild the cardname in a standardized way
	variation := ""
	vars := strings.Split(title, " - ")
	cardName := vars[0]
	if len(vars) > 1 {
		if vars[len(vars)-1] == edition {
			vars = vars[:len(vars)-1]
		}

		variation = strings.Join(vars[1:], " ")

		// Fix some untagged prerelease cards
		// Nahiri's Wrath, Tendershoot Dryad
		if strings.Contains(variation, edition+" FOIL") {
			variation = strings.Replace(variation, edition+" FOIL", "Prerelease", 1)
		}
	}

	// Split by ()
	vars = mtgmatcher.SplitVariants(cardName)
	cardName = vars[0]
	if len(vars) > 1 {
		oldVariation := variation
		variation = strings.Join(vars[1:], " ")
		if oldVariation != "" {
			variation += " " + oldVariation
		}
	}

	// Separate flavor names
	if strings.Contains(cardName, " | ") {
		cardName = strings.Split(cardName, " | ")[0]
	}

	// Cleanup variation as necessary
	if variation != "" {
		variation = strings.Replace(variation, "FOIL", "", -1)

		variation = strings.Replace(variation, "(", "", -1)
		variation = strings.Replace(variation, ")", "", -1)

		variation = strings.Replace(variation, "Not Tournament Legal", "", 1)

		variation = strings.TrimSpace(variation)

		isPromo := false
		for _, tag := range promoTags {
			if strings.Contains(variation, tag) {
				isPromo = true
				break
			}
		}
		if isPromo {
			// Handle promo cards appearing in multiple editions
			// like Sorcerous Spyglass
			if strings.Contains(variation, "Promo Pack") || strings.Contains(variation, "Prerelease") {
				variation += " " + edition
			}
			// Reset edition, and trust mtgmatcher to find it by its variation
			edition = "Promo"
		}
	}

	switch edition {
	case "":
		if mtgmatcher.IsBasicLand(cardName) {
			card.Edition = "GK2"
		} else {
			return nil, errors.New("missing edition")
		}
	case "Promo":
		switch cardName {
		case "Skirk Marauder":
			edition = "Arena League 2003"
		case "Damnation(Secret Lair":
			cardName = "Damnation"
			edition = "SLD"
		case "Elvish Aberration":
			if variation == "FNM" {
				variation = "Arena"
			}
		case "Elvish Lyrist":
			if variation == "FNM" {
				variation = "JSS Foil"
			}
		case "Island":
			if variation == "Arena 1999 No Symbol Promo" {
				variation = "Arena 1999 misprint"
			}
		case "Stocking Tiger":
			if variation == "Target" {
				variation = "misprint"
			}
		case "Psychatog":
			if variation == "FNM" {
				variation = "Textless"
			}
		case "Sol Ring":
			if variation == "Commander" {
				variation = "MagicFest 2019"
			}
		case "Mountain":
			if variation == "APAC a Phillippines" {
				variation = "APAC a Philippines"
			}
		case "Beast of Burden":
			if variation == "Prerelease No Expansion Symbol FOIL" {
				variation = "Prerelease misprint"
			}
		case "Mechagodzilla, Battle Fortress / Hangarback Walker":
			if variation == "Welcome Back" {
				cardName = "Hangarback Walker"
				edition = "PLG20"
			}
		case "Serra Angel":
			if strings.Contains(variation, "25th Anniversary") {
				edition = "PDOM"
			}
		case "Hidetsugu, Devouring Chaos":
			edition = "NEO"
		case "Rafiq of the Many":
			edition = "SHA"
			variation = "250"
		case "Swiftfoot Boots":
			if strings.Contains(variation, "Launch") {
				edition = "PW22"
				variation = "4"
			}
		case "Brood Sliver":
			edition = "SLD"
		case "Lavinia, Azorius Renegade":
			edition = "PRNA"
			variation = "189"
		}
		if strings.Contains(variation, "Secret") || strings.Contains(variation, "Lair") {
			num := mtgmatcher.ExtractNumber(variation)
			if num != "" {
				variation = num
			} else if strings.Contains(variation, "Seb McKinnon") {
				variation = "119"
			}
			edition = "Secret Lair Drop"
		} else if mtgmatcher.IsBasicLand(cardName) && strings.Contains(variation, "Full-Text") {
			edition = "SLD"
			variation = strings.TrimPrefix(variation, "Full-Text ")
		} else if strings.Contains(variation, "Play Promo") {
			variation = strings.Replace(variation, "FNM", "", 1)
		} else if card.Layout == "Planar" {
			edition = "Planechase Promos"
		} else if variation == "Preview" {
			if len(mtgmatcher.MatchInSet(cardName, "MGB")) > 0 {
				edition = "MGB"
			}
		}
	case "Secret Lair", "Secret Lair Drop":
		edition = "Secret Lair Drop"
		if len(mtgmatcher.MatchInSetNumber(cardName, "SLC", card.Number)) > 0 {
			edition = "SLC"
			variation = card.Number
		} else {
			// Check if variation contains a number as it's usually more
			// accurate. If not, check the card.Number property (yolo).
			// ExtractNumberAny (not the year-capped ExtractNumber) so a Secret
			// Lair collector number >= 1993 (e.g. 7010) is recognized and not
			// clobbered by a stale card.Number, while still ignoring dates and
			// ordinals.
			num := mtgmatcher.ExtractNumberAny(variation)
			if num == "" && card.Number != "" {
				variation += " " + card.Number
			}
		}
	case "Anthologies":
		if cardName == "Mountain" {
			if variation == "A" {
				variation = "B"
			} else if variation == "B" {
				variation = "A"
			}
		}
	case "World Championship":
		if cardName == "City of Brass" {
			if variation == "Leon Lindback 1996" {
				variation = "Sideboard Leon Lindback 1996"
			}
		}
	case "Ikoria: Lair of Behemoths":
		if strings.Contains(cardName, " / ") {
			cardName = card.SimpleTitle
			variation = "Godzilla"
		}
	case "Oath of the Gatewatch":
		if cardName == "Captain's Claws" && variation == "Goldnight Castigator Shadow FOIL" {
			variation = "misprint"
		}
	case "Core Set 2020 / M20":
		if cardName == "Corpse Knight" && variation == "2/3" {
			variation = "misprint"
		}
	case "Mystery Booster":
		if cardName == "Trial and Error" {
			// Hack to prevent aliasing with the real "Trial // Error"
			cardName = "Trial and Error "
		}
	case "Summer Magic / Edgar":
		if mtgmatcher.IsBasicLand(cardName) {
			return nil, errors.New("unsupported")
		}
	case "Streets of New Capenna Commander":
		if strings.Contains(cardName, "Spellbinding Soprano") && strings.Contains(cardName, "The List") {
			cardName = "Spellbinding Soprano"
			variation = "Promo Pack"
		}
	case "Fourth Edition Foreign Black Border":
		variation = strings.TrimSuffix(variation, " Japanese")
		variation = strings.TrimSuffix(variation, " BB")
		variations := strings.Fields(variation)
		if len(variations) > 1 && len(variations[0]) == 1 {
			variation = strings.Join(variations[1:], " ")
		}
	case "Fallen Empires":
		variations := strings.Fields(variation)
		if len(variations) > 1 && len(variations[0]) == 1 {
			variation = strings.Join(variations[1:], " ")
		}
	case "Modern Horizons 2 Timeshifts":
		edition = "MH2"
	// This set has some non standard way to tag foil, but luckily variations are all on different CNs
	case "Edge of Eternities: Stellar Sights":
		variation = card.Number
	case "Aetherdrift":
		if variation == "Borderless" {
			variation += " " + card.Number
		}
	case "Final Fantasy Commander":
		variation += " " + card.Number
	}

	// Either Promo or "European Land Program"
	if strings.Contains(variation, "Scandanavia") {
		variation = strings.Replace(variation, "Scandanavia", "Scandinavia", 1)
	} else if strings.Contains(variation, "Phillippines") {
		variation = strings.Replace(variation, "Phillippines", "Philippines", 1)
	} else if strings.Contains(variation, "Extented") {
		variation = strings.Replace(variation, "Extented", "Extended", 1)
	}

	// Use collector number data when the variation carries has none, unless for a couple of editions
	var numbered bool
	if card.Number != "" && mtgmatcher.ExtractNumberAny(variation) == "" {
		switch edition {
		case "Unfinity", "Promo":
		default:
			if variation != "" {
				variation += " "
			}
			variation += card.Number
			numbered = true
		}
	}
	// A basic land's artwork is the only thing telling one of its printings
	// from another, and this storefront names it with a letter in the title
	// - "Swamp (C)", "Island (A) - Waterfall". Its own collector number is
	// the same for every one of them, so the number just read buries the
	// letter under one that names the whole group, and the words beside the
	// letter are the storefront's own name for the art, which the catalog
	// spells its own way. The letter alone reaches every one of them.
	letter := artworkLetter(cardName, variation)
	if letter != "" {
		variation = letter
	}

	// Older sets name their several printings of one card by what is drawn
	// on them - "Urza's Mine (d Tower)" - and there the words are the whole
	// identity and the letter counts for nothing. Newer ones number the
	// printings and leave the words as flavour, so take the letter alone
	// only where it reaches a printing on its own.
	if letter == "" {
		variant := variantLetter(variation)
		if variant != "" && resolved(cardName, edition, variant, lang, isFoil) != nil {
			variation = variant
		}
	}

	// This storefront gives every frame of a card the base printing's own
	// collector number - the extended art of Platoon Dispenser is filed
	// under BRO 36 where the catalog has it at 310 - so a listing naming a
	// frame answers with the plain card the number just read states. Ask
	// again without that number, and keep the answer only where it reaches
	// a frame the catalog marks.
	//
	// Only a number this scraper appended is dropped: a listing spelling
	// its own is one where the wording and the number already agree.
	if numbered {
		bare := strings.TrimSpace(strings.TrimSuffix(variation, card.Number))
		plain := resolved(cardName, edition, variation, lang, isFoil)
		if bare != "" && plain != nil && len(plain.PromoTypes) == 0 {
			marked := resolved(cardName, edition, bare, lang, isFoil)
			if marked != nil && marked.HasPromoType(boosterFun) &&
				(finishNamed(marked) == isFoil || finishNamed(marked) == finishNamed(plain)) {
				variation = bare
			}
		}
	}

	// The same number is wrong the other way round too: a listing whose
	// whole variation is the number can carry the number of a frame, and then the plain card
	// answers with its own showcase. Ask again without the number, and keep
	// the answer only where it reaches a printing the catalog leaves
	// unmarked - a set whose every printing is marked, like Special Guests,
	// has no plain card to reach and keeps the number it came with.
	if numbered && strings.TrimSpace(strings.TrimSuffix(variation, card.Number)) == "" {
		framed := resolved(cardName, edition, variation, lang, isFoil)
		if framed != nil && framed.HasPromoType(boosterFun) {
			plain := resolved(cardName, edition, "", lang, isFoil)
			if plain != nil && len(plain.PromoTypes) == 0 &&
				plain.SetCode == framed.SetCode && finishNamed(plain) == isFoil {
				variation = ""
			}
		}
	}

	// Restore canonical name for split cards
	cardName = strings.ReplaceAll(cardName, " / ", " // ")
	if strings.Contains(cardName, " // ") {
		cardName = strings.Split(cardName, " // ")[0]
		cardName = strings.TrimSpace(cardName)
	}

	name, found := cardTable[cardName]
	if found {
		cardName = name
	}

	// Stash the language information (filtered earlier)
	if lang != "" && lang != "English" {
		if variation != "" {
			variation += " "
		}
		variation += lang
	}

	// A storefront selling a card in a language the catalog never printed it
	// in has nowhere to file it. The match falls back on the English
	// printing, and the shop's own price for the Italian copy lands beside
	// the English one on a single uuid - three prices on 9th Edition Elvish
	// Piper, English, Italian and Japanese, each of them real. A set that
	// does hold the printing is reached and kept, the Japanese Chronicles
	// and the foreign black borders among them.
	if lang != "" && lang != "English" {
		printing := resolved(cardName, edition, variation, lang, isFoil)
		if printing == nil || printing.Language != lang {
			return nil, errForeignListing
		}
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Variation: variation,
		Edition:   edition,
		Foil:      isFoil,
		Language:  lang,
	}, nil
}
