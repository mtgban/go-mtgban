package magic

import "github.com/mtgban/go-mtgban/mtgmatcher"

// Register the Magic (MTGJSON) datastore loader so that a blank import of this
// package makes it known to mtgmatcher.Open.
func init() {
	mtgmatcher.RegisterGame("magic", Load)
}
