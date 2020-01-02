package miniaturemarket

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"

	http "github.com/hashicorp/go-retryablehttp"
	"github.com/kodabb/go-mtgban/mtgban"
)

type mmJSON struct {
	Name      string `json:"name"`
	Set       string `json:"mtg_set"`
	Price     string `json:"price"`
	Condition string `json:"mtg_condition"`
	Rarity    string `json:"mtg_rarity"`
	IsFoil    bool   `json:"foil"`
}

var l = log.New(os.Stderr, "", 0)

const (
	maxConcurrency       = 4
	magicSinglesCategory = "1466"
	mmBuylistURL         = "https://www.miniaturemarket.com/buyback/data/products/"
	lastPage             = 10
)

type Miniaturemarket struct {
	inventory []mtgban.Entry
	buylist   []mtgban.Entry
}

func NewVendor() mtgban.Scraper {
	mm := Miniaturemarket{}
	return &mm
}

type resultChan struct {
	err   error
	cards []MMCard
}

func (mm *Miniaturemarket) processEntry(page int) (res resultChan) {
	resp, err := http.PostForm(mmBuylistURL, url.Values{
		"category": {magicSinglesCategory},
		"page":     {fmt.Sprintf("%d", page)},
	})
	if err != nil {
		res.err = err
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	var db []mmJSON
	err = json.Unmarshal(data, &db)
	if err != nil {
		res.err = err
		return
	}

	for _, card := range db {
		// This field is always "[Foil] Near Mint" or null for sealed
		if !strings.Contains(card.Condition, "Near Mint") ||
			card.Set == "Bulk MTG" || card.Rarity == "Sealed Product" {
			continue
		}

		// This field is always "<name> - <set> (<condition>)"
		names := strings.Split(card.Name, " - ")
		cardName := names[0]

		// Skip non-singles cards
		if strings.Contains(cardName, "Token") ||
			strings.Contains(cardName, "Emblem") ||
			strings.Contains(cardName, "Oversized") ||
			cardName == "Experience Counter" || cardName == "Poison Counter" {
			continue
		}

		// Skip foreign cards
		if (card.Set == "Legends" && strings.Contains(cardName, "Italian")) ||
			(card.Set == "Portal Three Kingdoms" && (strings.Contains(cardName, "Chinese") || strings.Contains(cardName, "Japanese"))) ||
			(card.Set == "Duel Decks: Jace vs. Chandra" && strings.Contains(cardName, "Japanese")) {
			continue
		}

		price, err := strconv.ParseFloat(card.Price, 64)
		if err != nil {
			res.err = err
			return
		}

		cc := MMCard{
			Name:    cardName,
			Set:     card.Set,
			Foil:    card.IsFoil,
			Pricing: price,
		}

		res.cards = append(res.cards, cc)
	}

	return
}

// Scrape returns an array of Entry, containing pricing and card information
func (mm *Miniaturemarket) Scrape() ([]mtgban.Entry, error) {
	pages := make(chan int)
	results := make(chan resultChan)
	var wg sync.WaitGroup

	for i := 0; i < maxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				results <- mm.processEntry(page)
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 1; i <= lastPage; i++ {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(results)
	}()

	for result := range results {
		if result.err != nil {
			l.Println(result.err)
			continue
		}
		for i := range result.cards {
			mm.buylist = append(mm.buylist, &result.cards[i])
		}
	}

	return mm.buylist, nil
}
