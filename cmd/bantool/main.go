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
	Enabled bool
	// Supports names every game this entry's Init can build a scraper for.
	// Checked in run() before Init is ever called, so a game missing from
	// this list is refused by name without Init running at all - nil
	// supports nothing, not everything, so a new entry that forgets to
	// set this is refused for every game rather than reaching Init
	// unchecked.
	Supports []string
	// OnlySeller and OnlyVendor name the games a store cannot honour the
	// other side for - Vegas Singles keeps no Magic singles shelf,
	// CoolStuffInc Sealed publishes no Yu-Gi-Oh buylist - where the same
	// store answers with both sides for every other game it prices. Most
	// entries leave both nil.
	OnlySeller []string
	OnlyVendor []string
	Init       func(game string) (mtgban.Scraper, error)
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
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := abugames.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"abugames_sealed": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := abugames.NewScraperSealed()
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"arcanafrisia": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := arcanafrisia.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"cardkingdom": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := cardkingdom.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CK_PARTNER")
			scraper.PreserveOOS = true
			return scraper, nil
		},
	},
	"cardkingdom_graded": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper, err := cardkingdom.NewScraperGraded()
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CK_PARTNER")
			return scraper, nil
		},
	},
	"cardkingdom_sealed": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := cardkingdom.NewScraperSealed()
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CK_PARTNER")
			scraper.PreserveOOS = true
			return scraper, nil
		},
	},
	"hareruya": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := hareruya.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"hareruya_sealed": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := hareruya.NewScraperSealed()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"magiccorner": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
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
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := manaleak.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"manapool": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := manapool.NewScraper()
			scraper.Partner = os.Getenv("MP_PARTNER")
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"manapool_index": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := manapool.NewScraperIndex()
			scraper.Partner = os.Getenv("MP_PARTNER")
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"manapool_sealed": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := manapool.NewScraperSealed()
			scraper.Partner = os.Getenv("MP_PARTNER")
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"mintcard": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
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
	"sealed_ev": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
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
	"trollandtoad": {
		Supports: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := trollandtoad.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"mtgseattle": {
		Supports:   []string{"magic"},
		OnlySeller: []string{"magic"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := mtgseattle.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"merlion": {
		Supports: []string{"riftbound"},
		Init: func(game string) (mtgban.Scraper, error) {
			scraper := merlion.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"cardmarket": {
		Supports: []string{"fleshandblood", "lorcana", "magic", "onepiece", "pokemon", "riftbound", "yugioh"},
		Init:     cardmarketScraper,
	},
	"cardmarket_sealed": {
		Supports: []string{"fleshandblood", "lorcana", "magic", "onepiece", "pokemon", "riftbound", "yugioh"},
		Init:     cardmarketSealedScraper,
	},
	"cardtrader": {
		Supports: []string{"fleshandblood", "gundam", "lorcana", "magic", "onepiece", "pokemon", "riftbound", "yugioh"},
		Init:     cardtraderMarketScraper,
	},
	"cardtrader_sealed": {
		Supports: []string{"fleshandblood", "gundam", "lorcana", "magic", "onepiece", "pokemon", "riftbound", "yugioh"},
		Init:     cardtraderSealedScraper,
	},
	"coolstuffinc": {
		Supports: []string{"gundam", "lorcana", "magic", "onepiece", "palworld", "pokemon", "riftbound", "yugioh"},
		Init:     coolstuffincScraper,
	},
	"coolstuffinc_sealed": {
		Supports:   []string{"lorcana", "magic", "onepiece", "pokemon", "riftbound", "yugioh"},
		OnlySeller: []string{"yugioh"},
		Init:       coolstuffincSealedScraper,
	},
	"gamenerdz": {
		Supports: []string{"fleshandblood", "lorcana", "magic", "onepiece", "pokemon"},
		Init:     gamenerdzScraper,
	},
	"miniaturemarket_sealed": {
		Supports: []string{"fleshandblood", "gundam", "lorcana", "magic", "onepiece", "riftbound"},
		Init:     miniaturemarketSealedScraper,
	},
	"starcitygames": {
		Supports: []string{"fleshandblood", "lorcana", "magic", "riftbound"},
		Init:     starcitygamesScraper,
	},
	"starcitygames_sealed": {
		Supports: []string{"fleshandblood", "lorcana", "magic", "riftbound"},
		Init:     starcitygamesSealedScraper,
	},
	"strikezone": {
		Supports: []string{"fleshandblood", "lorcana", "magic", "pokemon"},
		Init:     strikezoneScraper,
	},
	"tcg_index": {
		Supports: []string{"fleshandblood", "gundam", "lorcana", "magic", "onepiece", "palworld", "pokemon", "riftbound", "yugioh"},
		Init:     tcgIndexScraper,
	},
	"tcg_market": {
		Supports: []string{"fleshandblood", "gundam", "lorcana", "magic", "onepiece", "palworld", "pokemon", "riftbound", "yugioh"},
		Init:     tcgMarketScraper,
	},
	"tcg_sealed": {
		Supports: []string{"fleshandblood", "gundam", "lorcana", "magic", "onepiece", "palworld", "pokemon", "riftbound", "yugioh"},
		Init:     tcgSealedScraper,
	},
	"tcg_syplist": {
		Supports: []string{"magic", "pokemon"},
		Init:     tcgSYPScraper,
	},
	"vegassingles": {
		Supports:   []string{"gundam", "magic", "onepiece", "pokemon", "riftbound"},
		OnlyVendor: []string{"magic"},
		Init:       vegassinglesScraper,
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

// configureScraper turns off the half a target was not asked for.
//
// The type assertion is the only thing standing between an option and a
// scraper that cannot honour it, and it says nothing when it fails: a target
// asked for one half by a scraper holding both ran whole and published both.
// Refusing it names the target instead, where the empty half used to be found
// by reading the output.
func configureScraper(name string, onlySeller, onlyVendor bool, scraper mtgban.Scraper) error {
	config, ok := scraper.(mtgban.ScraperConfig)
	if ok {
		config.SetConfig(mtgban.ScraperOptions{
			DisableRetail:  onlyVendor,
			DisableBuylist: onlySeller,
		})
		return nil
	}
	if !onlyVendor && !onlySeller {
		return nil
	}
	half := "buylist"
	if onlySeller {
		half = "retail"
	}
	return fmt.Errorf("%s was asked for its %s alone, but does not implement "+
		"mtgban.ScraperConfig and would publish both halves", name, half)
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

	// The empty default mirrors mtgban.GameMagic: a scraper that never names
	// its game is Magic's, and so is a run that never names one.
	gameOpt := flag.String("game", "", "Canonical game to price (empty means Magic)")

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

	game := *gameOpt
	if game == "" {
		game = "magic"
	}

	// forceSeller and forceVendor are this run's -sellers/-vendors asking for
	// one half regardless of what the selected game's own data says; they
	// stand apart from OnlySeller/OnlyVendor, which name a fact about a
	// store's feed for one game and must not be overwritten by a flag that
	// only ever means "for this run".
	forceSeller := map[string]bool{}
	forceVendor := map[string]bool{}

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
			forceSeller[name] = true
			delete(forceVendor, name)
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
			forceVendor[name] = true
			delete(forceSeller, name)
		}
	}

	var anyEnabled bool
	for _, opt := range options {
		if opt.Enabled {
			anyEnabled = true
			break
		}
	}
	if !anyEnabled {
		log.Println("no scraper configured, run with -h for a list of commands")
		return 1
	}

	datastoreReader, err := simplecloud.InitReader(context.Background(), datastoreBucket, *datastoreOpt)
	if err != nil {
		log.Println(err)
		return 1
	}
	defer datastoreReader.Close()

	now := time.Now()
	backend, err := mtgmatcher.Open(game, datastoreReader)
	if err != nil {
		log.Println(err)
		return 1
	}
	mtgmatcher.SetGlobalDatastore(backend)
	log.Printf("loading datastore took: %v (%s)", time.Since(now), game)

	var scrapers []mtgban.Scraper

	// Initialize the enabled scrapers
	for name, opt := range options {
		if !opt.Enabled {
			continue
		}

		// Every entry names the games it answers for, so its refusal
		// names the store for free, from the name already in scope here.
		if !slices.Contains(opt.Supports, game) {
			log.Printf("%s does not support %q", name, game)
			return 1
		}

		scraper, err := opt.Init(game)
		if err != nil {
			log.Println(err)
			return 1
		}

		// Check if any sub data source needs to be disabled
		onlySeller := forceSeller[name] || slices.Contains(opt.OnlySeller, game)
		onlyVendor := forceVendor[name] || slices.Contains(opt.OnlyVendor, game)
		if onlySeller && onlyVendor {
			// -sellers/-vendors asked for the one half the selected game's
			// own data says this store cannot answer for the other -
			// forceSeller met a structural OnlyVendor, or forceVendor met a
			// structural OnlySeller. Both true would tell configureScraper
			// to publish neither, which is not what either flag asked for.
			asked, has := "retail", "a buylist"
			if forceVendor[name] {
				asked, has = "buylist", "a retail shelf"
			}
			log.Printf("%s was asked for its %s alone, but %s is all it has for %s", name, asked, has, game)
			return 1
		}
		err = configureScraper(name, onlySeller, onlyVendor, scraper)
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
