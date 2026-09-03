package magic

import (
	"strings"

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
	if !c.IsDuelDecks() {
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
	if c.IsMysteryList() && !c.Contains("Unfinity") {
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
	return isBasicLand(c) &&
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
		(c.Contains("Reward") && !c.IsJudge())
}

// isDuelsOfThePW reports a Duels of the Planeswalkers promo. It compares the
// raw strings so the fold does not equate Duels with Duel Decks.
func isDuelsOfThePW(c *mtgmatcher.InputCard) bool {
	// XXX: do not use c.Contains here
	return strings.Contains(c.Variation, "Duels") ||
		strings.Contains(c.Edition, "Duels") ||
		mtgmatcher.Contains(c.Variation, "DotP") // tat
}
