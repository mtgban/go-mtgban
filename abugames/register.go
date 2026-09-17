package abugames

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("abugames", []mtgban.Game{mtgban.GameMagic}, newScraper)
	mtgban.Register("abugames_sealed", []mtgban.Game{mtgban.GameMagic}, newScraperSealed)
}

func newScraper(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper := NewScraper(b)
	scraper.logCallback = opts.LogCallback
	if opts.MaxConcurrency != 0 {
		scraper.maxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}

func newScraperSealed(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	scraper := NewScraperSealed(b)
	scraper.logCallback = opts.LogCallback
	if opts.MaxConcurrency != 0 {
		scraper.maxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}
