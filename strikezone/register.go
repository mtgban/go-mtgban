package strikezone

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

func init() {
	mtgban.Register("strikezone",
		[]matcher.Game{
			matcher.GameMagic, matcher.GameLorcana, matcher.GamePokemon,
			matcher.GameFleshAndBlood,
		},
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper, err := NewScraper(b)
			if err != nil {
				return nil, err
			}
			scraper.logCallback = opts.LogCallback
			if opts.MaxConcurrency != 0 {
				scraper.maxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
}
