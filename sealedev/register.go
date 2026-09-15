package sealedev

import (
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// SecretBanKey is the env var name bantool has always read the BAN price
// API's signature from.
const SecretBanKey = "BAN_API_KEY"

func init() {
	mtgban.Register("sealed_ev", []mtgban.Game{mtgban.GameMagic}, newScraper)
}

func newScraper(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
	sig, err := auth.Secret(SecretBanKey)
	if err != nil {
		return nil, err
	}

	scraper := NewScraper(b, sig)
	scraper.LogCallback = opts.LogCallback
	scraper.Affiliate = opts.Affiliate
	scraper.BuylistAffiliate = opts.BuylistAffiliate
	if opts.MaxConcurrency != 0 {
		scraper.MaxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}
