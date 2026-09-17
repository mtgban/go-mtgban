package starcitygames

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// SecretAPIKey is the environment variable name bantool reads the API key
// from.
const SecretAPIKey = "SCG_API_KEY"

// starcitygamesGames are the games Star City Games' catalog covers.
var starcitygamesGames = []mtgban.Game{
	mtgban.GameMagic, mtgban.GameLorcana, mtgban.GameRiftbound, mtgban.GameFleshAndBlood,
}

func init() {
	mtgban.Register("starcitygames", starcitygamesGames,
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			apiKey, err := opts.Secret(SecretAPIKey)
			if err != nil {
				return nil, err
			}
			scraper, err := NewScraper(b, apiKey)
			if err != nil {
				return nil, err
			}
			scraper.logCallback = opts.LogCallback
			scraper.affiliate = opts.Affiliate
			scraper.targetEdition = opts.TargetEdition
			return scraper, nil
		})
	mtgban.Register("starcitygames_sealed", starcitygamesGames,
		func(b *mtgmatcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
			apiKey, err := opts.Secret(SecretAPIKey)
			if err != nil {
				return nil, err
			}
			scraper, err := NewScraperSealed(b, apiKey)
			if err != nil {
				return nil, err
			}
			scraper.logCallback = opts.LogCallback
			scraper.affiliate = opts.Affiliate
			return scraper, nil
		})
}
