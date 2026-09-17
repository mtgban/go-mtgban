package strikezone

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("strikezone",
		[]mtgban.Game{
			mtgban.GameMagic, mtgban.GameLorcana, mtgban.GamePokemon,
			mtgban.GameFleshAndBlood,
		},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
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
