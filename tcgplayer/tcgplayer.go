package tcgplayer

import (
	"compress/bzip2"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

type TCGPlayerMarket struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	buylistDate    time.Time
	Affiliate      string
	MaxConcurrency int

	inventory   mtgban.InventoryRecord
	buylist     mtgban.BuylistRecord
	marketplace map[string]mtgban.InventoryRecord

	client *TCGClient
}

type marketChan struct {
	UUID      string
	Condition string
	Printing  string
	Finish    string
	ProductId int
	SkuId     int
}

type responseChan struct {
	cardId string
	entry  mtgban.InventoryEntry
	bl     mtgban.BuylistEntry
}

const (
	allSkusURL       = "https://mtgjson.com/api/v5/TcgplayerSkus.json.bz2"
	allSkusBackupURL = "https://mtgjson.com/api/v5_backup/TcgplayerSkus.json.bz2"
)

var skuConditions = map[string]string{
	"NEAR MINT":         "NM",
	"LIGHTLY PLAYED":    "SP",
	"MODERATELY PLAYED": "MP",
	"HEAVILY PLAYED":    "HP",
	"DAMAGED":           "PO",
}

func (tcg *TCGPlayerMarket) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGMkt] "+format, a...)
	}
}

func NewScraperMarket(publicId, privateId string) *TCGPlayerMarket {
	tcg := TCGPlayerMarket{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.buylist = mtgban.BuylistRecord{}
	tcg.marketplace = map[string]mtgban.InventoryRecord{}
	tcg.client = NewTCGClient(publicId, privateId)
	tcg.MaxConcurrency = defaultConcurrency
	return &tcg
}

func (tcg *TCGPlayerMarket) processEntry(channel chan<- responseChan, reqs []marketChan, mode string) error {
	ids := make([]string, len(reqs))
	for i := range reqs {
		ids[i] = fmt.Sprint(reqs[i].SkuId)
	}

	var results []TCGSKUPrice
	var err error

	// Retrieve a list of skus with their prices
	if mode == "inventory" {
		results, err = tcg.client.TCGPricesForSKUs(ids)
	} else if mode == "buylist" {
		results, err = tcg.client.TCGBuylistPricesForSKUs(ids)
	}
	if err != nil {
		return err
	}

	for _, result := range results {
		var req marketChan
		for _, req = range reqs {
			if result.SkuId == req.SkuId {
				break
			}
		}

		theCard := &mtgmatcher.Card{
			Id:   req.UUID,
			Foil: req.Printing == "FOIL",
		}
		if req.Finish == "FOIL ETCHED" {
			theCard.Variation = "Etched"
		}
		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			tcg.printf("%s - (tcgId:%d / uuid:%s)", err.Error(), result.ProductId, req.UUID)
			continue
		}

		// Skip impossible entries, such as listing mistakes that list a foil
		// price for a foil-only card
		co, _ := mtgmatcher.GetUUID(cardId)
		if !co.Etched &&
			((co.Foil && req.Printing != "FOIL") ||
				(!co.Foil && req.Printing != "NON FOIL")) {
			continue
		}

		cond, found := skuConditions[req.Condition]
		if !found {
			tcg.printf("unknown condition %d for %d", req.Condition, req.SkuId)
			continue
		}

		if mode == "inventory" {
			prices := []float64{
				result.LowestListingPrice, result.DirectLowPrice,
			}
			names := []string{
				"TCG Player", "TCG Direct",
			}

			link := "https://www.tcgplayer.com/product/" + fmt.Sprint(req.ProductId)
			if tcg.Affiliate != "" {
				link += fmt.Sprintf("&utm_campaign=affiliate&utm_medium=%s&utm_source=%s&partner=%s", tcg.Affiliate, tcg.Affiliate, tcg.Affiliate)
			}
			if req.Printing == "FOIL" {
				link += "&Printing=Foil"
			} else {
				link += "&Printing=Normal"
			}

			for i := range names {
				if prices[i] == 0 {
					continue
				}
				out := responseChan{
					cardId: cardId,
					entry: mtgban.InventoryEntry{
						Conditions: cond,
						Price:      prices[i],
						Quantity:   1,
						URL:        link,
						SellerName: names[i],
						Bundle:     i == 1,
					},
				}

				channel <- out
			}
		} else if mode == "buylist" {
			price := result.BuylistPrices.High
			if price == 0 {
				continue
			}

			var sellPrice, priceRatio float64
			var backupPrice float64

			// Find the NM Market price of the same card id, if missing for
			// whatever reason use the tcg direct one
			invCards := tcg.inventory[cardId]
			for _, invCard := range invCards {
				if invCard.Conditions != "NM" {
					continue
				}
				if invCard.SellerName == "TCG Player" {
					backupPrice = invCard.Price
				}
				if invCard.SellerName == "TCG Direct" {
					sellPrice = invCard.Price
					break
				}
			}
			if sellPrice == 0 {
				sellPrice = backupPrice
			}

			if sellPrice > 0 {
				priceRatio = price / sellPrice * 100
			}
			out := responseChan{
				cardId: cardId,
				bl: mtgban.BuylistEntry{
					Conditions: cond,
					BuyPrice:   price,
					TradePrice: price,
					Quantity:   0,
					PriceRatio: priceRatio,
					URL:        "https://store.tcgplayer.com/buylist",
				},
			}

			channel <- out

		}
	}

	return nil
}

func (tcg *TCGPlayerMarket) scrape(mode string) error {
	tcg.printf("Retrieving skus")
	skusMap, err := getAllSKUs()
	if err != nil {
		return err
	}
	tcg.printf("Found skus for %d entries", len(skusMap))

	pages := make(chan marketChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < tcg.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			idFound := map[int]string{}
			buffer := make([]marketChan, 0, maxIdsInRequest)

			for page := range pages {
				// Skip dupes
				_, found := idFound[page.SkuId]
				if found {
					continue
				}
				idFound[page.SkuId] = ""

				// Add our data to the buffer
				buffer = append(buffer, page)

				// When buffer is full, process its contents and empty it
				if len(buffer) == maxIdsInRequest {
					err := tcg.processEntry(channel, buffer, mode)
					if err != nil {
						tcg.printf("%s", err.Error())
					}
					buffer = buffer[:0]
				}
			}
			// Process any spillover
			if len(buffer) != 0 {
				err := tcg.processEntry(channel, buffer, mode)
				if err != nil {
					tcg.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		sets := mtgmatcher.GetSets()
		i := 1
		for _, set := range sets {
			tcg.printf("Scraping %s (%d/%d)", set.Name, i, len(sets))
			i++

			for _, card := range set.Cards {
				uuid := card.UUID
				skus, found := skusMap[uuid]
				if !found {
					continue
				}
				for _, sku := range skus {
					// Skip languages that we do not track
					switch sku.Language {
					case "ENGLISH":
					case "ITALIAN":
						switch set.Name {
						case "Legends", "The Dark":
							uuid = card.UUID + "_ita"
						case "Rinascimento":
						default:
							continue
						}
					case "JAPANESE":
						switch set.Name {
						case "Chronicles":
							uuid = card.UUID + "_jpn"
						case "Fourth Edition Foreign Black Border":
						case "War of the Spark", "War of the Spark Promos":
							if !strings.Contains(card.Number, "★") {
								continue
							}
						case "Ikoria: Lair of Behemoths":
							switch card.Name {
							case "Mysterious Egg", "Dirge Bat", "Crystalline Giant":
							default:
								continue
							}
						case "Secrat Lair Drop":
							// No specific card because it's an "always open" set
						case "Kaldheim Promos":
							if card.Name != "Fiendish Duo" {
								continue
							}
						default:
							continue
						}
					default:
						continue
					}

					pages <- marketChan{
						UUID:      uuid,
						Condition: sku.Condition,
						Printing:  sku.Printing,
						Finish:    sku.Finish,
						ProductId: sku.ProductId,
						SkuId:     sku.SkuId,
					}
				}
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	if mode == "inventory" {
		for result := range channel {
			err := tcg.inventory.Add(result.cardId, &result.entry)
			if err != nil {
				tcg.printf("%s", err.Error())
				continue
			}
		}
		tcg.inventoryDate = time.Now()
	} else if mode == "buylist" {
		for result := range channel {
			err := tcg.buylist.Add(result.cardId, &result.bl)
			if err != nil {
				tcg.printf("%s", err.Error())
				continue
			}
		}
		tcg.buylistDate = time.Now()
	}

	return nil
}

func (tcg *TCGPlayerMarket) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	err := tcg.scrape("inventory")
	if err != nil {
		return nil, err
	}

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

func (tcg *TCGPlayerMarket) InitializeInventory(reader io.Reader) error {
	market, inventory, err := mtgban.LoadMarketFromCSV(reader)
	if err != nil {
		return err
	}
	if len(inventory) == 0 {
		return fmt.Errorf("nothing was loaded")
	}

	tcg.marketplace = market
	tcg.inventory = inventory

	tcg.printf("Loaded inventory from file")

	return nil
}

func (tcg *TCGPlayerMarket) Buylist() (mtgban.BuylistRecord, error) {
	if len(tcg.buylist) > 0 {
		return tcg.buylist, nil
	}

	err := tcg.scrape("buylist")
	if err != nil {
		return nil, err
	}

	return tcg.buylist, nil
}

func (tcg *TCGPlayerMarket) InitializeBuylist(reader io.Reader) error {
	buylist, err := mtgban.LoadBuylistFromCSV(reader)
	if err != nil {
		return err
	}
	if len(buylist) == 0 {
		return errors.New("empty buylist")
	}

	tcg.buylist = buylist

	tcg.printf("Loaded buylist from file")

	return nil
}

func (tcg *TCGPlayerMarket) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Market"
	info.Shorthand = "TCGMkt"
	info.InventoryTimestamp = tcg.inventoryDate
	info.BuylistTimestamp = tcg.buylistDate
	info.MultiCondBuylist = true
	info.NoQuantityInventory = true
	info.NoCredit = true
	return
}

func getAllSKUs() (map[string][]mtgjson.TCGSku, error) {
	resp, err := cleanhttp.DefaultClient().Get(allSkusURL)
	if err != nil {
		resp, err = cleanhttp.DefaultClient().Get(allSkusBackupURL)
		if err != nil {
			return nil, err
		}
	}
	defer resp.Body.Close()

	skus, err := mtgjson.LoadAllTCGSkus(bzip2.NewReader(resp.Body))
	if err != nil {
		return nil, err
	}
	return skus.Data, nil
}
