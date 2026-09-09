package mtgmatcher

import (
	"fmt"
	"io"
)

// A datastore entry's id is opaque. The builder that publishes it spells it,
// and no loader here reads one apart: what an id was once taken apart for -
// which product an entry belongs to, which printing of that product it is -
// every datastore publishes as a field of its own, so a loader asks for the
// fact instead of inferring it from a shape.
//
// This is not a style preference. Four loaders used to recover an entry's
// product by trimming a literal tail off its id, and when the builders began
// spelling those tails from the printing's own name - "_holo" became
// "_holofoil", "_1e" became "_1stedition" - the trimming stopped matching and
// printings of one product became separate cards. Nothing threw: the ids were
// still well formed, still unique, still stable, and one game's grouping test
// was the only thing that noticed. A grouping read off a published field
// cannot fail that way, and it leaves the builder free to respell an id, or
// to stop spelling it out of anything at all.

// GameLoader builds a Backend from a datastore reader for a particular game.
type GameLoader func(io.Reader) (*Backend, error)

type registeredGame struct {
	name string
	load GameLoader
}

var registeredGames []registeredGame

// RegisterGame registers a game's datastore loader under a unique name, in the
// style of database/sql's Register. Game packages (mtgmatcher/magic,
// mtgmatcher/lorcana) call this from their init(); a consumer activates a game
// with a blank import, e.g. import _ "github.com/mtgban/go-mtgban/mtgmatcher/magic".
// It panics on a duplicate name or a nil loader.
func RegisterGame(name string, load GameLoader) {
	if load == nil {
		panic("mtgmatcher: RegisterGame loader is nil for " + name)
	}
	for _, g := range registeredGames {
		if g.name == name {
			panic("mtgmatcher: RegisterGame called twice for " + name)
		}
	}
	registeredGames = append(registeredGames, registeredGame{name: name, load: load})
}

// RegisteredGames returns the names of the registered games in registration
// order.
func RegisteredGames() []string {
	names := make([]string, len(registeredGames))
	for i, g := range registeredGames {
		names[i] = g.name
	}
	return names
}

// Open loads the named game's datastore explicitly (sql.Open style) and returns
// the Backend without installing it as the global one.
func Open(name string, reader io.Reader) (*Backend, error) {
	for _, g := range registeredGames {
		if g.name == name {
			return g.load(reader)
		}
	}
	return nil, fmt.Errorf("mtgmatcher: unknown game %q (registered: %v)", name, RegisteredGames())
}
