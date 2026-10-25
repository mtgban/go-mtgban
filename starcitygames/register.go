package starcitygames

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// SecretAPIKey is the environment variable name bantool reads the API key
// from.
const SecretAPIKey = "SCG_API_KEY"

// starcitygamesGames are the games Star City Games' catalog covers.
var starcitygamesGames = []matcher.Game{
	matcher.GameMagic, matcher.GameLorcana, matcher.GameRiftbound, matcher.GameFleshAndBlood,
}

func init() {
	mtgban.Register("starcitygames", starcitygamesGames,
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
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
		func(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
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
