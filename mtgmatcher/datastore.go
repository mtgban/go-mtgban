package mtgmatcher

import (
	"fmt"
	"io"
	"strings"
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

// ProductKeyOf names the product a stored card is a printing of: the
// TCGplayer product id the datastore stamps on every printing it sells,
// and the entry's own uuid where it stamps none - an entry the builder
// mints is minted one printing at a time, so it is a product of one
// printing and stands for itself. The games whose datastores sell each
// finish as an entry of its own fold their candidates on it; see the note
// at the top of this file for why the uuid is never taken apart instead.
func ProductKeyOf(identifiers map[string]string, uuid string) string {
	if id := identifiers["tcgplayerProductId"]; id != "" {
		return id
	}
	return uuid
}

// SplitColors turns the colour value a Bandai-shaped catalog publishes into
// its components, "Red/Green" and "Red; Green" alike.
func SplitColors(color string) []string {
	if color == "" {
		return nil
	}
	fields := strings.FieldsFunc(color, func(r rune) bool {
		return r == ';' || r == '/'
	})
	for i := range fields {
		fields[i] = strings.TrimSpace(fields[i])
	}
	return fields
}

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
