package abugames

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

func init() {
	mtgban.Register("abugames", []matcher.Game{matcher.GameMagic}, newScraper)
	mtgban.Register("abugames_sealed", []matcher.Game{matcher.GameMagic}, newScraperSealed)
}

func newScraper(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper := NewScraper(b)
	scraper.logCallback = opts.LogCallback
	if opts.MaxConcurrency != 0 {
		scraper.maxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}

func newScraperSealed(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper := NewScraperSealed(b)
	scraper.logCallback = opts.LogCallback
	if opts.MaxConcurrency != 0 {
		scraper.maxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}
