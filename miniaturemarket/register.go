package miniaturemarket

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func init() {
	mtgban.Register("miniaturemarket_sealed",
		[]mtgban.Game{
			mtgban.GameMagic, mtgban.GameLorcana, mtgban.GameRiftbound,
			mtgban.GameOnePiece, mtgban.GameFleshAndBlood, mtgban.GameGundam,
		},
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper, err := NewScraperSealed(b)
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = opts.LogCallback
			scraper.Affiliate = opts.Affiliate
			if opts.MaxConcurrency != 0 {
				scraper.MaxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
}
