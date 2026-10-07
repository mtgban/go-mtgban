package manaleak

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("manaleak", []mtgmatcher.Game{mtgmatcher.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.logCallback = opts.LogCallback
			scraper.logRetries = opts.LogRetries
			if opts.MaxConcurrency != 0 {
				scraper.maxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
}
