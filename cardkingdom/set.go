package cardkingdom

import (
	"fmt"
	"strings"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

// Adjust the ck set name to the proper mtgjson name
func parseSet(cardName, setName, cardType string) string {
	// Split name according to the contents within ()
	variants := mtgban.SplitVariants(cardName)

	// Adjust the Set information
	switch {
	// Rebuild DDA deck type from the card name
	case setName == "Duel Decks: Anthology" && len(variants) > 1:
		version := strings.Replace(variants[1], " - Foil", "", 1)
		setName = "Duel Decks Anthology: " + version
		setName = strings.Replace(setName, " vs ", " vs. ", 1)
	case strings.HasPrefix(setName, "Duel Decks: "):
		setName = strings.Replace(setName, " Vs. ", " vs. ", 1)

	// Rework the WCD set names using the year in the card name
	case setName == "World Championships" && len(variants) > 1:
		year := wcdExp.ReplaceAllString(cardName, `$1`)
		setName = fmt.Sprintf("World Championship Decks %s", year)

	// Separate playtest cards in their own set
	case setName == "Mystery Booster":
		if len(variants) > 1 && variants[1] == "Not Tournament Legal" {
			setName = "Mystery Booster Playtest Cards"
		}

	// Separate planeswalker cards in their own set
	case setName == "Secret Lair":
		setName = "Secret Lair Drop Series"
		if strings.HasPrefix(cardType, "Legendary Planeswalker") {
			setName = "Secret Lair Promos"
		}

	// 'xxxx core set' -> 'magic xxxx' (the post-2019 ones are fine)
	case strings.HasSuffix(setName, "Core Set"):
		s := strings.Split(setName, " ")
		setName = "Magic " + s[0]
	}

	// Fix some wrong set attributions (mostly from promos)
	ed, found := card2setTable[cardName]
	if found {
		setName = ed
	}

	// Look up the Set
	ed, found = setTable[setName]
	if found {
		setName = ed
	}

	return setName
}

// Adjust the set name for Promotional sets, return a comparison function
// that will overrider the default one.
func parsePromotional(variants []string) (string, func(set mtgjson.Set) bool) {
	setName := promosetTable[variants[1]]
	setCheck := func(set mtgjson.Set) bool {
		return set.Name == setName
	}

	switch {
	// There way too many variations for Promo Pack promos, some are under
	// a different set type and some are under the normal expansion set.
	// For reference these cards can be identified only via:
	// - they have a 'p' suffix added to their collector number
	// - they are listed under a "Promos" or "Promo Pack" set
	// - they have an inverted frame effect
	// - they appear in normal sets, numbered after the normal size
	// but we can't use any of those methods reliably here, and they are
	// mutually exclusive, so instead we pollute setVariants with all the
	// possible cards and sets that could have a promo, if they appear in
	// a Promo Pack, and add extra checks to separate them from Prerelease.
	// Also need to check against Secret Lair Promos to avoid aliasing.
	case strings.HasPrefix(variants[1], "Promo Pack"):
		setCheck = func(set mtgjson.Set) bool {
			return set.Name != "Secret Lair Promos" &&
				(strings.HasSuffix(set.Name, "Promos") ||
					strings.HasSuffix(set.Name, "Promo Packs") ||
					set.Type == "expansion")
		}

	// The "Prerelease Events" are the old style prerelease promos, all
	// the others reside in the "Promos" section of the expansion.
	case strings.Contains(variants[1], "Prerelease"):
		setCheck = func(set mtgjson.Set) bool {
			return set.Name == "Prerelease Events" || strings.HasSuffix(set.Name, "Promos")
		}

	case strings.HasPrefix(variants[1], "Clash Pack"):
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Clash Pack")
		}
	case strings.HasPrefix(variants[1], "IDW"):
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "IDW Comics")
		}
	case strings.HasPrefix(variants[1], "MagicFest"):
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "MagicFest")
		}
	case strings.HasPrefix(variants[1], "Textless") || strings.HasPrefix(variants[1], "Player Reward"):
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Magic Player Rewards")
		}
	case strings.HasPrefix(variants[1], "Release") || strings.HasPrefix(variants[1], "Launch"):
		setCheck = func(set mtgjson.Set) bool {
			return set.Name == "Release Events" || set.Name == "Launch Parties" || strings.HasSuffix(set.Name, "Promos")
		}
	case strings.HasPrefix(variants[1], "Gateway") || strings.HasPrefix(variants[1], "WPN"):
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Gateway") || strings.HasPrefix(set.Name, "Wizards Play Network")
		}

	// Duels of the Planeswalkers (found in console games)
	case strings.HasPrefix(variants[1], "Duels ") || strings.Contains(strings.ToLower(variants[1]), "xbox") || strings.HasPrefix(variants[1], "PS"):
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Duels of the Planeswalkers Promos")
		}

	// Retrieve the year to rebuild the correct set name or do a blind search
	case strings.HasPrefix(variants[1], "Arena"):
		s := strings.Split(variants[1], " ")

		// Ignore tags that are not a year
		if len(s) > 1 && s[1] != "Foil" && s[1] != "Promo" {
			setName = "Arena League " + s[1]
		} else {
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Arena League")
			}
		}

	// Retrieve the year to rebuild the correct set name or do a blind search
	case strings.Contains(variants[1], "FNM"):
		s := strings.Split(variants[1], " ")

		// Use the year in the name
		if len(s) > 2 && s[2][0] == '\'' {
			setName = "Friday Night Magic 20" + s[2][1:]
		} else {
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Friday Night Magic")
			}
		}

	// Retrieve the year to rebuild the correct set name or do a blind search
	case strings.Contains(variants[1], "Judge"):
		if len(variants) > 2 {
			setName = "Judge Gift Cards " + variants[2]
		} else {
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Judge Gift Cards")
			}
		}

	// Retrieve the year to rebuild the correct set name
	case strings.HasPrefix(variants[1], "MPS"):
		s := strings.Split(variants[1], " ")
		if len(s) < 2 {
			setCheck = func(set mtgjson.Set) bool {
				return false
			}
		}
		setName = "Magic Premiere Shop " + s[1]

	// Retrieve the year to rebuild the correct set name
	case strings.Contains(variants[1], "SDCC"):
		s := strings.Split(variants[1], " ")
		if len(s) < 2 {
			setCheck = func(set mtgjson.Set) bool {
				return false
			}
		}
		setName = "San Diego Comic-Con " + s[1]

	// Rename the set, will find the number later on
	case strings.HasPrefix(variants[1], "Ravnica Weekend - A"):
		setName = "GRN Ravnica Weekend"
	case strings.HasPrefix(variants[1], "Ravnica Weekend - B"):
		setName = "RNA Ravnica Weekend"

	// JSS cards have a code in variants[2] which is useless.
	// Just drop the optional "Foil" tag and we can use it as is.
	case strings.Contains(variants[1], "Junior"):
		setName = strings.Replace(variants[1], " Foil", "", 1)

	default:
		// All these sets fall under some form of "xxx Promos"
		switch variants[1] {
		case "Bundle Foil",
			"Buy-A-Box Foil",
			"Buy-a-Box Foil",
			"Draft Weekend Foil",
			"Extended Art Foil",
			"Game Day Extended Art Foil",
			"Game Day Extended Art",
			"Game Day Foil",
			"Game Day Promo",
			"Gift Box Foil",
			"Intro Pack Rare Foil",
			"Launch Foil",
			"Launch Promo Foil",
			"Magic League Foil",
			"Open House Foil",
			"OpenHouse",
			"Store Championship Foil",
			"Planeswalker Weekend Foil":
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasSuffix(set.Name, "Promos")
			}
		}
	}

	return setName, setCheck
}
