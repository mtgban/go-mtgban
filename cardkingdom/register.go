package cardkingdom

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

func init() {
	mtgban.Register("cardkingdom", []matcher.Game{matcher.GameMagic}, newScraper)
	mtgban.Register("cardkingdom_graded", []matcher.Game{matcher.GameMagic}, newScraperGraded)
	mtgban.Register("cardkingdom_sealed", []matcher.Game{matcher.GameMagic}, newScraperSealed)
}

func newScraper(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper := NewScraper(b)
	scraper.logCallback = opts.LogCallback
	scraper.partner = opts.Affiliate
	// This was bantool's own setting, and is really the store's: Card
	// Kingdom's out-of-stock listings are worth publishing for their URL
	// alone.
	scraper.preserveOOS = true
	return scraper, nil
}

func newScraperGraded(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper, err := NewScraperGraded(b)
	if err != nil {
		return nil, err
	}
	scraper.logCallback = opts.LogCallback
	scraper.partner = opts.Affiliate
	return scraper, nil
}

func newScraperSealed(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper := NewScraperSealed(b)
	scraper.logCallback = opts.LogCallback
	scraper.partner = opts.Affiliate
	scraper.preserveOOS = true
	return scraper, nil
}
