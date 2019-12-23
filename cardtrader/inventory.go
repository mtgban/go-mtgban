package cardtrader

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	http "github.com/hashicorp/go-retryablehttp"
	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

const (
	maxCategoryId  = 62389
	maxConcurrency = 6
)

const ctInventoryURL = "https://www.cardtrader.com/cards/%d/filter.json"

// CardtraderInventory is the Scraper for the Card Trader vendor.
type CardtraderInventory struct{}

// NewInventory initializes a Scraper for retriving inventory information.
func NewInventory() mtgban.Scraper {
	ct := CardtraderInventory{}
	return &ct
}

type resultChan struct {
	err   error
	cards []CTCard
}

type ctJSON struct {
	Blueprint struct {
		CategoryId int    `json:"category_id"`
		Name       string `json:"name"`
		Variant    string `json:"display_name"`
	} `json:"blueprint"`
	Products []struct {
		Image      string `json:"image"`
		Cents      int    `json:"price_cents"`
		Quantity   int    `json:"quantity"`
		Properties struct {
			Condition string `json:"condition"`
			Language  string `json:"mtg_language"`
			Number    string `json:"collector_number"`
			Foil      bool   `json:"mtg_foil"`
			Altered   bool   `json:"altered"`
			Signed    bool   `json:"signed"`
		} `json:"properties_hash"`
		Expansion struct {
			Name string `json:"name"`
			Code string `json:"code"`
		} `json:"expansion"`
		User struct {
			Name string `json:"username"`
		} `json:"user"`
	} `json:"products"`
}

var l = log.New(os.Stderr, "", 0)

func (ct *CardtraderInventory) processEntry(categoryId int) (res resultChan) {
	resp, err := http.Post(fmt.Sprintf(ctInventoryURL, categoryId), "application/json", nil)
	if err != nil {
		res.err = fmt.Errorf("Error processing entry %d: %v", categoryId, err)
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		res.err = err
		return
	}

	var db ctJSON
	err = json.Unmarshal(data, &db)
	if err != nil {
		res.err = err
		return
	}

	// Skip anything that is not singles
	if db.Blueprint.CategoryId != 1 {
		return
	}

	for _, product := range db.Products {
		if product.Properties.Language != "en" || product.Properties.Altered {
			continue
		}

		if product.Quantity < 1 {
			continue
		}

		conditions := product.Properties.Condition
		if product.Properties.Signed {
			conditions = "HP"
		}

		maybeId := strings.TrimSuffix(filepath.Base(product.Image), filepath.Ext(product.Image))
		if len(maybeId) != 36 {
			maybeId = ""
		}

		maybeNumber := strings.Trim(product.Properties.Number, "0")
		if product.Expansion.Name == "Arabian Nights" || product.Expansion.Name == "Portal" {
			maybeNumber = strings.Replace(maybeNumber, "a", "", 1)
			maybeNumber = strings.Replace(maybeNumber, "b", mtgjson.SuffixLightMana, 1)
		}

		cc := CTCard{
			Name: strings.TrimSpace(db.Blueprint.Name),
			Set:  product.Expansion.Name,
			Foil: product.Properties.Foil,

			Vendor:    product.User.Name,
			Condition: conditions,
			Pricing:   float64(product.Cents) / 100,
			Qty:       product.Quantity,

			// Save information that will simplify lookup later
			scryfallId: maybeId,
			number:     maybeNumber,
			setCode:    strings.ToUpper(product.Expansion.Code),
		}

		res.cards = append(res.cards, cc)
	}

	return
}

// Scrape returns an array of Entry, containing pricing and card information
func (ct *CardtraderInventory) Scrape() ([]mtgban.Entry, error) {
	jobs := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for categoryId := range jobs {
				results <- ct.processEntry(categoryId)
			}
			wg.Done()
		}()
	}

	go func() {
		/* Start from a random place */
		rand.Seed(time.Now().UnixNano())
		start := rand.Intn(maxCategoryId)
		for i := start; i < start+maxCategoryId; i++ {
			jobs <- 1 + i%maxCategoryId
		}
		close(jobs)

		wg.Wait()
		close(results)
	}()

	db := []mtgban.Entry{}
	for result := range results {
		if result.err != nil {
			l.Println(result.err)
			continue
		}
		for i := range result.cards {
			db = append(db, &result.cards[i])
		}
	}

	return db, nil
}
