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

	// The leagues used to share one set code, so an edition only had to
	// reach the shelf they were collected on and the season lived in the
	// collector number. The datastore files each season as its own set
	// now, so the edition reaches the season it names.
	//
	// "Duelist League Promos Upperdeck" is gone rather than remapped: it
	// names the distributor and no season, and there is no longer one set
	// for it to mean. Those listings answer by collector number, which is
	// the only thing about them that says which league.
	"Duelist League Series 1":  "Duelist League Series 1 participation cards",
	"Duelist League Series 2":  "Duelist League Series 2 participation cards",
	"Duelist League Series 3":  "Duelist League Series 3 participation card",
	"Duelist League Series 4":  "Duelist League Series 4 participation card",
	"Duelist League Series 5":  "Duelist League Series 5 participation card",
	"Duelist League Series 6":  "Duelist League Series 6 participation card",
	"Duelist League Series 7":  "Duelist League Series 7 participation card",
	"Duelist League Series 8":  "Duelist League Series 8 participation card",
	"Duelist League Series 9":  "Duelist League Series 9 participation card",
	"Duelist League Series 10": "Duelist League Series 10 participation card",

	// Card Trader heads the Mega-Tin packs by one tin where the catalog
	// pluralises them, and drops the possessive off the video game.
	"2014 Mega-Tin Mega Pack":                     "2014 Mega-Tins Mega Pack",
	"2015 Mega-Tin Mega Pack":                     "2015 Mega-Tins Mega Pack",
	"2016 Mega-Tin Mega Pack":                     "2016 Mega-Tins Mega Pack",
	"2017 Mega-Tin Mega Pack":                     "2017 Mega-Tins Mega Pack",
	"2018 Mega-Tin Mega Pack":                     "2018 Mega-Tins Mega Pack",
	"Eternal Duelist's Soul":                      "Eternal Duelist Soul",
	"Forbidden Legacy 1":                          "Forbidden Legacy",
	"Kaiba's Collector Box":                       "Collector's Boxes (KACB)",
	"Reshef of Destruction Promos":                "Reshef of Destruction",
	"5D's Stardust Accelerator Promotional Cards": "Stardust Accelerator Promos",
	"Token Promos 3":                              "Yu-Gi-Oh! Tokens",
	"Token Promos 4":                              "Yu-Gi-Oh! Tokens",

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

	// The manga promos are a volume each, and the datastore files them
	// that way now - nine sets for the 5D's run, eight for GX. A wording
	// naming the run and not the volume has not said which of them it
	// means, so only ARC-V, which the catalog still keeps as one set,
	// is answered here. The rest answer by collector number, where the
	// volume is written down.
	"ARC-V Manga Promos": "Yu-Gi-Oh! ARC-V Promo Cards",

	// The subtitle or volume word the catalog spells out and the
	// storefronts drop.
	"Dark Revelation 1": "Dark Revelation Volume 1",
	"Dark Revelation 2": "Dark Revelation Volume 2",
	"Dark Revelation 3": "Dark Revelation Volume 3",
	"Dark Revelation 4": "Dark Revelation Volume 4",
	// Cool Stuff Inc pluralises the run. Graded against every name it
	// files under each: 125, 120 and 127 of them print in volumes 2, 3
	// and 4, none anywhere else.
	"Dark Revelations 1": "Dark Revelation Volume 1",
	"Dark Revelations 2": "Dark Revelation Volume 2",
	"Dark Revelations 3": "Dark Revelation Volume 3",
	"Dark Revelations 4": "Dark Revelation Volume 4",
	"Hidden Arsenal 5":   "Hidden Arsenal 5: Steelswarm Invasion",
	"Hidden Arsenal 6":   "Hidden Arsenal 6: Omega Xyz",
	"Hidden Arsenal 7":   "Hidden Arsenal 7: Knight of Stars",
	"Premium Gold 2":     "Premium Gold: Return of the Bling",

	"Battle Pack 2: Round 2":           "Battle Pack 2: War of the Giants Round 2",
	"Duelist Pack Collection Tin 2010": "Duelist Pack Collection Tin",
	"Gold Series 1":                    "Gold Series 2008",
	"McDonald's Promo Pack 1":          "McDonald's Promo",
	"World Championship 2006":          "World Championship 2006: Ultimate Masters",
	// The article and colon the catalog spells and the storefront drops;
	// all 28 names Cool Stuff Inc files under it print in the deck.
	"Structure Deck Dark Emperor": "Structure Deck: The Dark Emperor",

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

	// The cards Konami renamed, which the catalog files under the name
	// each set printed and Cardmarket writes under the current one
	// whichever set the product is from.
	{"Red-Eyes Black Dragon", "Red-Eyes B. Dragon"},
	{"Black Skull Dragon", "B. Skull Dragon"},
	{"Black Dragon's Chick", "Red-Eyes B. Chick"},
	{"Meteor Black Dragon", "Meteor B. Dragon"},
	{"Earthbound Immortal Revival", "Earthbound Revival"},
	{"Roar of the Earthbound Immortal", "Roar of the Earthbound"},
	// The catalog files the Rise of Destiny printing under the name
	// without its prefix.
	{"B.E.S. Big Core", "Big Core"},
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
	// The two half-decks the catalog files the 2-Player Starter Deck as,
	// Yuya's Saber Force and Declan's Dark Legion. The commons both decks
	// hold are two printings of one card in one box, and a listing naming
	// the box alone stays refused between them.
	"2-Player Starter Deck Yuya & Declan": {
		"Starter Deck: Saber Force",
		"Starter Deck: Dark Legion",
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
