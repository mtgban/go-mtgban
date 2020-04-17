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

	// The scryfall identifier of the card (output only)
	ImageId string

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
		ImageId: card.ScryfallId,
		Name:    card.Name,
		Edition: set.Name,
		Foil:    foil,
		Number:  card.Number,
	}

	switch {
	case c.isShowcase():
		out.Variation = "Showcase"
	case c.isBorderless():
		out.Variation = "Borderless"
	case c.isExtendedArt():
		out.Variation = "Extended Art"

	case c.isBundle():
		out.Variation = "Bundle"
	case c.isRelease():
		out.Variation = "Release"
	case c.isPrerelease():
		out.Variation = "Prerelease"
	case c.isBaB():
		out.Variation = "Buy-a-Box"

	case c.isARNLightMana():
		out.Variation = "Light Variant"
	case c.isARNDarkMana():
		out.Variation = "Dark Variant"
	case c.isBasicFullArt():
		out.Variation = "Full Art"
	}

	if c.isJPN() {
		if out.Variation != "" {
			out.Variation += " "
		}
		out.Variation += "Japanese"
	}

	// Append "_f" to the Id to distinguish from non-foil
	if (card.HasNonFoil || card.IsAlternative) && foil {
		out.Id += "_f"
	}

	return out
}

// Returns whether the card is a basic land
func (c *Card) IsBasicLand() bool {
	switch {
	case strings.HasPrefix(c.Name, "Plains"),
		strings.HasPrefix(c.Name, "Island"),
		strings.HasPrefix(c.Name, "Swamp"),
		strings.HasPrefix(c.Name, "Mountain"),
		strings.HasPrefix(c.Name, "Forest"),
		strings.HasPrefix(c.Name, "Wastes"):
		return true
	}
	return false
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
		c.Variation == "Dark Frame Promo"
}

func (c *Card) isBorderless() bool {
	return strings.Contains(c.Variation, "Borderless")
}

func (c *Card) isExtendedArt() bool {
	return strings.Contains(c.Variation, "Extended Art")
}

func (c *Card) isShowcase() bool {
	return strings.Contains(c.Variation, "Showcase")
}

func (c *Card) isFNM() bool {
	return strings.Contains(c.Variation, "FNM") ||
		strings.Contains(c.Variation, "Friday Night Magic")
}

func (c *Card) isJPN() bool {
	return strings.Contains(c.Variation, "JPN") ||
		mtgjson.NormContains(c.Variation, "Japanese") ||
		mtgjson.NormContains(c.Variation, "Gotta") ||
		mtgjson.NormContains(c.Variation, "Dengeki")
}

func (c *Card) isRelease() bool {
	return strings.Contains(c.Variation, "Release") ||
		strings.Contains(c.Variation, "Launch")
}

func (c *Card) isWPNGateway() bool {
	return strings.Contains(c.Variation, "WPN") ||
		strings.Contains(c.Variation, "Gateway") ||
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
		c.Variation == "Insert Foil" || // ck
		c.Variation == "Media Insert" // mm
}

func (c *Card) isRewards() bool {
	return strings.Contains(c.Variation, "Textless") ||
		strings.Contains(c.Variation, "Reward")
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

func (c *Card) isWorldChamp() bool {
	return mtgjson.NormContains(c.Edition, "Pro Tour Collect") ||
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
		if mtgjson.NormContains(c.Variation, player) {
			return players[player], mtgjson.NormContains(c.Variation, "Sideboard")
		}
	}
	return "", false
}

func (c *Card) isDuelsOfThePW() bool {
	return strings.Contains(c.Variation, "Duels") ||
		strings.Contains(c.Edition, "Duels")
}

func (c *Card) isBasicFullArt() bool {
	return c.isBasicLand() &&
		mtgjson.NormContains(c.Variation, "full art") &&
		!mtgjson.NormContains(c.Variation, "non")
}

func (c *Card) isBasicNonFullArt() bool {
	return c.isBasicLand() &&
		mtgjson.NormContains(c.Variation, "non-full art") ||
		mtgjson.NormContains(c.Variation, "Intro") // abu
}

func (c *Card) isPremiereShop() bool {
	return c.isBasicLand() &&
		(strings.Contains(c.Variation, "MPS") ||
			strings.Contains(c.Edition, "MPS"))
}

func (c *Card) isDuelDecks() bool {
	return (mtgjson.NormContains(c.Variation, " vs ") ||
		mtgjson.NormContains(c.Edition, " vs ")) &&
		!mtgjson.NormContains(c.Variation, "Anthology") &&
		!mtgjson.NormContains(c.Edition, "Anthology")
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
