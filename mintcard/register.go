package mintcard

import (
	"errors"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/tcgplayer"
)

// ResourceSKUs is the typed resource the constructor reads its sku catalog
// from: the same catalog tcgplayer's Magic scrapers resolve against, since
// mintcard prices skus TCGplayer itself hosts.
const ResourceSKUs = "mintcard.skus"

// WithSKUs hands the scraper the sku catalog LoadTCGSKUs returns, which is
// required: without it a listing has no sku id to resolve to a printing.
func WithSKUs(skus tcgplayer.SKUMap) mtgban.Option {
	return mtgban.WithResource(ResourceSKUs, skus)
}

func init() {
	mtgban.Register("mintcard", []mtgban.Game{mtgban.GameMagic}, newScraper)
}

func newScraper(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
	skus, err := mtgban.Resource[tcgplayer.SKUMap](opts, ResourceSKUs)
	if err != nil {
		return nil, err
	}
	if skus == nil {
		return nil, errors.New("mintcard needs WithSKUs")
	}

	scraper := NewScraper(b)
	scraper.SKUsData = skus
	scraper.LogCallback = opts.LogCallback
	scraper.Partner = opts.Affiliate
	return scraper, nil
}
