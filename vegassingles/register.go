package vegassingles

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("vegassingles", []mtgban.Game{
		mtgban.GameMagic, mtgban.GameRiftbound, mtgban.GameOnePiece,
		mtgban.GamePokemon, mtgban.GameGundam,
	}, func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
		scraper, err := NewScraper(b)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = opts.LogCallback
		if opts.MaxConcurrency != 0 {
			scraper.MaxConcurrency = opts.MaxConcurrency
		}
		return scraper, nil
	})
}
