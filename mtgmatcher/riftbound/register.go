package riftbound

import "github.com/mtgban/go-mtgban/mtgmatcher"

// Register the Riftbound datastore loader so that a blank import of this
// package makes the game available to mtgmatcher.Open.
func init() {
	mtgmatcher.RegisterGame("riftbound", Load)
}
