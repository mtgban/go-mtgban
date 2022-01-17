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

	// The card belongs to the extended side of the set, usually containing
	// variants with the same name of existing cards in the same set, but with
	// different frames or border effects
	beyondBaseSet bool

	// In case edition information is not accurate, use this flag to
	// perform a best-effor search, which will try to isolate promo
	// printings from the others
	promoWildcard bool
}

// Card implements the Stringer interface
func (c *Card) String() string {
	out := c.Name
	if c.Variation != "" {
		out = fmt.Sprintf("%s (%s)", out, c.Variation)
	}
	finish := ""
	if c.isEtched() {
		finish = " (etched)"
	} else if c.Foil {
		finish = " (foil)"
	}
	return fmt.Sprintf("%s [%s%s] {%s}", out, c.Edition, finish, c.Number)
}

func output(card mtgjson.Card, flags ...bool) string {
	hasNonfoil := card.HasFinish(mtgjson.FinishNonfoil)
	hasFoil := card.HasFinish(mtgjson.FinishFoil)
	hasEtched := card.HasFinish(mtgjson.FinishEtched)

	etched := len(flags) > 1 && flags[1]
	foil := len(flags) > 0 && flags[0] && !etched

	// In case the foiling information is incorrect
	if !foil && !hasNonfoil && !hasEtched {
		foil = true
	} else if foil && !hasFoil {
		foil = false
	}
	if hasFoil && !hasNonfoil && !hasEtched {
		foil = true
	} else if !hasFoil && (hasNonfoil || hasEtched) {
		foil = false
	}

	// In case the etching information is incorrect
	if !etched && !hasNonfoil && !hasFoil {
		etched = true
	} else if etched && !hasEtched {
		etched = false
	}
	if hasEtched && !hasNonfoil && !hasFoil {
		etched = true
	} else if !hasEtched && (hasNonfoil || hasFoil) {
		etched = false
	}

	// Prepare the output card
	id := card.UUID
	// Append suffixes to the Id to distinguish cards among finishes
	if etched && (hasNonfoil || hasFoil) {
		id += suffixEtched
	} else if foil && hasNonfoil {
		id += suffixFoil
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
func isToken(name string) bool {
	switch name {
	// Known token names
	case "A Threat to Alara: Nicol Bolas",
		"Acorn Stash",
		"Angel",
		"Ashaya, the Awoken World",
		"Bat",
		"Bear",
		"Bird",
		"Cat Dragon",
		"Cat Warrior",
		"Cat",
		"City's Blessing",
		"Cleric",
		"Clue",
		"Companion",
		"Construct",
		"Day // Night",
		"Demon",
		"Dinosaur",
		"Drake",
		"Eldrazi Spawn",
		"Elemental Shaman",
		"Elemental",
		"Elephant",
		"Elf Warrior",
		"Energy Reserve",
		"Faerie Rogue",
		"Faerie",
		"Food",
		"Foretell",
		"Fun Format: Pack Wars",
		"Germ",
		"Giant",
		"Gold",
		"Golem",
		"Human",
		"Human Cleric",
		"Human Rogue",
		"Human Soldier",
		"Human Warrior",
		"Insect",
		"Karox Bladewing",
		"Knight",
		"Kraken",
		"Manifest",
		"Marit Lage",
		"Merfolk",
		"Minion",
		"Minotaur",
		"Morph",
		"Mouse",
		"Myr",
		"Nightmare Horror",
		"On Your Turn",
		"On an Adventure",
		"Ooze",
		"Pirate",
		"Plant",
		"Poison Counter",
		"Pyromantic Pixels",
		"Rat",
		"Saproling",
		"Shapeshifter",
		"Sliver",
		"Snake",
		"Thopter",
		"Thrull",
		"Treasure",
		"Vampire",
		"Walker",
		"Wolf",
		"Wurm",
		"Zombie Knight",
		"Theme: WUBRG Cards":
		return true
	// WCD extra cards
	case "Biography",
		"Blank",
		"Overview":
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
		HasPrefix(name, "Leering Emblem"),
		// and with the `card` wildcard
		HasPrefix(name, "Our Market Research"):
		return false
	// Anything token
	case strings.Contains(name, " Card"),
		strings.Contains(name, "Card "),
		Contains(name, "Arena Code"),
		Contains(name, "Art Series"),
		Contains(name, "Charlie Brown"),
		Contains(name, "Checklist"),
		Contains(name, "Decklist"),
		Contains(name, "DFC Helper"),
		Contains(name, "Dungeon of the Mad Mage"),
		Contains(name, "Emblem"),
		Contains(name, "Experience C"),
		Contains(name, "Giant Teddy Bear"),
		Contains(name, "Guild Symbol"),
		Contains(name, "Magic Minigame"),
		Contains(name, "Monarch"),
		Contains(name, "Morph Overlay"),
		Contains(name, "On Your Turn"),
		Contains(name, "Online Code"),
		Contains(name, "Oversize"),
		Contains(name, "Punch Out"),
		Contains(name, "Token"),
		Contains(name, "Rules Tip"):
		return true
	// Alternative rules tip card names found on mkm
	case strings.HasPrefix(name, "Build a Deck: "),
		strings.HasPrefix(name, "Tip: "):
		return true
	}

	return false
}

func (c *Card) isUnsupported() bool {
	return c.Contains("Art Series") ||
		c.IsSurrounded("Complete", "Set") || // a complete collection
		c.Contains("Fallen Empires: Wyvern Misprints") ||
		c.Contains("Simplified Chinese Alternate Art Cards") ||
		c.Contains("Ultra-Pro Puzzle") ||
		c.Contains("Player Cards") || // scg pro players
		c.Contains("Foreign White Border") || // for REV and 4ED
		c.Contains("4th Edition - Alternate") ||
		c.Contains("Alternate 4th Edition") ||
		c.Contains("Fourth Edition: Alternate") ||
		c.Contains("Fourth (Alternate Edition)") ||
		c.Contains("Fourth Edition (Alt)") ||
		c.Contains("Filler Cards") || // Misprints from mkm and ct
		c.Contains("Salvat") || // Salvat-Hachette 2005/2011
		c.Contains("Redemption Program") || // PRES
		c.Contains("Heroes of the Realm") || // HTR*
		c.Contains("Memorabilia") ||
		c.Contains("Sealed") ||
		// Oversized are usually ok, but 8th and 9th ed box topper variants
		// conflict with the actual edition name, so skip them
		(c.Contains("Oversize") && (c.Contains("8th") || c.Contains("9th")))
}

func (c *Card) isSpecificUnsupported() bool {
	switch c.Name {
	case "Squire":
		return strings.Contains(c.Edition, "Secret Lair")
	case "Hero of Bladehold",
		"Rampaging Baloths",
		"Feral Hydra":
		return c.isRewards()
	case "Spined Wurm":
		return Contains(c.Edition, "Starter 2000")
	case "Drudge Skeletons",
		"Sapphire Medallion",
		"Thunderheads",
		"Winged Sliver":
		return c.Contains("Misprint")
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
		strings.Contains(name, "Fish"), strings.Contains(name, "Sanctuary"), // U
		strings.Contains(name, "Wak-Wak"): // U
	case strings.HasPrefix(name, "Plains"),
		strings.HasPrefix(name, "Island"),
		strings.HasPrefix(name, "Swamp"),
		strings.HasPrefix(name, "Mountain"),
		strings.HasPrefix(name, "Forest"),
		strings.HasPrefix(name, "Wastes"):
		return true
	case HasPrefix(name, "Snow-Covered"):
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
	return !c.isBaB() && !c.isPromoPack() && !c.isPrerelease() &&
		(c.Contains("Promo") || c.Contains("Game Day") ||
			c.Contains("Store Challenge") || // scg
			c.Contains("Store Championship")) // ck
}

func (c *Card) isDCIPromo() bool {
	return c.Contains("DCI") && !c.Contains("Judge")
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
	return c.Contains("Prerelease") ||
		c.Contains("Preview") // scg

}

func (c *Card) isPromoPack() bool {
	return c.Contains("Promo Pack") ||
		c.Contains("Play Promo") ||
		c.Variation == "Dark Frame Promo" ||
		Contains(c.Variation, "Planeswalker Stamp")
}

func (c *Card) isBorderless() bool {
	return Contains(c.Variation, "Borderless")
}

func (c *Card) isExtendedArt() bool {
	return Contains(c.Variation, "Extended")
}

func (c *Card) isShowcase() bool {
	return Contains(c.Variation, "Showcase") ||
		Contains(c.Variation, "Eternal Night") ||
		Contains(c.Variation, "Sketch") // binderpos
}

func (c *Card) isReskin() bool {
	return (Contains(c.Variation, "Reskin") ||
		Contains(c.Variation, "Dracula") ||
		Contains(c.Variation, "Godzilla")) &&
		// Needed to distinguish the SLD godizlla lands
		!c.isBasicLand()
}

func (c *Card) isFNM() bool {
	return c.Contains("FNM") ||
		c.Contains("Friday Night Magic")
}

func (c *Card) isJPN() bool {
	return strings.Contains(c.Variation, "JPN") ||
		strings.Contains(c.Variation, "JP") ||
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
		c.Variation == "Commander Party" || // scg
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
		c.Contains("Coro Coro") || // stks
		c.Contains("JP Graphic Novel") || // stks
		strings.Contains(c.Variation, "Book Promo") || // sz
		c.Contains("Top Deck") || // csi
		Contains(c.Edition, "CardZ") || // mkm
		Contains(c.Edition, "Dengeki") || // mkm
		c.Variation == "Insert Foil" || // ck
		c.Contains("Media Insert") // mm+nf
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
		(c.Contains("Box Promos") && // ha+sz
			!c.Contains("Xbox") && // ck+abu
			!c.Contains("Gift")) // csi
}

func (c *Card) isBundle() bool {
	return c.Contains("Bundle")
}

func (c *Card) isEtched() bool {
	// Note this can't be just "etch" because it would catch the "sketch" cards
	return Contains(c.Variation, "Etched") ||
		Contains(c.Variation, "Etching") // ha
}

func (c *Card) isARNLightMana() bool {
	return Contains(c.Variation, "light") || strings.Contains(c.Variation, "†")
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

func (c *Card) playerRewardsYear(maybeYear string) string {
	if maybeYear == "" {
		switch c.Name {
		case "Bear":
			if c.Variation == "Odyssey" {
				maybeYear = "2001"
			} else if c.Variation == "Onslaught" {
				maybeYear = "2003"
			}
		case "Beast":
			if c.Variation == "Odyssey" {
				maybeYear = "2001"
			} else if c.Variation == "Darksteel" {
				maybeYear = "2004"
			}
		case "Elephant":
			if c.Variation == "Invasion" {
				maybeYear = "2001"
			} else if c.Variation == "Odyssey" {
				maybeYear = "2002"
			}
		case "Spirit":
			if c.Variation == "Planeshift" {
				maybeYear = "2001"
			} else if c.Variation == "Champions" {
				maybeYear = "2004"
			}
		}
	}
	return maybeYear
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

func parseWorldChampPrefix(variation string) (string, bool) {
	players := map[string]string{
		"Aeo Paquette":         "ap",
		"Alex Borteh":          "ab",
		"Antoine Ruel":         "ar",
		"Ben Rubin":            "br",
		"Bertrand Lestree":     "bl",
		"Brian Hacker":         "bh",
		"Brian Kibler":         "bk",
		"Brian Selden":         "bs",
		"Brian Seldon":         "bs",
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

	// We cannot use HasPrefix for the second check due to mlp/ml aliasing
	variation = strings.ToLower(variation)
	idx := strings.IndexFunc(variation, func(c rune) bool {
		return unicode.IsDigit(c)
	})
	// Iterate over the player list and check if their name or their initials are present
	for player, tag := range players {
		if Contains(variation, player) || (idx > -1 && variation[:idx] == tag) {
			sb := strings.Contains(variation, "sb") || strings.Contains(variation, "sideboard")
			return tag, sb
		}
	}
	return "", false
}

func (c *Card) worldChampPrefix() (string, bool) {
	prefix, sideboard := parseWorldChampPrefix(c.Variation)
	if prefix == "" {
		return parseWorldChampPrefix(c.Edition)
	}
	return prefix, sideboard
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
	return ((c.Contains(" vs ")) ||
		(strings.Contains(c.Variation, " v. "))) && // tcg
		!c.Contains("Anthology")
}

func (c *Card) isDuelDecksAnthology() bool {
	return Contains(c.Edition, "Duel Decks Anthology") &&
		(c.Contains(" vs ") ||
			strings.Contains(c.Variation, " v. ")) // tcg
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

func (c *Card) isMysteryList() bool {
	return c.Contains("Mystery") || c.Contains("The List")
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

func (c *Card) IsSurrounded(prefix, suffix string) bool {
	return (HasPrefix(c.Edition, prefix) && HasSuffix(c.Edition, prefix)) ||
		(HasPrefix(c.Variation, prefix) && HasSuffix(c.Variation, prefix))
}

func (c *Card) Contains(prop string) bool {
	return Contains(c.Edition, prop) || Contains(c.Variation, prop)
}

func (c *Card) Equals(prop string) bool {
	return Equals(c.Edition, prop) || Equals(c.Variation, prop)
}

func ParseCommanderEdition(edition, variant string) string {
	if !strings.Contains(edition, "Commander") {
		return ""
	}

	isThick := strings.Contains(edition, "Display") || strings.Contains(edition, "Thick") ||
		strings.Contains(variant, "Display") || strings.Contains(variant, "Thick")

	// Well-known extra tags
	perSetCommander := map[string]string{
		"Launch":          "Commander 2011 Launch Party",
		"Arsenal":         "Commander's Arsenal",
		"Ikoria":          "Commander 2020",
		"Zendikar Rising": "Zendikar Rising Commander",
		"Legends":         "Commander Legends",
		"Kaldheim":        "Kaldheim Commander",
		"Strixhaven":      "Commander 2021",
		"Forgotten":       "Forgotten Realms Commander",
		"Midnight":        "Midnight Hunt Commander",
		"Crimson Vow":     "Crimson Vow Commander",
	}
	for key, ed := range perSetCommander {
		if strings.Contains(edition, key) {
			if isThick {
				ed += " Display Commanders"
			}
			return ed
		}
	}

	// Collection series
	if strings.Contains(edition, "Collection") {
		for _, color := range []string{"Green", "Black"} {
			if strings.Contains(edition, color) {
				return "Commander Collection: " + color
			}
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
		parsed := "Commander " + year
		if isThick {
			parsed += " Display Commanders"
		}
		return parsed
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
