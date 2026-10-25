package lorcana

import "github.com/mtgban/go-mtgban/matcher"

// Register the Lorcana datastore loader so that a blank import of this package
// makes it known to matcher.Open.
func init() {
	matcher.RegisterGame(matcher.GameLorcana, Load)
}
