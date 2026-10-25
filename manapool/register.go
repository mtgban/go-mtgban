package manapool

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

func init() {
	mtgban.Register("manapool", []matcher.Game{matcher.GameMagic},
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.partner = opts.Affiliate
			scraper.logCallback = opts.LogCallback
			return scraper, nil
		})
	mtgban.Register("manapool_index", []matcher.Game{matcher.GameMagic},
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraperIndex(b)
			scraper.partner = opts.Affiliate
			scraper.logCallback = opts.LogCallback
			return scraper, nil
		})
	mtgban.Register("manapool_sealed", []matcher.Game{matcher.GameMagic},
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraperSealed(b)
			scraper.partner = opts.Affiliate
			scraper.logCallback = opts.LogCallback
			return scraper, nil
		})
}
