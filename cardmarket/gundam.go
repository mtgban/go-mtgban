package cardmarket

import "strings"

// gundamRarities maps Cardmarket's spelling of a Gundam rarity onto the
// catalog's. The catalog marks a parallel by suffixing the rarity, where
// Cardmarket spells the rarity out and spaces the suffix off.
var gundamRarities = map[string]string{
	"Legendary Rare":    "Legend Rare",
	"Legendary Rare +":  "LR+",
	"Legendary Rare ++": "LR++",
	"Rare +":            "R+",
	"Uncommon +":        "U+",
	"Common +":          "C+",
	"Common ++":         "C++",
}

// gundamRarity spells a Cardmarket Gundam rarity the catalog's way.
func gundamRarity(rarity string) string {
	rarity = strings.TrimSpace(rarity)
	if spelled, found := gundamRarities[rarity]; found {
		return spelled
	}
	return rarity
}

// gundamLabels names, by product id, the promo print a product sells where
// Cardmarket names it like the card itself: the Premium Bandai products
// selling a Premium Card Collection 02 print, which CardTrader's blueprints
// identify, and the second Common of GD02-029's promo shelf, the one print
// of that card no other product prices.
var gundamLabels = map[int]string{
	908743: "Premium Card Collection 02", // Sword Strike Gundam, GD01-073
	908744: "Premium Card Collection 02", // A Healthy Curiosity, GD03-101
	908745: "Premium Card Collection 02", // GN Armor Type-E, GD04-063
	908746: "Premium Card Collection 02", // Darkness Finger, GD05-110
	908747: "Premium Card Collection 02", // Widespread Annihilation, GD05-114
	908748: "Premium Card Collection 02", // Char's Zaku II, ST03-006

	914392: "Store Tournament Participation Pack", // Gundam AGE-1 Normal, GD02-029
}

// gundamCrossedLinks gives the TCGplayer id of the two Beta Char's Zaku II
// products whose CardTrader blueprints list each other's Cardmarket id.
var gundamCrossedLinks = map[int]int{
	905979: 616641, // Rare +
	905980: 616640, // Rare
}

// gundamUnmade lists Cardmarket duplicates of a product that carry a code
// of another card, and name no printing at all.
var gundamUnmade = map[int]bool{
	908636: true, // a second Haman Karn tournament promo, coded GD02-092
}

// gundamNoRow lists the products of a printing the datastore does not carry,
// which the promo shelf pin would otherwise land on a sibling's row.
var gundamNoRow = map[int]bool{
	908683: true, // Char's Zaku II, ST03-006, a Special Tournament Promo
	908594: true, // GD01-013 V.2, a fifth print no catalog names
}

// gundamPromoSet is the one set the datastore files every promotional
// printing in, which Cardmarket splits over shelves of its own.
const gundamPromoSet = "Gundam Promotional Cards"

// gundamPromoShelves are the Cardmarket shelves holding promotional prints.
// Named as they are, the matcher reaches past them to the card's main-set
// printing, which the card's own product already prices.
var gundamPromoShelves = map[string]bool{
	"Unnumbered Promos":         true,
	"Special Tournament Promos": true,
	"Winner Cards":              true,
	"Premium Bandai Products":   true,
	"Judge Promos":              true,
	"Promos":                    true,
}

// gundamNumber answers the collector number of a product. Cardmarket's own
// number field can hold another card's code, where the code the name ends
// with is always the card's.
func gundamNumber(cardName, number string) string {
	fields := nameCode.FindStringSubmatch(cardName)
	if fields != nil && strings.Contains(number, "-") {
		return fields[1]
	}
	return number
}
