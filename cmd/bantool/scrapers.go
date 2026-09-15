package main

import (
	"errors"
	"log"
	"os"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/cardmarket"
	"github.com/mtgban/go-mtgban/cardtrader"
	"github.com/mtgban/go-mtgban/coolstuffinc"
	"github.com/mtgban/go-mtgban/gamenerdz"
	"github.com/mtgban/go-mtgban/miniaturemarket"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/starcitygames"
	"github.com/mtgban/go-mtgban/strikezone"
	"github.com/mtgban/go-mtgban/tcgplayer"
	"github.com/mtgban/go-mtgban/vegassingles"
)

// The scrapers below price one game each, and a vendor's games differ only in
// the constant naming them. Each family is written once and instanced per
// game at its entry in ScraperOptions, so a change to how a vendor is built
// is made in one place rather than in the six or seven copies a family had.

func tcgplayerCredentials() (publicID, privateID string, err error) {
	publicID = os.Getenv("TCGPLAYER_PUBLIC_KEY")
	privateID = os.Getenv("TCGPLAYER_PRIVATE_KEY")
	if publicID == "" || privateID == "" {
		return "", "", errors.New("missing TCGPLAYER_PUBLIC_KEY or TCGPLAYER_PRIVATE_KEY env vars")
	}
	return publicID, privateID, nil
}

func cardmarketCredentials() (appToken, appSecret string, err error) {
	appToken = os.Getenv("MKM_APP_TOKEN")
	appSecret = os.Getenv("MKM_APP_SECRET")
	if appToken == "" || appSecret == "" {
		return "", "", errors.New("missing MKM_APP_TOKEN or MKM_APP_SECRET env vars")
	}
	return appToken, appSecret, nil
}

func cardtraderToken() (string, error) {
	token := os.Getenv("CARDTRADER_TOKEN_BEARER")
	if token == "" {
		return "", errors.New("missing CARDTRADER_TOKEN_BEARER env var")
	}
	return token, nil
}

func starcitygamesKey() (string, error) {
	key := os.Getenv("SCG_API_KEY")
	if key == "" {
		return "", errors.New("missing SCG_API_KEY env var")
	}
	return key, nil
}

// cardmarketOptionallyBridgedIndexScraper is the bridged scraper for a game
// the bridge only improves. Where Yu-Gi-Oh and Flesh and Blood cannot name
// half their catalog without it, One Piece names most of its shelf from the
// catalog alone and asks the bridge for the printings a collector number
// cannot tell apart. So a cardtrader that will not answer costs those
// printings and nothing else, and the run goes ahead saying so rather than
// failing and pricing nothing.
func cardmarketOptionallyBridgedIndexScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		scraper, err := cardmarket.NewScraperIndex(b)
		if err != nil {
			return nil, err
		}
		err = loadCardmarketCatalog(scraper)
		if err != nil {
			return nil, err
		}
		bridge, err := cardtraderBridge(game)
		if err != nil {
			log.Printf("bridge unavailable, naming what the catalog can on its own: %v", err)
		} else {
			scraper.TCGBridge = bridge
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("MKM_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func cardmarketBridgedIndexScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		scraper, err := cardmarket.NewScraperIndex(b)
		if err != nil {
			return nil, err
		}
		err = loadCardmarketCatalog(scraper)
		if err != nil {
			return nil, err
		}
		scraper.TCGBridge, err = cardtraderBridge(game)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("MKM_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

// tcgSYPScraper reads Store Your Products, which is served per category and
// resolved against the catalog rather than an exported sku file, so every
// game the list covers reads the same way.
func tcgSYPScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		auth := os.Getenv("TCGPLAYER_AUTH")
		if auth == "" {
			return nil, errors.New("missing TCGPLAYER_AUTH env var")
		}
		catalogPath := os.Getenv("TCGPLAYER_CATALOG_PATH")
		if catalogPath == "" {
			return nil, errors.New("missing TCGPLAYER_CATALOG_PATH env var")
		}

		scraper, err := tcgplayer.NewScraperSYP(b, auth)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("TCG_PARTNER")

		// The list names skus and the datastore names products: the catalog
		// dump published beside each game's datastore is the step between,
		// and is the same file the datastore generators are built from.
		reader, err := openPath(catalogPath, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		catalog, err := tcgplayer.LoadSYPCatalog(reader)
		if err != nil {
			return nil, err
		}
		scraper.Catalog = catalog

		return scraper, nil
	}
}

func cardmarketIndexScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		scraper, err := cardmarket.NewScraperIndex(b)
		if err != nil {
			return nil, err
		}
		err = loadCardmarketCatalog(scraper)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("MKM_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

// loadCardmarketCatalog hands an index scraper the published catalog it prices
// from, whatever the game and whichever constructor built it: MTGJSON
// publishes Magic's, go-cardmarket's mkmcatalog builds the rest.
func loadCardmarketCatalog(scraper *cardmarket.Index) error {
	path := os.Getenv("MTGJSON_MKMID_PATH")
	if path == "" {
		return errors.New("missing MTGJSON_MKMID_PATH env var")
	}
	reader, err := openPath(path, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
	if err != nil {
		return err
	}
	defer reader.Close()
	catalog, err := cm.LoadCatalog(reader)
	if err != nil {
		return err
	}
	scraper.Catalog = catalog
	return nil
}

// loadCardmarketMarketCatalog is loadCardmarketCatalog for a Market scraper
// rather than an Index one - the two are unrelated concrete types, so
// nothing but the field they both promote from the shared resolver can be
// written in common between them.
func loadCardmarketMarketCatalog(scraper *cardmarket.Market) error {
	path := os.Getenv("MTGJSON_MKMID_PATH")
	if path == "" {
		return errors.New("missing MTGJSON_MKMID_PATH env var")
	}
	reader, err := openPath(path, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
	if err != nil {
		return err
	}
	defer reader.Close()
	catalog, err := cm.LoadCatalog(reader)
	if err != nil {
		return err
	}
	scraper.Catalog = catalog
	return nil
}

// cardmarketMarketScraper is the plain Market scraper, for the games whose
// catalog resolves without a TCGplayer bridge - the same games
// cardmarketIndexScraper covers without one.
func cardmarketMarketScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		appToken, appSecret, err := cardmarketCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := cardmarket.NewScraperMarket(b, appToken, appSecret)
		if err != nil {
			return nil, err
		}
		err = loadCardmarketMarketCatalog(scraper)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("MKM_PARTNER")
		scraper.BanPriceKey = os.Getenv("BAN_API_KEY")
		return scraper, nil
	}
}

// cardmarketBridgedMarketScraper is cardmarketMarketScraper for the games
// that cannot name half their catalog without the TCGplayer bridge - the
// same games cardmarketBridgedIndexScraper requires one for.
func cardmarketBridgedMarketScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		appToken, appSecret, err := cardmarketCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := cardmarket.NewScraperMarket(b, appToken, appSecret)
		if err != nil {
			return nil, err
		}
		err = loadCardmarketMarketCatalog(scraper)
		if err != nil {
			return nil, err
		}
		scraper.TCGBridge, err = cardtraderBridge(game)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("MKM_PARTNER")
		scraper.BanPriceKey = os.Getenv("BAN_API_KEY")
		return scraper, nil
	}
}

// cardmarketOptionallyBridgedMarketScraper is cardmarketMarketScraper for a
// game the bridge only improves - the same games
// cardmarketOptionallyBridgedIndexScraper degrades gracefully for.
func cardmarketOptionallyBridgedMarketScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		appToken, appSecret, err := cardmarketCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := cardmarket.NewScraperMarket(b, appToken, appSecret)
		if err != nil {
			return nil, err
		}
		err = loadCardmarketMarketCatalog(scraper)
		if err != nil {
			return nil, err
		}
		bridge, err := cardtraderBridge(game)
		if err != nil {
			log.Printf("bridge unavailable, naming what the catalog can on its own: %v", err)
		} else {
			scraper.TCGBridge = bridge
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("MKM_PARTNER")
		scraper.BanPriceKey = os.Getenv("BAN_API_KEY")
		return scraper, nil
	}
}

func cardmarketSealedScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		appToken, appSecret, err := cardmarketCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := cardmarket.NewScraperSealed(b, appToken, appSecret)
		if err != nil {
			return nil, err
		}
		scraper.TCGBridge, err = cardtraderBridge(game)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("MKM_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func cardtraderMarketScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		token, err := cardtraderToken()
		if err != nil {
			return nil, err
		}
		scraper, err := cardtrader.NewScraperMarket(game, token)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.ShareCode = os.Getenv("CT_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func cardtraderSealedScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		token, err := cardtraderToken()
		if err != nil {
			return nil, err
		}
		scraper, err := cardtrader.NewScraperSealed(game, token)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.ShareCode = os.Getenv("CT_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func coolstuffincScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		scraper, err := coolstuffinc.NewScraper(b)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Partner = os.Getenv("CSI_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func coolstuffincSealedScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		scraper, err := coolstuffinc.NewScraperSealed(b)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Partner = os.Getenv("CSI_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func gamenerdzScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		scraper, err := gamenerdz.NewScraper(game)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func miniaturemarketSealedScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		scraper, err := miniaturemarket.NewScraperSealed(game)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("MM_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func starcitygamesScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		apiKey, err := starcitygamesKey()
		if err != nil {
			return nil, err
		}
		scraper, err := starcitygames.NewScraper(b, apiKey)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("SCG_PARTNER")
		return scraper, nil
	}
}

func starcitygamesSealedScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		apiKey, err := starcitygamesKey()
		if err != nil {
			return nil, err
		}
		scraper, err := starcitygames.NewScraperSealed(b, apiKey)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("SCG_PARTNER")
		return scraper, nil
	}
}

func strikezoneScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		scraper, err := strikezone.NewScraper(game)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func tcgIndexScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		publicID, privateID, err := tcgplayerCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := tcgplayer.NewScraperGameIndex(b, publicID, privateID)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("TCG_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func tcgMarketScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		publicID, privateID, err := tcgplayerCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := tcgplayer.NewScraperGame(b, publicID, privateID)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("TCG_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func tcgSealedScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		publicID, privateID, err := tcgplayerCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := tcgplayer.NewScraperGameSealed(b, publicID, privateID)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("TCG_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func vegassinglesScraper(game mtgban.Game) func(*mtgmatcher.Backend) (mtgban.Scraper, error) {
	return func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
		scraper, err := vegassingles.NewScraper(game)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}
