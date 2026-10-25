package mtgseattle

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

func init() {
	mtgban.Register("mtgseattle", []matcher.Game{matcher.GameMagic},
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
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
