package tcgplayer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"sync"

	http "github.com/hashicorp/go-retryablehttp"

	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgdb"
)

type TCGPlayerMarket struct {
	LogCallback    mtgban.LogCallbackFunc
	InventoryDate  time.Time
	Affiliate      string
	MaxConcurrency int

	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord

	client *http.Client
}

func (tcg *TCGPlayerMarket) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGMkt] "+format, a...)
	}
}

func NewScraperMarket(publicId, privateId string) *TCGPlayerMarket {
	tcg := TCGPlayerMarket{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.marketplace = map[string]mtgban.InventoryRecord{}
	tcg.client = http.NewClient()
	tcg.client.Logger = nil
	tcg.client.HTTPClient.Transport = &authTransport{
		Parent:    tcg.client.HTTPClient.Transport,
		PublicId:  publicId,
		PrivateId: privateId,
	}
	tcg.MaxConcurrency = defaultConcurrency
	return &tcg
}

func (tcg *TCGPlayerMarket) processEntry(channel chan<- responseChan, req requestChan) error {
	resp, err := tcg.client.Get(tcgApiProductURL + fmt.Sprint(req.TCGProductId))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var response struct {
		Success bool     `json:"success"`
		Errors  []string `json:"errors"`
		Results []struct {
			LowPrice    float64 `json:"lowPrice"`
			MarketPrice float64 `json:"marketPrice"`
			MidPrice    float64 `json:"midPrice"`
			SubTypeName string  `json:"subTypeName"`
		} `json:"results"`
	}
	err = json.Unmarshal(data, &response)
	if err != nil {
		return err
	}
	if !response.Success {
		return errors.New(strings.Join(response.Errors, "|"))
	}

	for _, result := range response.Results {
		theCard := &mtgdb.Card{
			Id:   req.UUID,
			Foil: result.SubTypeName == "Foil",
		}
		cc, err := theCard.Match()
		if err != nil {
			return err
		}

		// This avoids duplicates for foil-only or nonfoil-only cards
		// in particular Tenth Edition and Unhinged
		if (cc.Foil && result.SubTypeName != "Foil") ||
			(!cc.Foil && result.SubTypeName != "Normal") {
			continue
		}

		prices := []float64{
			result.LowPrice, result.MarketPrice, result.MidPrice,
		}
		names := []string{
			"TCG Low", "TCG Market", "TCG Mid",
		}

		link := fmt.Sprintf("https://shop.tcgplayer.com/product/productsearch?id=%d", req.TCGProductId)
		if tcg.Affiliate != "" {
			link += fmt.Sprintf("&utm_campaign=affiliate&utm_medium=%s&utm_source=%s&partner=%s", tcg.Affiliate, tcg.Affiliate, tcg.Affiliate)
		}

		for i := range names {
			out := responseChan{
				card: *cc,
				entry: mtgban.InventoryEntry{
					Conditions: "NM",
					Price:      prices[i],
					Quantity:   1,
					URL:        link,
					SellerName: names[i],
				},
			}

			channel <- out
		}
	}

	return nil
}

func (tcg *TCGPlayerMarket) scrape() error {
	pages := make(chan requestChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := tcg.processEntry(channel, page)
				if err != nil {
					tcg.printf("%s", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, code := range mtgdb.AllSets() {
			set, _ := mtgdb.Set(code)
			tcg.printf("Scraping %s", set.Name)

			for _, card := range set.Cards {
				if card.TcgplayerProductId == 0 {
					continue
				}

				pages <- requestChan{
					TCGProductId: card.TcgplayerProductId,
					UUID:         card.UUID,
				}
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		err := tcg.inventory.Add(&result.card, &result.entry)
		if err != nil {
			tcg.printf(err.Error())
			continue
		}
	}

	tcg.InventoryDate = time.Now()

	return nil
}

func (tcg *TCGPlayerMarket) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	start := time.Now()
	tcg.printf("Inventory scraping started at %s", start)

	err := tcg.scrape()
	if err != nil {
		return nil, err
	}
	tcg.printf("Inventory scraping took %s", time.Since(start))

	return tcg.inventory, nil
}

func (tcg *TCGPlayerMarket) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) == 0 {
		_, err := tcg.Inventory()
		if err != nil {
			return nil, err
		}
	}

	inventory, found := tcg.marketplace[sellerName]
	if found {
		return inventory, nil
	}

	for card := range tcg.inventory {
		for i := range tcg.inventory[card] {
			if tcg.inventory[card][i].SellerName == sellerName {
				if tcg.inventory[card][i].Price == 0 {
					continue
				}
				if tcg.marketplace[sellerName] == nil {
					tcg.marketplace[sellerName] = mtgban.InventoryRecord{}
				}
				tcg.marketplace[sellerName][card] = append(tcg.marketplace[sellerName][card], tcg.inventory[card][i])
			}
		}
	}

	if len(tcg.marketplace[sellerName]) == 0 {
		return nil, fmt.Errorf("seller %s not found", sellerName)
	}
	return tcg.marketplace[sellerName], nil
}

func (tcg *TCGPlayerMarket) IntializeInventory(reader io.Reader) error {
	inventory, err := mtgban.LoadInventoryFromCSV(reader)
	if err != nil {
		return err
	}

	tcg.inventory = mtgban.InventoryRecord{}
	for card := range inventory {
		key, err := (&card).Match()
		if err != nil {
			tcg.printf("%s", err)
			continue
		}
		tcg.inventory[*key] = inventory[card]
	}
	if len(tcg.inventory) == 0 {
		return fmt.Errorf("nothing was loaded")
	}
	tcg.printf("Loaded inventory from file")

	return nil
}

func (tcg *TCGPlayerMarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Market"
	info.Shorthand = "TCGMkt"
	info.InventoryTimestamp = tcg.InventoryDate
	info.MetadataOnly = true
	return
}
