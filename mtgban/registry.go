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
// what NewScraper calls; the game is the datastore's, read with GameOf, and
// the secrets are the options', read with Options.Secret and
// Options.OptionalSecret.
type Constructor func(b *mtgmatcher.Backend, opts Options) (Scraper, error)

type registration struct {
	name  string
	games []Game
	build Constructor
}

var registeredScrapers []registration

// Register files a scraper under the name callers know it by, for the games
// it prices, in the style of database/sql's Register. A scraper package
// calls it from init, so importing the package is what puts its scrapers
// within NewScraper's reach. Registering a name twice for one game panics.
func Register(name string, games []Game, build Constructor) {
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

func lookup(game Game, name string) (registration, bool) {
	for _, reg := range registeredScrapers {
		if reg.name == name && slices.Contains(reg.games, game) {
			return reg, true
		}
	}
	return registration{}, false
}

// Registered lists the names registered for a game, sorted.
func Registered(game Game) []string {
	var names []string
	for _, reg := range registeredScrapers {
		if slices.Contains(reg.games, game) {
			names = append(names, reg.name)
		}
	}
	slices.Sort(names)
	return names
}

// GameOf names the game a datastore was loaded for. mtgmatcher spells the
// name its loader registered in lowercase ("magic") and Game capitalizes it,
// so the two are compared without case.
func GameOf(b *mtgmatcher.Backend) (Game, error) {
	if b.Game == "" {
		return "", errors.New("mtgban: the datastore names no game")
	}
	for _, game := range AllGames {
		if strings.EqualFold(string(game), b.Game) {
			return game, nil
		}
	}
	return "", fmt.Errorf("mtgban: the datastore's game %q is not one mtgban prices", b.Game)
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
	game, err := GameOf(b)
	if err != nil {
		return nil, err
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
