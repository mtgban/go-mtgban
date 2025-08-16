package main

import (
	"compress/bzip2"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/Backblaze/blazer/b2"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/scizorman/go-ndjson"
	"github.com/xi2/xz"
	"google.golang.org/api/option"

	_ "github.com/joho/godotenv/autoload"
	"github.com/mtgban/go-mtgban/abugames"
	"github.com/mtgban/go-mtgban/cardkingdom"
	"github.com/mtgban/go-mtgban/cardmarket"
	"github.com/mtgban/go-mtgban/cardsphere"
	"github.com/mtgban/go-mtgban/cardtrader"
	"github.com/mtgban/go-mtgban/coolstuffinc"
	"github.com/mtgban/go-mtgban/hareruya"
	"github.com/mtgban/go-mtgban/magiccorner"
	"github.com/mtgban/go-mtgban/manapool"
	"github.com/mtgban/go-mtgban/mintcard"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgseattle"
	"github.com/mtgban/go-mtgban/mtgstocks"
	"github.com/mtgban/go-mtgban/ninetyfive"
	"github.com/mtgban/go-mtgban/sealedev"
	"github.com/mtgban/go-mtgban/starcitygames"
	"github.com/mtgban/go-mtgban/strikezone"
	"github.com/mtgban/go-mtgban/tcgplayer"
	"github.com/mtgban/go-mtgban/toamagic"
	"github.com/mtgban/go-mtgban/trollandtoad"
	"github.com/mtgban/go-mtgban/wizardscupboard"

	"github.com/mtgban/go-mtgban/mtgban"
)

var GCSBucket *storage.BucketHandle
var B2Bucket *b2.Bucket

var GlobalLogCallback mtgban.LogCallbackFunc = log.Printf

var MaxConcurrency = os.Getenv("MAX_CONCURRENCY")

const AllPrintingsURL = "https://mtgjson.com/api/v5/AllPrintings.json.xz"

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

var options = map[string]*scraperOption{
	"abugames": {
		Init: func() (mtgban.Scraper, error) {
			scraper := abugames.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"cardkingdom": {
		Init: func() (mtgban.Scraper, error) {
			scraper := cardkingdom.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CK_PARTNER")
			return scraper, nil
		},
	},
	"cardkingdom_sealed": {
		Init: func() (mtgban.Scraper, error) {
			scraper := cardkingdom.NewScraperSealed()
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CK_PARTNER")
			return scraper, nil
		},
	},
	"cardsphere": {
		Init: func() (mtgban.Scraper, error) {
			csphereToken := os.Getenv("CARDSPHERE_TOKEN")
			if csphereToken == "" {
				return nil, errors.New("missing CARDSPHERE_TOKEN env var")
			}
			scraper := cardsphere.NewScraper(csphereToken)
			scraper.LogCallback = GlobalLogCallback
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
			scraper.LogCallback = GlobalLogCallback
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
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"coolstuffinc": {
		OnlyVendor: true,
		Init: func() (mtgban.Scraper, error) {
			scraper := coolstuffinc.NewScraper(coolstuffinc.GameMagic)
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CSI_PARTNER")
			return scraper, nil
		},
	},
	"coolstuffinc_official": {
		Init: func() (mtgban.Scraper, error) {
			csiKey := os.Getenv("CSI_KEY")
			if csiKey == "" {
				return nil, errors.New("missing CSI_KEY env var")
			}

			scraper := coolstuffinc.NewScraperOfficial(csiKey)
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CSI_PARTNER")
			return scraper, nil
		},
	},
	"coolstuffinc_sealed": {
		OnlyVendor: true,
		Init: func() (mtgban.Scraper, error) {
			scraper := coolstuffinc.NewScraperSealed()
			scraper.LogCallback = GlobalLogCallback
			scraper.Partner = os.Getenv("CSI_PARTNER")
			return scraper, nil
		},
	},
	"hareruya": {
		Init: func() (mtgban.Scraper, error) {
			scraper, err := hareruya.NewScraper()
			if err != nil {
				return nil, err
			}
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
			return scraper, nil
		},
	},
	"manapool": {
		Init: func() (mtgban.Scraper, error) {
			scraper := manapool.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"mintcard": {
		Init: func() (mtgban.Scraper, error) {
			scraper := mintcard.NewScraper()
			scraper.Partner = os.Getenv("MP_AFFILIATE")
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"mkm_index": {
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
			scraper.Affiliate = os.Getenv("MKM_AFFILIATE")
			return scraper, nil
		},
	},
	"mkm_sealed": {
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
			scraper.Affiliate = os.Getenv("MKM_AFFILIATE")
			return scraper, nil
		},
	},
	"mtgseattle": {
		Init: func() (mtgban.Scraper, error) {
			scraper := mtgseattle.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"mtgstocks": {
		Init: func() (mtgban.Scraper, error) {
			scraper := mtgstocks.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"ninetyfive": {
		Init: func() (mtgban.Scraper, error) {
			scraper, err := ninetyfive.NewScraper(ninetyfive.GameMagic)
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = GlobalLogCallback
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
			scraper.Affiliate = os.Getenv("TCG_AFFILIATE")
			scraper.BuylistAffiliate = os.Getenv("CK_PARTNER")
			scraper.LogCallback = GlobalLogCallback
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
			scraper.Affiliate = os.Getenv("SCG_AFFILIATE")
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
			scraper.Affiliate = os.Getenv("SCG_AFFILIATE")
			return scraper, nil
		},
	},
	"strikezone": {
		Init: func() (mtgban.Scraper, error) {
			scraper := strikezone.NewScraper(strikezone.GameMagic)
			scraper.LogCallback = GlobalLogCallback
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

			scraper := tcgplayer.NewScraperIndex(tcgPublicId, tcgPrivateId)
			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("TCG_AFFILIATE")
			num, _ := strconv.Atoi(MaxConcurrency)
			if num != 0 {
				scraper.MaxConcurrency = num
			}
			return scraper, nil
		},
	},
	"tcg_market": {
		Init: func() (mtgban.Scraper, error) {
			tcgPublicId := os.Getenv("TCGPLAYER_PUBLIC_ID")
			tcgPrivateId := os.Getenv("TCGPLAYER_PRIVATE_ID")
			mtgjsonTCGSKUPath := os.Getenv("MTGJSON_TCGSKU_PATH")
			if tcgPublicId == "" || tcgPrivateId == "" || mtgjsonTCGSKUPath == "" {
				return nil, errors.New("missing TCGPLAYER_PUBLIC_ID or TCGPLAYER_PRIVATE_ID or MTGJSON_TCGSKU_PATH env vars")
			}

			scraper := tcgplayer.NewScraperMarket(tcgPublicId, tcgPrivateId)
			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("TCG_AFFILIATE")
			num, _ := strconv.Atoi(MaxConcurrency)
			if num != 0 {
				scraper.MaxConcurrency = num
			}

			mtgjsonReader, err := loadData(mtgjsonTCGSKUPath)
			if err != nil {
				return nil, err
			}
			defer mtgjsonReader.Close()
			skus, err := tcgplayer.LoadTCGSKUs(mtgjsonReader)
			if err != nil {
				return nil, err
			}
			scraper.SKUsData = skus.Data
			return scraper, nil
		},
	},
	"tcg_sealed": {
		Init: func() (mtgban.Scraper, error) {
			tcgPublicId := os.Getenv("TCGPLAYER_PUBLIC_ID")
			tcgPrivateId := os.Getenv("TCGPLAYER_PRIVATE_ID")
			if tcgPublicId == "" || tcgPrivateId == "" {
				return nil, errors.New("missing TCGPLAYER_PUBLIC_ID or TCGPLAYER_PRIVATE_ID env vars")
			}

			scraper := tcgplayer.NewScraperSealed(tcgPublicId, tcgPrivateId)
			scraper.LogCallback = GlobalLogCallback
			scraper.Affiliate = os.Getenv("TCG_AFFILIATE")
			return scraper, nil
		},
	},
	"tcg_syplist": {
		Init: func() (mtgban.Scraper, error) {
			tcgAuth := os.Getenv("TCGPLAYER_AUTH")
			if tcgAuth == "" {
				return nil, errors.New("missing TCGPLAYER_AUTH env var")
			}
			scraper := tcgplayer.NewScraperSYP(tcgAuth)
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"toamagic": {
		Init: func() (mtgban.Scraper, error) {
			scraper := toamagic.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			return scraper, nil
		},
	},
	"trollandtoad": {
		Init: func() (mtgban.Scraper, error) {
			scraper := trollandtoad.NewScraper()
			scraper.LogCallback = GlobalLogCallback
			num, _ := strconv.Atoi(MaxConcurrency)
			if num != 0 {
				scraper.MaxConcurrency = num
			}
			return scraper, nil
		},
	},
	"wizardscupboard": {
		Init: func() (mtgban.Scraper, error) {
			scraper := wizardscupboard.NewScraper()
			scraper.LogCallback = GlobalLogCallback
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
	writer, err := putData(seller.Info().Shorthand+"_Retail."+format, outputPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	switch format {
	case "json":
		err = mtgban.WriteSellerToJSON(seller, writer)
	case "csv":
		err = mtgban.WriteSellerToCSV(seller, writer)
	case "ndjson":
		err = writeSellerToNDJSON(seller, writer)
	}

	return err
}

func dumpVendor(vendor mtgban.Vendor, outputPath, format string) error {
	writer, err := putData(vendor.Info().Shorthand+"_Buylist."+format, outputPath)
	if err != nil {
		return err
	}
	defer writer.Close()

	switch format {
	case "json":
		err = mtgban.WriteVendorToJSON(vendor, writer)
	case "csv":
		err = mtgban.WriteVendorToCSV(vendor, writer)
	case "ndjson":
		err = writeVendorToNDJSON(vendor, writer)
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

	mtgjsonOpt := flag.String("mtgjson", "", "Path to AllPrintings file")
	outputPathOpt := flag.String("output-path", "", "Path where to dump results")

	scrapersOpt := flag.String("scrapers", "", "Comma-separated list of scrapers to enable")
	sellersOpt := flag.String("sellers", "", "Comma-separated list of sellers to enable")
	vendorsOpt := flag.String("vendors", "", "Comma-separated list of vendors to enable")

	fileFormatOpt := flag.String("format", "json", "File format of the output files (json/csv/ndjson)")
	metaOpt := flag.Bool("meta", false, "When format is not json, output a second file for scraper metadata")

	versionOpt := flag.Bool("v", false, "Print version information")
	listOpt := flag.Bool("l", false, "List all scrapers available")
	flag.Parse()

	log.Println("bantool version", Commit)
	if *versionOpt {
		return 0
	}

	if *listOpt {
		keys := slices.Sorted(maps.Keys(options))
		for _, key := range keys {
			var list []string
			if options[key].OnlyVendor {
				list = append(list, "❌")
			} else {
				list = append(list, "✅")
			}
			if options[key].OnlySeller {
				list = append(list, "❌")
			} else {
				list = append(list, "✅")
			}
			fmt.Println(list, key)
		}
		return 0
	}

	switch *fileFormatOpt {
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
				log.Fatalln("Seller", name, "not found")
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
				log.Fatalln("Vendor", name, "not found")
			}
			options[name].Enabled = true
			options[name].OnlySeller = false
			options[name].OnlyVendor = true
		}
	}

	// Load static data
	if *mtgjsonOpt == "" {
		log.Println("No AllPrintings specified, loading from network...")
		*mtgjsonOpt = AllPrintingsURL
	}

	// Sanity check in case things are on different providers
	err = initializeBucket(*mtgjsonOpt)
	if err != nil {
		log.Println(err)
		return 1
	}

	now := time.Now()
	mtgjsonReader, err := loadData(*mtgjsonOpt)
	if err != nil {
		log.Println("Couldn't load MTGJSON/Allprintings")
		log.Println(err)
		return 1
	}
	defer mtgjsonReader.Close()

	now = time.Now()
	err = mtgmatcher.LoadDatastore(mtgjsonReader)
	if err != nil {
		log.Println("Couldn't parse MTGJSON/AllPrintings")
		log.Println(err)
		return 1
	}
	log.Println("loading mtgjson took:", time.Since(now))

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
	today := time.Now().Format("2006-01-02")
	filePath := fmt.Sprintf("%s/%s_%s", outputPath, today, suffix)

	var writer io.WriteCloser
	u, err := url.Parse(filePath)
	if err != nil {
		return nil, err
	}
	switch u.Scheme {
	case "gs":
		writer = GCSBucket.Object(u.Path).NewWriter(context.Background())
	case "b2":
		dst := strings.TrimPrefix(u.Path, "/")
		writer = B2Bucket.Object(dst).NewWriter(context.Background())
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
		xzReader, err := xz.NewReader(reader, 0)
		if err != nil {
			return nil, err
		}
		reader = io.NopCloser(xzReader)
	} else if strings.HasSuffix(pathOpt, "bz2") {
		reader = io.NopCloser(bzip2.NewReader(reader))
	}

	return reader, err
}
