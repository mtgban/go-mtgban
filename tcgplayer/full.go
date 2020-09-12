package tcgplayer

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	http "github.com/hashicorp/go-retryablehttp"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type TCGPlayerFull struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	MaxConcurrency int

	inventory   mtgban.InventoryRecord
	marketplace map[string]mtgban.InventoryRecord

	client *http.Client
}

func NewScraperFull() *TCGPlayerFull {
	tcg := TCGPlayerFull{}
	tcg.inventory = mtgban.InventoryRecord{}
	tcg.marketplace = map[string]mtgban.InventoryRecord{}
	tcg.client = http.NewClient()
	tcg.client.Logger = nil
	tcg.MaxConcurrency = defaultConcurrency
	return &tcg
}

func (tcg *TCGPlayerFull) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGF] "+format, a...)
	}
}

func (tcg *TCGPlayerFull) getPagesForProduct(productId string) (int, error) {
	num, err := getListingsNumber(tcg.client.StandardClient(), productId)
	if err != nil {
		return 0, err
	}
	return num/pagesPerRequest + 1, nil
}

func (tcg *TCGPlayerFull) processEntry(channel chan<- responseChan, req requestChan) error {
	theCard := &mtgmatcher.Card{
		Id: req.UUID,
	}
	cardId, err := mtgmatcher.Match(theCard)
	if err != nil {
		return err
	}
	var cardIdFoil string

	totalPages, err := tcg.getPagesForProduct(req.TCGProductId)
	if err != nil {
		return err
	}

	for i := 1; i <= totalPages; i++ {
		u, _ := url.Parse(tcgBaseURL)
		q := u.Query()
		q.Set("productId", req.TCGProductId)
		q.Set("pageSize", fmt.Sprintf("%d", pagesPerRequest))
		q.Set("page", fmt.Sprintf("%d", i))
		u.RawQuery = q.Encode()

		resp, err := tcg.client.Get(u.String())
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		doc, err := goquery.NewDocumentFromReader(resp.Body)
		if err != nil {
			return err
		}

		doc.Find(".product-listing").Each(func(i int, s *goquery.Selection) {
			cond := s.Find("a[class='condition']").Text()
			isFoil := false
			if strings.Contains(cond, " Foil") {
				isFoil = true
				cond = strings.Replace(cond, " Foil", "", 1)
				if cardIdFoil == "" {
					theCard.Foil = true
					cardIdFoil, _ = mtgmatcher.Match(theCard)
				}
			}

			co, _ := mtgmatcher.GetUUID(cardId)

			// Since we use the ID match, we can be sure that the foiling info
			// is appropriate, so we can skip anything that is not foil if our
			// card is foil. This is especially important for Tenth Edition and
			// Unhinged foils which mtgjson treats differently.
			if co.Foil && !isFoil {
				return
			}

			langs := strings.Split(cond, " - ")
			cond = langs[0]
			lang := ""
			if len(langs) > 1 {
				lang = langs[1]
			}
			switch lang {
			case "":
			case "Japanese":
				if co.Card.HasUniqueLanguage("Japanese") {
					return
				}
			default:
				return
			}

			switch cond {
			case "Near Mint":
				cond = "NM"
			case "Lightly Played":
				cond = "SP"
			case "Moderately Played":
				cond = "MP"
			case "Heavily Played":
				cond = "HP"
			case "Damaged":
				cond = "PO"
			default:
				tcg.printf("Unknown '%s' condition", cond)
				return
			}

			priceStr := s.Find("span[class='product-listing__price']").Text()
			priceStr = strings.Replace(priceStr, "$", "", 1)
			priceStr = strings.Replace(priceStr, ",", "", 1)
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				tcg.printf("%s - %v", co, err)
				return
			}

			qtyStr, _ := s.Find("input[name='quantityAvailable']").Attr("value")
			qty, err := strconv.Atoi(qtyStr)
			if err != nil {
				tcg.printf("%s - %v", co, err)
				return
			}

			sellerName := strings.TrimSpace(s.Find("a[class='seller__name']").Text())

			out := responseChan{
				cardId: cardId,
				entry: mtgban.InventoryEntry{
					Conditions: cond,
					Price:      price,
					Quantity:   qty,
					SellerName: sellerName,
					URL:        "https://shop.tcgplayer.com/product/productsearch?id=" + req.TCGProductId,
				},
			}
			if isFoil {
				out.cardId = cardIdFoil
			}

			channel <- out
		})
	}

	return nil
}

func (tcg *TCGPlayerFull) scrape() error {
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
		sets := mtgmatcher.GetSets()
		i := 1
		for _, set := range sets {
			tcg.printf("Scraping %s (%d/%d)", set.Name, i, len(sets))
			i++

			for _, card := range set.Cards {
				tcgId, found := card.Identifiers["tcgplayerProductId"]
				if !found {
					continue
				}

				pages <- requestChan{
					TCGProductId: tcgId,
					UUID:         card.UUID,
				}
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		err := tcg.inventory.Add(result.cardId, &result.entry)
		if err != nil {
			tcg.printf(err.Error())
			continue
		}
	}

	tcg.inventoryDate = time.Now()

	return nil
}

func (tcg *TCGPlayerFull) Inventory() (mtgban.InventoryRecord, error) {
	if len(tcg.inventory) > 0 {
		return tcg.inventory, nil
	}

	err := tcg.scrape()
	if err != nil {
		return nil, err
	}

	return tcg.inventory, nil
}

func (tcg *TCGPlayerFull) InventoryForSeller(sellerName string) (mtgban.InventoryRecord, error) {
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

func (tcg *TCGPlayerFull) IntializeInventory(reader io.Reader) error {
	inventory, err := mtgban.LoadInventoryFromCSV(reader)
	if err != nil {
		return err
	}

	tcg.inventory = mtgban.InventoryRecord{}
	for card := range inventory {
		tcg.inventory[card] = inventory[card]

		for i := range tcg.inventory[card] {
			sellerName := tcg.inventory[card][i].SellerName
			if tcg.marketplace[sellerName] == nil {
				tcg.marketplace[sellerName] = mtgban.InventoryRecord{}
			}
			tcg.marketplace[sellerName][card] = append(tcg.marketplace[sellerName][card], tcg.inventory[card][i])
		}
	}
	if len(tcg.inventory) == 0 {
		return fmt.Errorf("nothing was loaded")
	}
	tcg.printf("Loaded inventory from file")

	return nil
}

func (tcg *TCGPlayerFull) Info() (info mtgban.ScraperInfo) {
	info.Name = "TCG Player Full"
	info.Shorthand = "TCGF"
	info.InventoryTimestamp = tcg.inventoryDate
	return
}
