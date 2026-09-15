package arcanafrisia

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("arcanafrisia", []mtgban.Game{mtgban.GameMagic},
		func(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.LogCallback = opts.LogCallback
			return scraper, nil
		})
}
