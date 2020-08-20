package mtgdb

import (
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"unicode"

	"github.com/kodabb/go-mtgban/mtgjson"
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

	// Rarity of the card (output only)
	Rarity string
}

func (c *Card) Match() (outCard *Card, err error) {
	if internal == nil {
		return nil, fmt.Errorf("internal database is not initialized")
	}
	logger := log.New(ioutil.Discard, "", log.LstdFlags)
	return internal.Match(c, logger)
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

func (c *Card) output(card mtgjson.Card, set *mtgjson.Set) *Card {
	foil := c.Foil
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
	out := &Card{
		Id:      card.UUID,
		Name:    card.Name,
		Edition: set.Name,
		Foil:    foil,
		Number:  card.Number,
		Rarity:  strings.ToUpper(string(card.Rarity[0])),
	}

	// Append "_f" to the Id to distinguish from non-foil
	if (card.HasNonFoil || card.IsAlternative) && foil {
		out.Id += "_f"
	}

	return out
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
		(mtgjson.NormContains(c.Variation, "Promo") ||
			mtgjson.NormContains(c.Variation, "Game Day") ||
			mtgjson.NormContains(c.Edition, "Promo"))
}

func (c *Card) isGenericAltArt() bool {
	return mtgjson.NormContains(c.Variation, "Alt") && // includes Alternative
		mtgjson.NormContains(c.Variation, "Art")
}

func (c *Card) isGenericExtendedArt() bool {
	return mtgjson.NormContains(c.Variation, "Art") &&
		(mtgjson.NormContains(c.Variation, "Extended") ||
			mtgjson.NormContains(c.Variation, "Full"))
}

func (c *Card) isPrerelease() bool {
	return mtgjson.NormContains(c.Variation, "Prerelease")
}

func (c *Card) isPromoPack() bool {
	return mtgjson.NormContains(c.Edition, "Promo Pack") ||
		mtgjson.NormContains(c.Variation, "Promo Pack") ||
		c.Variation == "Dark Frame Promo" ||
		mtgjson.NormContains(c.Variation, "Planeswalker Stamp")
}

func (c *Card) isBorderless() bool {
	return mtgjson.NormContains(c.Variation, "Borderless")
}

func (c *Card) isExtendedArt() bool {
	return mtgjson.NormContains(c.Variation, "Extended Art") ||
		mtgjson.NormContains(c.Variation, "Extended Frame") //csi
}

func (c *Card) isShowcase() bool {
	return mtgjson.NormContains(c.Variation, "Showcase")
}

func (c *Card) isReskin() bool {
	return mtgjson.NormContains(c.Variation, "Godzilla")
}

func (c *Card) isFNM() bool {
	return mtgjson.NormContains(c.Variation, "FNM") ||
		strings.Contains(c.Variation, "Friday Night Magic")
}

func (c *Card) isJPN() bool {
	return strings.Contains(c.Variation, "JPN") ||
		mtgjson.NormContains(c.Variation, "Japanese") ||
		mtgjson.NormContains(c.Variation, "Gotta") ||
		mtgjson.NormContains(c.Variation, "Dengeki")
}

func (c *Card) isRelease() bool {
	return (!mtgjson.NormContains(c.Variation, "Prerelease") &&
		mtgjson.NormContains(c.Variation, "Release")) ||
		strings.Contains(c.Variation, "Launch") ||
		(!mtgjson.NormContains(c.Edition, "Prerelease") &&
			mtgjson.NormContains(c.Edition, "Release")) ||
		strings.Contains(c.Edition, "Launch")
}

func (c *Card) isWPNGateway() bool {
	return strings.Contains(c.Variation, "WPN") ||
		mtgjson.NormContains(c.Variation, "Gateway") ||
		mtgjson.NormContains(c.Variation, "Wizards Play Network") ||
		mtgjson.NormContains(c.Edition, "Gateway") ||
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
		mtgjson.NormContains(c.Variation, "Top Deck") || // csi
		c.Variation == "Insert Foil" || // ck
		c.Variation == "Media Insert" // mm
}

func (c *Card) isRewards() bool {
	return mtgjson.NormContains(c.Variation, "Textless") ||
		mtgjson.NormContains(c.Variation, "Reward") ||
		mtgjson.NormContains(c.Edition, "Reward")
}

func (c *Card) isMagicFest() bool {
	return mtgjson.NormContains(c.Variation, "Magic Fest") ||
		mtgjson.NormContains(c.Edition, "Magic Fest")
}

func (c *Card) isBaB() bool {
	return mtgjson.NormContains(c.Variation, "Buy-a-Box") ||
		strings.Contains(c.Variation, "BIBB") || // sz
		c.Variation == "Full Box Promo" // sz
}

func (c *Card) isBundle() bool {
	return mtgjson.NormContains(c.Variation, "Bundle")
}

func (c *Card) isARNLightMana() bool {
	return mtgjson.NormContains(c.Variation, "light")
}

func (c *Card) isARNDarkMana() bool {
	return mtgjson.NormContains(c.Variation, "dark")
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
	return mtgjson.NormContains(c.Edition, "Pro Tour Collect") ||
		mtgjson.NormContains(c.Edition, "Pro Tour 1996") ||
		mtgjson.NormContains(c.Edition, "World Championship") ||
		mtgjson.NormContains(c.Edition, "WCD")
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
		if mtgjson.NormContains(c.Variation, player) ||
			mtgjson.NormContains(c.Edition, player) {
			sb := strings.Contains(strings.ToLower(c.Variation), "sb") ||
				mtgjson.NormContains(c.Variation, "Sideboard")
			return players[player], sb
		}
	}
	return "", false
}

func (c *Card) isDuelsOfThePW() bool {
	return strings.Contains(c.Variation, "Duels") ||
		strings.Contains(c.Edition, "Duels") ||
		mtgjson.NormContains(c.Variation, "DotP") // tat
}

func (c *Card) isBasicFullArt() bool {
	return c.isBasicLand() &&
		(mtgjson.NormContains(c.Variation, "full art") ||
			c.Variation == "FA") && // csi
		!mtgjson.NormContains(c.Variation, "non") &&
		!mtgjson.NormContains(c.Variation, "not") // csi
}

func (c *Card) isBasicNonFullArt() bool {
	return c.isBasicLand() &&
		mtgjson.NormContains(c.Variation, "non-full art") ||
		mtgjson.NormContains(c.Variation, "Intro") || // abu
		mtgjson.NormContains(c.Variation, "NOT the full art") // csi
}

func (c *Card) isPremiereShop() bool {
	return c.isBasicLand() &&
		(strings.Contains(c.Variation, "MPS") ||
			strings.Contains(c.Variation, "Premier") || // csi
			strings.Contains(c.Edition, "MPS"))
}

func (c *Card) isPortalAlt() bool {
	return (mtgjson.NormContains(c.Variation, "Reminder Text") &&
		!mtgjson.NormContains(c.Variation, "No")) ||
		mtgjson.NormContains(c.Variation, "No Flavor Text") || // csi
		mtgjson.NormContains(c.Variation, "Without Flavor Text") // csi
}

func (c *Card) isDuelDecks() bool {
	return (mtgjson.NormContains(c.Variation, " vs ") ||
		mtgjson.NormContains(c.Edition, " vs ")) &&
		!mtgjson.NormContains(c.Variation, "Anthology") &&
		!mtgjson.NormContains(c.Edition, "Anthology")
}

func (c *Card) isDuelDecksAnthology() bool {
	return mtgjson.NormContains(c.Edition, "Duel Decks Anthology") &&
		(mtgjson.NormContains(c.Variation, " vs ") ||
			mtgjson.NormContains(c.Edition, " vs "))
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
