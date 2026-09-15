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
	"slices"
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
	"github.com/mtgban/go-mtgban/hareruya"
	"github.com/mtgban/go-mtgban/magiccorner"
	"github.com/mtgban/go-mtgban/manaleak"
	"github.com/mtgban/go-mtgban/manapool"
	"github.com/mtgban/go-mtgban/merlion"
	"github.com/mtgban/go-mtgban/mintcard"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgseattle"
	"github.com/mtgban/go-mtgban/sealedev"
	"github.com/mtgban/go-mtgban/tcgplayer"
	"github.com/mtgban/go-mtgban/trollandtoad"

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
	Init       func(b *mtgmatcher.Backend) (mtgban.Scraper, error)
}

// scraperFlagName is the external name a target is known by: the store's own
// name for Magic, and store+"_"+game for every other game - the name every
// workflow and -scrapers/-sellers/-vendors caller already uses. It is the one
// place that composes the two; nowhere else needs to. mtgmatcher registers
// its loaders and bantool names its flags in lowercase, where mtgban.Game
// spells every constant capitalized, so lowercasing it is the one conversion
// this needs.
func scraperFlagName(game mtgban.Game, name string) string {
	if game == mtgban.GameMagic {
		return name
	}
	return name + "_" + strings.ToLower(string(game))
}

// flattenOptions indexes every game's scrapers under the name each is enabled
// by, for the callers that just want "the target named X": flag registration
// and the -scrapers/-sellers/-vendors lookups. The pointers are shared with
// options, so enabling an entry here enables the same one runGame sees.
//
// Two entries landing on the same name is no longer a compile error the way a
// duplicate key in one flat literal was: the game and the store are two
// separate keys now, and nothing but this name stops them from colliding
// across games. Panicking here trades a scraper silently dropped - whichever
// pointer a random map iteration happened to write last - for a run that
// refuses to start at all, which is the failure worth having for a registry
// nothing else checks.
func flattenOptions(options map[mtgban.Game]map[string]*scraperOption) map[string]*scraperOption {
	flat := make(map[string]*scraperOption)
	for game, scrapers := range options {
		for name, opt := range scrapers {
			key := scraperFlagName(game, name)
			_, exists := flat[key]
			if exists {
				panic(fmt.Sprintf("bantool: %q is registered under more than one game", key))
			}
			flat[key] = opt
		}
	}
	return flat
}

// runGame names the one game the enabled scrapers price. A run loads one
// datastore and opens it by that name rather than trying every game's
// loader on it, so enabling scrapers of two games is refused up front.
func runGame(options map[mtgban.Game]map[string]*scraperOption) (mtgban.Game, error) {
	var games []mtgban.Game
	for game, scrapers := range options {
		for _, opt := range scrapers {
			if opt.Enabled {
				games = append(games, game)
				break
			}
		}
	}
	switch len(games) {
	case 0:
		return "", errors.New("no scraper configured, run with -h for a list of commands")
	case 1:
		return games[0], nil
	}
	names := make([]string, len(games))
	for i, game := range games {
		names[i] = strings.ToLower(string(game))
	}
	slices.Sort(names)
	return "", fmt.Errorf("the enabled scrapers price %s, and a run loads one datastore", strings.Join(names, " and "))
}

// cardtraderBridge maps every Cardmarket product id to the TCGplayer id of
// the same product, read off cardtrader's blueprints - the one source
// linking the two marketplaces' ids. The blueprints cover singles and
// sealed alike, so the same bridge serves both cardmarket scrapers; they
// receive it as plain data, and the composition of the two vendors happens
// here and nowhere else.
func cardtraderBridge(game mtgban.Game) (map[int]int, error) {
	ctTokenBearer := os.Getenv("CARDTRADER_TOKEN_BEARER")
	if ctTokenBearer == "" {
		return nil, errors.New("missing CARDTRADER_TOKEN_BEARER env var")
	}
	client := cardtrader.NewCTAuthClient(ctTokenBearer)

	blueprints, _, err := cardtrader.BlueprintsForGame(context.Background(), client, game, "", log.Printf)
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

var options = map[mtgban.Game]map[string]*scraperOption{
	mtgban.GameFleshAndBlood: {
		"cardmarket": {
			Init: cardmarketBridgedIndexScraper(mtgban.GameFleshAndBlood),
		},
		"cardmarket_market": {
			Init: cardmarketBridgedMarketScraper(mtgban.GameFleshAndBlood),
		},
		"cardmarket_sealed": {
			Init: cardmarketSealedScraper(mtgban.GameFleshAndBlood),
		},
		"cardtrader": {
			Init: cardtraderMarketScraper(mtgban.GameFleshAndBlood),
		},
		"cardtrader_sealed": {
			Init: cardtraderSealedScraper(mtgban.GameFleshAndBlood),
		},
		"miniaturemarket_sealed": {
			Init: miniaturemarketSealedScraper(mtgban.GameFleshAndBlood),
		},
		"starcitygames": {
			Init: starcitygamesScraper(mtgban.GameFleshAndBlood),
		},
		"starcitygames_sealed": {
			Init: starcitygamesSealedScraper(mtgban.GameFleshAndBlood),
		},
		"strikezone": {
			Init: strikezoneScraper(mtgban.GameFleshAndBlood),
		},
		"tcg_index": {
			Init: tcgIndexScraper(mtgban.GameFleshAndBlood),
		},
		"tcg_market": {
			Init: tcgMarketScraper(mtgban.GameFleshAndBlood),
		},
		"tcg_sealed": {
			Init: tcgSealedScraper(mtgban.GameFleshAndBlood),
		},
	},
	mtgban.GameGundam: {
		"cardtrader": {
			Init: cardtraderMarketScraper(mtgban.GameGundam),
		},
		"cardtrader_sealed": {
			Init: cardtraderSealedScraper(mtgban.GameGundam),
		},
		"coolstuffinc": {
			Init: coolstuffincScraper(mtgban.GameGundam),
		},
		"miniaturemarket_sealed": {
			Init: miniaturemarketSealedScraper(mtgban.GameGundam),
		},
		"tcg_index": {
			Init: tcgIndexScraper(mtgban.GameGundam),
		},
		"tcg_market": {
			Init: tcgMarketScraper(mtgban.GameGundam),
		},
		"tcg_sealed": {
			Init: tcgSealedScraper(mtgban.GameGundam),
		},
		"vegassingles": {
			Init: vegassinglesScraper(mtgban.GameGundam),
		},
	},
	mtgban.GameLorcana: {
		"cardmarket": {
			Init: cardmarketIndexScraper(mtgban.GameLorcana),
		},
		"cardmarket_market": {
			Init: cardmarketMarketScraper(mtgban.GameLorcana),
		},
		"cardmarket_sealed": {
			Init: cardmarketSealedScraper(mtgban.GameLorcana),
		},
		"cardtrader": {
			Init: cardtraderMarketScraper(mtgban.GameLorcana),
		},
		"cardtrader_sealed": {
			Init: cardtraderSealedScraper(mtgban.GameLorcana),
		},
		"coolstuffinc": {
			Init: coolstuffincScraper(mtgban.GameLorcana),
		},
		"coolstuffinc_sealed": {
			Init: coolstuffincSealedScraper(mtgban.GameLorcana),
		},
		"gamenerdz": {
			Init: gamenerdzScraper(mtgban.GameLorcana),
		},
		"miniaturemarket_sealed": {
			Init: miniaturemarketSealedScraper(mtgban.GameLorcana),
		},
		"starcitygames": {
			Init: starcitygamesScraper(mtgban.GameLorcana),
		},
		"starcitygames_sealed": {
			Init: starcitygamesSealedScraper(mtgban.GameLorcana),
		},
		"strikezone": {
			Init: strikezoneScraper(mtgban.GameLorcana),
		},
		"tcg_index": {
			Init: tcgIndexScraper(mtgban.GameLorcana),
		},
		"tcg_market": {
			Init: tcgMarketScraper(mtgban.GameLorcana),
		},
		"tcg_sealed": {
			Init: tcgSealedScraper(mtgban.GameLorcana),
		},
	},
	mtgban.GameMagic: {
		"abugames": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := abugames.NewScraper(b)
				scraper.LogCallback = GlobalLogCallback
				if MaxConcurrency != 0 {
					scraper.MaxConcurrency = MaxConcurrency
				}
				return scraper, nil
			},
		},
		"abugames_sealed": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := abugames.NewScraperSealed(b)
				scraper.LogCallback = GlobalLogCallback
				if MaxConcurrency != 0 {
					scraper.MaxConcurrency = MaxConcurrency
				}
				return scraper, nil
			},
		},
		"arcanafrisia": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := arcanafrisia.NewScraper()
				scraper.LogCallback = GlobalLogCallback
				return scraper, nil
			},
		},
		"cardkingdom": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := cardkingdom.NewScraper(b)
				scraper.LogCallback = GlobalLogCallback
				scraper.Partner = os.Getenv("CK_PARTNER")
				scraper.PreserveOOS = true
				return scraper, nil
			},
		},
		"cardkingdom_graded": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper, err := cardkingdom.NewScraperGraded(b)
				if err != nil {
					return nil, err
				}
				scraper.LogCallback = GlobalLogCallback
				scraper.Partner = os.Getenv("CK_PARTNER")
				return scraper, nil
			},
		},
		"cardkingdom_sealed": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := cardkingdom.NewScraperSealed(b)
				scraper.LogCallback = GlobalLogCallback
				scraper.Partner = os.Getenv("CK_PARTNER")
				scraper.PreserveOOS = true
				return scraper, nil
			},
		},
		"cardmarket": {
			Init: cardmarketIndexScraper(mtgban.GameMagic),
		},
		"cardmarket_market": {
			Init: cardmarketMarketScraper(mtgban.GameMagic),
		},
		"cardmarket_sealed": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				appToken, appSecret, err := cardmarketCredentials()
				if err != nil {
					return nil, err
				}
				scraper, err := cardmarket.NewScraperSealed(b, appToken, appSecret)
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
			Init: cardtraderMarketScraper(mtgban.GameMagic),
		},
		"cardtrader_sealed": {
			Init: cardtraderSealedScraper(mtgban.GameMagic),
		},
		"coolstuffinc": {
			Init: coolstuffincScraper(mtgban.GameMagic),
		},
		"coolstuffinc_sealed": {
			Init: coolstuffincSealedScraper(mtgban.GameMagic),
		},
		"gamenerdz": {
			Init: gamenerdzScraper(mtgban.GameMagic),
		},
		"hareruya": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := hareruya.NewScraper(b)
				scraper.LogCallback = GlobalLogCallback
				return scraper, nil
			},
		},
		"hareruya_sealed": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := hareruya.NewScraperSealed(b)
				scraper.LogCallback = GlobalLogCallback
				return scraper, nil
			},
		},
		"magiccorner": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
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
		"manaleak": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := manaleak.NewScraper()
				scraper.LogCallback = GlobalLogCallback
				if MaxConcurrency != 0 {
					scraper.MaxConcurrency = MaxConcurrency
				}
				return scraper, nil
			},
		},
		"manapool": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := manapool.NewScraper()
				scraper.Partner = os.Getenv("MP_PARTNER")
				scraper.LogCallback = GlobalLogCallback
				return scraper, nil
			},
		},
		"manapool_index": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := manapool.NewScraperIndex()
				scraper.Partner = os.Getenv("MP_PARTNER")
				scraper.LogCallback = GlobalLogCallback
				return scraper, nil
			},
		},
		"manapool_sealed": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := manapool.NewScraperSealed()
				scraper.Partner = os.Getenv("MP_PARTNER")
				scraper.LogCallback = GlobalLogCallback
				return scraper, nil
			},
		},
		"miniaturemarket_sealed": {
			Init: miniaturemarketSealedScraper(mtgban.GameMagic),
		},
		"mintcard": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				tcgSKUPath := os.Getenv("MTGJSON_TCGSKU_PATH")
				if tcgSKUPath == "" {
					return nil, errors.New("missing MTGJSON_TCGSKU_PATH env var")
				}

				scraper := mintcard.NewScraper(b)
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
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := mtgseattle.NewScraper()
				scraper.LogCallback = GlobalLogCallback
				if MaxConcurrency != 0 {
					scraper.MaxConcurrency = MaxConcurrency
				}
				return scraper, nil
			},
		},
		"sealed_ev": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				banKey := os.Getenv("BAN_API_KEY")
				if banKey == "" {
					return nil, errors.New("missing BAN_API_KEY env var")
				}
				scraper := sealedev.NewScraper(b, banKey)
				scraper.Affiliate = os.Getenv("TCG_PARTNER")
				scraper.LogCallback = GlobalLogCallback
				if MaxConcurrency != 0 {
					scraper.MaxConcurrency = MaxConcurrency
				}
				return scraper, nil
			},
		},
		"starcitygames": {
			Init: starcitygamesScraper(mtgban.GameMagic),
		},
		"starcitygames_sealed": {
			Init: starcitygamesSealedScraper(mtgban.GameMagic),
		},
		"strikezone": {
			Init: strikezoneScraper(mtgban.GameMagic),
		},
		"tcg_index": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				tcgPublicID := os.Getenv("TCGPLAYER_PUBLIC_KEY")
				tcgPrivateID := os.Getenv("TCGPLAYER_PRIVATE_KEY")
				if tcgPublicID == "" || tcgPrivateID == "" {
					return nil, errors.New("missing TCGPLAYER_PUBLIC_KEY or TCGPLAYER_PRIVATE_KEY env vars")
				}

				scraper, err := tcgplayer.NewScraperIndex(b, tcgPublicID, tcgPrivateID)
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
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				tcgPublicID := os.Getenv("TCGPLAYER_PUBLIC_KEY")
				tcgPrivateID := os.Getenv("TCGPLAYER_PRIVATE_KEY")
				tcgSKUPath := os.Getenv("MTGJSON_TCGSKU_PATH")
				if tcgPublicID == "" || tcgPrivateID == "" || tcgSKUPath == "" {
					return nil, errors.New("missing TCGPLAYER_PUBLIC_KEY or TCGPLAYER_PRIVATE_KEY or MTGJSON_TCGSKU_PATH env vars")
				}

				scraper, err := tcgplayer.NewScraperMarket(b, tcgPublicID, tcgPrivateID)
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
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				tcgPublicID := os.Getenv("TCGPLAYER_PUBLIC_KEY")
				tcgPrivateID := os.Getenv("TCGPLAYER_PRIVATE_KEY")
				tcgSKUPath := os.Getenv("MTGJSON_TCGSKU_PATH")
				if tcgPublicID == "" || tcgPrivateID == "" || tcgSKUPath == "" {
					return nil, errors.New("missing TCGPLAYER_PUBLIC_KEY or TCGPLAYER_PRIVATE_KEY or MTGJSON_TCGSKU_PATH env vars")
				}

				scraper, err := tcgplayer.NewScraperSealed(b, tcgPublicID, tcgPrivateID)
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
			Init: tcgSYPScraper(mtgban.GameMagic),
		},
		"trollandtoad": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := trollandtoad.NewScraper()
				scraper.LogCallback = GlobalLogCallback
				if MaxConcurrency != 0 {
					scraper.MaxConcurrency = MaxConcurrency
				}
				return scraper, nil
			},
		},
		// The store keeps no Magic singles shelf - not one variant in 960 sampled
		// had stock - so only the half it does answer for is asked here.
		"vegassingles": {
			OnlyVendor: true,
			Init:       vegassinglesScraper(mtgban.GameMagic),
		},
	},
	mtgban.GameOnePiece: {
		"cardmarket": {
			Init: cardmarketOptionallyBridgedIndexScraper(mtgban.GameOnePiece),
		},
		"cardmarket_market": {
			Init: cardmarketOptionallyBridgedMarketScraper(mtgban.GameOnePiece),
		},
		"cardmarket_sealed": {
			Init: cardmarketSealedScraper(mtgban.GameOnePiece),
		},
		"cardtrader": {
			Init: cardtraderMarketScraper(mtgban.GameOnePiece),
		},
		"cardtrader_sealed": {
			Init: cardtraderSealedScraper(mtgban.GameOnePiece),
		},
		"coolstuffinc": {
			Init: coolstuffincScraper(mtgban.GameOnePiece),
		},
		"coolstuffinc_sealed": {
			Init: coolstuffincSealedScraper(mtgban.GameOnePiece),
		},
		"gamenerdz": {
			Init: gamenerdzScraper(mtgban.GameOnePiece),
		},
		"miniaturemarket_sealed": {
			Init: miniaturemarketSealedScraper(mtgban.GameOnePiece),
		},
		"tcg_index": {
			Init: tcgIndexScraper(mtgban.GameOnePiece),
		},
		"tcg_market": {
			Init: tcgMarketScraper(mtgban.GameOnePiece),
		},
		"tcg_sealed": {
			Init: tcgSealedScraper(mtgban.GameOnePiece),
		},
		"vegassingles": {
			Init: vegassinglesScraper(mtgban.GameOnePiece),
		},
	},
	mtgban.GamePalworld: {
		"coolstuffinc": {
			Init: coolstuffincScraper(mtgban.GamePalworld),
		},
		"tcg_index": {
			Init: tcgIndexScraper(mtgban.GamePalworld),
		},
		"tcg_market": {
			Init: tcgMarketScraper(mtgban.GamePalworld),
		},
		"tcg_sealed": {
			Init: tcgSealedScraper(mtgban.GamePalworld),
		},
	},
	mtgban.GamePokemon: {
		"cardmarket": {
			Init: cardmarketBridgedIndexScraper(mtgban.GamePokemon),
		},
		"cardmarket_market": {
			Init: cardmarketBridgedMarketScraper(mtgban.GamePokemon),
		},
		"cardmarket_sealed": {
			Init: cardmarketSealedScraper(mtgban.GamePokemon),
		},
		"cardtrader": {
			Init: cardtraderMarketScraper(mtgban.GamePokemon),
		},
		"cardtrader_sealed": {
			Init: cardtraderSealedScraper(mtgban.GamePokemon),
		},
		"coolstuffinc": {
			Init: coolstuffincScraper(mtgban.GamePokemon),
		},
		"coolstuffinc_sealed": {
			Init: coolstuffincSealedScraper(mtgban.GamePokemon),
		},
		"gamenerdz": {
			Init: gamenerdzScraper(mtgban.GamePokemon),
		},
		"strikezone": {
			Init: strikezoneScraper(mtgban.GamePokemon),
		},
		"tcg_index": {
			Init: tcgIndexScraper(mtgban.GamePokemon),
		},
		"tcg_market": {
			Init: tcgMarketScraper(mtgban.GamePokemon),
		},
		"tcg_sealed": {
			Init: tcgSealedScraper(mtgban.GamePokemon),
		},
		"tcg_syplist": {
			Init: tcgSYPScraper(mtgban.GamePokemon),
		},
		"vegassingles": {
			Init: vegassinglesScraper(mtgban.GamePokemon),
		},
	},
	mtgban.GameRiftbound: {
		"cardmarket": {
			Init: cardmarketIndexScraper(mtgban.GameRiftbound),
		},
		"cardmarket_market": {
			Init: cardmarketMarketScraper(mtgban.GameRiftbound),
		},
		"cardmarket_sealed": {
			Init: cardmarketSealedScraper(mtgban.GameRiftbound),
		},
		"cardtrader": {
			Init: cardtraderMarketScraper(mtgban.GameRiftbound),
		},
		"cardtrader_sealed": {
			Init: cardtraderSealedScraper(mtgban.GameRiftbound),
		},
		"coolstuffinc": {
			Init: coolstuffincScraper(mtgban.GameRiftbound),
		},
		"coolstuffinc_sealed": {
			Init: coolstuffincSealedScraper(mtgban.GameRiftbound),
		},
		"merlion": {
			Init: func(b *mtgmatcher.Backend) (mtgban.Scraper, error) {
				scraper := merlion.NewScraper()
				scraper.LogCallback = GlobalLogCallback
				return scraper, nil
			},
		},
		"miniaturemarket_sealed": {
			Init: miniaturemarketSealedScraper(mtgban.GameRiftbound),
		},
		"starcitygames": {
			Init: starcitygamesScraper(mtgban.GameRiftbound),
		},
		"starcitygames_sealed": {
			Init: starcitygamesSealedScraper(mtgban.GameRiftbound),
		},
		"tcg_index": {
			Init: tcgIndexScraper(mtgban.GameRiftbound),
		},
		"tcg_market": {
			Init: tcgMarketScraper(mtgban.GameRiftbound),
		},
		"tcg_sealed": {
			Init: tcgSealedScraper(mtgban.GameRiftbound),
		},
		"vegassingles": {
			Init: vegassinglesScraper(mtgban.GameRiftbound),
		},
	},
	mtgban.GameYuGiOh: {
		"cardmarket": {
			Init: cardmarketBridgedIndexScraper(mtgban.GameYuGiOh),
		},
		"cardmarket_market": {
			Init: cardmarketBridgedMarketScraper(mtgban.GameYuGiOh),
		},
		"cardmarket_sealed": {
			Init: cardmarketSealedScraper(mtgban.GameYuGiOh),
		},
		"cardtrader": {
			Init: cardtraderMarketScraper(mtgban.GameYuGiOh),
		},
		"cardtrader_sealed": {
			Init: cardtraderSealedScraper(mtgban.GameYuGiOh),
		},
		"coolstuffinc": {
			Init: coolstuffincScraper(mtgban.GameYuGiOh),
		},
		"coolstuffinc_sealed": {
			Init:       coolstuffincSealedScraper(mtgban.GameYuGiOh),
			OnlySeller: true,
		},
		"tcg_index": {
			Init: tcgIndexScraper(mtgban.GameYuGiOh),
		},
		"tcg_market": {
			Init: tcgMarketScraper(mtgban.GameYuGiOh),
		},
		"tcg_sealed": {
			Init: tcgSealedScraper(mtgban.GameYuGiOh),
		},
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

func dumpSeller(backend *mtgmatcher.Backend, dataBucket simplecloud.Writer, seller mtgban.Seller, outputPath, format string) (err error) {
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
		err = mtgban.WriteInventoryToCSV(backend, seller.Inventory(), writer)
	case "ndjson":
		err = writeSellerToNDJSON(seller, writer)
	default:
		err = errors.New("invalid format")
	}

	return err
}

func dumpVendor(backend *mtgmatcher.Backend, dataBucket simplecloud.Writer, vendor mtgban.Vendor, outputPath, format string) (err error) {
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
		err = mtgban.WriteBuylistToCSV(backend, vendor.Buylist(), vendor.Info().CreditMultiplier, writer)
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

// reportCollapsedPricings says which cards a vendor buys at more than one
// price at one grade. A shop pays one price for one card, so a second is not
// a better offer but a second product: the feed named two printings and the
// match folded them onto one id. The run log says nothing about it either -
// both listings resolved, to the same card - and the pair of listing links
// is where the wording that tells them apart is published.
func reportCollapsedPricings(vendors []mtgban.Vendor) {
	for _, vendor := range vendors {
		collapsed := mtgban.CollapsedPricings(vendor.Buylist(), mtgban.CollapsedRatioThreshold)
		if len(collapsed) == 0 {
			continue
		}

		log.Printf("[%s] %d cards are bought at several prices at one grade",
			vendor.Info().Shorthand, len(collapsed))
		for _, entry := range collapsed {
			card, _ := mtgmatcher.GetUUID(entry.CardID)
			log.Printf("[%s] - %.1fx %d prices, $%.2f against $%.2f %s %s",
				vendor.Info().Shorthand, entry.Ratio, entry.Count,
				entry.High, entry.Low, entry.Conditions, card)
			log.Printf("[%s]   %s", vendor.Info().Shorthand, entry.HighURL)
			log.Printf("[%s]   %s", vendor.Info().Shorthand, entry.LowURL)
		}
	}
}

func dump(backend *mtgmatcher.Backend, dataBucket simplecloud.Writer, sellers []mtgban.Seller, vendors []mtgban.Vendor, outputPath, format string, meta bool) []error {
	log.Println("Writing results to", outputPath)

	var sellerErrs []error
	for _, seller := range sellers {
		err := dumpSeller(backend, dataBucket, seller, outputPath, format)
		if err != nil {
			log.Println(err)
			sellerErrs = append(sellerErrs, err)
			continue
		}

		if meta && format != "json" {
			sellerMeta := mtgban.NewSellerFromInventory(nil, seller.Info())
			err := dumpSeller(backend, dataBucket, sellerMeta, outputPath, "json")
			if err != nil {
				sellerErrs = append(sellerErrs, err)
				continue
			}
		}
	}

	var vendorErrs []error
	for _, vendor := range vendors {
		err := dumpVendor(backend, dataBucket, vendor, outputPath, format)
		if err != nil {
			log.Println(err)
			vendorErrs = append(vendorErrs, err)
			continue
		}

		if meta && format != "json" {
			vendorMeta := mtgban.NewVendorFromBuylist(nil, vendor.Info())
			err := dumpVendor(backend, dataBucket, vendorMeta, outputPath, "json")
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

// configureScraper turns off the half a target was not asked for.
//
// The type assertion is the only thing standing between an option and a
// scraper that cannot honour it, and it says nothing when it fails: a target
// asked for one half by a scraper holding both ran whole and published both.
// Refusing it names the target instead, where the empty half used to be found
// by reading the output.
//
// Asking for both halves alone is refused here too, because the two ways of
// asking meet on the same two fields. A few entries declare OnlySeller or
// OnlyVendor as a fact about the store - Vegas Singles keeps no Magic singles
// shelf, MTG Seattle no buylist - and -sellers/-vendors say the same thing for
// one run, so a flag can name a store whose own entry already answers for the
// other half, or both flags can name one store between them. Neither leaves
// anything to publish, and the empty record would surface at the dump, with
// the crawl already spent.
func configureScraper(name string, opt *scraperOption, scraper mtgban.Scraper) error {
	if opt.OnlySeller && opt.OnlyVendor {
		return fmt.Errorf("%s was asked for its retail alone and its buylist "+
			"alone at once, which leaves nothing to publish", name)
	}
	config, ok := scraper.(mtgban.ScraperConfig)
	if ok {
		config.SetConfig(mtgban.ScraperOptions{
			DisableRetail:  opt.OnlyVendor,
			DisableBuylist: opt.OnlySeller,
		})
		return nil
	}
	if !opt.OnlyVendor && !opt.OnlySeller {
		return nil
	}
	half := "buylist"
	if opt.OnlySeller {
		half = "retail"
	}
	return fmt.Errorf("%s was asked for its %s alone, but does not implement "+
		"mtgban.ScraperConfig and would publish both halves", name, half)
}

func run() int {
	start := time.Now()

	flatOptions := flattenOptions(options)

	for key, val := range flatOptions {
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
		if flatOptions[name] != nil {
			flatOptions[name].Enabled = true
		}
	}
	// Clearing the other half here would overwrite what the entry itself
	// says about the store rather than meet it; both are left standing, and
	// configureScraper refuses the pair that cannot hold.
	if *sellersOpt != "" {
		sells := strings.SplitSeq(*sellersOpt, ",")
		for name := range sells {
			if flatOptions[name] == nil {
				log.Println("Seller", name, "not found")
				return 1
			}
			flatOptions[name].Enabled = true
			flatOptions[name].OnlySeller = true
		}
	}
	if *vendorsOpt != "" {
		vends := strings.SplitSeq(*vendorsOpt, ",")
		for name := range vends {
			if flatOptions[name] == nil {
				log.Println("Vendor", name, "not found")
				return 1
			}
			flatOptions[name].Enabled = true
			flatOptions[name].OnlyVendor = true
		}
	}

	game, err := runGame(options)
	if err != nil {
		log.Println(err)
		return 1
	}

	datastoreReader, err := simplecloud.InitReader(context.Background(), datastoreBucket, *datastoreOpt)
	if err != nil {
		log.Println(err)
		return 1
	}
	defer datastoreReader.Close()

	now := time.Now()
	backend, err := mtgmatcher.Open(strings.ToLower(string(game)), datastoreReader)
	if err != nil {
		log.Println(err)
		return 1
	}
	mtgmatcher.SetGlobalDatastore(backend)
	log.Printf("loading datastore took: %v (%s)", time.Since(now), strings.ToLower(string(game)))

	var scrapers []mtgban.Scraper

	// Initialize the enabled scrapers
	for name, opt := range flatOptions {
		if !opt.Enabled {
			continue
		}

		scraper, err := opt.Init(backend)
		if err != nil {
			log.Println(err)
			return 1
		}

		// Check if any sub data source needs to be disabled
		err = configureScraper(name, opt, scraper)
		if err != nil {
			log.Println(err)
			return 1
		}

		scrapers = append(scrapers, scraper)
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
	reportCollapsedPricings(vendors)
	if retailResults == 0 && buylistResults == 0 {
		log.Println("No retail or buylist data retrieved")
		return 1
	}

	now = time.Now()
	// Dump the results
	dumpErrors := dump(backend, dataBucket, sellers, vendors, *outputPathOpt, *fileFormatOpt, *metaOpt)
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
