package mtgseattle

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("mtgseattle", []mtgban.Game{mtgban.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.logCallback = opts.LogCallback
			// The site throttles hard past defaultConcurrency, so a
			// caller may only lower it, never raise it.
			if opts.MaxConcurrency != 0 && opts.MaxConcurrency < scraper.maxConcurrency {
				scraper.maxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
}
