package stompinggrounds

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

func init() {
	mtgban.Register("stompinggrounds",
		[]matcher.Game{
			matcher.GameFleshAndBlood, matcher.GameLorcana, matcher.GameMagic,
			matcher.GameOnePiece, matcher.GamePokemon, matcher.GameRiftbound,
		},
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper, err := NewScraper(b)
			if err != nil {
				return nil, err
			}
			scraper.logCallback = opts.LogCallback
			return scraper, nil
		})
}
