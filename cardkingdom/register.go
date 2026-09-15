package cardkingdom

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("cardkingdom", []mtgban.Game{mtgban.GameMagic}, newScraper)
	mtgban.Register("cardkingdom_graded", []mtgban.Game{mtgban.GameMagic}, newScraperGraded)
	mtgban.Register("cardkingdom_sealed", []mtgban.Game{mtgban.GameMagic}, newScraperSealed)
}

func newScraper(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper := NewScraper(b)
	scraper.LogCallback = opts.LogCallback
	scraper.Partner = opts.Affiliate
	// This was bantool's own setting, and is really the store's: Card
	// Kingdom's out-of-stock listings are worth publishing for their URL
	// alone.
	scraper.PreserveOOS = true
	return scraper, nil
}

func newScraperGraded(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper, err := NewScraperGraded(b)
	if err != nil {
		return nil, err
	}
	scraper.LogCallback = opts.LogCallback
	scraper.Partner = opts.Affiliate
	return scraper, nil
}

func newScraperSealed(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper := NewScraperSealed(b)
	scraper.LogCallback = opts.LogCallback
	scraper.Partner = opts.Affiliate
	scraper.PreserveOOS = true
	return scraper, nil
}
