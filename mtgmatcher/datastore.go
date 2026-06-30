package mtgmatcher

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
)

// GameLoader builds a Backend from a datastore reader for a particular game.
type GameLoader func(io.Reader) (*Backend, error)

type registeredGame struct {
	name string
	load GameLoader
}

var registeredGames []registeredGame

// RegisterGame registers a game's datastore loader under a unique name, in the
// style of database/sql's Register. Game packages (mtgmatcher/mtgjson,
// mtgmatcher/lorcana) call this from their init(); a consumer activates a game
// with a blank import, e.g. import _ "github.com/mtgban/go-mtgban/mtgmatcher/mtgjson".
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

// LoadDatastore auto-detects the datastore's game among the registered games
// and installs it as the global backend. At least one game package must be
// blank-imported. Preserved for source compatibility with the pre-sub-package
// loading API: each registered loader is tried in registration order and the
// first that succeeds wins (loaders reject formats they don't recognize).
func LoadDatastore(reader io.Reader) error {
	if len(registeredGames) == 0 {
		return errors.New("mtgmatcher: no game registered; blank-import a game package such as github.com/mtgban/go-mtgban/mtgmatcher/mtgjson")
	}
	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}
	var firstErr error
	for _, g := range registeredGames {
		b, err := g.load(bytes.NewReader(data))
		if err == nil && b != nil {
			SetGlobalDatastore(b)
			return nil
		}
		if firstErr == nil && err != nil {
			firstErr = err
		}
	}
	if firstErr == nil {
		firstErr = errors.New("unrecognized datastore format")
	}
	return fmt.Errorf("mtgmatcher: no registered game could load the datastore: %w", firstErr)
}

// LoadDatastoreFile is LoadDatastore over a file path.
func LoadDatastoreFile(filename string) error {
	reader, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer reader.Close()
	return LoadDatastore(reader)
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
