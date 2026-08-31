package yugioh

import (
	"sync"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// editionAliases maps the set names storefronts write onto the set the
// catalog files those printings under. Every entry names a set the datastore
// does not carry under that spelling, so an edition already naming a set is
// never rewritten - the lookups in AdjustEdition ask the backend first.
//
// The entries are the storefront's own decoration of a name the catalog
// spells differently: the number in a Duelist Pack's name ("Duelist Pack:
// Crow" is the catalog's "Duelist Pack 11: Crow"), the "Mega Pack" a
// Legendary Collection's cards are sold in, the "Starter" a Speed Duel deck
// picks up, and the comic the manga promos came out of. Each was checked
// against the printings in that expansion that resolve through their
// TCGplayer id, which name the set outright.
//
// A storefront that publishes collector numbers proves its own spellings:
// a Yu-Gi-Oh number's prefix is the set code ("DCR-EN021" is DCR), so the
// products a storefront files under an edition name the set they belong to.
// The entries below were graded that way against Cool Stuff Inc's buylist
// feed, and an edition whose products spread over sibling sets one name
// cannot pick - the print-run twins reissued under the same numbers - is
// deliberately absent: only the copyright date printed on the card tells
// those apart, which is the scraper's to read and not a name's to guess.
var editionAliases = map[string]string{
	"Duelist Pack: Jaden Yuki":      "Duelist Pack 1: Jaden Yuki",
	"Duelist Pack: Chazz Princeton": "Duelist Pack 2: Chazz Princeton",
	"Duelist Pack: Jaden Yuki 2":    "Duelist Pack 3: Jaden Yuki 2",
	"Duelist Pack: Zane Truesdale":  "Duelist Pack 4: Zane Truesdale",
	"Duelist Pack: Aster Phoenix":   "Duelist Pack 5: Aster Phoenix",
	"Duelist Pack: Jaden Yuki 3":    "Duelist Pack 6: Jaden Yuki 3",
	"Duelist Pack: Jesse Anderson":  "Duelist Pack 7: Jesse Anderson",
	"Duelist Pack: Yusei Fudo":      "Duelist Pack 8: Yusei Fudo",
	"Duelist Pack: Yusei Fudo 2":    "Duelist Pack 9: Yusei 2",
	"Duelist Pack: Yusei Fudo 3":    "Duelist Pack 10: Yusei 3",
	"Duelist Pack: Crow":            "Duelist Pack 11: Crow",
	"Duelist Pack 9 Yusei Fudo 2":   "Duelist Pack 9: Yusei 2",
	"Duelist Pack 10 Yusei Fudo 3":  "Duelist Pack 10: Yusei 3",

	// Every Duelist League shares the one set code, so the league's own
	// number lives in the collector number and the edition only has to
	// reach the set the leagues are collected in.
	"Duelist League Promos Upperdeck": "Duelist League Promo",
	"Duelist League Series 1":         "Duelist League Promo",
	"Duelist League Series 2":         "Duelist League Promo",
	"Duelist League Series 3":         "Duelist League Promo",
	"Duelist League Series 4":         "Duelist League Promo",
	"Duelist League Series 5":         "Duelist League Promo",
	"Duelist League Series 6":         "Duelist League Promo",
	"Duelist League Series 7":         "Duelist League Promo",
	"Duelist League Series 8":         "Duelist League Promo",
	"Duelist League Series 9":         "Duelist League Promo",
	"Duelist League Series 10":        "Duelist League Promo",

	"Legendary Collection 2: Mega Pack":                   "Legendary Collection 2",
	"Legendary Collection 2: The Duel Academy Years":      "Legendary Collection 2",
	"Legendary Collection 3: Mega Pack":                   "Legendary Collection 3: Yugi's World",
	"Legendary Collection 4: Mega Pack":                   "Legendary Collection 4: Joey's World",
	"Legendary Collection 5D's: Mega Pack":                "Legendary Collection 5D's",
	"Legendary Collection Kaiba Mega Pack":                "Legendary Collection Kaiba",
	"Legendary Collection Kaiba (2025 Reprint)":           "Legendary Collection Kaiba (2020 Date Reprint)",
	"Legendary Collection Kaiba Mega Pack (2025 Reprint)": "Legendary Collection Kaiba (2020 Date Reprint)",

	"Speed Duel Starter Decks: Destiny Masters":         "Speed Duel Decks: Destiny Masters",
	"Speed Duel Starter Decks: Duelists of Tomorrow":    "Speed Duel Decks: Duelists of Tomorrow",
	"Speed Duel Starter Decks: Match of the Millennium": "Speed Duel Decks: Match of the Millennium",
	"Speed Duel Starter Decks: Twisted Nightmares":      "Speed Duel Decks: Twisted Nightmares",
	"Starter Deck: Ultimate Predators":                  "Speed Duel Decks: Ultimate Predators",
	"Speed Duel GX: Duel Academy":                       "Speed Duel GX: Duel Academy Box",
	// The same decks headed as a starter deck naming its speed duel.
	"Starter Deck: Speed Duel - Battle City Box":         "Speed Duel: Battle City Box",
	"Starter Deck: Speed Duel - Match of the Millennium": "Speed Duel Decks: Match of the Millennium",
	"Starter Deck: Speed Duel - Twisted Nightmares":      "Speed Duel Decks: Twisted Nightmares",
	"Starter Deck: Speed Duel - Ultimate Predators":      "Speed Duel Decks: Ultimate Predators",

	"5D's Manga Promos":   "Yu-Gi-Oh! 5D's Manga Promotional Cards",
	"ARC-V Manga Promos":  "Yu-Gi-Oh! ARC-V Promo Cards",
	"GX Manga Promos":     "Yu-Gi-Oh! GX Manga Promotional Cards",
	"R Comic Book Promos": "Yu-Gi-Oh! R Manga Promo",
	"ZEXAL Manga Promos":  "Yu-Gi-Oh! ZEXAL Manga Promotional Cards",

	// The subtitle or volume word the catalog spells out and the
	// storefronts drop.
	"Dark Revelation 1": "Dark Revelation Volume 1",
	"Dark Revelation 2": "Dark Revelation Volume 2",
	"Dark Revelation 3": "Dark Revelation Volume 3",
	"Dark Revelation 4": "Dark Revelation Volume 4",
	"Hidden Arsenal 5":  "Hidden Arsenal 5: Steelswarm Invasion",
	"Hidden Arsenal 6":  "Hidden Arsenal 6: Omega Xyz",
	"Hidden Arsenal 7":  "Hidden Arsenal 7: Knight of Stars",
	"Premium Gold 2":    "Premium Gold: Return of the Bling",

	"Battle Pack 2: Round 2":           "Battle Pack 2: War of the Giants Round 2",
	"Duelist Pack Collection Tin 2010": "Duelist Pack Collection Tin",
	"Gold Series 1":                    "Gold Series 2008",
	"McDonald's Promo Pack 1":          "McDonald's Promo",
	"World Championship 2006":          "World Championship 2006: Ultimate Masters",

	// The booster the catalog numbers in words.
	"Turbo Pack 1": "Turbo Pack Booster One Pack",
	"Turbo Pack 2": "Turbo Pack: Booster Two",
	"Turbo Pack 3": "Turbo Pack: Booster Three",
	"Turbo Pack 4": "Turbo Pack: Booster Four",
	"Turbo Pack 5": "Turbo Pack: Booster Five",
	"Turbo Pack 6": "Turbo Pack: Booster Six",
	"Turbo Pack 7": "Turbo Pack: Booster Seven",
	"Turbo Pack 8": "Turbo Pack: Booster Eight",

	// The starter decks a storefront heads by their year or their series
	// where the catalog heads them by their subtitle, and the other way
	// around. "Two-Player Starter Set" is the digit the catalog writes.
	"Starter Deck 2011 Dawn of the Xyz":       "Starter Deck: Dawn of the Xyz",
	"Starter Deck 2012 XYZ Symphony":          "Starter Deck: Xyz Symphony",
	"Starter Deck Yu-Gi-Oh! 5Ds":              "5D's 2008 Starter Deck",
	"Starter Deck Yu-Gi-Oh! 5Ds 2009":         "5D's Starter Deck 2009",
	"Starter Deck Yu-Gi-Oh! GX":               "Starter Deck 2006",
	"Super-Starter 2013 V For Victory":        "Super Starter: V for Victory",
	"Super-Starter 2014 Space-Time Showdown!": "Super Starter: Space-Time Showdown",
	"Two-Player Starter Set":                  "2-Player Starter Set",

	// The box the storefronts sell a set of decks in, which the catalog
	// files the cards under the set's own name.
	"2025 Mega-Pack Bundle":               "2025 Mega-Pack",
	"Duel Devastator Box":                 "Duel Devastator",
	"Duel Overload Box Set":               "Duel Overload",
	"Legendary 5D's Decks Box Set":        "Legendary 5D's Decks",
	"Legendary Modern 2026 Decks Box Set": "Legendary Modern Decks 2026",

	// The video games the storefronts name by the game alone.
	"Destiny Board Traveler":        "Destiny Board Traveler Promo",
	"GX Duel Academy":               "GX Duel Academy GBA Promo",
	"GX Next Generation":            "GX Next Generation Blister Pack Promo",
	"GX Spirit Caller":              "GX Spirit Caller Promo",
	"Yu-Gi-Oh! The Movie Ani-Manga": "Yu-Gi-Oh! The Movie Ani-Manga Promo",
}

// nameRespellings pairs the card names a storefront and the catalog spell
// differently. Both spellings of each pair are real cards — cardtrader
// writes the old Vampire monsters with the adjective their DASA retrains
// carry — so no pair is ever a plain rename: respellName only crosses a
// pair for an edition that prints the other side, in whichever direction.
var nameRespellings = [][2]string{
	{"Vampire Orchis", "Vampiric Orchis"},
	{"Vampire Koala", "Vampiric Koala"},
}

// pooledEditions maps a storefront name spanning two of the catalog's sets
// onto the pair. Cool Stuff Inc files both Speed Duel starter decks under
// one "Starter Deck: Speed Dueling", and a listing does not say which of
// the two it means - the deck's own name is what the catalog files it
// under, and the storefront drops it.
//
// The name is left unrewritten, since a single-valued alias cannot carry a
// pair, and FilterCards restricts the candidates to the pool instead. The
// two decks share no card: every one of the eighty-three names the feed
// files here is printed in one of them and not the other, so narrowing to
// the pool is all it takes for the name to pick a set.
var pooledEditions = map[string][]string{
	"Starter Deck: Speed Dueling": {
		"Speed Duel Decks: Destiny Masters",
		"Speed Duel Decks: Duelists of Tomorrow",
	},
}

// normalizedPooledEditions indexes the pooled names the same way.
var normalizedPooledEditions = sync.OnceValue(func() map[string][]string {
	pools := make(map[string][]string, len(pooledEditions))
	for name, sets := range pooledEditions {
		pools[mtgmatcher.Normalize(name)] = sets
	}
	return pools
})

// normalizedEditionAliases indexes the table the way an edition arrives, so
// the punctuation and casing a storefront writes never decide. It is built
// once: the table is a constant.
var normalizedEditionAliases = sync.OnceValue(func() map[string]string {
	aliases := make(map[string]string, len(editionAliases))
	for name, set := range editionAliases {
		aliases[mtgmatcher.Normalize(name)] = set
	}
	return aliases
})
