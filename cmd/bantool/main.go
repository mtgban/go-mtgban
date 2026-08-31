// Command bantool runs the scrapers and writes what they return, either to
// disk or to a cloud bucket, one file per scraper and price kind.
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/scizorman/go-ndjson"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mtgban/go-mtgban/abugames"
	"github.com/mtgban/go-mtgban/arcanafrisia"
	"github.com/mtgban/go-mtgban/cardkingdom"
	"github.com/mtgban/go-mtgban/cardmarket"
	"github.com/mtgban/go-mtgban/cardtrader"
	"github.com/mtgban/go-mtgban/coolstuffinc"
	"github.com/mtgban/go-mtgban/hareruya"
	"github.com/mtgban/go-mtgban/magiccorner"
	"github.com/mtgban/go-mtgban/manapool"
	"github.com/mtgban/go-mtgban/merlion"
	"github.com/mtgban/go-mtgban/miniaturemarket"
	"github.com/mtgban/go-mtgban/mintcard"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgseattle"
	"github.com/mtgban/go-mtgban/sealedev"
	"github.com/mtgban/go-mtgban/starcitygames"
	"github.com/mtgban/go-mtgban/strikezone"
	"github.com/mtgban/go-mtgban/tcgplayer"
	"github.com/mtgban/go-mtgban/trollandtoad"
	"github.com/mtgban/go-mtgban/vegassingles"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/simplecloud"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// GlobalLogCallback is handed to every scraper, so one run logs in one place.
var GlobalLogCallback mtgban.LogCallbackFunc = log.Printf

// MaxConcurrency overrides each scraper's own default when set, so a whole run
// can be slowed down at once.
var MaxConcurrency int

// Commit is the revision this binary was built from, read out of the build
// info rather than stamped at link time.
var Commit = func() string {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value
			}
		}
	}
	return ""
}()

type scraperOption struct {
	Enabled    bool
	OnlySeller bool
	OnlyVendor bool
	Init       func() (mtgban.Scraper, error)
}

// cardtraderBridge maps every Cardmarket product id to the TCGplayer id of
// the same product, read off cardtrader's blueprints - the one source
// linking the two marketplaces' ids. The blueprints cover singles and
// sealed alike, so the same bridge serves both cardmarket scrapers; they
// receive it as plain data, and the composition of the two vendors happens
// here and nowhere else.
func cardtraderBridge(gameID int) (map[int]int, error) {
	ctTokenBearer := os.Getenv("CARDTRADER_TOKEN_BEARER")
	if ctTokenBearer == "" {
		return nil, errors.New("missing CARDTRADER_TOKEN_BEARER env var")
	}
	client := cardtrader.NewCTAuthClient(ctTokenBearer)

	blueprints, _, err := cardtrader.BlueprintsForGame(context.Background(), client, gameID, "", log.Printf)
	if err != nil {
		return nil, err
	}

	bridge := map[int]int{}
	for _, bp := range blueprints {
		if bp.TCGplayerID == 0 {
			continue
		}
		for _, mkmID := range bp.CardMarketIDs {
			bridge[mkmID] = bp.TCGplayerID
		}
	}
	log.Printf("bridge: %d cardmarket ids linked to a tcgplayer id", len(bridge))
	return bridge, nil
}

func init() {
	MaxConcurrency, _ = strconv.Atoi(os.Getenv("MAX_CONCURRENCY"))

	log.Println("Workers running with", MaxConcurrency, "parallel threads")
}

var options = map[string]*scraperOption{
	"abugames": {
		Init: func() (mtgban.Scraper, error) {
			scraper := abugames.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"abugames_sealed": {
		Init: func() (mtgban.Scraper, error) {
			scraper := abugames.NewScraperSealed()
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"arcanafrisia": {
		Init: func() (mtgban.Scraper, error) {
			scraper := arcanafrisia.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"cardkingdom": {
		Init: func() (mtgban.Scraper, error) {
			scraper := cardkingdom.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CK_PARTNER")
			scraper.PreserveOOS = true
			return scraper, nil
		},
	},
	"cardkingdom_sealed": {
		Init: func() (mtgban.Scraper, error) {
			scraper := cardkingdom.NewScraperSealed()
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CK_PARTNER")
			scraper.PreserveOOS = true
			return scraper, nil
		},
	},
	"cardkingdom_graded": {
		Init: func() (mtgban.Scraper, error) {
			scraper, err := cardkingdom.NewScraperGraded()
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CK_PARTNER")
			return scraper, nil
		},
	},
	"cardmarket": {
		Init: cardmarketIndexScraper(cardmarket.GameMagic),
	},
	"cardmarket_sealed": {
		Init: func() (mtgban.Scraper, error) {
			mkmAppToken := os.Getenv("MKM_APP_TOKEN")
			mkmAppSecret := os.Getenv("MKM_APP_SECRET")
			if mkmAppToken == "" || mkmAppSecret == "" {
				return nil, errors.New("missing MKM_APP_TOKEN or MKM_APP_SECRET env vars")
			}

			scraper, err := cardmarket.NewScraperSealed(cardmarket.GameMagic, mkmAppToken, mkmAppSecret)
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("MKM_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"cardtrader": {
		Init: cardtraderMarketScraper(cardtrader.GameMagic),
	},
	"cardtrader_sealed": {
		Init: cardtraderSealedScraper(cardtrader.GameMagic),
	},
	"coolstuffinc": {
		Init: coolstuffincScraper(coolstuffinc.GameMagic),
	},
	"coolstuffinc_sealed": {
		Init: coolstuffincSealedScraper(coolstuffinc.GameMagic),
	},
	"hareruya": {
		Init: func() (mtgban.Scraper, error) {
			scraper := hareruya.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"hareruya_sealed": {
		Init: func() (mtgban.Scraper, error) {
			scraper := hareruya.NewScraperSealed()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"magiccorner": {
		Init: func() (mtgban.Scraper, error) {
			scraper, err := magiccorner.NewScraper()
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"manapool": {
		Init: func() (mtgban.Scraper, error) {
			scraper := manapool.NewScraper()
			scraper.Partner = os.Getenv("MP_PARTNER")
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"manapool_index": {
		Init: func() (mtgban.Scraper, error) {
			scraper := manapool.NewScraperIndex()
			scraper.Partner = os.Getenv("MP_PARTNER")
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"manapool_sealed": {
		Init: func() (mtgban.Scraper, error) {
			scraper := manapool.NewScraperSealed()
			scraper.Partner = os.Getenv("MP_PARTNER")
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"miniaturemarket_sealed_lorcana": {
		Init: miniaturemarketSealedScraper(miniaturemarket.GameLorcana),
	},
	"miniaturemarket_sealed_riftbound": {
		Init: miniaturemarketSealedScraper(miniaturemarket.GameRiftbound),
	},
	"miniaturemarket_sealed_onepiece": {
		Init: miniaturemarketSealedScraper(miniaturemarket.GameOnePiece),
	},
	"miniaturemarket_sealed_fleshandblood": {
		Init: miniaturemarketSealedScraper(miniaturemarket.GameFleshAndBlood),
	},
	"miniaturemarket_sealed": {
		Init: func() (mtgban.Scraper, error) {
			scraper := miniaturemarket.NewScraperSealed(miniaturemarket.GameMagic)
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"mintcard": {
		Init: func() (mtgban.Scraper, error) {
			tcgSKUPath := os.Getenv("MTGJSON_TCGSKU_PATH")
			if tcgSKUPath == "" {
				return nil, errors.New("missing MTGJSON_TCGSKU_PATH env var")
			}

			scraper := mintcard.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("MINT_PARTNER")

			start := time.Now()
			skuReader, err := openPath(tcgSKUPath, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
			if err != nil {
				return nil, err
			}
			defer skuReader.Close()
			skus, err := tcgplayer.LoadTCGSKUs(skuReader)
			if err != nil {
				return nil, err
			}
			scraper.SKUsData = skus
			log.Println("loading skus took:", time.Since(start))

			return scraper, nil
		},
	},
	"mtgseattle": {
		OnlySeller: true,
		Init: func() (mtgban.Scraper, error) {
			scraper := mtgseattle.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"sealed_ev": {
		Init: func() (mtgban.Scraper, error) {
			banKey := os.Getenv("BAN_API_KEY")
			if banKey == "" {
				return nil, errors.New("missing BAN_API_KEY env var")
			}
			scraper := sealedev.NewScraper(banKey)
			scraper.Affiliate = os.Getenv("TCG_PARTNER")
			scraper.BuylistAffiliate = os.Getenv("CK_PARTNER")
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"starcitygames": {
		Init: starcitygamesScraper(starcitygames.GameMagic),
	},
	"starcitygames_sealed": {
		Init: starcitygamesSealedScraper(starcitygames.GameMagic),
	},
	"strikezone": {
		Init: strikezoneScraper(strikezone.GameMagic),
	},
	"tcg_index": {
		Init: func() (mtgban.Scraper, error) {
			tcgPublicID := os.Getenv("TCGPLAYER_PUBLIC_KEY")
			tcgPrivateID := os.Getenv("TCGPLAYER_PRIVATE_KEY")
			if tcgPublicID == "" || tcgPrivateID == "" {
				return nil, errors.New("missing TCGPLAYER_PUBLIC_KEY or TCGPLAYER_PRIVATE_KEY env vars")
			}

			scraper, err := tcgplayer.NewScraperIndex(tcgPublicID, tcgPrivateID)
			if err != nil {
				return nil, err
			}

			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("TCG_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"tcg_market": {
		Init: func() (mtgban.Scraper, error) {
			tcgPublicID := os.Getenv("TCGPLAYER_PUBLIC_KEY")
			tcgPrivateID := os.Getenv("TCGPLAYER_PRIVATE_KEY")
			tcgSKUPath := os.Getenv("MTGJSON_TCGSKU_PATH")
			if tcgPublicID == "" || tcgPrivateID == "" || tcgSKUPath == "" {
				return nil, errors.New("missing TCGPLAYER_PUBLIC_KEY or TCGPLAYER_PRIVATE_KEY or MTGJSON_TCGSKU_PATH env vars")
			}

			scraper, err := tcgplayer.NewScraperMarket(tcgPublicID, tcgPrivateID)
			if err != nil {
				return nil, err
			}

			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("TCG_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}

			start := time.Now()
			skuReader, err := openPath(tcgSKUPath, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
			if err != nil {
				return nil, err
			}
			defer skuReader.Close()
			skus, err := tcgplayer.LoadTCGSKUs(skuReader)
			if err != nil {
				return nil, err
			}
			scraper.SKUsData = skus
			log.Println("loading skus took:", time.Since(start))

			return scraper, nil
		},
	},
	"tcg_sealed": {
		Init: func() (mtgban.Scraper, error) {
			tcgPublicID := os.Getenv("TCGPLAYER_PUBLIC_KEY")
			tcgPrivateID := os.Getenv("TCGPLAYER_PRIVATE_KEY")
			tcgSKUPath := os.Getenv("MTGJSON_TCGSKU_PATH")
			if tcgPublicID == "" || tcgPrivateID == "" || tcgSKUPath == "" {
				return nil, errors.New("missing TCGPLAYER_PUBLIC_KEY or TCGPLAYER_PRIVATE_KEY or MTGJSON_TCGSKU_PATH env vars")
			}

			scraper, err := tcgplayer.NewScraperSealed(tcgPublicID, tcgPrivateID)
			if err != nil {
				return nil, err
			}

			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("TCG_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}

			start := time.Now()
			skuReader, err := openPath(tcgSKUPath, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
			if err != nil {
				return nil, err
			}
			defer skuReader.Close()
			skus, err := tcgplayer.LoadTCGSKUs(skuReader)
			if err != nil {
				return nil, err
			}
			scraper.SKUsData = skus
			log.Println("loading skus took:", time.Since(start))

			return scraper, nil
		},
	},
	"tcg_syplist": {
		Init: func() (mtgban.Scraper, error) {
			tcgAuth := os.Getenv("TCGPLAYER_AUTH")
			tcgSKUPath := os.Getenv("MTGJSON_TCGSKU_PATH")
			if tcgAuth == "" || tcgSKUPath == "" {
				return nil, errors.New("missing TCGPLAYER_AUTH or MTGJSON_TCGSKU_PATH env var")
			}

			scraper, err := tcgplayer.NewScraperSYP(mtgban.GameMagic, tcgAuth)
			if err != nil {
				return nil, err
			}

			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("TCG_PARTNER")

			start := time.Now()
			skuReader, err := openPath(tcgSKUPath, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
			if err != nil {
				return nil, err
			}
			defer skuReader.Close()
			skus, err := tcgplayer.LoadTCGSKUs(skuReader)
			if err != nil {
				return nil, err
			}
			scraper.SKUsData = skus
			log.Println("loading skus took:", time.Since(start))

			return scraper, nil
		},
	},
	"trollandtoad": {
		Init: func() (mtgban.Scraper, error) {
			scraper := trollandtoad.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"cardmarket_lorcana": {
		Init: cardmarketIndexScraper(cardmarket.GameLorcana),
	},
	"cardtrader_lorcana": {
		Init: cardtraderMarketScraper(cardtrader.GameLorcana),
	},
	"cardmarket_sealed_lorcana": {
		Init: cardmarketSealedScraper(cardmarket.GameLorcana, cardtrader.GameLorcana),
	},
	"cardtrader_sealed_lorcana": {
		Init: cardtraderSealedScraper(cardtrader.GameLorcana),
	},
	"coolstuffinc_sealed_lorcana": {
		Init: coolstuffincSealedScraper(coolstuffinc.GameLorcana),
	},
	"coolstuffinc_lorcana": {
		Init: coolstuffincScraper(coolstuffinc.GameLorcana),
	},
	"starcitygames_sealed_lorcana": {
		Init: starcitygamesSealedScraper(starcitygames.GameLorcana),
	},
	"starcitygames_lorcana": {
		Init: starcitygamesScraper(starcitygames.GameLorcana),
	},
	"strikezone_lorcana": {
		Init: strikezoneScraper(strikezone.GameLorcana),
	},
	"strikezone_pokemon": {
		Init: strikezoneScraper(strikezone.GamePokemon),
	},
	"strikezone_fleshandblood": {
		Init: strikezoneScraper(strikezone.GameFleshAndBlood),
	},
	"tcg_index_lorcana": {
		Init: tcgIndexScraper(mtgban.GameLorcana),
	},
	"tcg_market_lorcana": {
		Init: tcgMarketScraper(mtgban.GameLorcana),
	},
	"tcg_sealed_lorcana": {
		Init: tcgSealedScraper(mtgban.GameLorcana),
	},

	"cardmarket_riftbound": {
		Init: cardmarketIndexScraper(cardmarket.GameRiftbound),
	},
	"cardmarket_onepiece": {
		Init: cardmarketIndexScraper(cardmarket.GameOnePiece),
	},
	"cardmarket_yugioh": {
		Init: cardmarketBridgedIndexScraper(cardmarket.GameYuGiOh, cardtrader.GameYuGiOh),
	},
	"cardmarket_fleshandblood": {
		Init: cardmarketBridgedIndexScraper(cardmarket.GameFleshAndBlood, cardtrader.GameFleshAndBlood),
	},
	"cardmarket_pokemon": {
		Init: cardmarketBridgedIndexScraper(cardmarket.GamePokemon, cardtrader.GamePokemon),
	},
	"cardtrader_riftbound": {
		Init: cardtraderMarketScraper(cardtrader.GameRiftbound),
	},
	"cardtrader_onepiece": {
		Init: cardtraderMarketScraper(cardtrader.GameOnePiece),
	},
	"cardtrader_yugioh": {
		Init: cardtraderMarketScraper(cardtrader.GameYuGiOh),
	},
	"cardtrader_fleshandblood": {
		Init: cardtraderMarketScraper(cardtrader.GameFleshAndBlood),
	},
	"cardtrader_pokemon": {
		Init: cardtraderMarketScraper(cardtrader.GamePokemon),
	},
	"cardmarket_sealed_riftbound": {
		Init: cardmarketSealedScraper(cardmarket.GameRiftbound, cardtrader.GameRiftbound),
	},
	"cardmarket_sealed_onepiece": {
		Init: cardmarketSealedScraper(cardmarket.GameOnePiece, cardtrader.GameOnePiece),
	},
	"cardmarket_sealed_yugioh": {
		Init: cardmarketSealedScraper(cardmarket.GameYuGiOh, cardtrader.GameYuGiOh),
	},
	"cardmarket_sealed_fleshandblood": {
		Init: cardmarketSealedScraper(cardmarket.GameFleshAndBlood, cardtrader.GameFleshAndBlood),
	},
	"cardmarket_sealed_pokemon": {
		Init: cardmarketSealedScraper(cardmarket.GamePokemon, cardtrader.GamePokemon),
	},
	"cardtrader_sealed_riftbound": {
		Init: cardtraderSealedScraper(cardtrader.GameRiftbound),
	},
	"cardtrader_sealed_onepiece": {
		Init: cardtraderSealedScraper(cardtrader.GameOnePiece),
	},
	"cardtrader_sealed_yugioh": {
		Init: cardtraderSealedScraper(cardtrader.GameYuGiOh),
	},
	"cardtrader_sealed_fleshandblood": {
		Init: cardtraderSealedScraper(cardtrader.GameFleshAndBlood),
	},
	"cardtrader_sealed_pokemon": {
		Init: cardtraderSealedScraper(cardtrader.GamePokemon),
	},
	"coolstuffinc_sealed_riftbound": {
		Init: coolstuffincSealedScraper(coolstuffinc.GameRiftbound),
	},
	"coolstuffinc_onepiece": {
		Init: coolstuffincScraper(coolstuffinc.GameOnePiece),
	},
	"coolstuffinc_pokemon": {
		Init: coolstuffincScraper(coolstuffinc.GamePokemon),
	},
	"coolstuffinc_yugioh": {
		Init: coolstuffincScraper(coolstuffinc.GameYuGiOh),
	},
	"coolstuffinc_sealed_onepiece": {
		Init: coolstuffincSealedScraper(coolstuffinc.GameOnePiece),
	},
	"coolstuffinc_sealed_pokemon": {
		Init: coolstuffincSealedScraper(coolstuffinc.GamePokemon),
	},
	"coolstuffinc_sealed_yugioh": {
		Init:       coolstuffincSealedScraper(coolstuffinc.GameYuGiOh),
		OnlySeller: true,
	},
	"coolstuffinc_riftbound": {
		Init: coolstuffincScraper(coolstuffinc.GameRiftbound),
	},
	"merlion_riftbound": {
		Init: func() (mtgban.Scraper, error) {
			scraper := merlion.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"starcitygames_sealed_riftbound": {
		Init: starcitygamesSealedScraper(starcitygames.GameRiftbound),
	},
	"starcitygames_riftbound": {
		Init: starcitygamesScraper(starcitygames.GameRiftbound),
	},
	"starcitygames_sealed_fleshandblood": {
		Init: starcitygamesSealedScraper(starcitygames.GameFleshAndBlood),
	},
	"starcitygames_fleshandblood": {
		Init: starcitygamesScraper(starcitygames.GameFleshAndBlood),
	},
	"tcg_index_riftbound": {
		Init: tcgIndexScraper(mtgban.GameRiftbound),
	},
	"tcg_index_onepiece": {
		Init: tcgIndexScraper(mtgban.GameOnePiece),
	},
	"tcg_index_yugioh": {
		Init: tcgIndexScraper(mtgban.GameYuGiOh),
	},
	"tcg_index_fleshandblood": {
		Init: tcgIndexScraper(mtgban.GameFleshAndBlood),
	},
	"tcg_index_pokemon": {
		Init: tcgIndexScraper(mtgban.GamePokemon),
	},
	"tcg_market_riftbound": {
		Init: tcgMarketScraper(mtgban.GameRiftbound),
	},
	"vegassingles_riftbound": {
		Init: vegassinglesScraper(vegassingles.GameRiftbound),
	},
	"tcg_market_onepiece": {
		Init: tcgMarketScraper(mtgban.GameOnePiece),
	},
	"vegassingles_onepiece": {
		Init: vegassinglesScraper(vegassingles.GameOnePiece),
	},
	"tcg_market_yugioh": {
		Init: tcgMarketScraper(mtgban.GameYuGiOh),
	},
	"tcg_market_fleshandblood": {
		Init: tcgMarketScraper(mtgban.GameFleshAndBlood),
	},
	"tcg_market_pokemon": {
		Init: tcgMarketScraper(mtgban.GamePokemon),
	},
	"vegassingles_pokemon": {
		Init: vegassinglesScraper(vegassingles.GamePokemon),
	},
	"tcg_sealed_riftbound": {
		Init: tcgSealedScraper(mtgban.GameRiftbound),
	},
	"tcg_sealed_onepiece": {
		Init: tcgSealedScraper(mtgban.GameOnePiece),
	},
	"tcg_sealed_yugioh": {
		Init: tcgSealedScraper(mtgban.GameYuGiOh),
	},
	"tcg_sealed_fleshandblood": {
		Init: tcgSealedScraper(mtgban.GameFleshAndBlood),
	},
	"tcg_sealed_pokemon": {
		Init: tcgSealedScraper(mtgban.GamePokemon),
	},
}

type inventoryElement struct {
	UUID string
	mtgban.InventoryEntry
}

type buylistElement struct {
	UUID string
	mtgban.BuylistEntry
}

func writeSellerToNDJSON(seller mtgban.Seller, w io.Writer) error {
	inventory := seller.Inventory()

	var inventoryFlat []inventoryElement
	for uuid, entries := range inventory {
		for _, entry := range entries {
			inventoryFlat = append(inventoryFlat, inventoryElement{
				UUID:           uuid,
				InventoryEntry: entry,
			})
		}
	}

	output, err := ndjson.Marshal(inventoryFlat)
	if err != nil {
		return err
	}

	_, err = w.Write(output)
	return err
}

func writeVendorToNDJSON(vendor mtgban.Vendor, w io.Writer) error {
	buylist := vendor.Buylist()

	var buylistFlat []buylistElement
	for uuid, entries := range buylist {
		for _, entry := range entries {
			buylistFlat = append(buylistFlat, buylistElement{
				UUID:         uuid,
				BuylistEntry: entry,
			})
		}
	}

	output, err := ndjson.Marshal(buylistFlat)
	if err != nil {
		return err
	}

	_, err = w.Write(output)
	return err
}

func dumpSeller(dataBucket simplecloud.Writer, seller mtgban.Seller, outputPath, format string) (err error) {
	if len(seller.Inventory()) == 0 {
		return fmt.Errorf("seller %s has no data", seller.Info().Shorthand)
	}

	target := fmt.Sprintf("%s/retail/%s.%s", outputPath, seller.Info().Shorthand, format)
	log.Println("Writing", target)

	writer, err := simplecloud.InitWriter(context.Background(), dataBucket, target)
	if err != nil {
		return err
	}
	// Close is where a buffered cloud writer commits the upload, so it
	// reports whether anything was durably written at all
	defer func() {
		cerr := writer.Close()
		if err == nil {
			err = cerr
		}
	}()

	switch strings.Split(format, ".")[0] {
	case "json":
		err = mtgban.WriteSellerToJSON(seller, writer)
	case "csv":
		err = mtgban.WriteInventoryToCSV(seller.Inventory(), writer)
	case "ndjson":
		err = writeSellerToNDJSON(seller, writer)
	default:
		err = errors.New("invalid format")
	}

	return err
}

func dumpVendor(dataBucket simplecloud.Writer, vendor mtgban.Vendor, outputPath, format string) (err error) {
	if len(vendor.Buylist()) == 0 {
		return fmt.Errorf("vendor %s has no data", vendor.Info().Shorthand)
	}

	target := fmt.Sprintf("%s/buylist/%s.%s", outputPath, vendor.Info().Shorthand, format)
	log.Println("Writing", target)

	writer, err := simplecloud.InitWriter(context.Background(), dataBucket, target)
	if err != nil {
		return err
	}
	// Close is where a buffered cloud writer commits the upload, so it
	// reports whether anything was durably written at all
	defer func() {
		cerr := writer.Close()
		if err == nil {
			err = cerr
		}
	}()

	switch strings.Split(format, ".")[0] {
	case "json":
		err = mtgban.WriteVendorToJSON(vendor, writer)
	case "csv":
		err = mtgban.WriteBuylistToCSV(vendor.Buylist(), vendor.Info().CreditMultiplier, writer)
	case "ndjson":
		err = writeVendorToNDJSON(vendor, writer)
	default:
		err = errors.New("invalid format")
	}

	return err
}

// countResults sums the distinct cards each unfolded half holds, so a run
// can say what it found before writing any of it.
func countResults(sellers []mtgban.Seller, vendors []mtgban.Vendor) (int, int) {
	var retail, buylist int
	for _, seller := range sellers {
		retail += len(seller.Inventory())
	}
	for _, vendor := range vendors {
		buylist += len(vendor.Buylist())
	}
	return retail, buylist
}

// reportSuspectPricings says which cards a scraper priced on both sides at
// prices too close to be the same printing. A shop's buy price sits well under
// what it asks, so a pairing that does not is usually two products meeting on
// one id - and nothing in the run log says so otherwise, because both halves
// resolved perfectly well on their own.
func reportSuspectPricings(sellers []mtgban.Seller, vendors []mtgban.Vendor) {
	for _, seller := range sellers {
		for _, vendor := range vendors {
			if seller.Info().Shorthand != vendor.Info().Shorthand {
				continue
			}

			suspects := mtgban.SuspectPricings(seller.Inventory(), vendor.Buylist(), mtgban.SuspectRatioThreshold)
			if len(suspects) == 0 {
				continue
			}

			log.Printf("[%s] %d cards are bought at %.0f%% or more of their asking price",
				seller.Info().Shorthand, len(suspects), mtgban.SuspectRatioThreshold)
			for _, suspect := range suspects {
				card, _ := mtgmatcher.GetUUID(suspect.CardID)
				log.Printf("[%s] - %.0f%% buy $%.2f ask $%.2f %s %s",
					seller.Info().Shorthand, suspect.Ratio, suspect.BuyPrice, suspect.Price,
					suspect.Conditions, card)
			}
		}
	}
}

func dump(dataBucket simplecloud.Writer, sellers []mtgban.Seller, vendors []mtgban.Vendor, outputPath, format string, meta bool) []error {
	log.Println("Writing results to", outputPath)

	var sellerErrs []error
	for _, seller := range sellers {
		err := dumpSeller(dataBucket, seller, outputPath, format)
		if err != nil {
			log.Println(err)
			sellerErrs = append(sellerErrs, err)
			continue
		}

		if meta && format != "json" {
			sellerMeta := mtgban.NewSellerFromInventory(nil, seller.Info())
			err := dumpSeller(dataBucket, sellerMeta, outputPath, "json")
			if err != nil {
				sellerErrs = append(sellerErrs, err)
				continue
			}
		}
	}

	var vendorErrs []error
	for _, vendor := range vendors {
		err := dumpVendor(dataBucket, vendor, outputPath, format)
		if err != nil {
			log.Println(err)
			vendorErrs = append(vendorErrs, err)
			continue
		}

		if meta && format != "json" {
			vendorMeta := mtgban.NewVendorFromBuylist(nil, vendor.Info())
			err := dumpVendor(dataBucket, vendorMeta, outputPath, "json")
			if err != nil {
				vendorErrs = append(vendorErrs, err)
				continue
			}
		}
	}

	return append(sellerErrs, vendorErrs...)
}

// HTTPBucket reads a datastore over plain HTTP, for the files served rather
// than kept in a cloud bucket. It implements only the reading half.
type HTTPBucket struct {
	Client *http.Client
	URL    *url.URL
}

// NewHTTPBucket returns a bucket rooted at the given URL.
func NewHTTPBucket(client *http.Client, path string) (*HTTPBucket, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	return &HTTPBucket{
		Client: client,
		URL:    u,
	}, nil
}

// NewReader opens the object at path for reading.
func (h *HTTPBucket) NewReader(ctx context.Context, path string) (io.ReadCloser, error) {
	u := new(url.URL)
	*u = *h.URL
	if h.URL.User != nil {
		u.User = new(url.Userinfo)
		*u.User = *h.URL.User
	}

	u.Path = path

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := h.Client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// NewWriter always fails: nothing can be written back over plain HTTP.
func (h *HTTPBucket) NewWriter(ctx context.Context, path string) (io.WriteCloser, error) {
	return nil, errors.New("an http bucket cannot be written to")
}

// concurrentDownloads is how many ranged parts B2 is asked for at once.
const concurrentDownloads = 20

// openPath reads the object at path, resolving the scheme the same way
// initializeBucket does and taking the same trailing environment values:
// one names a GCS service account, two are a B2 key pair. Reading needs no
// bucket of its own, so a path that is only ever read goes through here
// rather than being built into one first.
func openPath(path string, env ...string) (io.ReadCloser, error) {
	opts := []simplecloud.OpenOption{
		simplecloud.WithHTTPClient(cleanhttp.DefaultClient()),
		simplecloud.WithConcurrentDownloads(concurrentDownloads),
		simplecloud.WithS3Credentials(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY")),
		simplecloud.WithS3Endpoint(os.Getenv("AWS_ENDPOINT")),
	}
	if len(env) > 0 {
		opts = append(opts, simplecloud.WithGCSServiceAccount(env[0]))
	}
	if len(env) > 1 {
		opts = append(opts, simplecloud.WithB2Credentials(env[0], env[1]))
	}
	return simplecloud.Open(context.Background(), path, opts...)
}

func initializeBucket(outputPath string, env ...string) (simplecloud.ReadWriter, error) {
	u, err := url.Parse(outputPath)
	if err != nil {
		return nil, err
	}

	var bucket simplecloud.ReadWriter

	switch u.Scheme {
	case "":
		_, err := os.Stat(u.Path)
		if os.IsNotExist(err) {
			return nil, errors.New("path does not exist")
		}
		bucket = &simplecloud.FileBucket{}
	case "http", "https":
		bucket, err = NewHTTPBucket(cleanhttp.DefaultClient(), outputPath)
		if err != nil {
			return nil, err
		}
	case "gs":
		if len(env) < 1 {
			return nil, errors.New("missing required environment variable")
		}
		serviceAcc := env[0]

		bucket, err = simplecloud.NewGCSClient(context.Background(), serviceAcc, u.Host)
		if err != nil {
			return nil, err
		}
	case "b2":
		if len(env) < 2 {
			return nil, errors.New("missing required environment variables")
		}
		accessKey := env[0]
		secretKey := env[1]

		b2Bucket, err := simplecloud.NewB2Client(context.Background(), accessKey, secretKey, u.Host)
		if err != nil {
			return nil, err
		}
		b2Bucket.ConcurrentDownloads = concurrentDownloads
		bucket = b2Bucket
	case "s3":
		accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
		secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
		endpoint := os.Getenv("AWS_ENDPOINT")

		bucket, err = simplecloud.NewS3Client(context.Background(), accessKey, secretKey, u.Host, endpoint, "")
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported path scheme %s", u.Scheme)
	}

	return bucket, nil
}

func run() int {
	start := time.Now()

	for key, val := range options {
		label := key
		if label != "" {
			label = strings.ToUpper(label[:1]) + label[1:]
		}
		flag.BoolVar(&val.Enabled, key, false, "Enable "+label)
	}

	datastoreOpt := flag.String("datastore", "", "Path to AllPrintings file")
	outputPathOpt := flag.String("output-path", "", "Path where to dump results")

	scrapersOpt := flag.String("scrapers", "", "Comma-separated list of scrapers to enable")
	sellersOpt := flag.String("sellers", "", "Comma-separated list of sellers to enable")
	vendorsOpt := flag.String("vendors", "", "Comma-separated list of vendors to enable")

	fileFormatOpt := flag.String("format", "json", "File format of the output files (json/csv/ndjson)")
	metaOpt := flag.Bool("meta", false, "When format is not json, output a second file for scraper metadata")

	signOpt := flag.String("sign", "", "Sign input")
	versionOpt := flag.Bool("v", false, "Print version information")
	flag.Parse()

	log.Println("bantool version", Commit)
	if *versionOpt {
		return 0
	}

	if *signOpt != "" {
		sig, err := signAPI(*signOpt)
		if err != nil {
			log.Println(err)
			return 1
		}
		fmt.Fprintln(os.Stdout, *signOpt+"?sig="+sig)
		return 0
	}

	switch strings.Split(*fileFormatOpt, ".")[0] {
	case "json", "csv", "ndjson":
	default:
		log.Println("Invalid -format option, see -h for supported values")
		return 1
	}

	if *outputPathOpt == "" {
		log.Println("Missing output-path argument")
		return 1
	}

	dataBucket, err := initializeBucket(*outputPathOpt, os.Getenv("B2_KEY_ID"), os.Getenv("B2_APP_KEY"))
	if err != nil {
		log.Println("cannot initilize buckets:", err)
		return 1
	}

	if *datastoreOpt == "" {
		log.Println("Missing datatore argument")
		return 1
	}

	if os.Getenv("B2_KEY_ID_DATASTORE") == "" {
		os.Setenv("B2_KEY_ID_DATASTORE", os.Getenv("B2_KEY_ID"))
	}
	if os.Getenv("B2_APP_KEY_DATASTORE") == "" {
		os.Setenv("B2_APP_KEY_DATASTORE", os.Getenv("B2_APP_KEY"))
	}

	datastoreBucket, err := initializeBucket(*datastoreOpt, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
	if err != nil {
		log.Println(err)
		return 1
	}

	// Enable Scrapers or Sellers/Vendors
	scraps := strings.SplitSeq(*scrapersOpt, ",")
	for name := range scraps {
		if options[name] != nil {
			options[name].Enabled = true
		}
	}
	if *sellersOpt != "" {
		sells := strings.SplitSeq(*sellersOpt, ",")
		for name := range sells {
			if options[name] == nil {
				log.Println("Seller", name, "not found")
				return 1
			}
			options[name].Enabled = true
			options[name].OnlySeller = true
			options[name].OnlyVendor = false
		}
	}
	if *vendorsOpt != "" {
		vends := strings.SplitSeq(*vendorsOpt, ",")
		for name := range vends {
			if options[name] == nil {
				log.Println("Vendor", name, "not found")
				return 1
			}
			options[name].Enabled = true
			options[name].OnlySeller = false
			options[name].OnlyVendor = true
		}
	}

	datastoreReader, err := simplecloud.InitReader(context.Background(), datastoreBucket, *datastoreOpt)
	if err != nil {
		log.Println(err)
		return 1
	}
	defer datastoreReader.Close()

	now := time.Now()
	err = mtgmatcher.LoadDatastore(datastoreReader)
	if err != nil {
		log.Println(err)
		return 1
	}
	log.Println("loading datastore took:", time.Since(now))

	var scrapers []mtgban.Scraper

	// Initialize the enabled scrapers
	for _, opt := range options {
		if !opt.Enabled {
			continue
		}

		scraper, err := opt.Init()
		if err != nil {
			log.Println(err)
			return 1
		}

		// Check if any sub data source needs to be disabled
		config, ok := scraper.(mtgban.ScraperConfig)
		if ok {
			config.SetConfig(mtgban.ScraperOptions{
				DisableRetail:  opt.OnlyVendor,
				DisableBuylist: opt.OnlySeller,
			})
		}

		scrapers = append(scrapers, scraper)
	}

	if len(scrapers) == 0 {
		log.Println("No scraper configured, run with -h for a list of commands")
		return 1
	}
	countSellers, countVendors := mtgban.CountScrapers(scrapers)
	log.Println("Configured with", countSellers, "sellers and", countVendors, "vendors")

	now = time.Now()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var nonFatalErrors []error

	// Load the data
	for _, scraper := range scrapers {
		err := scraper.Load(ctx)
		if err != nil {
			log.Println(err)
			nonFatalErrors = append(nonFatalErrors, err)
		}
	}

	log.Println("loading scraper data took:", time.Since(now))

	sellers, vendors := mtgban.UnfoldScrapers(scrapers)
	retailResults, buylistResults := countResults(sellers, vendors)
	log.Println("Found", retailResults, "retail results and", buylistResults, "buylist results")

	reportSuspectPricings(sellers, vendors)
	if retailResults == 0 && buylistResults == 0 {
		log.Println("No retail or buylist data retrieved")
		return 1
	}

	now = time.Now()
	// Dump the results
	dumpErrors := dump(dataBucket, sellers, vendors, *outputPathOpt, *fileFormatOpt, *metaOpt)
	nonFatalErrors = append(nonFatalErrors, dumpErrors...)

	log.Println("uploading data took:", time.Since(now))

	log.Println("Completed in", time.Since(start))

	// Check for non-fatal errors and exit accordingly
	if nonFatalErrors != nil {
		log.Println("There were non-fatal errors:")
		for _, err := range nonFatalErrors {
			log.Println("-", err)
		}
		return 2
	}

	return 0
}

func main() {
	os.Exit(run())
}

func signAPI(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		return "", err
	}

	v := url.Values{}
	v.Set("API", path.Base(u.Path))
	v.Set("APImode", "load")

	expires := time.Now().Add(1 * time.Minute)
	v.Set("Expires", fmt.Sprintf("%d", expires.Unix()))

	path := u.Scheme + "://" + u.Host
	if !strings.Contains(u.Host, "localhost") {
		path = "http://www.mtgban.com"
	}

	data := fmt.Sprintf("GET%d%s%s", expires.Unix(), path, v.Encode())
	key := os.Getenv("BAN_SECRET")
	if key == "" {
		return "", errors.New("missing BAN_SECRET")
	}

	// signHMACSHA1Base64
	h := hmac.New(sha1.New, []byte(key))
	h.Write([]byte(data))
	sig := base64.StdEncoding.EncodeToString(h.Sum(nil))

	v.Set("Signature", sig)
	str := base64.StdEncoding.EncodeToString([]byte(v.Encode()))

	return str, nil
}
