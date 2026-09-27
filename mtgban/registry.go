package mtgban

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Constructor builds a scraper against the datastore it is handed, from the
// options its caller supplied. It is what a scraper package registers and
// what NewScraper calls; the game is the datastore's own, b.Game, and the
// secrets are the options', read with Options.Secret.
type Constructor func(b *mtgmatcher.Backend, opts Options) (Scraper, error)

type registration struct {
	name  string
	games []mtgmatcher.Game
	build Constructor
}

var registeredScrapers []registration

// Register files a scraper under the name callers know it by, for the games
// it prices, in the style of database/sql's Register. A scraper package
// calls it from init, so importing the package is what puts its scrapers
// within NewScraper's reach. Registering a name twice for one game panics.
func Register(name string, games []mtgmatcher.Game, build Constructor) {
	if build == nil {
		panic("mtgban: Register constructor is nil for " + name)
	}
	if len(games) == 0 {
		panic("mtgban: Register called with no games for " + name)
	}
	for _, game := range games {
		if _, found := lookup(game, name); found {
			panic(fmt.Sprintf("mtgban: Register called twice for %s on %s", name, game))
		}
	}
	registeredScrapers = append(registeredScrapers, registration{
		name:  name,
		games: slices.Clone(games),
		build: build,
	})
}

func lookup(game mtgmatcher.Game, name string) (registration, bool) {
	for _, reg := range registeredScrapers {
		if reg.name == name && slices.Contains(reg.games, game) {
			return reg, true
		}
	}
	return registration{}, false
}

// Registered lists the names registered for a game, sorted.
func Registered(game mtgmatcher.Game) []string {
	var names []string
	for _, reg := range registeredScrapers {
		if slices.Contains(reg.games, game) {
			names = append(names, reg.name)
		}
	}
	slices.Sort(names)
	return names
}

// NewScraper builds the named scraper against the datastore, configured and
// ready for Load. The name is the one the scraper registered ("cardmarket",
// "tcg_market"); the game is read off the datastore, which is what the
// scraper will match against, so the two cannot disagree. A scraper that
// asks for a secret is built with WithAuthenticator; most ask for none.
func NewScraper(b *mtgmatcher.Backend, name string, opts ...Option) (Scraper, error) {
	if b == nil {
		return nil, errors.New("mtgban: NewScraper needs a datastore")
	}
	game := b.Game
	if game == "" {
		return nil, errors.New("mtgban: the datastore names no game")
	}
	reg, found := lookup(game, name)
	if !found {
		return nil, fmt.Errorf("mtgban: no scraper %q for %s (registered: %s)",
			name, game, strings.Join(Registered(game), ", "))
	}

	var options Options
	for _, opt := range opts {
		opt.apply(&options)
	}
	if options.DisableRetail && options.DisableBuylist {
		return nil, fmt.Errorf("%s was asked for its retail alone and its buylist "+
			"alone at once, which leaves nothing to publish", name)
	}

	scraper, err := reg.build(b, options)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	if !options.DisableRetail && !options.DisableBuylist {
		return scraper, nil
	}
	config, ok := scraper.(ScraperConfig)
	if !ok {
		half := "buylist"
		if options.DisableBuylist {
			half = "retail"
		}
		return nil, fmt.Errorf("%s was asked for its %s alone, but does not implement "+
			"mtgban.ScraperConfig and would publish both halves", name, half)
	}
	config.SetConfig(ScraperOptions{
		DisableRetail:  options.DisableRetail,
		DisableBuylist: options.DisableBuylist,
	})
	return scraper, nil
}
