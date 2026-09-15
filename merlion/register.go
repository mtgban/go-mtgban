package merlion

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("merlion", []mtgban.Game{mtgban.GameRiftbound},
		func(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper := NewScraper(b)
			scraper.LogCallback = opts.LogCallback
			return scraper, nil
		})
}
