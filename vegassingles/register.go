package vegassingles

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("vegassingles", []mtgmatcher.Game{
		mtgmatcher.GameMagic, mtgmatcher.GameRiftbound, mtgmatcher.GameOnePiece,
		mtgmatcher.GamePokemon, mtgmatcher.GameGundam,
	}, func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
		scraper, err := NewScraper(b)
		if err != nil {
			return nil, err
		}
		scraper.logCallback = opts.LogCallback
		scraper.logRetries = opts.LogRetries
		if opts.MaxConcurrency != 0 {
			scraper.maxConcurrency = opts.MaxConcurrency
		}
		return scraper, nil
	})
}
