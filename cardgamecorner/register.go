package cardgamecorner

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("cardgamecorner",
		[]mtgmatcher.Game{
			mtgmatcher.GameMagic, mtgmatcher.GamePokemon, mtgmatcher.GameYuGiOh,
			mtgmatcher.GameOnePiece, mtgmatcher.GameLorcana, mtgmatcher.GameRiftbound,
			mtgmatcher.GameFleshAndBlood,
		},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper, err := NewScraper(b)
			if err != nil {
				return nil, err
			}
			scraper.logCallback = opts.LogCallback
			if opts.MaxConcurrency != 0 {
				scraper.maxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
}
