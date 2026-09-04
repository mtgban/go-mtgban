package magic

import (
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The functions here are InputCard vocabulary that magic alone reads,
// migrating out of the core card type one method at a time - each was an
// exported method with this file's caller as its only user.

// arenaYear returns the year of an Arena league printing, deducing it from the
// artist or set named in the variation when the listing gives no year.
func arenaYear(c *mtgmatcher.InputCard, maybeYear string) string {
	if maybeYear == "" {
		switch {
		case strings.Contains(c.Variation, "Tony Roberts"):
			maybeYear = "1996"
		case strings.Contains(c.Variation, "Urza"),
			strings.Contains(c.Variation, "Saga"),
			strings.Contains(c.Variation, "Anthony S. Waters"),
			strings.Contains(c.Variation, "Donato Giancola"):
			maybeYear = "1999"
		case strings.Contains(c.Variation, "Mercadian"),
			strings.Contains(c.Variation, "Masques"):
			maybeYear = "2000"
		case strings.Contains(c.Variation, "Ice Age"),
			strings.Contains(c.Variation, "IA"),
			strings.Contains(c.Variation, "Pat Morrissey"),
			strings.Contains(c.Variation, "Anson Maddocks"),
			strings.Contains(c.Variation, "Tom Wanerstrand"),
			strings.Contains(c.Variation, "Christopher Rush"),
			strings.Contains(c.Variation, "Douglas Shuler"):
			maybeYear = "2001"
		case strings.Contains(c.Variation, "Mark Poole"):
			maybeYear = "2002"
		case strings.Contains(c.Variation, "Rob Alexander"):
			maybeYear = "2003"
		case strings.Contains(c.Variation, "Don Thompson"):
			maybeYear = "2005"
		case strings.Contains(c.Variation, "Beta"):
			switch c.Name {
			case "Forest":
				maybeYear = "2001"
			case "Island":
				maybeYear = "2002"
			}
		}
	} else if c.Name == "Forest" && strings.Contains(maybeYear, "2002") {
		maybeYear = "2001"
	} else if c.Name == "Island" && strings.Contains(maybeYear, "2001") && strings.Contains(c.Variation, "Poole") {
		maybeYear = "2002"
	}
	return maybeYear
}

// duelDecksVariant returns which half of a Duel Decks pairing the listing
// names, or an empty string if it is not a Duel Decks card.
func duelDecksVariant(c *mtgmatcher.InputCard) string {
	if !isDuelDecks(c) {
		return ""
	}

	// Variation might contain numbers, strip them away
	variant := c.Variation
	num := mtgmatcher.ExtractNumber(variant)
	variant = strings.TrimSpace(strings.Replace(variant, num, "", 1))
	if len(variant) < len("Duel Deck") {
		variant = c.Edition
	}

	if strings.Contains(variant, ": ") {
		fields := strings.Split(variant, ": ")
		variant = fields[len(fields)-1]
	}

	return variant
}

// worldChampPrefix returns the World Championship deck code for this listing,
// looking in the variation first and falling back to the edition.
func worldChampPrefix(c *mtgmatcher.InputCard) (string, bool) {
	prefix, sideboard := mtgmatcher.ParseWorldChampPrefix(c.Variation)
	if prefix == "" {
		return mtgmatcher.ParseWorldChampPrefix(c.Edition)
	}
	return prefix, sideboard
}

// playerRewardsYear returns the year of a player rewards printing, falling
// back to the set or artist named in the variation when the listing gives no
// year of its own.
func playerRewardsYear(c *mtgmatcher.InputCard, maybeYear string) string {
	if maybeYear == "" {
		switch c.Name {
		case "Bear":
			if mtgmatcher.Contains(c.Variation, "Odyssey") {
				maybeYear = "2001"
			} else if mtgmatcher.Contains(c.Variation, "Onslaught") {
				maybeYear = "2003"
			}
		case "Beast":
			if mtgmatcher.Contains(c.Variation, "Odyssey") {
				maybeYear = "2001"
			} else if mtgmatcher.Contains(c.Variation, "Darksteel") {
				maybeYear = "2004"
			}
		case "Elephant":
			if mtgmatcher.Contains(c.Variation, "Invasion") {
				maybeYear = "2001"
			} else if mtgmatcher.Contains(c.Variation, "Odyssey") {
				maybeYear = "2002"
			}
		case "Spirit":
			if mtgmatcher.Contains(c.Variation, "Planeshift") {
				maybeYear = "2001"
			} else if mtgmatcher.Contains(c.Variation, "Champions") {
				maybeYear = "2004"
			}
		case "Lightning Bolt":
			if c.Contains("Oversize") {
				maybeYear = "2009"
			} else {
				maybeYear = "2010"
			}
		}
	}
	return maybeYear
}

// shouldIgnoreNumber reports whether the collector number, where one was
// given, is too unreliable to narrow with: some storefronts publish a number
// that belongs to a different printing of the same set.
func shouldIgnoreNumber(c *mtgmatcher.InputCard, setName, num string) bool {
	// No misprints or WCD
	if c.Contains("Misprint") || c.IsWorldChamp() {
		return true
	}

	// This is better handled in thelistCheck()
	if isMysteryList(c) && !c.Contains("Unfinity") {
		return true
	}

	// Unfinity numbers could refer to Attractions
	if mtgmatcher.Contains(c.Edition, "unf") {
		if mtgmatcher.HasPrinting(c.Name, "field", "attractionLights", "UNF") && (strings.Contains(c.Variation, "/") || strings.Contains(c.Variation, "-")) {
			return true
		}
	}

	// If the number is the same as in the edition, there might be
	// variation pollution, therefore unreliable (unless they are years)
	if num != "" && strings.Contains(setName, num) && mtgmatcher.ExtractYear(setName) == "" {
		return true
	}

	return false

}

// isPremiereShop reports a Magic Premiere Shop basic land. It compares the raw
// strings because the folded form is too short to be safe.
func isPremiereShop(c *mtgmatcher.InputCard) bool {
	return isBasicLand(c.Name) &&
		// XXX: do not use c.Contains here
		(strings.Contains(c.Variation, "MPS") ||
			strings.Contains(c.Variation, "Premier") || // csi
			strings.Contains(c.Edition, "MPS") ||
			strings.Contains(c.Edition, "Premiere Shop")) // mkm
}

// isFNM reports a Friday Night Magic promo, abbreviated or spelled out.
func isFNM(c *mtgmatcher.InputCard) bool {
	return c.Contains("FNM") ||
		c.Contains("Friday Night Magic")
}

// isRewards reports a player rewards promo: the textless ones, minus the
// unrelated series that are also textless, or anything else calling itself a
// reward that is not a judge card.
func isRewards(c *mtgmatcher.InputCard) bool {
	return (mtgmatcher.Contains(c.Variation, "Textless") &&
		!mtgmatcher.Contains(c.Variation, "Year of") &&
		!mtgmatcher.Contains(c.Variation, "Lunar") &&
		!mtgmatcher.Contains(c.Variation, "Store")) ||
		(c.Contains("Reward") && !isJudge(c))
}

// isDuelsOfThePW reports a Duels of the Planeswalkers promo. It compares the
// raw strings so the fold does not equate Duels with Duel Decks.
func isDuelsOfThePW(c *mtgmatcher.InputCard) bool {
	// XXX: do not use c.Contains here
	return strings.Contains(c.Variation, "Duels") ||
		strings.Contains(c.Edition, "Duels") ||
		mtgmatcher.Contains(c.Variation, "DotP") // tat
}

// isBorderless reports a borderless printing, from the variation only.
func isBorderless(c *mtgmatcher.InputCard) bool {
	return mtgmatcher.Contains(c.Variation, "Borderless")
}

// isShowcase reports a showcase frame; binderpos storefronts say Sketch.
func isShowcase(c *mtgmatcher.InputCard) bool {
	return mtgmatcher.Contains(c.Variation, "Showcase") ||
		mtgmatcher.Contains(c.Variation, "Sketch") // binderpos
}

// isExtendedArt reports extended art, from the variation only.
func isExtendedArt(c *mtgmatcher.InputCard) bool {
	return mtgmatcher.Contains(c.Variation, "Extended")
}

// isGenericExtendedArt reports extended or full art, from the variation only.
func isGenericExtendedArt(c *mtgmatcher.InputCard) bool {
	return mtgmatcher.Contains(c.Variation, "Art") &&
		(mtgmatcher.Contains(c.Variation, "Extended") ||
			mtgmatcher.Contains(c.Variation, "Full"))
}

// isThickDisplay reports the thick display commander cards.
func isThickDisplay(c *mtgmatcher.InputCard) bool {
	return c.Contains("Display") || c.Contains("Thick")
}

// isSerialized reports a serialized printing, in either field.
func isSerialized(c *mtgmatcher.InputCard) bool {
	return strings.Contains(strings.ToLower(c.Variation), "serial") ||
		strings.Contains(strings.ToLower(c.Edition), "serial")
}

// isGalaxyFoil reports the galaxy foiling.
func isGalaxyFoil(c *mtgmatcher.InputCard) bool {
	return mtgmatcher.Contains(c.Variation, "Galaxy")
}

// isSurgeFoil reports the surge foiling, in either field.
func isSurgeFoil(c *mtgmatcher.InputCard) bool {
	return strings.Contains(strings.ToLower(c.Variation), "surge") ||
		strings.Contains(strings.ToLower(c.Edition), "surge")
}

// isOilSlick reports the oil slick foiling, in either field.
func isOilSlick(c *mtgmatcher.InputCard) bool {
	return strings.Contains(strings.ToLower(c.Variation), "slick") ||
		strings.Contains(strings.ToLower(c.Edition), "slick")
}

// isStepAndCompleat reports the step-and-compleat foiling.
func isStepAndCompleat(c *mtgmatcher.InputCard) bool {
	return mtgmatcher.Contains(c.Variation, "Compleat")
}

// isPhyrexian reports a Phyrexian language printing.
func isPhyrexian(c *mtgmatcher.InputCard) bool {
	return mtgmatcher.Contains(c.Variation, "Phyrexian")
}

// isChineseAltArt reports the Chinese alternate art printings.
func isChineseAltArt(c *mtgmatcher.InputCard) bool {
	return (c.Contains("Chinese") || strings.Contains(c.Variation, "CS")) && isGenericAltArt(c)
}

// isBasicFullArt reports a full art basic land, refusing the negations that
// storefronts write in the same field.
func isBasicFullArt(c *mtgmatcher.InputCard) bool {
	return isBasicLand(c.Name) &&
		(mtgmatcher.Contains(c.Variation, "full art") ||
			c.Variation == "FA") && // csi
		!mtgmatcher.Contains(c.Variation, "non") &&
		!mtgmatcher.Contains(c.Variation, "not") // csi
}

// isBasicNonFullArt reports a basic land explicitly marked as not full art.
func isBasicNonFullArt(c *mtgmatcher.InputCard) bool {
	return isBasicLand(c.Name) &&
		mtgmatcher.Contains(c.Variation, "non-full art") ||
		mtgmatcher.Contains(c.Variation, "Intro") || // abu
		mtgmatcher.Contains(c.Variation, "NOT the full art") // csi
}

// isPortalAlt reports the Portal alternates, which differ by carrying reminder
// text or by lacking flavor text.
func isPortalAlt(c *mtgmatcher.InputCard) bool {
	return (mtgmatcher.Contains(c.Variation, "Reminder Text") &&
		!mtgmatcher.Contains(c.Variation, "No")) ||
		mtgmatcher.Contains(c.Variation, "No Flavor Text") || // csi
		mtgmatcher.Contains(c.Variation, "Without Flavor Text") // csi
}

// isARNDarkMana reports the dark mana symbol variant of Arabian Nights.
func isARNDarkMana(c *mtgmatcher.InputCard) bool {
	return mtgmatcher.Contains(c.Variation, "dark")
}

// isARNLightMana reports the light mana symbol variant of Arabian Nights,
// which some storefronts mark with a dagger instead of a word.
func isARNLightMana(c *mtgmatcher.InputCard) bool {
	return mtgmatcher.Contains(c.Variation, "light") || strings.Contains(c.Variation, "†")
}

// isDCIPromo reports a DCI promo, excluding judge rewards, which carry the
// same mark.
func isDCIPromo(c *mtgmatcher.InputCard) bool {
	return c.Contains("DCI") && !c.Contains("Judge")
}

// isWPNGateway reports a Wizards Play Network or Gateway promo, including the
// Commander Party and Moonlit Lands series that ran under it.
func isWPNGateway(c *mtgmatcher.InputCard) bool {
	return c.Contains("WPN") ||
		c.Contains("Gateway") ||
		mtgmatcher.Contains(c.Variation, "Wizards Play Network") ||
		mtgmatcher.Contains(c.Variation, "Commander Party") || // scg
		mtgmatcher.Contains(c.Variation, "Moonlit Lands") // ck
}

// isRelease reports a release or launch promo, and refuses a prerelease,
// whose wording otherwise contains this one.
func isRelease(c *mtgmatcher.InputCard) bool {
	return !c.Contains("Prerelease") &&
		(c.Contains("Release") ||
			c.Contains("Draft Weekend") ||
			c.Contains("Launch"))
}

// isJudge reports a judge reward.
func isJudge(c *mtgmatcher.InputCard) bool {
	return c.Contains("Judge")
}

// isMagicFest reports a MagicFest or MagicCon promo, including TCGplayer's
// MFP code.
func isMagicFest(c *mtgmatcher.InputCard) bool {
	return c.Contains("Magic Fest") ||
		c.Contains("MagicCon") || // scg
		strings.Contains(c.Edition, "MFP") || // tcg collection
		strings.Contains(c.Variation, "MFP") // tcg collection
}

// isArena reports an Arena league promo.
func isArena(c *mtgmatcher.InputCard) bool {
	return c.Contains("Arena")
}

// isResale reports a resale or repack promo, refusing championship cards,
// whose wording collides.
func isResale(c *mtgmatcher.InputCard) bool {
	return !c.Contains("Championship") && (c.Contains("Repack") || c.Contains("Store") || c.Contains("Resale"))
}

// isMysteryList reports a Mystery Booster or The List printing. The List is
// matched raw, since folded it also matches The Little.
func isMysteryList(c *mtgmatcher.InputCard) bool {
	return saysMysteryList(c.Edition, c.Variation)
}

// saysMysteryList is isMysteryList over the wording alone, for the edition
// alias, which has no card to read.
func saysMysteryList(edition, variation string) bool {
	return mtgmatcher.Contains(edition, "Mystery") || mtgmatcher.Contains(variation, "Mystery") ||
		mtgmatcher.Contains(edition, "Planeswalker Symbol Reprints") ||
		mtgmatcher.Contains(variation, "Planeswalker Symbol Reprints") ||
		strings.Contains(edition, "The List") || strings.Contains(variation, "The List")
}

// isDuelDecks reports a Duel Decks printing, named by the two sides it pits
// against each other, and refuses the Anthology reprints.
func isDuelDecks(c *mtgmatcher.InputCard) bool {
	return ((c.Contains(" vs ")) ||
		(strings.Contains(c.Variation, " v. "))) && // tcg
		!c.Contains("Anthology")
}

// isDuelDecksAnthology reports the Duel Decks Anthology reprints.
func isDuelDecksAnthology(c *mtgmatcher.InputCard) bool {
	return strings.Contains(c.Edition, "DDA") ||
		(mtgmatcher.Contains(c.Edition, "Duel Decks") && mtgmatcher.Contains(c.Edition, "Anthology"))
}

// hasSecretLairTag reports whether the listing belongs to the given Secret
// Lair set, which each need their own rule: the drops differ in what they
// reprint and in how storefronts spell them.
func hasSecretLairTag(c *mtgmatcher.InputCard, code string) bool {
	var tag bool
	switch code {
	case "SLU":
		// SLU is mostly static and cards are unlikely to reappear elsewhere
		tag = c.Contains("Ultimate") || len(mtgmatcher.MatchInSet(c.Name, "SLU")) == 1
	case "SLX":
		// SLX only has plain cards, if they are reskinned, they are from SLD
		tag = !isReskin(c) || c.Contains("Within") || c.Contains("SLX")
	case "SLC":
		// Some of these cards are numbered after the year they represent.
		// The same numbers double as plain collector numbers in SLD (e.g.
		// Sol Ring is SLD #1993), so only treat the year as an SLC signal
		// when that exact card actually exists in SLC at that number.
		yearStr := mtgmatcher.ExtractYear(c.Variation)
		tag = c.Contains("30th") || c.Contains("Countdown") ||
			(yearStr != "" && len(mtgmatcher.MatchInSetNumber(c.Name, "SLC", yearStr)) > 0)
	case "SLP":
		// Simple check the variations
		tag = c.Contains("Showdown") || c.Contains("Prize") || c.Contains("Finish") || c.Contains("Play")
	}

	return c.IsSecretLair() && tag
}

// isIDWMagazineBook reports a promo that came with print media: comics,
// magazines, novels, and the retail tie-ins storefronts file alongside them.
func isIDWMagazineBook(c *mtgmatcher.InputCard) bool {
	return strings.HasPrefix(c.Variation, "IDW") || strings.HasPrefix(c.Edition, "IDW") ||
		c.Contains("Magazine") ||
		c.Contains("Duelist") ||
		// Catches Comic and Comics, but skips San Diego Comic-Con
		(c.Contains("Comic") && !c.Contains("Diego")) ||
		// Cannot use Contains because it may trigger a false positive
		// for cards with "book" in their variation (insidious bookworms)
		c.Variation == "Book" ||
		c.Variation == "Insert" || // mmc
		c.Variation == "Japanese Promo" || // tcg
		c.Contains("Book Insert") ||
		c.Contains("Walmart") ||
		c.Contains("Coro Coro") || // stks
		c.Contains("Graphic Novel") || // stks
		strings.Contains(c.Variation, "Book Promo") || // sz
		c.Contains("Top Deck") || // csi
		c.Contains("Hobby Japan") || // abu+tcg
		mtgmatcher.Contains(c.Edition, "CardZ") || // mkm
		mtgmatcher.Contains(c.Edition, "Dengeki") || // mkm
		c.Variation == "Insert Foil" || // ck
		c.Contains("Beadle & Grimm Phyrexian") || // scg
		c.Contains("Stance Socks") || // scg
		c.Contains("Manga Promo") || // csi
		c.Contains("Media Promo") || // tcg
		c.Contains("Media Insert") // mm+nf
}

// possibleNumberSuffix returns a lone letter from the variation, lowercased,
// which is how storefronts often carry the suffix of a collector number.
func possibleNumberSuffix(c *mtgmatcher.InputCard) string {
	fields := strings.FieldsSeq(c.Variation)
	for field := range fields {
		if len(field) == 1 && unicode.IsLetter(rune(field[0])) {
			return strings.ToLower(field)
		}
	}
	return ""
}

// isReskin reports a reskinned printing, named outright or by its Dracula or
// Godzilla series. Basic lands are excluded so the Secret Lair Godzilla lands
// stay ordinary lands.
func isReskin(c *mtgmatcher.InputCard) bool {
	return (mtgmatcher.Contains(c.Variation, "Reskin") ||
		mtgmatcher.Contains(c.Variation, "Dracula") ||
		mtgmatcher.Contains(c.Variation, "Godzilla")) &&
		// Needed to distinguish the SLD godzilla lands
		!isBasicLand(c.Name)
}

// isGenericAltArt reports alternate art, matching Alternative as well as Alt.
func isGenericAltArt(c *mtgmatcher.InputCard) bool {
	// "Alt" includes Alternative
	return c.Contains("Alt") && c.Contains("Art")
}

// isUnsupported reports whether the listing is for something with no printing
// at all behind it: art cards, memorabilia, misprint lots, sealed product and
// the like.
func isUnsupported(c *mtgmatcher.InputCard) bool {
	return c.Contains("Art Series") ||
		strings.HasSuffix(c.Edition, "Art Variants") || // toa
		(c.Contains("Art Card") && !c.Contains("Chinese")) || // Art Series, except a well-known edition
		c.Contains("Complete") || // a complete collection
		c.Contains("Fallen Empires: Wyvern Misprints") ||
		c.Contains("Ultra-Pro Puzzle") ||
		c.Contains("Player Cards") || // scg pro players
		c.Contains("Foreign White Border") || // for REV and 4ED
		c.Contains("Filler Cards") || // Misprints from mkm and ct
		c.Contains("Salvat") || // Salvat-Hachette 2005/2011
		c.Contains("Redemption Program") || // PRES
		c.Contains("Heroes of the Realm") || // HTR*
		c.Contains("Memorabilia") ||
		c.Contains("Front Card") || // Jumpstart
		(c.Contains("Duel Masters") && c.Contains("Not Tournament Legal")) || // scg
		c.Contains("Sealed") ||
		c.Contains("Un-Known Event Playtest") ||
		c.Contains("Charlie Brown") || // abu
		// Oversized are usually ok, but 8th and 9th ed box topper variants
		// conflict with the actual edition name, so skip them
		(c.Contains("Oversize") && (c.Contains("8th") || c.Contains("9th")))
}

// isSpecificUnsupported reports the named cards unsupported only in one
// edition or one misprint, rather than as a whole class.
func isSpecificUnsupported(c *mtgmatcher.InputCard) bool {
	switch c.Name {
	case "Spined Wurm":
		return mtgmatcher.Contains(c.Edition, "Starter 2000")
	case "Drudge Skeletons",
		"Emerald Medallion",
		"Forest",
		"Sapphire Medallion",
		"Serra Angel",
		"Time Elemental",
		"Winged Sliver":
		return c.Contains("Misprint")
	// Erroneous release information
	case "Zombify":
		return c.Contains("Game Night")
	default:
		if mtgmatcher.Contains(c.Name, "Sticker Sheet") {
			return true
		}
	}
	return false
}
