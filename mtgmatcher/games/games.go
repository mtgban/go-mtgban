// Package games activates every built-in mtgmatcher game by registering their
// datastore loaders through their init functions. Blank-import it to make all
// games available with a single import instead of one per game:
//
//	import _ "github.com/mtgban/go-mtgban/mtgmatcher/games"
//
// This links every game (and its transitive dependencies) into the binary and
// makes every game's name known to mtgmatcher.Open. For a leaner build,
// blank-import only the specific mtgmatcher/<game> packages you need.
package games

import (
	// Each blank import runs a game's init, which registers its loader; see
	// the package comment above for why they are gathered here.
	_ "github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/gundam"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/onepiece"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/palworld"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/pokemon"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/riftbound"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/yugioh"
)
