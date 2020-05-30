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
	"github.com/kodabb/go-mtgban/mtgdb"
)

type TCGPlayerFull struct {
	LogCallback   mtgban.LogCallbackFunc
	InventoryDate time.Time

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
	return &tcg
}

func (tcg *TCGPlayerFull) printf(format string, a ...interface{}) {
	if tcg.LogCallback != nil {
		tcg.LogCallback("[TCGF] "+format, a...)
	}
}

func (tcg *TCGPlayerFull) getPagesForProduct(productId int) (int, error) {
	num, err := getListingsNumber(tcg.client.StandardClient(), productId)
	if err != nil {
		return 0, err
	}
	return num/pagesPerRequest + 1, nil
}

func (tcg *TCGPlayerFull) processEntry(channel chan<- responseChan, req requestChan) error {
	theCard := &mtgdb.Card{
		Id: req.UUID,
	}
	cc, err := theCard.Match()
	if err != nil {
		return err
	}
	var ccfoil *mtgdb.Card

	totalPages, err := tcg.getPagesForProduct(req.TCGProductId)
	if err != nil {
		return err
	}

	for i := 1; i <= totalPages; i++ {
		u, _ := url.Parse(tcgBaseURL)
		q := u.Query()
		q.Set("productId", fmt.Sprintf("%d", req.TCGProductId))
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
				if ccfoil == nil {
					theCard.Foil = true
					ccfoil, _ = theCard.Match()
				}
			}

			// Since we use the ID match, we can be sure that the foiling info
			// is appropriate, so we can skip anything that is not foil if our
			// card is foil. This is especially important for Tenth Edition and
			// Unhinged foils which mtgjson treats differently.
			if cc.Foil && !isFoil {
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
				if !strings.Contains(cc.Variation, "Japanese") {
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
				tcg.printf("%s - %v", cc, err)
				return
			}

			qtyStr, _ := s.Find("input[name='quantityAvailable']").Attr("value")
			qty, err := strconv.Atoi(qtyStr)
			if err != nil {
				tcg.printf("%s - %v", cc, err)
				return
			}

			sellerName := strings.TrimSpace(s.Find("a[class='seller__name']").Text())

			out := responseChan{
				card: *cc,
				entry: mtgban.InventoryEntry{
					Conditions: cond,
					Price:      price,
					Quantity:   qty,
					SellerName: sellerName,
					URL:        fmt.Sprintf("https://shop.tcgplayer.com/product/productsearch?id=%d", req.TCGProductId),
				},
			}
			if isFoil {
				out.card = *ccfoil
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

	for i := 0; i < maxConcurrency; i++ {
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

func (tcg *TCGPlayerFull) Inventory() (mtgban.InventoryRecord, error) {
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
	info.InventoryTimestamp = tcg.InventoryDate
	return
}
