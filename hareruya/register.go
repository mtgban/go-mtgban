package hareruya

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("hareruya", []mtgban.Game{mtgban.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.logCallback = opts.LogCallback
			scraper.targetEdition = opts.TargetEdition
			if opts.MaxConcurrency != 0 {
				scraper.maxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
	mtgban.Register("hareruya_sealed", []mtgban.Game{mtgban.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraperSealed(b)
			scraper.logCallback = opts.LogCallback
			return scraper, nil
		})
}
