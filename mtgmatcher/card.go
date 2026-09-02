package mtgmatcher

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

// InputCard is a card as a storefront described it: a name plus whatever
// edition and variation text they published, which the matcher resolves to one
// printing. Field names follow the MTGJSON project.
type InputCard struct {
	// The unique identifier of the card
	// When used as input it can host or scryfall id
	ID string `json:"id,omitempty"`

	// The canonical name of the card
	Name string `json:"name,omitempty"`

	// The hint or commonly know variation
	Variation string `json:"variant,omitempty"`

	// The set the card comes from, or a portion of it
	Edition string `json:"edition,omitempty"`

	// Whether the card is foil or not
	Foil bool `json:"foil,omitempty"`

	// The finish being priced, spelled the way the source spells it
	// ("Normal", "Cold Foil", "Holofoil", …): the game's rules place the
	// name (GameRules.CanonicalFinish), so a caller pricing one printing in
	// one finish can name the finish instead of choosing between the two the
	// Foil flag has a bit for. It refines the flag rather than replacing it,
	// and saying nothing leaves the flag to answer alone.
	Finish string `json:"finish,omitempty"`

	// The card belongs to the extended side of the set, usually containing
	// variants with the same name of existing cards in the same set, but with
	// different frames or border effects
	// Internal matcher state, not part of the serialized input.
	BeyondBaseSet bool `json:"-"`

	// In case edition information is not accurate, use this flag to
	// perform a best-effort search, which will try to isolate promo
	// printings from the others
	PromoWildcard bool `json:"PromoWildcard,omitempty"`

	// In case card got renamed in some way, this contains the original
	// card name, instead of the sanitized version
	// Internal matcher state, not part of the serialized input.
	OriginalName string `json:"-"`

	// The language as parsed
	Language string `json:"language,omitempty"`
}

// Card implements the Stringer interface
func (c *InputCard) String() string {
	name := c.Name
	edition := c.Edition

	if name == "" {
		co, err := GetUUID(c.ID)
		if err == nil {
			name = co.Name
			edition = co.Edition
		}
	}

	if c.Variation != "" {
		name = fmt.Sprintf("%s ('%s')", name, c.Variation)
	}
	finish := ""
	if c.IsEtched() {
		finish = " (etched)"
	} else if c.Foil {
		finish = " (foil)"
	}
	lang := ""
	if c.Language != "" && c.Language != "English" {
		lang = " {" + c.Language + "}"
	}
	return fmt.Sprintf("%s [%s%s]%s", name, edition, finish, lang)
}

// AddToVariant appends a tag to the variation, keeping what is already there
// and separating with a space.
func (c *InputCard) AddToVariant(tag string) {
	if c.Variation != "" {
		c.Variation += " "
	}
	c.Variation += tag
}

// IsToken reports whether the name may represent a token.
func IsToken(name string) bool {
	return defaultBackend.IsToken(name)
}

// The Is* predicates below read the free text a storefront published, not the
// datastore: they decide what a listing claims, and the rules then decide
// whether a printing can honour the claim.
//
// Two spellings appear throughout and they differ in reach. c.Contains(x)
// looks in both Edition and Variation, while Contains(c.Variation, x) looks
// only at the variation, which matters when an edition name would otherwise
// match every card in it. Both fold case and punctuation, and a few
// predicates need the raw strings.Contains instead, and say so where they do.
//
// The vendor abbreviations in the clauses name the storefront whose wording
// forced that clause: each is a real listing someone published.

// IsUnsupported reports whether the listing is for something with no printing
// at all behind it: art cards, memorabilia, misprint lots, sealed product and
// the like.
func (c *InputCard) IsUnsupported() bool {
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

// IsSpecificUnsupported reports the named cards unsupported only in one
// edition or one misprint, rather than as a whole class.
func (c *InputCard) IsSpecificUnsupported() bool {
	switch c.Name {
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
	// Erroneous release information
	case "Zombify":
		return c.Contains("Game Night")
	default:
		if Contains(c.Name, "Sticker Sheet") {
			return true
		}
	}
	return false
}

// IsBasicLand reports whether the name may represent a basic land.
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

// IsBasicLand reports whether the name is a basic land.
func (c *InputCard) IsBasicLand() bool {
	return IsBasicLand(c.Name)
}

// More specific version of the above, for internal use only
func (c *InputCard) isBasicLand() bool {
	switch c.Name {
	case "Plains", "Island", "Swamp", "Mountain", "Forest", "Wastes":
		return true
	}
	return false
}

// IsGenericPromo reports a promo with no more specific kind, one that
// probably needs further analysis to categorize: it excludes every promo the
// other predicates recognise, and tokens, then accepts the leftovers that say
// Promo or name a store event.
func (c *InputCard) IsGenericPromo() bool {
	return !c.IsBaB() && !c.IsPromoPack() && !c.IsPrerelease() && !c.IsSDCC() &&
		!c.IsRetro() &&
		!c.Contains("Year of the") && // tcg
		!c.Contains("Deckmasters") && // no real promos here, just foils
		!c.Contains("Token") && !IsToken(c.Name) &&
		(Contains(c.Variation, "Promo") || // catch-all (*not* Edition)
			c.Contains("Gift Box") || // ck+scg
			(c.Contains("Promo") && c.Contains("Intro Pack")) || // scg
			c.Contains("League") ||
			c.Contains("Play Draft") || // scg
			c.Contains("Miscellaneous") ||
			c.Contains("Open House") || // tcg
			(c.Contains("Other") && !c.Contains("Brother")) ||
			c.Contains("Planeswalker Event") || // tcg
			c.Contains("Planeswalker Weekend") || // scg
			c.Contains("Store Challenge") || // scg
			c.Contains("Unique")) // mtgs
}

// IsDCIPromo reports a DCI promo, excluding judge rewards, which carry the
// same mark.
func (c *InputCard) IsDCIPromo() bool {
	return c.Contains("DCI") && !c.Contains("Judge")
}

// IsGenericAltArt reports alternate art, matching Alternative as well as Alt.
func (c *InputCard) IsGenericAltArt() bool {
	// "Alt" includes Alternative
	return c.Contains("Alt") && c.Contains("Art")
}

// IsGenericExtendedArt reports extended or full art, from the variation only.
func (c *InputCard) IsGenericExtendedArt() bool {
	return Contains(c.Variation, "Art") &&
		(Contains(c.Variation, "Extended") ||
			Contains(c.Variation, "Full"))
}

// IsPrerelease reports a prerelease printing; SCG spells it Preview.
func (c *InputCard) IsPrerelease() bool {
	return c.Contains("Prerelease") ||
		c.Contains("Preview") // scg
}

// IsPromoPack reports a promo pack printing, by name, by the stamp it carries,
// or by a collector number ending in p, which the 30th Anniversary numbers
// reuse for something else.
func (c *InputCard) IsPromoPack() bool {
	return c.Contains("Promo Pack") ||
		c.Variation == "Dark Frame Promo" ||
		Contains(c.Variation, "Planeswalker Stamp") ||
		Contains(c.Variation, "Silver Stamped") ||
		(strings.HasSuffix(ExtractNumber(c.Variation), "p") && !c.Contains("30th"))
}

// IsBorderless reports a borderless printing, from the variation only.
func (c *InputCard) IsBorderless() bool {
	return Contains(c.Variation, "Borderless")
}

// IsExtendedArt reports extended art, from the variation only.
func (c *InputCard) IsExtendedArt() bool {
	return Contains(c.Variation, "Extended")
}

// IsShowcase reports a showcase frame; binderpos storefronts say Sketch.
func (c *InputCard) IsShowcase() bool {
	return Contains(c.Variation, "Showcase") ||
		Contains(c.Variation, "Sketch") // binderpos
}

// IsReskin reports a reskinned printing, named outright or by its Dracula or
// Godzilla series. Basic lands are excluded so the Secret Lair Godzilla lands
// stay ordinary lands.
func (c *InputCard) IsReskin() bool {
	return (Contains(c.Variation, "Reskin") ||
		Contains(c.Variation, "Dracula") ||
		Contains(c.Variation, "Godzilla")) &&
		// Needed to distinguish the SLD godzilla lands
		!c.isBasicLand()
}

// IsStepAndCompleat reports the step-and-compleat foiling.
func (c *InputCard) IsStepAndCompleat() bool {
	return Contains(c.Variation, "Compleat")
}

// IsOilSlick reports the oil slick foiling, in either field.
func (c *InputCard) IsOilSlick() bool {
	return strings.Contains(strings.ToLower(c.Variation), "slick") ||
		strings.Contains(strings.ToLower(c.Edition), "slick")
}

// IsFNM reports a Friday Night Magic promo, abbreviated or spelled out.
func (c *InputCard) IsFNM() bool {
	return c.Contains("FNM") ||
		c.Contains("Friday Night Magic")
}

// IsJPN reports a Japanese printing, by language or by the magazines that
// carried them, Gotta and Dengeki.
func (c *InputCard) IsJPN() bool {
	return strings.Contains(c.Variation, "JPN") ||
		strings.Contains(c.Variation, "JP") ||
		c.Contains("Japanese") ||
		Contains(c.Variation, "Gotta") ||
		Contains(c.Variation, "Dengeki")
}

// IsChineseAltArt reports the Chinese alternate art printings.
func (c *InputCard) IsChineseAltArt() bool {
	return (c.Contains("Chinese") || strings.Contains(c.Variation, "CS")) && c.IsGenericAltArt()
}

// IsRelease reports a release or launch promo, and refuses a prerelease,
// whose wording otherwise contains this one.
func (c *InputCard) IsRelease() bool {
	return !c.Contains("Prerelease") &&
		(c.Contains("Release") ||
			c.Contains("Draft Weekend") ||
			c.Contains("Launch"))
}

// IsWPNGateway reports a Wizards Play Network or Gateway promo, including the
// Commander Party and Moonlit Lands series that ran under it.
func (c *InputCard) IsWPNGateway() bool {
	return c.Contains("WPN") ||
		c.Contains("Gateway") ||
		Contains(c.Variation, "Wizards Play Network") ||
		Contains(c.Variation, "Commander Party") || // scg
		Contains(c.Variation, "Moonlit Lands") // ck
}

// IsIDWMagazineBook reports a promo that came with print media: comics,
// magazines, novels, and the retail tie-ins storefronts file alongside them.
func (c *InputCard) IsIDWMagazineBook() bool {
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
		Contains(c.Edition, "CardZ") || // mkm
		Contains(c.Edition, "Dengeki") || // mkm
		c.Variation == "Insert Foil" || // ck
		c.Contains("Beadle & Grimm Phyrexian") || // scg
		c.Contains("Stance Socks") || // scg
		c.Contains("Manga Promo") || // csi
		c.Contains("Media Promo") || // tcg
		c.Contains("Media Insert") // mm+nf
}

// IsResale reports a resale or repack promo, refusing championship cards,
// whose wording collides.
func (c *InputCard) IsResale() bool {
	return !c.Contains("Championship") && (c.Contains("Repack") || c.Contains("Store") || c.Contains("Resale"))
}

// IsJudge reports a judge reward.
func (c *InputCard) IsJudge() bool {
	return c.Contains("Judge")
}

// IsRewards reports a player rewards promo: the textless ones, minus the
// unrelated series that are also textless, or anything else calling itself a
// reward that is not a judge card.
func (c *InputCard) IsRewards() bool {
	return (Contains(c.Variation, "Textless") &&
		!Contains(c.Variation, "Year of") &&
		!Contains(c.Variation, "Lunar") &&
		!Contains(c.Variation, "Store")) ||
		(c.Contains("Reward") && !c.IsJudge())
}

// IsMagicFest reports a MagicFest or MagicCon promo, including TCGplayer's
// MFP code.
func (c *InputCard) IsMagicFest() bool {
	return c.Contains("Magic Fest") ||
		c.Contains("MagicCon") || // scg
		strings.Contains(c.Edition, "MFP") || // tcg collection
		strings.Contains(c.Variation, "MFP") // tcg collection
}

// IsBaB reports a buy-a-box promo, by name, by TCGplayer's BABP or
// Strikezone's BIBB, or by Box Promos where it is not an Xbox tie-in or a gift
// box.
func (c *InputCard) IsBaB() bool {
	return c.Contains("Buy a Box") ||
		strings.Contains(c.Variation, "BABP") || // tcg collection
		strings.Contains(c.Variation, "BIBB") || // sz
		(c.Contains("Box Promos") && // ha+sz
			!c.Contains("Xbox") && // ck+abu
			!c.Contains("Gift")) // csi
}

// IsBundle reports a bundle promo.
func (c *InputCard) IsBundle() bool {
	return c.Contains("Bundle")
}

// IsFoil reports a foil printing from the variation, refusing Non-Foil and
// leaving etched to IsEtched.
func (c *InputCard) IsFoil() bool {
	return Contains(c.Variation, "Foil") && !Contains(c.Variation, "Non") && !c.IsEtched()
}

// IsEtched reports etched foiling. It matches the whole word because the stem
// would also catch Sketch.
func (c *InputCard) IsEtched() bool {
	// Note this can't be just "etch" because it would catch the "sketch" cards
	return Contains(c.Variation, "Etched")
}

// IsARNLightMana reports the light mana symbol variant of Arabian Nights,
// which some storefronts mark with a dagger instead of a word.
func (c *InputCard) IsARNLightMana() bool {
	return Contains(c.Variation, "light") || strings.Contains(c.Variation, "†")
}

// IsARNDarkMana reports the dark mana symbol variant of Arabian Nights.
func (c *InputCard) IsARNDarkMana() bool {
	return Contains(c.Variation, "dark")
}

// IsArena reports an Arena league promo.
func (c *InputCard) IsArena() bool {
	return c.Contains("Arena")
}

// IsSDCC reports a San Diego Comic-Con promo.
func (c *InputCard) IsSDCC() bool {
	return c.Contains("SDCC") ||
		c.Contains("San Diego Comic-Con")
}

// IsRetro reports a retro frame printing.
func (c *InputCard) IsRetro() bool {
	return c.Contains("Retro")
}

// PlayerRewardsYear returns the year of a player rewards printing, falling
// back to the set or artist named in the variation when the listing gives no
// year of its own.
func (c *InputCard) PlayerRewardsYear(maybeYear string) string {
	if maybeYear == "" {
		switch c.Name {
		case "Bear":
			if Contains(c.Variation, "Odyssey") {
				maybeYear = "2001"
			} else if Contains(c.Variation, "Onslaught") {
				maybeYear = "2003"
			}
		case "Beast":
			if Contains(c.Variation, "Odyssey") {
				maybeYear = "2001"
			} else if Contains(c.Variation, "Darksteel") {
				maybeYear = "2004"
			}
		case "Elephant":
			if Contains(c.Variation, "Invasion") {
				maybeYear = "2001"
			} else if Contains(c.Variation, "Odyssey") {
				maybeYear = "2002"
			}
		case "Spirit":
			if Contains(c.Variation, "Planeshift") {
				maybeYear = "2001"
			} else if Contains(c.Variation, "Champions") {
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

// ArenaYear returns the year of an Arena league printing, deducing it from the
// artist or set named in the variation when the listing gives no year.
func (c *InputCard) ArenaYear(maybeYear string) string {
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

// IsWorldChamp reports a World Championship or Pro Tour deck card, from the
// edition alone.
func (c *InputCard) IsWorldChamp() bool {
	return Contains(c.Edition, "Pro Tour Collect") ||
		Contains(c.Edition, "Pro Tour 1996") ||
		Contains(c.Edition, "World Championship") ||
		Contains(c.Edition, "Championship Deck") ||
		Contains(c.Edition, "WCD")
}

// ParseWorldChampPrefix returns the deck code for the player named in the
// text, and whether the card was in their sideboard.
func ParseWorldChampPrefix(variation string) (string, bool) {
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

// WorldChampPrefix returns the World Championship deck code for this listing,
// looking in the variation first and falling back to the edition.
func (c *InputCard) WorldChampPrefix() (string, bool) {
	prefix, sideboard := ParseWorldChampPrefix(c.Variation)
	if prefix == "" {
		return ParseWorldChampPrefix(c.Edition)
	}
	return prefix, sideboard
}

// IsDuelsOfThePW reports a Duels of the Planeswalkers promo. It compares the
// raw strings so the fold does not equate Duels with Duel Decks.
func (c *InputCard) IsDuelsOfThePW() bool {
	// XXX: do not use c.Contains here
	return strings.Contains(c.Variation, "Duels") ||
		strings.Contains(c.Edition, "Duels") ||
		Contains(c.Variation, "DotP") // tat
}

// IsBasicFullArt reports a full art basic land, refusing the negations that
// storefronts write in the same field.
func (c *InputCard) IsBasicFullArt() bool {
	return c.isBasicLand() &&
		(Contains(c.Variation, "full art") ||
			c.Variation == "FA") && // csi
		!Contains(c.Variation, "non") &&
		!Contains(c.Variation, "not") // csi
}

// IsBasicNonFullArt reports a basic land explicitly marked as not full art.
func (c *InputCard) IsBasicNonFullArt() bool {
	return c.isBasicLand() &&
		Contains(c.Variation, "non-full art") ||
		Contains(c.Variation, "Intro") || // abu
		Contains(c.Variation, "NOT the full art") // csi
}

// IsPremiereShop reports a Magic Premiere Shop basic land. It compares the raw
// strings because the folded form is too short to be safe.
func (c *InputCard) IsPremiereShop() bool {
	return c.isBasicLand() &&
		// XXX: do not use c.Contains here
		(strings.Contains(c.Variation, "MPS") ||
			strings.Contains(c.Variation, "Premier") || // csi
			strings.Contains(c.Edition, "MPS") ||
			strings.Contains(c.Edition, "Premiere Shop")) // mkm
}

// IsPortalAlt reports the Portal alternates, which differ by carrying reminder
// text or by lacking flavor text.
func (c *InputCard) IsPortalAlt() bool {
	return (Contains(c.Variation, "Reminder Text") &&
		!Contains(c.Variation, "No")) ||
		Contains(c.Variation, "No Flavor Text") || // csi
		Contains(c.Variation, "Without Flavor Text") // csi
}

// IsDuelDecks reports a Duel Decks printing, named by the two sides it pits
// against each other, and refuses the Anthology reprints.
func (c *InputCard) IsDuelDecks() bool {
	return ((c.Contains(" vs ")) ||
		(strings.Contains(c.Variation, " v. "))) && // tcg
		!c.Contains("Anthology")
}

// IsDuelDecksAnthology reports the Duel Decks Anthology reprints.
func (c *InputCard) IsDuelDecksAnthology() bool {
	return strings.Contains(c.Edition, "DDA") ||
		(Contains(c.Edition, "Duel Decks") && Contains(c.Edition, "Anthology"))
}

// DuelDecksVariant returns which half of a Duel Decks pairing the listing
// names, or an empty string if it is not a Duel Decks card.
func (c *InputCard) DuelDecksVariant() string {
	if !c.IsDuelDecks() {
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

// IsMysteryList reports a Mystery Booster or The List printing. The List is
// matched raw, since folded it also matches The Little.
func (c *InputCard) IsMysteryList() bool {
	return c.Contains("Mystery") || c.Contains("Planeswalker Symbol Reprints") ||
		// Cannot use c.Contains because it trips with "The Little"
		strings.Contains(c.Edition, "The List") || strings.Contains(c.Variation, "The List")
}

// IsSecretLair reports a Secret Lair printing, by name or by set code.
func (c *InputCard) IsSecretLair() bool {
	return c.Contains("Secret Lair") || strings.Contains(c.Edition, "SLD")
}

// HasSecretLairTag reports whether the listing belongs to the given Secret
// Lair set, which each need their own rule: the drops differ in what they
// reprint and in how storefronts spell them.
func (c *InputCard) HasSecretLairTag(code string) bool {
	var tag bool
	switch code {
	case "SLU":
		// SLU is mostly static and cards are unlikely to reappear elsewhere
		tag = c.Contains("Ultimate") || len(MatchInSet(c.Name, "SLU")) == 1
	case "SLX":
		// SLX only has plain cards, if they are reskinned, they are from SLD
		tag = !c.IsReskin() || c.Contains("Within") || c.Contains("SLX")
	case "SLC":
		// Some of these cards are numbered after the year they represent.
		// The same numbers double as plain collector numbers in SLD (e.g.
		// Sol Ring is SLD #1993), so only treat the year as an SLC signal
		// when that exact card actually exists in SLC at that number.
		yearStr := ExtractYear(c.Variation)
		tag = c.Contains("30th") || c.Contains("Countdown") ||
			(yearStr != "" && len(MatchInSetNumber(c.Name, "SLC", yearStr)) > 0)
	case "SLP":
		// Simple check the variations
		tag = c.Contains("Showdown") || c.Contains("Prize") || c.Contains("Finish") || c.Contains("Play")
	}

	return c.IsSecretLair() && tag
}

// IsThickDisplay reports the thick display commander cards.
func (c *InputCard) IsThickDisplay() bool {
	return c.Contains("Display") || c.Contains("Thick")
}

// IsPhyrexian reports a Phyrexian language printing.
func (c *InputCard) IsPhyrexian() bool {
	return Contains(c.Variation, "Phyrexian")
}

// IsGalaxyFoil reports the galaxy foiling.
func (c *InputCard) IsGalaxyFoil() bool {
	return Contains(c.Variation, "Galaxy")
}

// IsSurgeFoil reports the surge foiling, in either field.
func (c *InputCard) IsSurgeFoil() bool {
	return strings.Contains(strings.ToLower(c.Variation), "surge") ||
		strings.Contains(strings.ToLower(c.Edition), "surge")
}

// IsSerialized reports a serialized printing, in either field.
func (c *InputCard) IsSerialized() bool {
	return strings.Contains(strings.ToLower(c.Variation), "serial") ||
		strings.Contains(strings.ToLower(c.Edition), "serial")
}

// PossibleNumberSuffix returns a lone letter from the variation, lowercased,
// which is how storefronts often carry the suffix of a collector number.
func (c *InputCard) PossibleNumberSuffix() string {
	fields := strings.FieldsSeq(c.Variation)
	for field := range fields {
		if len(field) == 1 && unicode.IsLetter(rune(field[0])) {
			return strings.ToLower(field)
		}
	}
	return ""
}

// RavnicaGuildKit returns which Guild Kit the listing names, by set name or set
// code, or an empty string if it names none.
func (c *InputCard) RavnicaGuildKit() string {
	if !c.Contains("Guild Kit") {
		return ""
	}

	if c.Contains("Guilds of Ravnica") || c.Contains("GRN") {
		return "GRN Guild Kit"
	}
	if c.Contains("Ravnica Allegiance") || c.Contains("RNA") {
		return "RNA Guild Kit"
	}

	if slices.ContainsFunc(GRNGuilds, c.Contains) {
		return "GRN Guild Kit"
	}
	if slices.ContainsFunc(ARNGuilds, c.Contains) {
		return "RNA Guild Kit"
	}

	if c.isBasicLand() {
		return "Guild Kit"
	}
	if len(MatchInSet(c.Name, "GK1")) > 0 {
		return "GRN Guild Kit"
	}
	if len(MatchInSet(c.Name, "GK2")) > 0 {
		return "RNA Guild Kit"
	}

	return ""
}

// Contains reports whether either the edition or the variation contains the
// property, ignoring case and punctuation. Prefer Contains(c.Variation, prop)
// where an edition name would match every card printed in it.
func (c *InputCard) Contains(prop string) bool {
	return Contains(c.Edition, prop) || Contains(c.Variation, prop)
}

// Equals reports whether either the edition or the variation is exactly the
// property, ignoring case and punctuation.
func (c *InputCard) Equals(prop string) bool {
	return Equals(c.Edition, prop) || Equals(c.Variation, prop)
}

// ParseCommanderEdition returns the Commander edition the text names, using
// the default datastore.
func ParseCommanderEdition(edition, variant string) string {
	return defaultBackend.ParseCommanderEdition(edition, variant)
}

// ShouldIgnoreNumber reports whether the collector number, where one was
// given, is too unreliable to narrow with: some storefronts publish a number
// that belongs to a different printing of the same set.
func (c *InputCard) ShouldIgnoreNumber(setName, num string) bool {
	// No misprints or WCD
	if c.Contains("Misprint") || c.IsWorldChamp() {
		return true
	}

	// This is better handled in thelistCheck()
	if c.IsMysteryList() && !c.Contains("Unfinity") {
		return true
	}

	// Unfinity numbers could refer to Attractions
	if Contains(c.Edition, "unf") {
		if HasPrinting(c.Name, "field", "attractionLights", "UNF") && (strings.Contains(c.Variation, "/") || strings.Contains(c.Variation, "-")) {
			return true
		}
	}

	// If the number is the same as in the edition, there might be
	// variation pollution, therefore unreliable (unless they are years)
	if num != "" && strings.Contains(setName, num) && ExtractYear(setName) == "" {
		return true
	}

	return false

}

// IsToken reports whether the name is a token in this datastore. The names a
// game knows as tokens without carrying a token type of their own are its own
// business, so the game's rules are asked as well.
func (b *Backend) IsToken(name string) bool {
	if slices.Contains(b.Tokens, name) {
		return true
	}
	if b.rules == nil {
		return false
	}
	return b.rules.IsToken(b, name)
}

// ParseCommanderEdition returns the Commander edition the text names, or an
// empty string when it names none.
func (b *Backend) ParseCommanderEdition(edition, variant string) string {
	if !strings.Contains(edition, "Commander") {
		return ""
	}

	// An edition already naming a carried token set is exact: parsing it
	// down to the commander set it stems from would lose the tokens
	if strings.Contains(strings.ToLower(edition), "token") {
		_, found := b.NormalizedSets[Normalize(edition)]
		if found {
			return ""
		}
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
	// Double Strixhaven
	if strings.Contains(edition, "Strixhaven") {
		if strings.Contains(edition, "Secret") {
			return "Secrets of Strixhaven Commander"
		}
		return "Commander 2021"
	}

	// Well-known extra tags
	perSetCommander := map[string]string{
		"Launch":  "Commander 2011 Launch Party",
		"Arsenal": "Commander's Arsenal",
		"Ikoria":  "Commander 2020",
		"Starter": "Starter Commander Decks",
	}
	for key, ed := range perSetCommander {
		if strings.Contains(edition, key) {
			return ed
		}
	}
	for key, ed := range b.CommanderKeywordMap {
		if strings.Contains(strings.ToLower(edition), strings.ToLower(key)) {
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

func (b *Backend) output(card Card, flags ...bool) string {
	hasNonfoil := card.HasFinish(FinishNonfoil)
	hasFoil := card.HasFinish(FinishFoil)
	hasEtched := card.HasFinish(FinishEtched)

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

	// Resolve to the finish the caller is asking for and pull the uuid the
	// loader registered for it. Loaders register every finish a card carries
	// (and, for Lorcana, every foil sub-type), so this is the common path.
	finish := FinishNonfoil
	if etched {
		finish = FinishEtched
	} else if foil {
		finish = FinishFoil
	}
	if id, ok := card.FoilUUIDs[finish]; ok {
		return id
	}

	// Fall back to the suffix rules for cards without a registered map.
	id := card.UUID
	// Append suffixes to the Id to distinguish cards among finishes
	if etched && (hasNonfoil || hasFoil) {
		// Retrieve the base id if it's already tagged (only for this and the case below)
		if strings.HasSuffix(id, suffixFoil) || strings.HasSuffix(id, suffixEtched) {
			id = id[:len(id)-2]
		}
		id += suffixEtched
	} else if foil && hasNonfoil {
		if strings.HasSuffix(id, suffixFoil) || strings.HasSuffix(id, suffixEtched) {
			id = id[:len(id)-2]
		}
		id += suffixFoil
	}
	return id
}

// hasOversizedPrinting reports whether the datastore carries an oversized
// printing of the card. A listing that says oversize is asking for one, and
// the sheets that were never built - the championship prizes, the oversized
// dungeons - hold nothing it could mean, so the word marks it unsupported.
//
// The sets that carry them used to be named by the words in their titles,
// which read "Commander Legends: Battle for Baldur's Gate" as an oversized
// Commander product and priced its dungeon as the ordinary token. The edition
// is the wrong thing to ask: a storefront writes "Magic Player Rewards" for a
// card filed under "Magic Player Rewards 2009", and a set can hold one
// oversized printing among ordinary ones. Ask whether the card has such a
// printing at all, and leave which one to the edition filter below.
func (b *Backend) hasOversizedPrinting(name string) bool {
	printings, err := b.Printings4Card(name)
	if err != nil {
		return false
	}
	for _, code := range printings {
		set, found := b.Sets[code]
		if !found {
			continue
		}
		for i := range set.Cards {
			if set.Cards[i].IsOversized && Equals(set.Cards[i].Name, name) {
				return true
			}
		}
	}
	return false
}
