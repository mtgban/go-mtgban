package hareruya

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

func init() {
	mtgban.Register("hareruya", []matcher.Game{matcher.GameMagic},
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.logCallback = opts.LogCallback
			scraper.targetEdition = opts.TargetEdition
			if opts.MaxConcurrency != 0 {
				scraper.maxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
	mtgban.Register("hareruya_sealed", []matcher.Game{matcher.GameMagic},
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraperSealed(b)
			scraper.logCallback = opts.LogCallback
			return scraper, nil
		})
}
