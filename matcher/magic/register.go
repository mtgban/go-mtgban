package magic

import (
	"strings"

	"github.com/mtgban/go-mtgban/matcher"
)

// Register the Magic (MTGJSON) datastore loader so that a blank import of this
// package makes it known to matcher.Open.
func init() {
	matcher.RegisterGame(matcher.GameMagic, Load)
	matcher.ParenthesizedNames = append(matcher.ParenthesizedNames, parenthesizedName)
}

// parenthesizedName recognises the Magic cards whose names carry parentheses
// of their own, for matcher.SplitVariants.
func parenthesizedName(str string) (tail string, found bool) {
	switch {
	case strings.HasPrefix(str, "Erase (Not the Urza's Legacy One)"),
		strings.HasPrefix(str, "Hazmat Suit (Used"),
		strings.HasPrefix(str, "B.O.B."),
		matcher.Contains(str, "B.F.M.") && strings.Contains(str, "Big Furry Monster"):
		return "", true
	case strings.HasPrefix(str, "Dwight, Assistant (to the)"):
		return " King", true
	}
	return "", false
}
