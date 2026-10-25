package sealedev

import (
	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// SecretBanKey is the env var name bantool has always read the BAN price
// API's signature from.
const SecretBanKey = "BAN_API_KEY"

// ResourceTargetProduct names the optional single sealed product filter.
const ResourceTargetProduct = "sealed_ev.target_product"

// WithTargetProduct limits the EV scraper to one product name or UUID.
func WithTargetProduct(product string) mtgban.Option {
	return mtgban.WithResource(ResourceTargetProduct, product)
}

func init() {
	mtgban.Register("sealed_ev", []matcher.Game{matcher.GameMagic}, newScraper)
}

func newScraper(b *matcher.Backend, opts mtgban.Options) (mtgban.Scraper, error) {
	sig, err := opts.Secret(SecretBanKey)
	if err != nil {
		return nil, err
	}

	scraper := NewScraper(b, sig)
	scraper.logCallback = opts.LogCallback
	scraper.affiliate = opts.Affiliate
	scraper.targetEdition = opts.TargetEdition
	targetProduct, err := mtgban.Resource[string](opts, ResourceTargetProduct)
	if err != nil {
		return nil, err
	}
	scraper.targetProduct = targetProduct
	if opts.MaxConcurrency != 0 {
		scraper.maxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}
