package bazaarofmagic

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

func init() {
	mtgban.Register("bazaarofmagic", []matcher.Game{matcher.GameMagic},
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.logCallback = opts.LogCallback
			// The site rate limits, so a caller may only lower the
			// concurrency, never raise it.
			if opts.MaxConcurrency != 0 && opts.MaxConcurrency < scraper.maxConcurrency {
				scraper.maxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
	mtgban.Register("bazaarofmagic_sealed", []matcher.Game{matcher.GameMagic},
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraperSealed(b)
			scraper.logCallback = opts.LogCallback
			return scraper, nil
		})
}
