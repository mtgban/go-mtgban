package strikezone

import (
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// neonInkColors are the Neon Ink treatments the catalog files under a colour.
var neonInkColors = []string{"Red", "Green", "Blue", "Yellow", "Pink"}

// neonInkWording spells a Neon Ink colour the way the catalog files it. The
// buylist drops the word the treatment is named for and writes "Neon Red",
// which names no treatment at all and reads as the plain printing standing
// beside the four coloured ones.
func neonInkWording(variation string) string {
	if !strings.HasPrefix(variation, "Neon ") || strings.HasPrefix(variation, "Neon Ink") {
		return variation
	}
	for _, color := range neonInkColors {
		if strings.HasPrefix(variation, "Neon "+color) {
			return "Neon Ink " + strings.TrimPrefix(variation, "Neon ")
		}
	}
	return variation
}

func preprocess(cardName, edition, notes string) (*mtgmatcher.InputCard, error) {
	var variation string

	// Skip tokens, too many variations
	if strings.Contains(cardName, "Token") {
		return nil, mtgmatcher.ErrUnsupported
	}

	cn, found := cardTable[cardName]
	if found {
		cardName = cn
	}

	// Sometimes the buylist specifies tags at the end of the card name,
	// but without parenthesis, so make sure they are present.
	for _, tag := range tagsTable {
		if strings.HasSuffix(cardName, tag) {
			cardName = strings.Replace(cardName, tag, "("+tag+")", 1)
			break
		}
	}

	cardName = strings.Replace(cardName, "[", "(", 1)
	cardName = strings.Replace(cardName, "]", ")", 1)

	switch {
	// Tyop
	case strings.HasPrefix(cardName, "Snow-Cover "):
		cardName = strings.Replace(cardName, "Snow-Cover ", "Snow-Covered ", 1)
	// Found at beginning, move it to variation
	case strings.HasPrefix(cardName, "Borderless"):
		cardName = strings.TrimPrefix(cardName, "Borderless ")
		cardName = strings.TrimPrefix(cardName, "Alt Art ")
		variation = "Borderless"
	// Found at beginning, move it to variation
	case strings.HasPrefix(cardName, "Showcase"):
		cardName = strings.TrimPrefix(cardName, "Showcase ")
		variation = "Showcase"
	// Found at beginning, move it to variation
	case strings.HasPrefix(cardName, "Extended Art"):
		cardName = strings.TrimPrefix(cardName, "Extended Art ")
		variation = "Extended Art"
	// Found at end, move it to variation
	case strings.HasSuffix(cardName, "JPN ALT ART PRERELEASE"):
		cardName = strings.TrimSuffix(cardName, " JPN ALT ART PRERELEASE")
		variation = "Prerelease Japanese"
	// Found at end, move it to edition
	case strings.HasSuffix(cardName, "Ultimate Edition"):
		cardName = strings.TrimSuffix(cardName, " Ultimate Edition")
		edition = "Secret Lair: Ultimate Edition"
	// Found at end, move it to edition
	case strings.HasSuffix(cardName, "Godzilla") && mtgmatcher.IsBasicLand(cardName):
		cardName = strings.TrimSuffix(cardName, " Godzilla")
		edition = "SLD"
	// Found at beginning, just drop it
	case strings.HasPrefix(cardName, "Alt Art"):
		cardName = strings.TrimPrefix(cardName, "Alt Art ")
		edition = "SLD"
	// APAC and EURO lands, drop specifier
	case strings.Contains(cardName, "APAC") || strings.Contains(cardName, "EURO"):
		variants := mtgmatcher.SplitVariants(cardName)
		cardName = variants[0]
		if mtgmatcher.IsBasicLand(cardName) {
			edition = "European Land Program"
			if strings.Contains(cardName, "APAC") {
				edition = "Asia Pacific Land Program"
			}
			fields := strings.Fields(cardName)
			cardName = fields[0]
			variation = variants[1]
		}
	// Ravnica weekend lands, move to variation
	case strings.Contains(cardName, "Ravnica Weekend"):
		fields := strings.Fields(cardName)
		cardName = fields[0]
		variation = strings.Join(fields[1:], " ")
	case strings.HasPrefix(cardName, "B.F.M.") && strings.Contains(cardName, "#"):
		for field := range strings.FieldsSeq(cardName) {
			if strings.HasPrefix(field, "#") {
				cardName = "B.F.M."
				variation = field[1:]
				break
			}
		}
	}

	ed, found := card2setTable[cardName]
	if found {
		edition = ed
	}

	variants := mtgmatcher.SplitVariants(cardName)
	cardName = variants[0]
	if len(variants) > 1 {
		if variation != "" {
			variation += " "
		}
		variation += strings.Join(variants[1:], " ")
	}

	variation = neonInkWording(variation)

	// Repeat to catch numbers
	if mtgmatcher.IsBasicLand(cardName) {
		num := mtgmatcher.ExtractNumber(cardName)
		if num != "" {
			cardName = strings.Replace(cardName, num, "", 1)
			cardName = strings.TrimSpace(cardName)
			if variation != "" {
				variation += " "
			}
			variation += num
		}

		for _, tag := range tagsTable {
			if strings.HasSuffix(cardName, tag) {
				cardName = strings.TrimSuffix(cardName, " "+tag)
				if variation != "" {
					variation += " "
				}
				variation += tag
				break
			}
		}
	}

	switch variation {
	case "6E", "VI DCI", "DCI", "US":
		switch cardName {
		case "Crusade",
			"Lord of Atlantis",
			"Serra Avatar",
			"Thran Quarry",
			"Two-Headed Dragon":
			edition = "Junior Super Series"
		case "Forest",
			"Island",
			"Mountain",
			"Swamp",
			"Plains":
			edition = "Arena League 1999"
		case "Argothian Enchantress",
			"Balance",
			"Ball Lightning",
			"Gaea's Cradle",
			"Intuition",
			"Living Death",
			"Memory Lapse",
			"Oath of Druids",
			"Stroke of Genius",
			"Tradewind Rider":
			variation = "Judge"
		case "Vampiric Tutor":
			variation = "Judge 2000"
		case "Arc Lightning",
			"Chill",
			"Duress",
			"Enlightened Tutor",
			"Karn, Silver Golem",
			"Mana Leak",
			"Rewind",
			"Serum Visions",
			"Stupor":
			variation = "Arena"
		case "Powder Keg",
			"Voidmage Prodigy",
			"Wasteland":
			variation = "Magic Player Rewards"
		case "Zoetic Cavern":
			variation = "DCI Promos"
		default:
			variation = "FNM"
		}
	case "DotP 2012 - Xbox", "X Box Promo 2013", "X Box Promo",
		"Playstation", "PS3 Promo",
		"Duels of the Planeswalkers - PC",
		"Duel of the Planeswalkers":
		variation = "Duels"
	}

	switch {
	case strings.HasPrefix(variation, "The "):
		edition = "Magic Premiere Shop 2005"
	case strings.Contains(variation, "Holiday"):
		edition = "Happy Holidays"
	case mtgmatcher.HasPrefix(cardName, "Teferi Master of Time"):
		if edition == "Promotional Cards" {
			variation += "s"
		} else if edition == "Promo Pack" {
			variation += "p"
		}
	}

	switch edition {
	case "Promotional Cards":
		if variation == "" {
			ed, found = promo2setTable[cardName]
			if found {
				edition = ed
			}
		}
	case "Ikoria: Lair of Behemoths":
		if strings.Contains(cardName, " - ") {
			s := strings.Split(cardName, " - ")
			cardName = s[1]
			variation = "Godzilla"
			if strings.Contains(notes, "Japanese") {
				variation += " Japanese"
			}
		}
	case "Promos: Play":
		edition = "Promotional"
		variation = "playpromo"
	case "Promos: Standard Showdown":
		if len(mtgmatcher.MatchInSet(cardName, "PSS1")) > 0 {
			edition = "PSS1"
		}
	case "Promos: Champs":
		edition = "PCMP"
	case "Promos: Pro Tour":
		for _, code := range []string{"PPRO", "SLP", "LTR", "PRCQ", "PR23"} {
			if len(mtgmatcher.MatchInSet(cardName, code)) > 0 {
				edition = code
			}
		}
		// The Secret Lair sets are only considered when the listing spells the
		// drop out, and this category never does, so say it on its behalf
		if edition == "SLP" {
			variation = strings.TrimSpace(variation + " Play")
		}
	case "Promos: Media":
		for _, code := range []string{
			"PHPR", "PMEI", "PURL",
			"PDTP", "PDP10", "PDP12", "PDP13", "PDP14", "PDP15",
		} {
			if len(mtgmatcher.MatchInSet(cardName, code)) > 0 {
				edition = code
			}
		}
	case "Promos: Junior Series":
		for _, code := range []string{"PSUS"} {
			if len(mtgmatcher.MatchInSet(cardName, code)) > 0 {
				edition = code
			}
		}
	case "Promos: Arena":
		cardName = strings.TrimSuffix(cardName, " Arena Promo")
	case "Promos: Judge":
		cardName = strings.TrimSuffix(cardName, " Full Art")
	case "Promos: Planeswalker Event":
		edition = "PWCS"
		if variation == "Top 8" {
			variation = ""
		}
	case "Promos: Unique and Miscellaneous":
		switch cardName {
		case "Lotus Petal":
			edition = "P30M"
		case "Serra Angel":
			edition = "PWOS"
		}
	case "Promos: Launch Party and Release Event":
		if mtgmatcher.IsBasicLand(cardName) {
			edition = "Ravnica Weekend"
		}
	case "Promos: WPN and Gateway":
		switch cardName {
		case "Orb of Dragonkind":
			edition = "PLG21"
			variation = "J" + strings.TrimLeft(variation, "0")
		}
	case "Hours of Devestation":
		edition = "HOU"
	case "Secret Lair Commander: Heads I Win":
		edition = "Secret Lair Commander: Heads I Win, Tails You Lose"
	case "Battlebond":
		if strings.HasSuffix(cardName, "Alternate Art") {
			cardName = strings.TrimSuffix(cardName, " Alternate Art")
		}
	}

	// Second pass in case some tags interfered with the lookup
	cn, found = cardTable[cardName]
	if found {
		cardName = cn
	}

	if variation == "Extemded Art" {
		variation = "Extended Art"
	}

	// Set finish
	isFoil := strings.Contains(strings.ToLower(notes), "foil")
	if strings.Contains(strings.ToLower(notes), "etched") {
		if variation != "" {
			variation += " "
		}
		variation += "Etched"
	}

	var language string
	for lang := range mtgmatcher.LanguageTag2LanguageCode {
		// Skip empty (or it would match everything) and skip English as sometimes
		// non-English cards are mistakenly tagged as such
		if lang != "" && lang != "English" && strings.Contains(notes, lang) {
			language = lang
			break
		}
	}

	// Secret Lair files a card under one drop or under several, and the store
	// says which only where it writes something beside the name. Where the
	// set holds several, the match that follows picks one of them for no
	// reason and prices the others as it, so refuse instead of choosing.
	if edition == "Secret Lair" && variation == "" && hasSeveralDrops(cardName) {
		return nil, mtgmatcher.ErrUnsupported
	}

	return &mtgmatcher.InputCard{
		Name:      cardName,
		Variation: variation,
		Edition:   edition,
		Foil:      isFoil,
		Language:  language,
	}, nil
}

// hasSeveralDrops reports whether the set files a card under more than one
// drop. The suffixes a number can end on - the star of a foil twin, the phi
// of a step-and-compleat - mark twins the wording picks, not drops of their
// own, and OriginalNumber is the number with all of them already stripped.
func hasSeveralDrops(cardName string) bool {
	cards := mtgmatcher.MatchInSet(cardName, "SLD")
	if len(cards) < 2 {
		return false
	}
	first := cards[0].OriginalNumber
	for _, card := range cards[1:] {
		if card.OriginalNumber != first {
			return true
		}
	}
	return false
}

// retroFrameVersion is the frame the catalog files the retro printings under.
const retroFrameVersion = "1997"

// treatmentClaims pairs a word the store writes with what the catalog files
// that treatment under. The store spells these out only where it is selling
// the printing wearing one, so they are read as claims and never as denials.
//
// Only the words the catalog answers as a promo type belong here. The store
// also writes "Rainbow Foil" and "Confetti Foil", which name the finish a
// Secret Lair is sold in rather than a treatment the printing wears, and
// reading those as claims refuses 62 of its listings for saying so.
var treatmentClaims = []struct {
	tag   string
	wears func(co *mtgmatcher.CardObject) bool
}{
	{"Retro Frame", func(co *mtgmatcher.CardObject) bool {
		return co.FrameVersion == retroFrameVersion
	}},
	{"Textured", func(co *mtgmatcher.CardObject) bool {
		return co.HasPromoType(magic.PromoTypeTextured)
	}},
	{"Surge Foil", func(co *mtgmatcher.CardObject) bool {
		return co.HasPromoType(magic.PromoTypeSurgeFoil)
	}},
}

// namesAbsentTreatment reports whether the listing spells out a treatment the
// printing it was answered with does not wear. The store says these words
// about the card in hand, so an answer without one is a different card, and
// the price it carries belongs to that card rather than this listing.
func namesAbsentTreatment(variation string, co *mtgmatcher.CardObject) bool {
	for _, claim := range treatmentClaims {
		if mtgmatcher.Contains(variation, claim.tag) && !claim.wears(co) {
			return true
		}
	}
	return false
}
