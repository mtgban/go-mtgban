package magic

import "github.com/mtgban/go-mtgban/matcher"

// Register the Magic (MTGJSON) datastore loader so that a blank import of this
// package makes it known to matcher.Open.
func init() {
	matcher.RegisterGame(matcher.GameMagic, Load)
}
