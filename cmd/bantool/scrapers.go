package main

import (
	"errors"
	"log"
	"os"

	"github.com/mtgban/go-mtgban/cardmarket"
	"github.com/mtgban/go-mtgban/cardtrader"
	"github.com/mtgban/go-mtgban/coolstuffinc"
	"github.com/mtgban/go-mtgban/gamenerdz"
	"github.com/mtgban/go-mtgban/miniaturemarket"
	"github.com/mtgban/go-mtgban/mtgban"
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
func cardmarketOptionallyBridgedIndexScraper(game int, bridgedGame int) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		appToken, appSecret, err := cardmarketCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := cardmarket.NewScraperIndex(game, appToken, appSecret)
		if err != nil {
			return nil, err
		}
		bridge, err := cardtraderBridge(bridgedGame)
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
		err = loadCardmarketIDMap(scraper)
		if err != nil {
			return nil, err
		}
		return scraper, nil
	}
}

func cardmarketBridgedIndexScraper(game int, bridgedGame int) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		appToken, appSecret, err := cardmarketCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := cardmarket.NewScraperIndex(game, appToken, appSecret)
		if err != nil {
			return nil, err
		}
		scraper.TCGBridge, err = cardtraderBridge(bridgedGame)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("MKM_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		err = loadCardmarketIDMap(scraper)
		if err != nil {
			return nil, err
		}
		return scraper, nil
	}
}

// tcgSYPScraper reads Store Your Products, which is served per category and
// resolved against the catalog rather than an exported sku file, so every
// game the list covers reads the same way.
func tcgSYPScraper(game string) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		auth := os.Getenv("TCGPLAYER_AUTH")
		if auth == "" {
			return nil, errors.New("missing TCGPLAYER_AUTH env var")
		}
		catalogPath := os.Getenv("TCGPLAYER_CATALOG_PATH")
		if catalogPath == "" {
			return nil, errors.New("missing TCGPLAYER_CATALOG_PATH env var")
		}

		scraper, err := tcgplayer.NewScraperSYP(game, auth)
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

func cardmarketIndexScraper(game int) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		appToken, appSecret, err := cardmarketCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := cardmarket.NewScraperIndex(game, appToken, appSecret)
		if err != nil {
			return nil, err
		}
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("MKM_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		err = loadCardmarketIDMap(scraper)
		if err != nil {
			return nil, err
		}
		return scraper, nil
	}
}

// loadCardmarketIDMap hands an index scraper the published id map when one
// is addressed, whatever the game and whichever constructor built it:
// MTGJSON publishes Magic's, mkmcatalog builds the rest, and the map
// replaces the API crawl.
func loadCardmarketIDMap(scraper *cardmarket.Index) error {
	idMapPath := os.Getenv("MTGJSON_MKMID_PATH")
	if idMapPath == "" {
		return nil
	}
	reader, err := openPath(idMapPath, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
	if err != nil {
		return err
	}
	defer reader.Close()
	idMap, err := cardmarket.LoadIDMap(reader)
	if err != nil {
		return err
	}
	scraper.IDMap = idMap
	return nil
}

// bridgedGame is the CardTrader game whose catalog stands in for a Cardmarket
// one, for the games where Cardmarket publishes no ids of its own.

func cardmarketSealedScraper(game int, bridgedGame int) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		appToken, appSecret, err := cardmarketCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := cardmarket.NewScraperSealed(game, appToken, appSecret)
		if err != nil {
			return nil, err
		}
		scraper.TCGBridge, err = cardtraderBridge(bridgedGame)
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

func cardtraderMarketScraper(game int) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
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

func cardtraderSealedScraper(game int) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
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

func coolstuffincScraper(game string) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		scraper := coolstuffinc.NewScraper(game)
		scraper.LogCallback = GlobalLogCallback
		scraper.Partner = os.Getenv("CSI_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func coolstuffincSealedScraper(game string) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		scraper := coolstuffinc.NewScraperSealed(game)
		scraper.LogCallback = GlobalLogCallback
		scraper.Partner = os.Getenv("CSI_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func gamenerdzScraper(game string) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		scraper := gamenerdz.NewScraper(game)
		scraper.LogCallback = GlobalLogCallback
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func miniaturemarketSealedScraper(game string) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		scraper := miniaturemarket.NewScraperSealed(game)
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("MM_PARTNER")
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func starcitygamesScraper(game int) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		apiKey, err := starcitygamesKey()
		if err != nil {
			return nil, err
		}
		scraper := starcitygames.NewScraper(game, apiKey)
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("SCG_PARTNER")
		return scraper, nil
	}
}

func starcitygamesSealedScraper(game int) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		apiKey, err := starcitygamesKey()
		if err != nil {
			return nil, err
		}
		scraper := starcitygames.NewScraperSealed(game, apiKey)
		scraper.LogCallback = GlobalLogCallback
		scraper.Affiliate = os.Getenv("SCG_PARTNER")
		return scraper, nil
	}
}

func strikezoneScraper(game string) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		scraper := strikezone.NewScraper(game)
		scraper.LogCallback = GlobalLogCallback
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}

func tcgIndexScraper(game string) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		publicID, privateID, err := tcgplayerCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := tcgplayer.NewScraperGameIndex(game, publicID, privateID)
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

func tcgMarketScraper(game string) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		publicID, privateID, err := tcgplayerCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := tcgplayer.NewScraperGame(game, publicID, privateID)
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

func tcgSealedScraper(game string) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		publicID, privateID, err := tcgplayerCredentials()
		if err != nil {
			return nil, err
		}
		scraper, err := tcgplayer.NewScraperGameSealed(game, publicID, privateID)
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

func vegassinglesScraper(game string) func() (mtgban.Scraper, error) {
	return func() (mtgban.Scraper, error) {
		scraper := vegassingles.NewScraper(game)
		scraper.LogCallback = GlobalLogCallback
		if MaxConcurrency != 0 {
			scraper.MaxConcurrency = MaxConcurrency
		}
		return scraper, nil
	}
}
