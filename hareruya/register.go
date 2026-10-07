package hareruya

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("hareruya", []mtgmatcher.Game{mtgmatcher.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.logCallback = opts.LogCallback
			scraper.logRetries = opts.LogRetries
			scraper.targetEdition = opts.TargetEdition
			if opts.MaxConcurrency != 0 {
				scraper.maxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
	mtgban.Register("hareruya_sealed", []mtgmatcher.Game{mtgmatcher.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraperSealed(b)
			scraper.logCallback = opts.LogCallback
			scraper.logRetries = opts.LogRetries
			return scraper, nil
		})
}
