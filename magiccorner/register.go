package magiccorner

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("magiccorner", []mtgban.Game{mtgban.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper, err := NewScraper(b)
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = opts.LogCallback
			if opts.MaxConcurrency != 0 {
				scraper.MaxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
}
