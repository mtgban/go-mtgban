package hareruya

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("hareruya", []mtgban.Game{mtgban.GameMagic},
		func(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.LogCallback = opts.LogCallback
			scraper.TargetEdition = opts.TargetEdition
			return scraper, nil
		})
	mtgban.Register("hareruya_sealed", []mtgban.Game{mtgban.GameMagic},
		func(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraperSealed(b)
			scraper.LogCallback = opts.LogCallback
			return scraper, nil
		})
}
