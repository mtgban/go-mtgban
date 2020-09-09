package mtgmatcher

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/kodabb/go-mtgmatcher/mtgmatcher/mtgjson"
)

// Card is a generic card representation using fields defined by the MTGJSON project.
type Card struct {
	// The mtgjson unique identifier of the card
	// When used as input it can host mtgjson or scryfall id
	Id string

	// The canonical name of the card
	Name string

	// The hint or commonly know variation
	Variation string

	// The set the card comes from, or a portion of it
	Edition string

	// Whether the card is foil or not
	Foil bool

	// The collector number of the card (output only)
	Number string
}

// Card implements the Stringer interface
func (c *Card) String() string {
	out := c.Name
	if c.Variation != "" {
		out = fmt.Sprintf("%s (%s)", out, c.Variation)
	}
	foil := ""
	if c.Foil {
		foil = " " + mtgjson.SuffixSpecial
	}
	return fmt.Sprintf("%s [%s%s] {%s}", out, c.Edition, foil, c.Number)
}

func output(card mtgjson.Card, foil bool) string {
	// In case the foiling information is incorrect
	if !foil && !card.HasNonFoil {
		foil = true
	} else if foil && !card.HasFoil {
		foil = false
	}
	if card.HasFoil && !card.HasNonFoil {
		foil = true
	} else if !card.HasFoil && card.HasNonFoil && !card.IsAlternative {
		foil = false
	}

	// Prepare the output card
	id := card.UUID
	// Append "_f" to the Id to distinguish from non-foil
	if (card.HasNonFoil || card.IsAlternative) && foil {
		id += "_f"
	}

	return id
}

func (c *Card) addToVariant(tag string) {
	if c.Variation != "" {
		c.Variation += " "
	}
	c.Variation += tag
}

// Returns whether the input string may represent a basic land
func IsBasicLand(name string) bool {
	switch {
	case strings.Contains(name, "Bear") && !strings.Contains(name, "Beard"), // G
		strings.Contains(name, "Mosquito"),                                     // B
		strings.Contains(name, "Stronghold"), strings.Contains(name, "Bandit"), // R
		strings.Contains(name, "Yeti"), strings.Contains(name, "Titan"), // R
		strings.Contains(name, "Valley"), strings.Contains(name, "Goat"), // R
		strings.Contains(name, "Fish"), strings.Contains(name, "Sanctuary"): // U
	case strings.HasPrefix(name, "Plains"),
		strings.HasPrefix(name, "Island"),
		strings.HasPrefix(name, "Swamp"),
		strings.HasPrefix(name, "Mountain"),
		strings.HasPrefix(name, "Forest"),
		strings.HasPrefix(name, "Wastes"):
		return true
	}
	return false
}

// Returns whether the card is a basic land
func (c *Card) IsBasicLand() bool {
	return IsBasicLand(c.Name)
}

// More specific version of the above, for internal use only
func (c *Card) isBasicLand() bool {
	switch c.Name {
	case "Plains", "Island", "Swamp", "Mountain", "Forest", "Wastes":
		return true
	}
	return false
}

func (c *Card) isGenericPromo() bool {
	return !c.isBaB() && !c.isPromoPack() &&
		(Contains(c.Variation, "Promo") ||
			Contains(c.Variation, "Game Day") ||
			Contains(c.Edition, "Promo"))
}

func (c *Card) isGenericAltArt() bool {
	return Contains(c.Variation, "Alt") && // includes Alternative
		Contains(c.Variation, "Art")
}

func (c *Card) isGenericExtendedArt() bool {
	return Contains(c.Variation, "Art") &&
		(Contains(c.Variation, "Extended") ||
			Contains(c.Variation, "Full"))
}

func (c *Card) isPrerelease() bool {
	return Contains(c.Variation, "Prerelease")
}

func (c *Card) isPromoPack() bool {
	return Contains(c.Edition, "Promo Pack") ||
		Contains(c.Variation, "Promo Pack") ||
		c.Variation == "Dark Frame Promo" ||
		Contains(c.Variation, "Planeswalker Stamp")
}

func (c *Card) isBorderless() bool {
	return Contains(c.Variation, "Borderless")
}

func (c *Card) isExtendedArt() bool {
	return Contains(c.Variation, "Extended Art") ||
		Contains(c.Variation, "Extended Frame") //csi
}

func (c *Card) isShowcase() bool {
	return Contains(c.Variation, "Showcase")
}

func (c *Card) isReskin() bool {
	return Contains(c.Variation, "Godzilla")
}

func (c *Card) isFNM() bool {
	return Contains(c.Variation, "FNM") ||
		strings.Contains(c.Variation, "Friday Night Magic")
}

func (c *Card) isJPN() bool {
	return strings.Contains(c.Variation, "JPN") ||
		Contains(c.Variation, "Japanese") ||
		Contains(c.Variation, "Gotta") ||
		Contains(c.Variation, "Dengeki")
}

func (c *Card) isRelease() bool {
	return (!Contains(c.Variation, "Prerelease") &&
		Contains(c.Variation, "Release")) ||
		strings.Contains(c.Variation, "Launch") ||
		(!Contains(c.Edition, "Prerelease") &&
			Contains(c.Edition, "Release")) ||
		strings.Contains(c.Edition, "Launch")
}

func (c *Card) isWPNGateway() bool {
	return strings.Contains(c.Variation, "WPN") ||
		Contains(c.Variation, "Gateway") ||
		Contains(c.Variation, "Wizards Play Network") ||
		Contains(c.Edition, "Gateway") ||
		c.Variation == "Euro Promo" // cfb
}

func (c *Card) isIDWMagazineBook() bool {
	return strings.HasPrefix(c.Variation, "IDW") ||
		strings.Contains(c.Variation, "Magazine") ||
		strings.Contains(c.Variation, "Duelist") ||
		// Catches Comic and Comics, but skips San Diego Comic-Con
		(strings.Contains(c.Variation, "Comic") && !strings.Contains(c.Variation, "Diego")) ||
		// Cannot use Contains because it may trigger a false positive
		// for cards with "book" in their variation (insidious bookworms)
		c.Variation == "Book" ||
		strings.Contains(c.Variation, "Book Insert") || // cfb
		strings.Contains(c.Variation, "Book Promo") || // sz
		Contains(c.Variation, "Top Deck") || // csi
		c.Variation == "Insert Foil" || // ck
		c.Variation == "Media Insert" // mm
}

func (c *Card) isRewards() bool {
	return Contains(c.Variation, "Textless") ||
		Contains(c.Variation, "Reward") ||
		Contains(c.Edition, "Reward")
}

func (c *Card) isMagicFest() bool {
	return Contains(c.Variation, "Magic Fest") ||
		Contains(c.Edition, "Magic Fest")
}

func (c *Card) isBaB() bool {
	return Contains(c.Variation, "Buy-a-Box") ||
		strings.Contains(c.Variation, "BIBB") || // sz
		c.Variation == "Full Box Promo" // sz
}

func (c *Card) isBundle() bool {
	return Contains(c.Variation, "Bundle")
}

func (c *Card) isARNLightMana() bool {
	return Contains(c.Variation, "light")
}

func (c *Card) isARNDarkMana() bool {
	return Contains(c.Variation, "dark")
}

func (c *Card) isArena() bool {
	return strings.Contains(c.Variation, "Arena") ||
		strings.Contains(c.Edition, "Arena")
}

func (c *Card) arenaYear(maybeYear string) string {
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
	}
	return maybeYear
}

func (c *Card) isWorldChamp() bool {
	return Contains(c.Edition, "Pro Tour Collect") ||
		Contains(c.Edition, "Pro Tour 1996") ||
		Contains(c.Edition, "World Championship") ||
		Contains(c.Edition, "WCD")
}

func (c *Card) worldChampPrefix() (string, bool) {
	players := map[string]string{
		"Aeo Paquette":         "ap",
		"Alex Borteh":          "ab",
		"Antoine Ruel":         "ar",
		"Ben Rubin":            "br",
		"Bertrand Lestree":     "bl",
		"Brian Hacker":         "bh",
		"Brian Kibler":         "bk",
		"Brian Selden":         "bs",
		"Carlos Romao":         "cr",
		"Daniel Zink":          "dz",
		"Dave Humpherys":       "dh",
		"Eric Tam":             "et",
		"Gabriel Nassif":       "gn",
		"George Baxter":        "gb",
		"Jakub Slemr":          "js",
		"Jan Tomcani":          "jt",
		"Janosch Kuhn":         "jk",
		"Janosch Kuehn":        "jk",
		"Jon Finkel":           "jf",
		"Julien Nuijten":       "jn",
		"Kai Budde":            "kb",
		"Leon Lindback":        "ll",
		"Manuel Bevand":        "mb",
		"Mark Justice":         "mj",
		"Mark Le Pine":         "mlp",
		"Matt Linde":           "ml",
		"Michael Locanto":      "ml",
		"Nicolas Labarre":      "nl",
		"Paul McCabe":          "pm",
		"Peer Kröger":          "pk",
		"Preston Poulter":      "pp",
		"Randy Buehler":        "rb",
		"Raphael Levy":         "rl",
		"Shawn Hammer Regnier": "shr",
		"Sim Han How":          "shh",
		"Svend Geertsen":       "sg",
		"Tom van de Logt":      "tvdl",
		"Wolfgang Eder":        "we",
	}
	for player := range players {
		if Contains(c.Variation, player) || Contains(c.Edition, player) {
			sb := strings.Contains(strings.ToLower(c.Variation), "sb") ||
				Contains(c.Variation, "Sideboard")
			return players[player], sb
		}
	}
	return "", false
}

func (c *Card) isDuelsOfThePW() bool {
	return strings.Contains(c.Variation, "Duels") ||
		strings.Contains(c.Edition, "Duels") ||
		Contains(c.Variation, "DotP") // tat
}

func (c *Card) isBasicFullArt() bool {
	return c.isBasicLand() &&
		(Contains(c.Variation, "full art") ||
			c.Variation == "FA") && // csi
		!Contains(c.Variation, "non") &&
		!Contains(c.Variation, "not") // csi
}

func (c *Card) isBasicNonFullArt() bool {
	return c.isBasicLand() &&
		Contains(c.Variation, "non-full art") ||
		Contains(c.Variation, "Intro") || // abu
		Contains(c.Variation, "NOT the full art") // csi
}

func (c *Card) isPremiereShop() bool {
	return c.isBasicLand() &&
		(strings.Contains(c.Variation, "MPS") ||
			strings.Contains(c.Variation, "Premier") || // csi
			strings.Contains(c.Edition, "MPS"))
}

func (c *Card) isPortalAlt() bool {
	return (Contains(c.Variation, "Reminder Text") &&
		!Contains(c.Variation, "No")) ||
		Contains(c.Variation, "No Flavor Text") || // csi
		Contains(c.Variation, "Without Flavor Text") // csi
}

func (c *Card) isDuelDecks() bool {
	return (Contains(c.Variation, " vs ") ||
		Contains(c.Edition, " vs ")) &&
		!Contains(c.Variation, "Anthology") &&
		!Contains(c.Edition, "Anthology")
}

func (c *Card) isDuelDecksAnthology() bool {
	return Contains(c.Edition, "Duel Decks Anthology") &&
		(Contains(c.Variation, " vs ") ||
			Contains(c.Edition, " vs "))
}

func (c *Card) duelDecksVariant() string {
	if !c.isDuelDecks() {
		return ""
	}

	// Variation might contain numbers, strip them away
	variant := c.Variation
	num := ExtractNumber(variant)
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

func (c *Card) possibleNumberSuffix() string {
	fields := strings.Fields(c.Variation)
	for _, field := range fields {
		if len(field) == 1 && unicode.IsLetter(rune(field[0])) {
			return strings.ToLower(field)
		}
	}
	return ""
}
