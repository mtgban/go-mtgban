package cardtrader

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// SecretToken is the environment variable bantool reads the API token from.
const SecretToken = "CARDTRADER_TOKEN_BEARER"

// registeredGames are the games ctGames answers for that bantool schedules a
// Card Trader run under.
var registeredGames = []mtgban.Game{
	mtgban.GameMagic, mtgban.GameLorcana, mtgban.GameRiftbound,
	mtgban.GameOnePiece, mtgban.GamePokemon, mtgban.GameYuGiOh,
	mtgban.GameFleshAndBlood, mtgban.GameGundam,
}

func init() {
	mtgban.Register("cardtrader", registeredGames,
		func(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
			token, err := auth.Secret(SecretToken)
			if err != nil {
				return nil, err
			}
			scraper, err := NewScraperMarket(b, token)
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = opts.LogCallback
			scraper.ShareCode = opts.Affiliate
			if opts.MaxConcurrency != 0 {
				scraper.MaxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})

	mtgban.Register("cardtrader_sealed", registeredGames,
		func(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
			token, err := auth.Secret(SecretToken)
			if err != nil {
				return nil, err
			}
			scraper, err := NewScraperSealed(b, token)
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = opts.LogCallback
			scraper.ShareCode = opts.Affiliate
			if opts.MaxConcurrency != 0 {
				scraper.MaxConcurrency = opts.MaxConcurrency
			}
			return scraper, nil
		})
}
