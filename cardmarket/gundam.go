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

// gundamLabels names, by product id, the Premium Bandai products selling a
// card's Premium Card Collection 02 print, which Cardmarket names like the
// card itself; CardTrader's blueprints say which they are.
var gundamLabels = map[int]string{
	908743: "Premium Card Collection 02", // Sword Strike Gundam, GD01-073
	908744: "Premium Card Collection 02", // A Healthy Curiosity, GD03-101
	908745: "Premium Card Collection 02", // GN Armor Type-E, GD04-063
	908746: "Premium Card Collection 02", // Darkness Finger, GD05-110
	908747: "Premium Card Collection 02", // Widespread Annihilation, GD05-114
}
