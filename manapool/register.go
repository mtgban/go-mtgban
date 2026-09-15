package manapool

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("manapool", []mtgban.Game{mtgban.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.Partner = opts.Affiliate
			scraper.LogCallback = opts.LogCallback
			return scraper, nil
		})
	mtgban.Register("manapool_index", []mtgban.Game{mtgban.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraperIndex(b)
			scraper.Partner = opts.Affiliate
			scraper.LogCallback = opts.LogCallback
			return scraper, nil
		})
	mtgban.Register("manapool_sealed", []mtgban.Game{mtgban.GameMagic},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraperSealed(b)
			scraper.Partner = opts.Affiliate
			scraper.LogCallback = opts.LogCallback
			return scraper, nil
		})
}
