package manaleak

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("manaleak", []mtgban.Game{mtgban.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.LogCallback = opts.LogCallback
			if opts.MaxConcurrency != 0 {
				scraper.MaxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
}
