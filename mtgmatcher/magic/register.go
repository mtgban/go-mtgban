package magic

import "github.com/mtgban/go-mtgban/mtgmatcher"

// Register the Magic (MTGJSON) datastore loader so that a blank import of this
// package makes mtgmatcher.LoadDatastore able to auto-detect and load it.
func init() {
	mtgmatcher.RegisterGame("magic", Load)
}
