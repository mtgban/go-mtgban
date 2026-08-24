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

	// Every Duelist League shares the one set code, so the league's own
	// number lives in the collector number and the edition only has to
	// reach the set the leagues are collected in.
	"Duelist League 09":               "Duelist League Promo",
	"Duelist League 11":               "Duelist League Promo",
	"Duelist League 12":               "Duelist League Promo",
	"Duelist League 13":               "Duelist League Promo",
	"Duelist League 14":               "Duelist League Promo",
	"Duelist League 15":               "Duelist League Promo",
	"Duelist League 16":               "Duelist League Promo",
	"Duelist League 17":               "Duelist League Promo",
	"Duelist League 18":               "Duelist League Promo",
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

	"Duelist Pack Collection Tin 2010": "Duelist Pack Collection Tin",
	"McDonald's Promo Pack 1":          "McDonald's Promo",
	"World Championship 2006":          "World Championship 2006: Ultimate Masters",

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
