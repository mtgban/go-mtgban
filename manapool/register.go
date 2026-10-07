package manapool

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("manapool", []mtgmatcher.Game{mtgmatcher.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.partner = opts.Affiliate
			scraper.logCallback = opts.LogCallback
			scraper.logRetries = opts.LogRetries
			return scraper, nil
		})
	mtgban.Register("manapool_index", []mtgmatcher.Game{mtgmatcher.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraperIndex(b)
			scraper.partner = opts.Affiliate
			scraper.logCallback = opts.LogCallback
			scraper.logRetries = opts.LogRetries
			return scraper, nil
		})
	mtgban.Register("manapool_sealed", []mtgmatcher.Game{mtgmatcher.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraperSealed(b)
			scraper.partner = opts.Affiliate
			scraper.logCallback = opts.LogCallback
			scraper.logRetries = opts.LogRetries
			return scraper, nil
		})
}
