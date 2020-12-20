package mtgmatcher

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
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
	} else if !card.HasFoil && card.HasNonFoil {
		foil = false
	}

	// Prepare the output card
	id := card.UUID
	// Append "_f" to the Id to distinguish from non-foil
	if card.HasNonFoil && foil {
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

// Returns whether the input string may represent a token
func IsToken(name string) bool {
	switch name {
	// Known token names
	case "A Threat to Alara: Nicol Bolas",
		"Acorn Stash",
		"Ashaya, the Awoken World",
		"City's Blessing",
		"Companion",
		"Energy Reserve",
		"Fun Format: Pack Wars",
		"Manifest",
		"Morph",
		"On Your Turn",
		"On an Adventure",
		"Poison Counter",
		"Theme: WUBRG Cards":
		return true
	// Un-tokens
	case "Beast",
		"Beeble",
		"Dragon",
		"Goblin",
		"Pegasus",
		"Spirit",
		"Sheep",
		"Soldier",
		"Squirrel",
		"Zombie":
		return true
	}
	switch {
	// Avoid confusion with Monarch and Emblem below
	case HasPrefix(name, "Emblem of the Warmind"),
		HasPrefix(name, "Kavu Monarch"),
		HasPrefix(name, "Leering Emblem"):
		return false
	// Anything token
	case Contains(name, "Arena Code"),
		Contains(name, "Art Card"),
		Contains(name, "Art Series"),
		Contains(name, "Blank Card"),
		Contains(name, "Card List"),
		Contains(name, "Checklist"),
		Contains(name, "DFC Helper"),
		Contains(name, "Emblem"),
		Contains(name, "Experience C"),
		Contains(name, "Guild Symbol"),
		Contains(name, "Helper Card"),
		Contains(name, "Magic Minigame"),
		Contains(name, "Monarch"),
		Contains(name, "Online Code"),
		Contains(name, "Oversize"),
		Contains(name, "Punch Card"),
		Contains(name, "Punch Out"),
		Contains(name, "Rules Card"),
		Contains(name, "Strategy Card"),
		Contains(name, "Token"),
		Contains(name, "Rules Tip"):
		return true
	// Alternative rules tip card names found on mkm
	case strings.HasPrefix(name, "Tip: "):
		return true
	// One more generic un-token
	case Contains(name, "Teddy Bear"):
		return true
	}

	return false
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
		(c.Contains("Promo") ||
			Contains(c.Variation, "Game Day") ||
			Contains(c.Edition, "Game Day"))
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
	return c.Contains("Prerelease")
}

func (c *Card) isPromoPack() bool {
	return c.Contains("Promo Pack") ||
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
	return Contains(c.Variation, "Godzilla") &&
		// Needed to distinguish the SLD godizlla lands
		!c.isBasicLand()
}

func (c *Card) isFNM() bool {
	return c.Contains("FNM") ||
		c.Contains("Friday Night Magic")
}

func (c *Card) isJPN() bool {
	return strings.Contains(c.Variation, "JPN") ||
		c.Contains("Japanese") ||
		Contains(c.Variation, "Gotta") ||
		Contains(c.Variation, "Dengeki")
}

func (c *Card) isRelease() bool {
	return !c.Contains("Prerelease") &&
		(c.Contains("Release") ||
			c.Contains("Launch"))
}

func (c *Card) isWPNGateway() bool {
	return c.Contains("WPN") ||
		c.Contains("Gateway") ||
		Contains(c.Variation, "Wizards Play Network") ||
		c.Variation == "Euro Promo" // cfb
}

func (c *Card) isIDWMagazineBook() bool {
	return strings.HasPrefix(c.Variation, "IDW") || strings.HasPrefix(c.Edition, "IDW") ||
		c.Contains("Magazine") ||
		c.Contains("Duelist") ||
		// Catches Comic and Comics, but skips San Diego Comic-Con
		(c.Contains("Comic") && !c.Contains("Diego")) ||
		// Cannot use Contains because it may trigger a false positive
		// for cards with "book" in their variation (insidious bookworms)
		c.Variation == "Book" ||
		c.Contains("Book Insert") ||
		strings.Contains(c.Variation, "Book Promo") || // sz
		c.Contains("Top Deck") || // csi
		Contains(c.Edition, "CardZ") || // mkm
		Contains(c.Edition, "Dengeki") || // mkm
		c.Variation == "Insert Foil" || // ck
		c.Variation == "Media Insert" // mm
}

func (c *Card) isJudge() bool {
	return c.Contains("Judge")
}

func (c *Card) isRewards() bool {
	return Contains(c.Variation, "Textless") ||
		(c.Contains("Reward") && !c.isJudge())
}

func (c *Card) isMagicFest() bool {
	return c.Contains("Magic Fest")
}

func (c *Card) isBaB() bool {
	return c.Contains("Buy a Box") ||
		strings.Contains(c.Variation, "BIBB") || // sz
		c.Variation == "Full Box Promo" // sz
}

func (c *Card) isBundle() bool {
	return c.Contains("Bundle")
}

func (c *Card) isFoilEtched() bool {
	return Contains(c.Variation, "Etched") && Contains(c.Variation, "Foil")
}

func (c *Card) isARNLightMana() bool {
	return Contains(c.Variation, "light")
}

func (c *Card) isARNDarkMana() bool {
	return Contains(c.Variation, "dark")
}

func (c *Card) isArena() bool {
	return c.Contains("Arena")
}

func (c *Card) isSDCC() bool {
	return c.Contains("SDCC") ||
		c.Contains("San Diego Comic-Con")
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
		"Michael Loconto":      "ml",
		"Nicolas Labarre":      "nl",
		"Paul McCabe":          "pm",
		"Peer Kroger":          "pk",
		"Preston Poulter":      "pp",
		"Randy Buehler":        "rb",
		"Raphael Levy":         "rl",
		"Shawn Regnier":        "shr",
		"Shawn Hammer Regnier": "shr",
		"Sim Han How":          "shh",
		"Svend Geertsen":       "sg",
		"Tom van de Logt":      "tvdl",
		"Wolfgang Eder":        "we",
	}
	for player := range players {
		if c.Contains(player) {
			sb := strings.Contains(strings.ToLower(c.Variation), "sb") ||
				Contains(c.Variation, "Sideboard")
			return players[player], sb
		}
	}
	return "", false
}

func (c *Card) isDuelsOfThePW() bool {
	// XXX: do not use c.Contains here
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
		// XXX: do not use c.Contains here
		(strings.Contains(c.Variation, "MPS") ||
			strings.Contains(c.Variation, "Premier") || // csi
			strings.Contains(c.Edition, "MPS") ||
			strings.Contains(c.Edition, "Premiere Shop")) // mkm
}

func (c *Card) isPortalAlt() bool {
	return (Contains(c.Variation, "Reminder Text") &&
		!Contains(c.Variation, "No")) ||
		Contains(c.Variation, "No Flavor Text") || // csi
		Contains(c.Variation, "Without Flavor Text") // csi
}

func (c *Card) isDuelDecks() bool {
	return (c.Contains(" vs ")) &&
		!c.Contains("Anthology")
}

func (c *Card) isDuelDecksAnthology() bool {
	return Contains(c.Edition, "Duel Decks Anthology") &&
		(c.Contains(" vs "))
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

func (c *Card) ravnicaWeekend() (string, string) {
	num := ExtractNumber(c.Variation)
	if strings.HasPrefix(num, "A") {
		return "GRN Ravnica Weekend", num
	} else if strings.HasPrefix(num, "B") {
		return "RNA Ravnica Weekend", num
	}

	for _, guild := range []string{
		"boros", "dimir", "golgari", "izzet", "selesnya",
	} {
		if c.Contains(guild) {
			return "GRN Ravnica Weekend", prwkVariants[c.Name][guild]
		}
	}
	for _, guild := range []string{
		"azorius", "gruul", "orzhov", "rakdos", "simic",
	} {
		if c.Contains(guild) {
			return "RNA Ravnica Weekend", prw2Variants[c.Name][guild]
		}
	}
	return "", ""
}

func (c *Card) ravnicaGuidKit() string {
	if !c.Contains("Guild Kit") {
		return ""
	}

	if c.Contains("Guilds of Ravnica") || c.Contains("GRN") {
		return "GRN Guild Kit"
	}
	if c.Contains("Ravnica Allegiance") || c.Contains("RNA") {
		return "RNA Guild Kit"
	}

	for _, guild := range []string{
		"boros", "dimir", "golgari", "izzet", "selesnya",
	} {
		if c.Contains(guild) {
			return "GRN Guild Kit"
		}
	}
	for _, guild := range []string{
		"azorius", "gruul", "orzhov", "rakdos", "simic",
	} {
		if c.Contains(guild) {
			return "RNA Guild Kit"
		}
	}

	if !c.isBasicLand() {
		if len(MatchInSet(c.Name, "GK1")) > 0 {
			return "GRN Guild Kit"
		}
		if len(MatchInSet(c.Name, "GK2")) > 0 {
			return "RNA Guild Kit"
		}
	} else {
		return "Guild Kit"
	}

	return ""
}

func (c *Card) Contains(prop string) bool {
	return Contains(c.Edition, prop) || Contains(c.Variation, prop)
}

func ParseCommanderEdition(edition string) string {
	if !strings.Contains(edition, "Commander") {
		return ""
	}

	// Well-known extra tags
	perSetCommander := map[string]string{
		"Arsenal":         "Commander's Arsenal",
		"Ikoria":          "Commander 2020",
		"Zendikar Rising": "Zendikar Rising Commander",
		"Legends":         "Commander Legends",
		"Green":           "Commander Collection: Green",
		"Kaldheim":        "Kaldheim Commander",
	}
	for key, ed := range perSetCommander {
		if strings.Contains(edition, key) {
			return ed
		}
	}

	// Check Anthology, but decouple from volume 2
	if strings.Contains(edition, "Anthology") {
		for _, tag := range []string{"2018", "II", "Vol"} {
			if strings.Contains(edition, tag) {
				return "Commander Anthology Volume II"
			}
		}
		return "Commander Anthology"
	}

	// Is there a year available?
	year := ExtractYear(edition)
	if year != "" {
		return "Commander " + year
	}

	// Special fallbacks
	switch edition {
	case "Commander",
		"Commander Decks",
		"Commander Singles":
		return "Commander 2011"
	}

	return ""
}
