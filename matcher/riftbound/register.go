package riftbound

import "github.com/mtgban/go-mtgban/matcher"

// Register the Riftbound datastore loader so that a blank import of this
// package makes the game available to matcher.Open.
func init() {
	matcher.RegisterGame(matcher.GameRiftbound, Load)
}
