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
	"WOTC Black Star Promos":    "WoTC Promo",
	`"W" Promos`:                "WoTC Promo",

	// The era-prefixed promo spellings the catalog writes with the era's
	// long title. SV, MEP, SM and XY are absent on purpose: their replay
	// puts products on Jumbo, e-League and McDonald's printings the id
	// route contradicts.
	//
	// Nintendo was absent for the same reason and is back, because the
	// replay says the opposite of what was feared for it: over the whole
	// Cool Stuff Inc Pokemon buylist it gains 7 and moves 4, and three of
	// those four move OFF the Jumbo printings rather than onto them - a
	// listing filed under "Nintendo Black Star Promos" is not a jumbo.
	// The fourth is Pikachu 035, which had been answering with the
	// Evolutions card at a fortieth of the price. Nothing is lost, and 23
	// of the 40 listings on that shelf already reached the set without
	// the alias.
	"SWSH Black Star Promos":     "SWSH: Sword & Shield Promo Cards",
	"BW Black Star Promos":       "Black and White Promos",
	"DP Black Star Promos":       "Diamond and Pearl Promos",
	"HGSS Black Star Promos":     "HGSS Promos",
	"Nintendo Black Star Promos": "Nintendo Promos",

	// The same promo runs written with the era behind the word rather than
	// in front of it. Neither of the general readings reaches these: the
	// tail reading asks what the wording and a set name end alike, and
	// "Sword and Shield" ends no set's name, while "SM Promos" and "Sun and
	// Moon" share no tail at all; the trailing-"Promos" reading only looks
	// at the end of the wording, and these spell it at the front.
	"Promos Sword and Shield": "SWSH: Sword & Shield Promo Cards",
	"Promos Sun and Moon":     "SM Promos",

	// The 1999 set under the print run a storefront heads it with. The run
	// itself rides in the row's own details, so naming the set here does not
	// spend it: an unlimited listing still reaches the unlimited printing.
	"Base Set Unlimited": "Base Set",

	// Mega Evolution's base set, which is not the 1999 one. Left alone it
	// reaches "Base Set" by tail and prices a 2025 card as a 1999 one, which
	// is a wrong answer rather than a refusal, so this entry is the only
	// thing standing between the two.
	"Mega Evolution Base Set": "ME01: Mega Evolution",

	// The McDonald's campaigns, which the storefronts name and the catalog
	// files by year. The years the catalog has since started carrying under
	// the storefront's own spelling are not here: they name sets of their
	// own now, so the lookup above reaches them and an entry would only be
	// a name this table promises not to hold.
	"McDonald's Collection 25th Anniversary": "McDonald's 25th Anniversary Promos",
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

// pooledEditions maps the storefront names spanning two of the catalog's
// sets onto the pair: "Theme Deck & Blisters Exclusives" files what the
// catalog splits into Blister Exclusives and Deck Exclusives, and a listing
// does not say which half it means. The name is left unrewritten - a
// single-valued edition cannot carry the pair - and FilterCards restricts
// the candidates to the pool, so the collector number picks within it and
// nothing falls through to a global guess.
var pooledEditions = map[string][]string{
	"Theme Deck & Blisters Exclusives": {"Blister Exclusives", "Deck Exclusives"},
}

// normalizedPooledEditions indexes the pooled names the same way.
var normalizedPooledEditions = sync.OnceValue(func() map[string][]string {
	pools := make(map[string][]string, len(pooledEditions))
	for name, sets := range pooledEditions {
		pools[mtgmatcher.Normalize(name)] = sets
	}
	return pools
})
