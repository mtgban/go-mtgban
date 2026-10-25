// Package games activates every built-in matcher game by registering their
// datastore loaders through their init functions. Blank-import it to make all
// games available with a single import instead of one per game:
//
//	import _ "github.com/mtgban/go-mtgban/matcher/games"
//
// This links every game (and its transitive dependencies) into the binary and
// makes every game's name known to matcher.Open. For a leaner build,
// blank-import only the specific matcher/<game> packages you need.
package games

import (
	// Each blank import runs a game's init, which registers its loader; see
	// the package comment above for why they are gathered here.
	_ "github.com/mtgban/go-mtgban/matcher/fleshandblood"
	_ "github.com/mtgban/go-mtgban/matcher/gundam"
	_ "github.com/mtgban/go-mtgban/matcher/lorcana"
	_ "github.com/mtgban/go-mtgban/matcher/magic"
	_ "github.com/mtgban/go-mtgban/matcher/onepiece"
	_ "github.com/mtgban/go-mtgban/matcher/palworld"
	_ "github.com/mtgban/go-mtgban/matcher/pokemon"
	_ "github.com/mtgban/go-mtgban/matcher/riftbound"
	_ "github.com/mtgban/go-mtgban/matcher/yugioh"
)
