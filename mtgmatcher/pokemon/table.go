package pokemon

import (
	"sync"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// editionAliases maps the set names storefronts write onto the set the
// catalog files those printings under. Every entry names a set the datastore
// does not carry under that spelling, so an edition already naming a set is
// never rewritten - AdjustEdition asks the backend first. Each entry was
// replayed against the printings in that expansion that resolve through
// their TCGplayer id, which name the set outright; spellings whose replay
// put even one product on a printing the id route contradicts stayed out.
var editionAliases = map[string]string{
	// The prize pack series all print into the one set the catalog
	// collects them in, and the series lives in the collector number
	// ("BRS 077"), so the edition only has to reach the shared set.
	// Series Five through Nine are absent on purpose: their replay lands
	// 4-34 of the comparable products on printings the id route
	// contradicts.
	"Play! Pokémon Prize Pack Series One":   "Prize Pack Series Cards",
	"Play! Pokémon Prize Pack Series Two":   "Prize Pack Series Cards",
	"Play! Pokémon Prize Pack Series Three": "Prize Pack Series Cards",
	"Play! Pokémon Prize Pack Series Four":  "Prize Pack Series Cards",

	// The Wizards-era promos, which the catalog files under the publisher.
	"Wizards Black Star Promos": "WoTC Promo",
	`"W" Promos`:                "WoTC Promo",

	// The era-prefixed promo spellings the catalog writes with the era's
	// long title. SV, MEP, SM, XY and Nintendo are absent on purpose:
	// their replay puts products on Jumbo, e-League and McDonald's
	// printings the id route contradicts.
	"SWSH Black Star Promos": "SWSH: Sword & Shield Promo Cards",
	"BW Black Star Promos":   "Black and White Promos",
	"DP Black Star Promos":   "Diamond and Pearl Promos",
	"HGSS Black Star Promos": "HGSS Promos",

	// The McDonald's series, which the storefronts name by campaign and
	// the catalog by year.
	"McDonald's Collection 25th Anniversary": "McDonald's 25th Anniversary Promos",
	"McDonald's Collection 2011":             "McDonald's Promos 2011",
	"McDonald's Collection 2012":             "McDonald's Promos 2012",
	"McDonald's Collection 2014":             "McDonald's Promos 2014",
	"McDonald's Collection 2015":             "McDonald's Promos 2015",
	"McDonald's Collection 2016":             "McDonald's Promos 2016",
	"McDonald's Collection 2017":             "McDonald's Promos 2017",
	"McDonald's Collection 2018":             "McDonald's Promos 2018",
	"McDonald's Collection 2019":             "McDonald's Promos 2019",
	"McDonald's Match Battle 2022":           "McDonald's Promos 2022",
	"McDonald's Match Battle 2023":           "McDonald's Promos 2023",
	"McDonald's Dragon Discovery":            "McDonald's Promos 2024",

	// The wordplay the storefronts drop.
	"Trick or Trade":      "Trick or Trade BOOster Bundle",
	"Trick or Trade 2023": "Trick or Trade BOOster Bundle 2023",
	"Trick or Trade 2024": "Trick or Trade BOOster Bundle 2024",

	// The decks the catalog collects in one set.
	"Pokémon Trading Card Game Classic: Venusaur & Lugia ex Deck":    "Trading Card Game Classic",
	"Pokémon Trading Card Game Classic: Charizard & Ho-Oh ex Deck":   "Trading Card Game Classic",
	"Pokémon Trading Card Game Classic: Blastoise & Suicune ex Deck": "Trading Card Game Classic",

	// The subtitle or year the two sides spell differently.
	"Battle Academy 2020":              "Battle Academy",
	"Pikachu World Collection":         "Pikachu World Collection Promos",
	"EX Trainer Kit":                   "EX Trainer Kit 1: Latias & Latios",
	"EX Trainer Kit 2":                 "EX Trainer Kit 2: Plusle & Minun",
	"DP Trainer Kit":                   "DP Trainer Kit: Manaphy & Lucario",
	"HS Trainer Kit":                   "HGSS Trainer Kit: Gyarados & Raichu",
	"BW Trainer Kit":                   "BW Trainer Kit: Excadrill & Zoroark",
	"Burger King DP Promos 2008":       "Burger King Promos",
	"Burger King Platinum Promos 2009": "Burger King Promos",
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
