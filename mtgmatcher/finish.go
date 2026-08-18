package mtgmatcher

import "strings"

// The finishes every game shares, spelled the way the matcher spells every
// finish name: lower case, no separators. A game names the finishes past
// these itself (Lorcana's rainbow pillars, the reverse holos and first
// editions the pending games bring) through GameRules.CanonicalFinish, and
// the names it hands back are the keys of Card.FoilUUIDs and the value of
// CardObject.Finish, so one spelling reaches a printing from every source.
const (
	FinishNonfoil = "nonfoil"
	FinishFoil    = "foil"
	FinishEtched  = "etched"
)

// NormalizeFinish spells a finish name the way finish names are spelled here,
// dropping the separators and the case a vendor writes it with, so "Cold
// Foil", "cold foil" and "COLD-FOIL" are one name. A game whose finish
// vocabulary is open - Lorcana's foil types are data, not a fixed list -
// canonicalizes its own names with this.
func NormalizeFinish(name string) string {
	var out strings.Builder
	for _, r := range strings.ToLower(name) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			out.WriteRune(r)
		}
	}
	return out.String()
}

// CanonicalFinish maps the finish names every game shares onto the constants
// above, including the spellings vendors reach them by, and answers "" for
// anything else - a name only the game can place. It is the fallback every
// GameRules.CanonicalFinish defers to, so a game only has to name what is
// its own.
func CanonicalFinish(name string) string {
	switch NormalizeFinish(name) {
	// "Normal" is what TCGplayer calls a plain printing in every category
	case "nonfoil", "normal":
		return FinishNonfoil
	case "foil":
		return FinishFoil
	case "etched", "foiletched", "etchedfoil":
		return FinishEtched
	}
	return ""
}
