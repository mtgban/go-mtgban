package main

import (
	"compress/gzip"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/dsnet/compress/bzip2"
	"github.com/ulikunitz/xz"

	"cloud.google.com/go/storage"
	"github.com/Backblaze/blazer/b2"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/scizorman/go-ndjson"
	xzReader "github.com/xi2/xz"
	"google.golang.org/api/option"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mtgban/go-mtgban/abugames"
	"github.com/mtgban/go-mtgban/cardkingdom"
	"github.com/mtgban/go-mtgban/cardmarket"
	"github.com/mtgban/go-mtgban/cardtrader"
	"github.com/mtgban/go-mtgban/coolstuffinc"
	"github.com/mtgban/go-mtgban/magiccorner"
	"github.com/mtgban/go-mtgban/manapool"
	"github.com/mtgban/go-mtgban/miniaturemarket"
	"github.com/mtgban/go-mtgban/mintcard"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgseattle"
	"github.com/mtgban/go-mtgban/sealedev"
	"github.com/mtgban/go-mtgban/starcitygames"
	"github.com/mtgban/go-mtgban/strikezone"
	"github.com/mtgban/go-mtgban/tcgplayer"
	"github.com/mtgban/go-mtgban/trollandtoad"

	"github.com/mtgban/go-mtgban/mtgban"
)

var GCSBucket *storage.BucketHandle
var B2Bucket *b2.Bucket

var GlobalLogCallback mtgban.LogCallbackFunc = log.Printf

var MaxConcurrency int

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
	"abugames_sealed": &scraperOption{
		Init: func() (mtgban.Scraper, error) {
			scraper := abugames.NewScraperSealed()
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
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
	"cardtrader": {
		Init: func() (mtgban.Scraper, error) {
			ctTokenBearer := os.Getenv("CARDTRADER_TOKEN_BEARER")
			if ctTokenBearer == "" {
				return nil, errors.New("missing CARDTRADER_TOKEN_BEARER env var")
			}

			scraper, err := cardtrader.NewScraperMarket(cardtrader.GameIdMagic, ctTokenBearer)
			if err != nil {
				return nil, err
			}
			scraper.ShareCode = os.Getenv("CT_PARTNER")
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"cardtrader_sealed": {
		Init: func() (mtgban.Scraper, error) {
			ctTokenBearer := os.Getenv("CARDTRADER_TOKEN_BEARER")
			if ctTokenBearer == "" {
				return nil, errors.New("missing CARDTRADER_TOKEN_BEARER env var")
			}

			scraper, err := cardtrader.NewScraperSealed(ctTokenBearer)
			if err != nil {
				return nil, err
			}
			scraper.ShareCode = os.Getenv("CT_PARTNER")
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"coolstuffinc": {
		Init: func() (mtgban.Scraper, error) {
			scraper := coolstuffinc.NewScraper(coolstuffinc.GameMagic)
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CSI_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"coolstuffinc_sealed": {
		Init: func() (mtgban.Scraper, error) {
			scraper := coolstuffinc.NewScraperSealed()
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CSI_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
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
	"manapool_sealed": {
		Init: func() (mtgban.Scraper, error) {
			scraper := manapool.NewScraperSealed()
			scraper.Partner = os.Getenv("MP_PARTNER")
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"miniaturemarket_sealed": {
		Init: func() (mtgban.Scraper, error) {
			scraper := miniaturemarket.NewScraperSealed()
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"mintcard": {
		Init: func() (mtgban.Scraper, error) {
			scraper := mintcard.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"cardmarket": {
		Init: func() (mtgban.Scraper, error) {
			mkmAppToken := os.Getenv("MKM_APP_TOKEN")
			mkmAppSecret := os.Getenv("MKM_APP_SECRET")
			if mkmAppToken == "" || mkmAppSecret == "" {
				return nil, errors.New("missing MKM_APP_TOKEN or MKM_APP_SECRET env vars")
			}

			scraper, err := cardmarket.NewScraperIndex(cardmarket.GameIdMagic, mkmAppToken, mkmAppSecret)
			if err != nil {
				return nil, err
			}
			scraper.Affiliate = "mtgban"
			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("MKM_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"cardmarket_sealed": {
		Init: func() (mtgban.Scraper, error) {
			mkmAppToken := os.Getenv("MKM_APP_TOKEN")
			mkmAppSecret := os.Getenv("MKM_APP_SECRET")
			if mkmAppToken == "" || mkmAppSecret == "" {
				return nil, errors.New("missing MKM_APP_TOKEN or MKM_APP_SECRET env vars")
			}

			scraper, err := cardmarket.NewScraperSealed(mkmAppToken, mkmAppSecret)
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
		Init: func() (mtgban.Scraper, error) {
			scgGUID := os.Getenv("SCG_GUID")
			scgBearer := os.Getenv("SCG_BEARER")
			if scgGUID == "" || scgBearer == "" {
				return nil, errors.New("missing SCG_GUID or SCG_BEARER env var")
			}

			scraper := starcitygames.NewScraper(starcitygames.GameMagic, scgGUID, scgBearer)
			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("SCG_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"starcitygames_sealed": {
		Init: func() (mtgban.Scraper, error) {
			scgGUID := os.Getenv("SCG_GUID")
			scgBearer := os.Getenv("SCG_BEARER")
			if scgGUID == "" || scgBearer == "" {
				return nil, errors.New("missing SCG_GUID or SCG_BEARER env var")
			}

			scraper := starcitygames.NewScraperSealed(scgGUID, scgBearer)
			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("SCG_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"strikezone": {
		Init: func() (mtgban.Scraper, error) {
			scraper := strikezone.NewScraper(strikezone.GameMagic)
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"tcg_index": {
		Init: func() (mtgban.Scraper, error) {
			tcgPublicId := os.Getenv("TCGPLAYER_PUBLIC_ID")
			tcgPrivateId := os.Getenv("TCGPLAYER_PRIVATE_ID")
			if tcgPublicId == "" || tcgPrivateId == "" {
				return nil, errors.New("missing TCGPLAYER_PUBLIC_ID or TCGPLAYER_PRIVATE_ID env vars")
			}

			scraper, err := tcgplayer.NewScraperIndex(tcgPublicId, tcgPrivateId)
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
			tcgPublicId := os.Getenv("TCGPLAYER_PUBLIC_ID")
			tcgPrivateId := os.Getenv("TCGPLAYER_PRIVATE_ID")
			tcgSKUPath := os.Getenv("MTGJSON_TCGSKU_PATH")
			if tcgPublicId == "" || tcgPrivateId == "" || tcgSKUPath == "" {
				return nil, errors.New("missing TCGPLAYER_PUBLIC_ID or TCGPLAYER_PRIVATE_ID or MTGJSON_TCGSKU_PATH env vars")
			}

			scraper, err := tcgplayer.NewScraperMarket(tcgPublicId, tcgPrivateId)
			if err != nil {
				return nil, err
			}

			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("TCG_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}

			start := time.Now()
			skuReader, err := loadData(tcgSKUPath)
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
			tcgPublicId := os.Getenv("TCGPLAYER_PUBLIC_ID")
			tcgPrivateId := os.Getenv("TCGPLAYER_PRIVATE_ID")
			tcgSKUPath := os.Getenv("MTGJSON_TCGSKU_PATH")
			if tcgPublicId == "" || tcgPrivateId == "" || tcgSKUPath == "" {
				return nil, errors.New("missing TCGPLAYER_PUBLIC_ID or TCGPLAYER_PRIVATE_ID or MTGJSON_TCGSKU_PATH env vars")
			}

			scraper, err := tcgplayer.NewScraperSealed(tcgPublicId, tcgPrivateId)
			if err != nil {
				return nil, err
			}

			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("TCG_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}

			start := time.Now()
			skuReader, err := loadData(tcgSKUPath)
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
			scraper := tcgplayer.NewScraperSYP(tcgAuth)
			scraper.LogCallback = GlobalLogCallback

			start := time.Now()
			skuReader, err := loadData(tcgSKUPath)
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

	"cardmarket_lorcana": &scraperOption{
		Init: func() (mtgban.Scraper, error) {
			mkmAppToken := os.Getenv("MKM_APP_TOKEN")
			mkmAppSecret := os.Getenv("MKM_APP_SECRET")
			if mkmAppToken == "" || mkmAppSecret == "" {
				return nil, errors.New("missing MKM_APP_TOKEN or MKM_APP_SECRET env vars")
			}

			scraper, err := cardmarket.NewScraperIndex(cardmarket.GameIdLorcana, mkmAppToken, mkmAppSecret)
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
	"cardtrader_lorcana": &scraperOption{
		Init: func() (mtgban.Scraper, error) {
			ctTokenBearer := os.Getenv("CARDTRADER_TOKEN_BEARER")
			if ctTokenBearer == "" {
				return nil, errors.New("missing CARDTRADER_TOKEN_BEARER env var")
			}

			scraper, err := cardtrader.NewScraperMarket(cardtrader.GameIdLorcana, ctTokenBearer)
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = GlobalLogCallback
			scraper.ShareCode = os.Getenv("CT_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"coolstuffinc_lorcana": &scraperOption{
		Init: func() (mtgban.Scraper, error) {
			scraper := coolstuffinc.NewScraper(coolstuffinc.GameLorcana)
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CSI_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"starcitygames_lorcana": &scraperOption{
		Init: func() (mtgban.Scraper, error) {
			scgGUID := os.Getenv("SCG_GUID")
			scgBearer := os.Getenv("SCG_BEARER")
			if scgGUID == "" || scgBearer == "" {
				return nil, errors.New("missing SCG_GUID or SCG_BEARER env var")
			}

			scraper := starcitygames.NewScraper(starcitygames.GameLorcana, scgGUID, scgBearer)
			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("SCG_PARTNER")
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"strikezone_lorcana": &scraperOption{
		Init: func() (mtgban.Scraper, error) {
			scraper := strikezone.NewScraper(strikezone.GameLorcana)
			scraper.LogCallback = GlobalLogCallback
			if MaxConcurrency != 0 {
				scraper.MaxConcurrency = MaxConcurrency
			}
			return scraper, nil
		},
	},
	"tcg_index_lorcana": &scraperOption{
		Init: func() (mtgban.Scraper, error) {
			tcgPublicId := os.Getenv("TCGPLAYER_PUBLIC_ID")
			tcgPrivateId := os.Getenv("TCGPLAYER_PRIVATE_ID")
			if tcgPublicId == "" || tcgPrivateId == "" {
				return nil, errors.New("missing TCGPLAYER_PUBLIC_ID or TCGPLAYER_PRIVATE_ID env vars")
			}
			scraper, err := tcgplayer.NewLorcanaIndex(tcgPublicId, tcgPrivateId)
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
	"tcg_market_lorcana": &scraperOption{
		Init: func() (mtgban.Scraper, error) {
			tcgPublicId := os.Getenv("TCGPLAYER_PUBLIC_ID")
			tcgPrivateId := os.Getenv("TCGPLAYER_PRIVATE_ID")
			if tcgPublicId == "" || tcgPrivateId == "" {
				return nil, errors.New("missing TCGPLAYER_PUBLIC_ID or TCGPLAYER_PRIVATE_ID env vars")
			}
			scraper, err := tcgplayer.NewLorcanaScraper(tcgPublicId, tcgPrivateId)
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
	inventory, err := seller.Inventory()
	if err != nil {
		return err
	}

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
	buylist, err := vendor.Buylist()
	if err != nil {
		return err
	}

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

func dumpSeller(seller mtgban.Seller, outputPath, format string) error {
	writer, err := putData("retail/"+seller.Info().Shorthand+"."+format, outputPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	if strings.HasSuffix(format, ".xz") {
		xzWriter, err := xz.NewWriter(writer)
		if err != nil {
			return err
		}
		defer xzWriter.Close()
		writer = xzWriter
	} else if strings.HasSuffix(format, ".bz2") {
		bz2Writer, err := bzip2.NewWriter(writer, nil)
		if err != nil {
			return err
		}
		defer bz2Writer.Close()
		writer = bz2Writer
	}

	switch strings.Split(format, ".")[0] {
	case "json":
		err = mtgban.WriteSellerToJSON(seller, writer)
	case "csv":
		err = mtgban.WriteSellerToCSV(seller, writer)
	case "ndjson":
		err = writeSellerToNDJSON(seller, writer)
	default:
		err = errors.New("invalid format")
	}

	return err
}

func dumpVendor(vendor mtgban.Vendor, outputPath, format string) error {
	writer, err := putData("buylist/"+vendor.Info().Shorthand+"."+format, outputPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	if strings.HasSuffix(format, ".xz") {
		xzWriter, err := xz.NewWriter(writer)
		if err != nil {
			return err
		}
		defer xzWriter.Close()
		writer = xzWriter
	} else if strings.HasSuffix(format, ".bz2") {
		bz2Writer, err := bzip2.NewWriter(writer, nil)
		if err != nil {
			return err
		}
		defer bz2Writer.Close()
		writer = bz2Writer
	}

	switch strings.Split(format, ".")[0] {
	case "json":
		err = mtgban.WriteVendorToJSON(vendor, writer)
	case "csv":
		err = mtgban.WriteVendorToCSV(vendor, writer)
	case "ndjson":
		err = writeVendorToNDJSON(vendor, writer)
	default:
		err = errors.New("invalid format")
	}

	return err
}

func dump(bc *mtgban.BanClient, outputPath, format string, meta bool) error {
	log.Println("Writing results to", outputPath)

	for _, seller := range bc.Sellers() {
		err := dumpSeller(seller, outputPath, format)
		if err != nil {
			return err
		}

		if meta && format != "json" {
			sellerMeta := mtgban.NewSellerFromInventory(nil, seller.Info())
			err := dumpSeller(sellerMeta, outputPath, "json")
			if err != nil {
				return err
			}
		}
	}

	for _, vendor := range bc.Vendors() {
		err := dumpVendor(vendor, outputPath, format)
		if err != nil {
			return err
		}

		if meta && format != "json" {
			vendorMeta := mtgban.NewVendorFromBuylist(nil, vendor.Info())
			err := dumpVendor(vendorMeta, outputPath, "json")
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// TODO: potentially this function could initialize more than one client
func initializeBucket(outputPath string) error {
	u, err := url.Parse(outputPath)
	if err != nil {
		return err
	}

	switch u.Scheme {
	case "http", "https":
		// nothing to do here
	case "gs":
		if GCSBucket != nil {
			return nil
		}

		serviceAcc := os.Getenv("GCS_SVC_ACC")
		if serviceAcc == "" {
			return errors.New("missing GCS_SVC_ACC for GCS access")
		}

		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(serviceAcc))
		if err != nil {
			return fmt.Errorf("error creating the GCS client %w", err)
		}

		GCSBucket = client.Bucket(u.Host)
	case "b2":
		if B2Bucket != nil {
			return nil
		}

		accessKey := os.Getenv("B2_KEY_ID")
		secretKey := os.Getenv("B2_APP_KEY")
		if accessKey == "" || secretKey == "" {
			return errors.New("missing required B2 environment variables")
		}

		client, err := b2.NewClient(context.TODO(), accessKey, secretKey)
		if err != nil {
			return err
		}

		B2Bucket, err = client.Bucket(context.TODO(), u.Host)
		if err != nil {
			return err
		}
	default:
		_, err := os.Stat(u.Path)
		if os.IsNotExist(err) {
			return errors.New("path does not exist")
		}
	}

	return nil
}

func run() int {
	start := time.Now()

	for key, val := range options {
		flag.BoolVar(&val.Enabled, key, false, "Enable "+strings.Title(key))
	}

	datastoreOpt := flag.String("datastore", "", "Path to AllPrintings file")
	outputPathOpt := flag.String("output-path", "", "Path where to dump results")

	scrapersOpt := flag.String("scrapers", "", "Comma-separated list of scrapers to enable")
	sellersOpt := flag.String("sellers", "", "Comma-separated list of sellers to enable")
	vendorsOpt := flag.String("vendors", "", "Comma-separated list of vendors to enable")

	fileFormatOpt := flag.String("format", "json", "File format of the output files (json/csv/ndjson)")
	metaOpt := flag.Bool("meta", false, "When format is not json, output a second file for scraper metadata")

	listOpt := flag.Bool("list", false, "List all items present in the specified bucket path")
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

	u, err := url.Parse(*outputPathOpt)
	if err != nil {
		log.Println("cannot parse output-path", err)
		return 1
	}
	u.Path = filepath.Dir(u.Path)

	err = initializeBucket(u.String())
	if err != nil {
		log.Println("cannot initilize buckets:", err)
		return 1
	}

	if *listOpt {
		if u.Scheme == "b2" {
			iterator := B2Bucket.List(context.TODO())
			for iterator.Next() {
				fmt.Println(iterator.Object().Name())
			}
		}
		return 0
	}

	if *datastoreOpt == "" {
		log.Println("Missing datatore argument")
		return 1
	}
	// Sanity check in case things are on different providers
	err = initializeBucket(*datastoreOpt)
	if err != nil {
		log.Println(err)
		return 1
	}

	// Enable Scrapers or Sellers/Vendors
	scraps := strings.Split(*scrapersOpt, ",")
	for _, name := range scraps {
		if options[name] != nil {
			options[name].Enabled = true
		}
	}
	if *sellersOpt != "" {
		sells := strings.Split(*sellersOpt, ",")
		for _, name := range sells {
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
		vends := strings.Split(*vendorsOpt, ",")
		for _, name := range vends {
			if options[name] == nil {
				log.Println("Vendor", name, "not found")
				return 1
			}
			options[name].Enabled = true
			options[name].OnlySeller = false
			options[name].OnlyVendor = true
		}
	}

	now := time.Now()
	datastoreReader, err := loadData(*datastoreOpt)
	if err != nil {
		log.Println(err)
		return 1
	}
	defer datastoreReader.Close()

	now = time.Now()
	err = mtgmatcher.LoadDatastore(datastoreReader)
	if err != nil {
		log.Println(err)
		return 1
	}
	log.Println("loading datastore took:", time.Since(now))

	bc := mtgban.NewClient()

	// Initialize the enabled scrapers
	for _, opt := range options {
		if opt.Enabled {
			scraper, err := opt.Init()
			if err != nil {
				log.Println(err)
				return 1
			}
			if opt.OnlySeller {
				bc.RegisterSeller(scraper)
			} else if opt.OnlyVendor {
				bc.RegisterVendor(scraper)
			} else {
				bc.Register(scraper)
			}
		}
	}

	if len(bc.Scrapers()) == 0 {
		log.Println("No scraper configured, run with -h for a list of commands")
		return 1
	}
	log.Println("BAN client configured with", len(bc.Sellers()), "sellers and", len(bc.Vendors()), "vendors")

	now = time.Now()
	// Load the data
	err = bc.Load()
	if err != nil {
		log.Println("Something didn't work while scraping...")
		log.Println(err)
		return 1
	}
	log.Println("loading scraper data took:", time.Since(now))

	now = time.Now()
	// Dump the results
	err = dump(bc, *outputPathOpt, *fileFormatOpt, *metaOpt)
	if err != nil {
		log.Println(err)
		return 1
	}
	log.Println("uploading data took:", time.Since(now))

	log.Println("Completed in", time.Since(start))

	return 0
}

func main() {
	os.Exit(run())
}

func putData(suffix, outputPath string) (io.WriteCloser, error) {
	filePath := fmt.Sprintf("%s/%s", outputPath, suffix)

	var writer io.WriteCloser
	u, err := url.Parse(filePath)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "gs":
		writer = GCSBucket.Object(u.Path).NewWriter(context.TODO())
	case "b2":
		dst := strings.TrimPrefix(u.Path, "/")
		obj := B2Bucket.Object(dst).NewWriter(context.TODO())

		writer = obj
	default:
		file, err := os.Create(filePath)
		if err != nil {
			return nil, err
		}
		writer = file
	}

	return writer, nil
}

func loadData(pathOpt string) (io.ReadCloser, error) {
	var reader io.ReadCloser

	u, err := url.Parse(pathOpt)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "http", "https":
		resp, err := cleanhttp.DefaultClient().Get(pathOpt)
		if err != nil {
			return nil, err
		}

		reader = resp.Body
	case "b2":
		src := strings.TrimPrefix(u.Path, "/")
		obj := B2Bucket.Object(src).NewReader(context.TODO())
		obj.ConcurrentDownloads = 20

		reader = obj
	default:
		file, err := os.Open(pathOpt)
		if err != nil {
			return nil, err
		}

		reader = file
	}

	if strings.HasSuffix(pathOpt, "xz") {
		xzReader, err := xzReader.NewReader(reader, 0)
		if err != nil {
			return nil, err
		}
		reader = io.NopCloser(xzReader)
	} else if strings.HasSuffix(pathOpt, "bz2") {
		bz2Reader, err := bzip2.NewReader(reader, nil)
		if err != nil {
			return nil, err
		}
		reader = bz2Reader
	} else if strings.HasSuffix(pathOpt, "gz") {
		zipReader, err := gzip.NewReader(reader)
		if err != nil {
			return nil, err
		}
		reader = zipReader
	}

	return reader, err
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
