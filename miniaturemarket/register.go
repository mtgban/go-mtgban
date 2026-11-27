package miniaturemarket

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// sealedGames are the games "miniaturemarket_sealed" registers for.
var sealedGames = []matcher.Game{
	matcher.GameMagic, matcher.GameLorcana, matcher.GameRiftbound,
	matcher.GameOnePiece, matcher.GameFleshAndBlood, matcher.GameGundam,
}

func init() {
	mtgban.Register("miniaturemarket_sealed", sealedGames,
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			scraper, err := NewScraperSealed(b)
			if err != nil {
				return nil, err
			}
			scraper.logCallback = opts.LogCallback
			scraper.affiliate = opts.Affiliate
			if opts.MaxConcurrency != 0 {
				scraper.maxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
}
