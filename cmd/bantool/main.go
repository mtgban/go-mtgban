package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/kodabb/go-mtgban/abugames"
	"github.com/kodabb/go-mtgban/amazon"
	"github.com/kodabb/go-mtgban/blueprint"
	"github.com/kodabb/go-mtgban/cardkingdom"
	"github.com/kodabb/go-mtgban/cardmarket"
	"github.com/kodabb/go-mtgban/cardshark"
	"github.com/kodabb/go-mtgban/cardsphere"
	"github.com/kodabb/go-mtgban/cardtrader"
	"github.com/kodabb/go-mtgban/coolstuffinc"
	"github.com/kodabb/go-mtgban/hareruya"
	"github.com/kodabb/go-mtgban/jupitergames"
	"github.com/kodabb/go-mtgban/magiccorner"
	"github.com/kodabb/go-mtgban/mtgseattle"
	"github.com/kodabb/go-mtgban/mtgstocks"
	"github.com/kodabb/go-mtgban/mythicmtg"
	"github.com/kodabb/go-mtgban/ninetyfive"
	"github.com/kodabb/go-mtgban/starcitygames"
	"github.com/kodabb/go-mtgban/strikezone"
	"github.com/kodabb/go-mtgban/tcgplayer"
	"github.com/kodabb/go-mtgban/toamagic"
	"github.com/kodabb/go-mtgban/trollandtoad"
	"github.com/kodabb/go-mtgban/wizardscupboard"
	"google.golang.org/api/option"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var date = time.Now().Format("2006-01-02")
var GCSBucket *storage.BucketHandle

type scraperOption struct {
	Enabled    bool
	OnlySeller bool
	OnlyVendor bool
	Init       func() (mtgban.Scraper, error)
}

var options = map[string]*scraperOption{
	"magiccorner": {
		Init: func() (mtgban.Scraper, error) {
			scraper, err := magiccorner.NewScraper()
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"strikezone": {
		OnlySeller: true,
		Init: func() (mtgban.Scraper, error) {
			scraper := strikezone.NewScraper()
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"cardkingdom": {
		Init: func() (mtgban.Scraper, error) {
			scraper := cardkingdom.NewScraper()
			scraper.LogCallback = log.Printf
			scraper.Partner = os.Getenv("CK_PARTNER")
			return scraper, nil
		},
	},
	"abugames": {
		Init: func() (mtgban.Scraper, error) {
			scraper := abugames.NewScraper()
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"ninetyfive": {
		Init: func() (mtgban.Scraper, error) {
			scraper, err := ninetyfive.NewScraper(true)
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"mythicmtg": {
		Init: func() (mtgban.Scraper, error) {
			scraper := mythicmtg.NewScraper()
			scraper.LogCallback = log.Printf
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
			scraper.LogCallback = log.Printf
			scraper.Affiliate = os.Getenv("TCG_AFFILIATE")
			scraper.MaxConcurrency = 6
			return scraper, nil
		},
	},
	"tcg_market": {
		Init: func() (mtgban.Scraper, error) {
			tcgPublicId := os.Getenv("TCGPLAYER_PUBLIC_ID")
			tcgPrivateId := os.Getenv("TCGPLAYER_PRIVATE_ID")
			if tcgPublicId == "" || tcgPrivateId == "" {
				return nil, errors.New("missing TCGPLAYER_PUBLIC_ID or TCGPLAYER_PRIVATE_ID env vars")
			}

			scraper := tcgplayer.NewScraperMarket(tcgPublicId, tcgPrivateId)
			scraper.LogCallback = log.Printf
			scraper.Affiliate = os.Getenv("TCG_AFFILIATE")
			scraper.MaxConcurrency = 6
			return scraper, nil
		},
	},
	"coolstuffinc": {
		OnlyVendor: true,
		Init: func() (mtgban.Scraper, error) {
			scraper := coolstuffinc.NewScraper()
			scraper.LogCallback = log.Printf
			scraper.Partner = os.Getenv("CSI_PARTNER")
			return scraper, nil
		},
	},
	"starcitygames": {
		OnlyVendor: true,
		Init: func() (mtgban.Scraper, error) {
			scgUsername := os.Getenv("SCG_USERNAME")
			scgPassword := os.Getenv("SCG_PASSWORD")
			if scgUsername == "" || scgPassword == "" {
				return nil, errors.New("missing SCG_USERNAME or SCG_PASSWORD env vars")
			}

			scraper, err := starcitygames.NewScraper(scgUsername, scgPassword)
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"trollandtoad": {
		Init: func() (mtgban.Scraper, error) {
			scraper := trollandtoad.NewScraper()
			scraper.LogCallback = log.Printf
			scraper.MaxConcurrency = 6
			return scraper, nil
		},
	},
	"jupitergames": {
		OnlySeller: true,
		Init: func() (mtgban.Scraper, error) {
			scraper := jupitergames.NewScraper()
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"mtgstocks": {
		Init: func() (mtgban.Scraper, error) {
			scraper := mtgstocks.NewScraper()
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"mtgstocks_index": {
		Init: func() (mtgban.Scraper, error) {
			scraper := mtgstocks.NewScraperIndex()
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"tcgplayer_syp": {
		Init: func() (mtgban.Scraper, error) {
			scraper := tcgplayer.NewScraperSYP()
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"wizardscupboard": {
		Init: func() (mtgban.Scraper, error) {
			scraper := wizardscupboard.NewScraper()
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"cardtrader": {
		Init: func() (mtgban.Scraper, error) {
			ctTokenBearer := os.Getenv("CARDTRADER_TOKEN_BEARER")
			if ctTokenBearer == "" {
				return nil, errors.New("missing CARDTRADER_TOKEN_BEARER env var")
			}

			scraper, err := cardtrader.NewScraperMarket(ctTokenBearer)
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = log.Printf
			scraper.ShareCode = os.Getenv("CT_SHARECODE")
			return scraper, nil
		},
	},
	"blueprint": {
		Init: func() (mtgban.Scraper, error) {
			scraper := blueprint.NewScraper()
			scraper.LogCallback = log.Printf
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

			scraper, err := cardmarket.NewScraperIndex(mkmAppToken, mkmAppSecret)
			if err != nil {
				return nil, err
			}
			scraper.Affiliate = "mtgban"
			scraper.LogCallback = log.Printf
			scraper.Affiliate = os.Getenv("MKM_AFFILIATE")
			return scraper, nil
		},
	},
	"cardshark": {
		Init: func() (mtgban.Scraper, error) {
			scraper := cardshark.NewScraper()
			scraper.LogCallback = log.Printf
			scraper.Referral = "kodamtg"
			return scraper, nil
		},
	},
	"cardsphere": {
		Init: func() (mtgban.Scraper, error) {
			csphereEmail := os.Getenv("CARDSPHERE_EMAIL")
			cspherePassword := os.Getenv("CARDSPHERE_PASSWORD")
			if csphereEmail == "" || cspherePassword == "" {
				return nil, errors.New("missing CARDSPHERE_EMAIL or CARDSPHERE_PASSWORD env vars")
			}

			scraper, err := cardsphere.NewScraperFull(csphereEmail, cspherePassword)
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"amazon": {
		Init: func() (mtgban.Scraper, error) {
			amzToken := os.Getenv("AMAZON_TOKEN")
			if amzToken == "" {
				return nil, errors.New("missing AMAZON_TOKEN env var")
			}

			scraper := amazon.NewScraper(amzToken)
			scraper.LogCallback = log.Printf
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
			scraper.LogCallback = log.Printf
			scraper.Partner = os.Getenv("CSI_PARTNER")
			return scraper, nil
		},
	},
	"mtgseattle": {
		Init: func() (mtgban.Scraper, error) {
			scraper := mtgseattle.NewScraper()
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"hareruya": {
		Init: func() (mtgban.Scraper, error) {
			scraper, err := hareruya.NewScraper()
			if err != nil {
				return nil, err
			}
			scraper.LogCallback = log.Printf
			return scraper, nil
		},
	},
	"toamagic": {
		Init: func() (mtgban.Scraper, error) {
			scraper := toamagic.NewScraper()
			scraper.LogCallback = log.Printf
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
			scraper.LogCallback = log.Printf
			scraper.Affiliate = os.Getenv("TCG_AFFILIATE")
			return scraper, nil
		},
	},
	"cardkingdom_sealed": {
		Init: func() (mtgban.Scraper, error) {
			scraper := cardkingdom.NewScraperSealed()
			scraper.LogCallback = log.Printf
			scraper.Partner = os.Getenv("CK_PARTNER")
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
			scraper.LogCallback = log.Printf
			scraper.Affiliate = os.Getenv("MKM_AFFILIATE")
			return scraper, nil
		},
	},
}

func dump(bc *mtgban.BanClient, outputPath string) error {
	for _, seller := range bc.Sellers() {

		fname := fmt.Sprintf("%s_%s_Inventory.json", date, seller.Info().Shorthand)
		filePath := path.Join(outputPath, fname)

		var writer io.WriteCloser
		if GCSBucket == nil {
			log.Println("Dumping seller", seller.Info().Shorthand)
			file, err := os.Create(filePath)
			if err != nil {
				return err
			}
			writer = file
		} else {
			log.Println("Uploading seller", seller.Info().Shorthand)
			writer = GCSBucket.Object(filePath).NewWriter(context.Background())
		}

		err := mtgban.WriteSellerToJSON(seller, writer)
		if err != nil {
			writer.Close()
			return err
		}
		writer.Close()
	}

	for _, vendor := range bc.Vendors() {
		fname := fmt.Sprintf("%s_%s_Buylist.json", date, vendor.Info().Shorthand)
		filePath := path.Join(outputPath, fname)

		var writer io.WriteCloser
		if GCSBucket == nil {
			log.Println("Dumping vendor", vendor.Info().Shorthand)
			file, err := os.Create(filePath)
			if err != nil {
				return err
			}
			writer = file
		} else {
			log.Println("Uploading vendor", vendor.Info().Shorthand)
			writer = GCSBucket.Object(filePath).NewWriter(context.Background())
		}

		err := mtgban.WriteVendorToJSON(vendor, writer)
		if err != nil {
			writer.Close()
			return err
		}
		writer.Close()
	}

	return nil
}

func run() int {
	for key, val := range options {
		flag.BoolVar(&val.Enabled, key, false, "Enable "+strings.Title(key))
	}

	mtgjsonOpt := flag.String("mtgjson", "allprintings5.json", "Path to AllPrintings file")
	outputPathOpt := flag.String("output-path", "", "Path where to dump results")
	serviceAccOpt := flag.String("svc-acc", "", "Service account with write permission on the bucket")

	scrapersOpt := flag.String("scrapers", "", "Comma-separated list of scrapers to enable")
	sellersOpt := flag.String("sellers", "", "Comma-separated list of sellers to enable")
	vendorsOpt := flag.String("vendors", "", "Comma-separated list of vendors to enable")

	flag.Parse()

	// Load static data
	err := mtgmatcher.LoadDatastoreFile(*mtgjsonOpt)
	if err != nil {
		log.Println("Couldn't load MTGJSON file...")
		log.Println(err)
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

	// If a service account file is passed in, create a bucket object and update the output path
	if *serviceAccOpt != "" {
		client, err := storage.NewClient(context.Background(), option.WithCredentialsFile(*serviceAccOpt))
		if err != nil {
			log.Println("error creating the GCS client", err)
			return 1
		}

		if u.Scheme != "gs" {
			log.Println("unsupported scheme in output-path")
			return 1
		}

		GCSBucket = client.Bucket(u.Host)

		// Trim to avoid creating an empty directory in the bucket
		*outputPathOpt = strings.TrimPrefix(u.Path, "/")
	} else if u.Scheme != "" {
		log.Println("missing svc-acc file for cloud access")
		return 1
	} else {
		_, err := os.Stat(*outputPathOpt)
		if os.IsNotExist(err) {
			log.Println("output-path does not exist")
			return 1
		}
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

	// Load the data
	err = bc.Load()
	if err != nil {
		log.Println("Something didn't work while scraping...")
		log.Println(err)
		return 1
	}

	// Dump the results
	err = dump(bc, *outputPathOpt)
	if err != nil {
		log.Println(err)
		return 1
	}

	log.Println("Completed")

	return 0
}

func main() {
	os.Exit(run())
}
