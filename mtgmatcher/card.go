package mtgmatcher

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
)

// Card is a generic card representation using fields defined by the MTGJSON project.
type Card struct {
	// The mtgjson unique identifier of the card
	// When used as input it can host mtgjson or scryfall id
	Id string `json:"id,omitempty"`

	// The canonical name of the card
	Name string `json:"name,omitempty"`

	// The hint or commonly know variation
	Variation string `json:"variant,omitempty"`

	// The set the card comes from, or a portion of it
	Edition string `json:"edition,omitempty"`

	// Whether the card is foil or not
	Foil bool `json:"foil,omitempty"`

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
	return fmt.Sprintf("%s [%s%s]", out, c.Edition, finish)
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
func IsToken(name string) bool {
	// Check main table first
	if backend.Tokens[name] {
		return true
	}
	switch name {
	// Custom token names
	case "A Threat to Alara: Nicol Bolas",
		"Fun Format: Pack Wars",
		"On An Adventure",
		"Pyromantic Pixels",
		"Theme: WUBRG Cards":
		return true
	// WCD extra cards
	case "Biography",
		"Blank",
		"Overview":
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
		Contains(name, "Copy"),
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

// List of editions and specific cards supported in a non-English language
func SkipLanguage(cardName, edition, language string) bool {
	card := Card{
		Name:      cardName,
		Edition:   edition,
		Variation: language,
	}
	adjustEdition(&card)
	edition = strings.ToLower(card.Edition)
	cardName = strings.ToLower(card.Name)

	switch {
	case strings.HasPrefix(edition, "30th anniversary"):
		return false
	}

	switch strings.ToLower(language) {
	case "en", "english", "":
	case "it", "italian":
		switch edition {
		case "foreign black border",
			"legends italian",
			"rinascimento",
			"the dark italian":
		default:
			return true
		}
	case "ja", "jp", "japanese":
		switch edition {
		case "chronicles japanese",
			"dominaria united japanese promo tokens",
			"fourth edition foreign black border",
			"magic premiere shop",
			"strixhaven mystical archive",
			"war of the spark",
			"war of the spark promos":
		case "ikoria: lair of behemoths":
			switch cardName {
			case "mysterious egg", "mothra's great cocoon",
				"dirge bat", "battra, dark destroyer",
				"crystalline giant", "mechagodzilla, the weapon":
			default:
				return true
			}
		case "kaldheim promos":
			if cardName != "fiendish duo" {
				return true
			}
		case "secrat lair drop",
			"url/convention promos",
			"unique and miscellaneous promos",
			"resale promos",
			"media inserts":
			// No specific card because these are a evolving sets,
			// with new cards added every now and then
		default:
			return true
		}
	case "zhs", "zh-CN", "chinese", "simplified chinese", "chinese simplified":
		switch edition {
		case "simplified chinese alternate art cards":
			if !HasChineseAltArtPrinting(cardName) {
				return true
			}
		default:
			return true
		}
	default:
		return true
	}
	return false
}

func (c *Card) isUnsupported() bool {
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
		c.Contains("Sealed") ||
		c.Contains("Un-Known Event Playtest") ||
		c.Contains("Charlie Brown") || // abu
		// Oversized are usually ok, but 8th and 9th ed box topper variants
		// conflict with the actual edition name, so skip them
		(c.Contains("Oversize") && (c.Contains("8th") || c.Contains("9th")))
}

func (c *Card) isSpecificUnsupported() bool {
	switch c.Name {
	case "Hero of Bladehold",
		"Rampaging Baloths",
		"Feral Hydra":
		return c.isRewards()
	case "Spined Wurm":
		return Contains(c.Edition, "Starter 2000")
	case "Drudge Skeletons",
		"Emerald Medallion",
		"Forest",
		"Sapphire Medallion",
		"Serra Angel",
		"Time Elemental",
		"Winged Sliver":
		return c.Contains("Misprint")
	// SLD JPN non-unique cards
	case "Thoughtseize",
		"Plaguecrafter",
		"Doomsday",
		"Carrion Feeder",
		"Solemn Simulacrum",
		"Skullclamp",
		"Tezzeret the Seeker",
		"Phyrexian Metamorph":
		return c.Contains("Secret Lair") && c.isJPN()
	// Erraneous release information
	case "Zombify":
		return c.Contains("Game Night")
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

// Returns whether the cards is a "generic" promo, that probably needs
// further analysis to be fully categorized. Tokens are excluded.
func (c *Card) isGenericPromo() bool {
	return !c.isBaB() && !c.isPromoPack() && !c.isPrerelease() && !c.isSDCC() &&
		!c.isRetro() &&
		!c.Contains("Deckmasters") && // no real promos here, just foils
		!c.Contains("Token") && !IsToken(c.Name) &&
		(Contains(c.Variation, "Promo") || // catch-all (*not* Edition)
			c.Contains("Game Day") ||
			c.Contains("Gift Box") || // ck+scg
			(c.Contains("Promo") && c.Contains("Intro Pack")) || // scg
			c.Contains("League") ||
			c.Contains("Miscellaneous") ||
			c.Contains("Open House") || // tcg
			(c.Contains("Other") && !c.Contains("Brother")) ||
			c.Contains("Planeswalker Event") || // tcg
			c.Contains("Planeswalker Weekend") || // scg
			c.Contains("Store Challenge") || // scg
			c.Contains("Store Championship") || // ck
			c.Contains("Unique")) // mtgs
}

func (c *Card) isDCIPromo() bool {
	return c.Contains("DCI") && !c.Contains("Judge")
}

func (c *Card) isGenericAltArt() bool {
	// "Alt" includes Alternative
	return c.Contains("Alt") && c.Contains("Art")
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
		c.Variation == "Dark Frame Promo" ||
		Contains(c.Variation, "Planeswalker Stamp") ||
		(strings.HasSuffix(ExtractNumber(c.Variation), "p") && !c.Contains("30th"))
}

func (c *Card) isPlayPromo() bool {
	return c.Contains("Play Promo")
}

func (c *Card) isBorderless() bool {
	return Contains(c.Variation, "Borderless")
}

func (c *Card) isExtendedArt() bool {
	return Contains(c.Variation, "Extended")
}

func (c *Card) isShowcase() bool {
	return Contains(c.Variation, "Showcase") ||
		Contains(c.Variation, "Sketch") // binderpos
}

func (c *Card) isReskin() bool {
	return (Contains(c.Variation, "Reskin") ||
		Contains(c.Variation, "Dracula") ||
		Contains(c.Variation, "Godzilla")) &&
		// Needed to distinguish the SLD godizlla lands
		!c.isBasicLand()
}

func (c *Card) isGilded() bool {
	return Contains(c.Variation, "Gilded")
}

func (c *Card) isStepAndCompleat() bool {
	return Contains(c.Variation, "Compleat")
}

func (c *Card) isOilSlick() bool {
	return strings.Contains(strings.ToLower(c.Variation), "slick") ||
		strings.Contains(strings.ToLower(c.Edition), "slick")
}

func (c *Card) isConcept() bool {
	return Contains(c.Variation, "Concept")
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

func (c *Card) isChineseAltArt() bool {
	return (c.Contains("Chinese") || strings.Contains(c.Variation, "CS")) && c.isGenericAltArt()
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
		Contains(c.Variation, "Commander Party") || // scg
		Contains(c.Variation, "Moonlit Lands") // ck
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
	return (Contains(c.Variation, "Textless") && !Contains(c.Variation, "Lunar") && !Contains(c.Variation, "Store")) ||
		(c.Contains("Reward") && !c.isJudge())
}

func (c *Card) isMagicFest() bool {
	return c.Contains("Magic Fest") ||
		strings.Contains(c.Edition, "MFP") || // tcg collection
		strings.Contains(c.Variation, "MFP") // tcg collection
}

func (c *Card) isBaB() bool {
	return c.Contains("Buy a Box") ||
		strings.Contains(c.Variation, "BABP") || // tcg collection
		strings.Contains(c.Variation, "BIBB") || // sz
		(c.Contains("Box Promos") && // ha+sz
			!c.Contains("Xbox") && // ck+abu
			!c.Contains("Gift")) // csi
}

func (c *Card) isBundle() bool {
	return c.Contains("Bundle")
}

func (c *Card) isFoil() bool {
	return Contains(c.Variation, "Foil") && !Contains(c.Variation, "Non") && !c.isEtched()
}

func (c *Card) isEtched() bool {
	// Note this can't be just "etch" because it would catch the "sketch" cards
	return Contains(c.Variation, "Etched")
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

func (c *Card) isRetro() bool {
	return c.Contains("Retro")
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
		Contains(c.Edition, "Championship Deck") ||
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
	return strings.Contains(c.Edition, "DDA") ||
		(Contains(c.Edition, "Duel Decks") && Contains(c.Edition, "Anthology"))
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
	return c.Contains("Mystery") || c.Contains("The List") ||
		c.Contains("Planeswalker Symbol Reprints") ||
		c.Contains("Heads I Win, Tails You Lose") ||
		c.Contains("From Cute to Brute") ||
		c.Contains("They're Just Like Us")
}

func (c *Card) isThickDisplay() bool {
	return c.Contains("Display") || c.Contains("Thick")
}

func (c *Card) isPhyrexian() bool {
	return Contains(c.Variation, "Phyrexian")
}

func (c *Card) isTextured() bool {
	return Contains(c.Variation, "Textured")
}

func (c *Card) isGalaxyFoil() bool {
	return Contains(c.Variation, "Galaxy")
}

func (c *Card) isSurgeFoil() bool {
	return strings.Contains(strings.ToLower(c.Variation), "surge") ||
		strings.Contains(strings.ToLower(c.Edition), "surge")
}

func (c *Card) isSerialized() bool {
	return strings.Contains(strings.ToLower(c.Variation), "serial") ||
		strings.Contains(strings.ToLower(c.Edition), "serial")
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
	if strings.HasPrefix(num, "a") {
		return "GRN Ravnica Weekend", num
	} else if strings.HasPrefix(num, "b") {
		return "RNA Ravnica Weekend", num
	}

	for _, guild := range GRNGuilds {
		if c.Contains(guild) {
			return "GRN Ravnica Weekend", prwkVariants[c.Name][guild]
		}
	}
	for _, guild := range ARNGuilds {
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

	for _, guild := range GRNGuilds {
		if c.Contains(guild) {
			return "GRN Guild Kit"
		}
	}
	for _, guild := range ARNGuilds {
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

	// Append a custom display tag to avoid including the main set during filtering
	if strings.Contains(edition, "Display") || strings.Contains(edition, "Thick") ||
		strings.Contains(variant, "Display") || strings.Contains(variant, "Thick") {
		return edition + " Display"
	}

	// Legends series
	if strings.Contains(edition, "Legends") {
		if edition == "Commander Legends" {
			return "Commander Legends"
		} else if strings.Contains(edition, "Baldur's Gate") {
			edition = "Commander Legends: Battle for Baldur's Gate"
			return edition
		}
	}

	// Well-known extra tags
	perSetCommander := map[string]string{
		"Launch":      "Commander 2011 Launch Party",
		"Arsenal":     "Commander's Arsenal",
		"Ikoria":      "Commander 2020",
		"Strixhaven":  "Commander 2021",
		"Heads I Win": "Heads I Win, Tails You Lose",
		"Starter":     "Starter Commander Decks",
	}
	for key, ed := range perSetCommander {
		if strings.Contains(edition, key) {
			return ed
		}
	}
	for key, ed := range backend.CommanderKeywordMap {
		if strings.Contains(edition, key) {
			if strings.Contains(edition, "Promo") || strings.Contains(variant, "Promo") {
				ed += " Promos"
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
