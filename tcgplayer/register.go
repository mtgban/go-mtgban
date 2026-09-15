package tcgplayer

import (
	"errors"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The environment variable names bantool has always read these secrets
// under.
const (
	SecretPublicKey  = "TCGPLAYER_PUBLIC_KEY"
	SecretPrivateKey = "TCGPLAYER_PRIVATE_KEY"
	SecretAuth       = "TCGPLAYER_AUTH"
)

// The typed resources tcg_market, tcg_sealed and tcg_syplist read out of
// their Options.
const (
	ResourceSKUs       = "tcgplayer.skus"
	ResourceSYPCatalog = "tcgplayer.sypcatalog"
)

// WithSKUs hands the Magic market and sealed scrapers the sku catalog
// LoadTCGSKUs returns, which is what resolves a card to the skus the
// partner API prices. The other games identify a sku by name and number
// instead and read no sku catalog.
func WithSKUs(skus SKUMap) mtgban.Option {
	return mtgban.WithResource(ResourceSKUs, skus)
}

// WithSYPCatalog hands the SYP scraper the catalog LoadSYPCatalog returns,
// which is the step between a sku id the list names and the product and
// finish the datastore knows it by.
func WithSYPCatalog(catalog SYPCatalog) mtgban.Option {
	return mtgban.WithResource(ResourceSYPCatalog, catalog)
}

func tcgCredentials(auth mtgban.Authenticator) (publicID, privateID string, err error) {
	publicID, err = auth.Secret(SecretPublicKey)
	if err != nil {
		return "", "", err
	}
	privateID, err = auth.Secret(SecretPrivateKey)
	if err != nil {
		return "", "", err
	}
	return publicID, privateID, nil
}

// newTCGIndexScraper builds tcg_index: Magic is priced by its own sku-driven
// scraper, every other game by the shared TCGGameIndex.
func newTCGIndexScraper(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
	publicID, privateID, err := tcgCredentials(auth)
	if err != nil {
		return nil, err
	}

	game, err := mtgban.GameOf(b)
	if err != nil {
		return nil, err
	}

	if game == mtgban.GameMagic {
		scraper, err := NewScraperIndex(b, publicID, privateID)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = opts.LogCallback
		scraper.Affiliate = opts.Affiliate
		if opts.MaxConcurrency != 0 {
			scraper.MaxConcurrency = opts.MaxConcurrency
		}
		return scraper, nil
	}

	scraper, err := NewScraperGameIndex(b, publicID, privateID)
	if err != nil {
		return nil, err
	}
	scraper.LogCallback = opts.LogCallback
	scraper.Affiliate = opts.Affiliate
	if opts.MaxConcurrency != 0 {
		scraper.MaxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}

// newTCGMarketScraper builds tcg_market: Magic's Market scraper needs the
// sku catalog to run, every other game's TCGGame resolves by name and
// number alone.
func newTCGMarketScraper(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
	publicID, privateID, err := tcgCredentials(auth)
	if err != nil {
		return nil, err
	}

	game, err := mtgban.GameOf(b)
	if err != nil {
		return nil, err
	}

	if game == mtgban.GameMagic {
		skus, err := mtgban.Resource[SKUMap](opts, ResourceSKUs)
		if err != nil {
			return nil, err
		}
		if skus == nil {
			return nil, errors.New("tcg_market needs WithSKUs on Magic")
		}

		scraper, err := NewScraperMarket(b, publicID, privateID)
		if err != nil {
			return nil, err
		}
		scraper.SKUsData = skus
		scraper.LogCallback = opts.LogCallback
		scraper.Affiliate = opts.Affiliate
		if opts.MaxConcurrency != 0 {
			scraper.MaxConcurrency = opts.MaxConcurrency
		}
		return scraper, nil
	}

	scraper, err := NewScraperGame(b, publicID, privateID)
	if err != nil {
		return nil, err
	}
	scraper.LogCallback = opts.LogCallback
	scraper.Affiliate = opts.Affiliate
	if opts.MaxConcurrency != 0 {
		scraper.MaxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}

// newTCGSealedScraper builds tcg_sealed: Magic's Sealed scraper needs the
// sku catalog the same way its Market sibling does, every other game's
// TCGGame resolves sealed products through the datastore's sealed product
// map instead.
func newTCGSealedScraper(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
	publicID, privateID, err := tcgCredentials(auth)
	if err != nil {
		return nil, err
	}

	game, err := mtgban.GameOf(b)
	if err != nil {
		return nil, err
	}

	if game == mtgban.GameMagic {
		skus, err := mtgban.Resource[SKUMap](opts, ResourceSKUs)
		if err != nil {
			return nil, err
		}
		if skus == nil {
			return nil, errors.New("tcg_sealed needs WithSKUs on Magic")
		}

		scraper, err := NewScraperSealed(b, publicID, privateID)
		if err != nil {
			return nil, err
		}
		scraper.SKUsData = skus
		scraper.LogCallback = opts.LogCallback
		scraper.Affiliate = opts.Affiliate
		if opts.MaxConcurrency != 0 {
			scraper.MaxConcurrency = opts.MaxConcurrency
		}
		return scraper, nil
	}

	scraper, err := NewScraperGameSealed(b, publicID, privateID)
	if err != nil {
		return nil, err
	}
	scraper.LogCallback = opts.LogCallback
	scraper.Affiliate = opts.Affiliate
	if opts.MaxConcurrency != 0 {
		scraper.MaxConcurrency = opts.MaxConcurrency
	}
	return scraper, nil
}

// newTCGSYPListScraper builds tcg_syplist, served the same way for every
// game it covers: the list names a sku and the catalog resolves it to a
// product and finish the datastore knows.
func newTCGSYPListScraper(b *mtgmatcher.Backend, auth mtgban.Authenticator, opts mtgban.Options) (mtgban.Scraper, error) {
	authTicket, err := auth.Secret(SecretAuth)
	if err != nil {
		return nil, err
	}
	catalog, err := mtgban.Resource[SYPCatalog](opts, ResourceSYPCatalog)
	if err != nil {
		return nil, err
	}
	if catalog == nil {
		return nil, errors.New("tcg_syplist needs WithSYPCatalog")
	}

	scraper, err := NewScraperSYP(b, authTicket)
	if err != nil {
		return nil, err
	}
	scraper.Catalog = catalog
	scraper.LogCallback = opts.LogCallback
	scraper.Affiliate = opts.Affiliate
	return scraper, nil
}

func init() {
	mtgban.Register("tcg_index", mtgban.AllGames, newTCGIndexScraper)
	mtgban.Register("tcg_market", mtgban.AllGames, newTCGMarketScraper)
	mtgban.Register("tcg_sealed", mtgban.AllGames, newTCGSealedScraper)
	mtgban.Register("tcg_syplist", []mtgban.Game{mtgban.GameMagic, mtgban.GamePokemon}, newTCGSYPListScraper)
}
