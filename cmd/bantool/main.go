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

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/cardmarket"
	"github.com/mtgban/go-mtgban/cardtrader"
	"github.com/mtgban/go-mtgban/mintcard"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/tcgplayer"
	"github.com/mtgban/simplecloud"

	_ "github.com/mtgban/go-mtgban/abugames"
	_ "github.com/mtgban/go-mtgban/arcanafrisia"
	_ "github.com/mtgban/go-mtgban/cardkingdom"
	_ "github.com/mtgban/go-mtgban/coolstuffinc"
	_ "github.com/mtgban/go-mtgban/gamenerdz"
	_ "github.com/mtgban/go-mtgban/hareruya"
	_ "github.com/mtgban/go-mtgban/magiccorner"
	_ "github.com/mtgban/go-mtgban/manaleak"
	_ "github.com/mtgban/go-mtgban/manapool"
	_ "github.com/mtgban/go-mtgban/merlion"
	_ "github.com/mtgban/go-mtgban/miniaturemarket"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
	_ "github.com/mtgban/go-mtgban/mtgseattle"
	_ "github.com/mtgban/go-mtgban/sealedev"
	_ "github.com/mtgban/go-mtgban/starcitygames"
	_ "github.com/mtgban/go-mtgban/strikezone"
	_ "github.com/mtgban/go-mtgban/trollandtoad"
	_ "github.com/mtgban/go-mtgban/vegassingles"
)

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

// targets lists every registered scraper for every game, keyed by game then
// name. Everything mtgban.Registered(game) names starts out enabled for both
// halves; three targets across the whole registry do not answer for one of
// them, and are marked here rather than left for a run to find out the hard
// way.
func targets() map[mtgban.Game]map[string]*scraperOption {
	all := make(map[mtgban.Game]map[string]*scraperOption)
	for _, game := range mtgban.AllGames {
		scrapers := make(map[string]*scraperOption)
		for _, name := range mtgban.Registered(game) {
			scrapers[name] = &scraperOption{}
		}
		all[game] = scrapers
	}

	// The store keeps no Magic singles shelf - not one variant in 960 sampled
	// had stock - so only the half it does answer for is asked here.
	all[mtgban.GameMagic]["vegassingles"].OnlyVendor = true
	all[mtgban.GameMagic]["mtgseattle"].OnlySeller = true
	all[mtgban.GameYuGiOh]["coolstuffinc_sealed"].OnlySeller = true

	return all
}

var options = targets()

// mkmCatalog reads Cardmarket's own published id-map catalog, whatever the
// game and whichever of cardmarket's scrapers asks for it: MTGJSON publishes
// Magic's, go-cardmarket's mkmcatalog builds the rest.
func mkmCatalog() (*cm.Catalog, error) {
	path := os.Getenv("MTGJSON_MKMID_PATH")
	if path == "" {
		return nil, errors.New("missing MTGJSON_MKMID_PATH env var")
	}
	reader, err := openPath(path, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return cm.LoadCatalog(reader)
}

// tcgSKUs reads the sku catalog tcg_market and tcg_sealed resolve their
// Magic listings against, and mintcard resolves all of its listings against,
// since mintcard prices skus TCGplayer itself hosts.
func tcgSKUs() (tcgplayer.SKUMap, error) {
	path := os.Getenv("MTGJSON_TCGSKU_PATH")
	if path == "" {
		return nil, errors.New("missing MTGJSON_TCGSKU_PATH env var")
	}
	start := time.Now()
	reader, err := openPath(path, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	skus, err := tcgplayer.LoadTCGSKUs(reader)
	if err != nil {
		return nil, err
	}
	log.Println("loading skus took:", time.Since(start))
	return skus, nil
}

// tcgSYPCatalog reads the catalog tcg_syplist resolves its skus against: the
// list names a sku and the datastore names products, and the catalog is the
// step between the two - the same file the datastore generators build from.
func tcgSYPCatalog() (tcgplayer.SYPCatalog, error) {
	path := os.Getenv("TCGPLAYER_CATALOG_PATH")
	if path == "" {
		return nil, errors.New("missing TCGPLAYER_CATALOG_PATH env var")
	}
	reader, err := openPath(path, os.Getenv("B2_KEY_ID_DATASTORE"), os.Getenv("B2_APP_KEY_DATASTORE"))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return tcgplayer.LoadSYPCatalog(reader)
}

// scraperResources loads the catalogs, sku lists and bridge a key needs
// beyond its own secrets and flags, and wraps each as the typed
// mtgban.Option its package exports. Loading stays bantool's job; the
// constructor a key registered receives loaded data only.
func scraperResources(game mtgban.Game, key string) ([]mtgban.Option, error) {
	var opts []mtgban.Option

	switch key {
	case "cardmarket", "cardmarket_market":
		catalog, err := mkmCatalog()
		if err != nil {
			return nil, err
		}
		opts = append(opts, cardmarket.WithCatalog(catalog))

		switch cardmarket.BridgeUseOf(game) {
		case cardmarket.BridgeRequired:
			bridge, err := cardtraderBridge(game)
			if err != nil {
				return nil, err
			}
			opts = append(opts, cardmarket.WithBridge(bridge))
		case cardmarket.BridgeHelps:
			// The catalog names most of the shelf by itself; the bridge only
			// settles what a collector number cannot, so a cardtrader that
			// will not answer costs those printings and nothing else, and
			// the run goes ahead saying so rather than failing and pricing
			// nothing.
			bridge, err := cardtraderBridge(game)
			if err != nil {
				log.Printf("bridge unavailable, naming what the catalog can on its own: %v", err)
			} else {
				opts = append(opts, cardmarket.WithBridge(bridge))
			}
		}

	case "cardmarket_sealed":
		bridge, err := cardtraderBridge(game)
		if err != nil {
			return nil, err
		}
		opts = append(opts, cardmarket.WithBridge(bridge))

	case "tcg_market", "tcg_sealed":
		if game != mtgban.GameMagic {
			break
		}
		skus, err := tcgSKUs()
		if err != nil {
			return nil, err
		}
		opts = append(opts, tcgplayer.WithSKUs(skus))

	case "mintcard":
		skus, err := tcgSKUs()
		if err != nil {
			return nil, err
		}
		opts = append(opts, mintcard.WithSKUs(skus))

	case "tcg_syplist":
		catalog, err := tcgSYPCatalog()
		if err != nil {
			return nil, err
		}
		opts = append(opts, tcgplayer.WithSYPCatalog(catalog))
	}

	return opts, nil
}

// scraperOptions turns the flags and environment bantool reads into the
// options the key's own registered constructor understands: always the log
// callback and, where set, a concurrency cap; one half if the target asked
// for it; the partner code its key's family has always read from the
// environment; and any catalog, sku list or bridge scraperResources loads.
// Secrets are not read here: EnvAuthenticator answers each constructor's own
// Secret* names directly.
func scraperOptions(game mtgban.Game, key string, opt *scraperOption, maxConcurrency int) ([]mtgban.Option, error) {
	opts := []mtgban.Option{mtgban.WithLogCallback(log.Printf)}
	if maxConcurrency != 0 {
		opts = append(opts, mtgban.WithMaxConcurrency(maxConcurrency))
	}
	if opt.OnlySeller {
		opts = append(opts, mtgban.WithRetailOnly())
	}
	if opt.OnlyVendor {
		opts = append(opts, mtgban.WithBuylistOnly())
	}

	switch {
	case strings.HasPrefix(key, "cardmarket"):
		opts = append(opts, mtgban.WithAffiliate(os.Getenv("MKM_PARTNER")))
	case strings.HasPrefix(key, "tcg_"):
		opts = append(opts, mtgban.WithAffiliate(os.Getenv("TCG_PARTNER")))
	case strings.HasPrefix(key, "cardkingdom"):
		opts = append(opts, mtgban.WithAffiliate(os.Getenv("CK_PARTNER")))
	case strings.HasPrefix(key, "coolstuffinc"):
		opts = append(opts, mtgban.WithAffiliate(os.Getenv("CSI_PARTNER")))
	case strings.HasPrefix(key, "cardtrader"):
		opts = append(opts, mtgban.WithAffiliate(os.Getenv("CT_PARTNER")))
	case strings.HasPrefix(key, "manapool"):
		opts = append(opts, mtgban.WithAffiliate(os.Getenv("MP_PARTNER")))
	case key == "miniaturemarket_sealed":
		opts = append(opts, mtgban.WithAffiliate(os.Getenv("MM_PARTNER")))
	case key == "mintcard":
		opts = append(opts, mtgban.WithAffiliate(os.Getenv("MINT_PARTNER")))
	case strings.HasPrefix(key, "starcitygames"):
		opts = append(opts, mtgban.WithAffiliate(os.Getenv("SCG_PARTNER")))
	case key == "sealed_ev":
		opts = append(opts,
			mtgban.WithAffiliate(os.Getenv("TCG_PARTNER")),
			mtgban.WithBuylistAffiliate(os.Getenv("CK_PARTNER")))
	}

	resources, err := scraperResources(game, key)
	if err != nil {
		return nil, err
	}
	return append(opts, resources...), nil
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
func reportSuspectPricings(backend *mtgmatcher.Backend, sellers []mtgban.Seller, vendors []mtgban.Vendor) {
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
				card, _ := backend.GetUUID(suspect.CardID)
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
func reportCollapsedPricings(backend *mtgmatcher.Backend, vendors []mtgban.Vendor) {
	for _, vendor := range vendors {
		collapsed := mtgban.CollapsedPricings(vendor.Buylist(), mtgban.CollapsedRatioThreshold)
		if len(collapsed) == 0 {
			continue
		}

		log.Printf("[%s] %d cards are bought at several prices at one grade",
			vendor.Info().Shorthand, len(collapsed))
		for _, entry := range collapsed {
			card, _ := backend.GetUUID(entry.CardID)
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

func run() int {
	start := time.Now()

	maxConcurrency, _ := strconv.Atoi(os.Getenv("MAX_CONCURRENCY"))
	log.Println("Workers running with", maxConcurrency, "parallel threads")

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
	// mtgban.NewScraper refuses the pair that cannot hold.
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
	log.Printf("loading datastore took: %v (%s)", time.Since(now), strings.ToLower(string(game)))

	var scrapers []mtgban.Scraper

	auth := mtgban.EnvAuthenticator{}
	// Initialize the enabled scrapers
	for key, opt := range options[game] {
		if !opt.Enabled {
			continue
		}

		opts, err := scraperOptions(game, key, opt, maxConcurrency)
		if err != nil {
			log.Println(err)
			return 1
		}

		scraper, err := mtgban.NewScraper(backend, key, auth, opts...)
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

	reportSuspectPricings(backend, sellers, vendors)
	reportCollapsedPricings(backend, vendors)
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
