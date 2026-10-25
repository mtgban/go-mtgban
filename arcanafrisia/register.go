package arcanafrisia

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

func init() {
	mtgban.Register("arcanafrisia", []matcher.Game{matcher.GameMagic},
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.logCallback = opts.LogCallback
			return scraper, nil
		})
}
