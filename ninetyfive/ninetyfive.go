package ninetyfive

import (
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

type Ninetyfive struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int

	client *NFClient

	exchangeRate float64

	inventoryDate time.Time
	inventory     mtgban.InventoryRecord
	buylistDate   time.Time
	buylist       mtgban.BuylistRecord
}

func NewScraper(altHost bool) (*Ninetyfive, error) {
	nf := Ninetyfive{}
	nf.inventory = mtgban.InventoryRecord{}
	nf.buylist = mtgban.BuylistRecord{}
	nf.client = NewNFClient(altHost)
	nf.MaxConcurrency = defaultConcurrency
	nf.exchangeRate = 1.0
	if altHost {
		rate, err := mtgban.GetExchangeRate("EUR")
		if err != nil {
			return nil, err
		}
		nf.exchangeRate = rate
	}
	return &nf, nil
}

type respChan struct {
	cardId   string
	invEntry *mtgban.InventoryEntry
	buyEntry *mtgban.BuylistEntry
}

func (nf *Ninetyfive) printf(format string, a ...interface{}) {
	if nf.LogCallback != nil {
		nf.LogCallback("[95] "+format, a...)
	}
}

func (nf *Ninetyfive) processPage(channel chan<- respChan, start int, mode string) error {
	var products []NFProduct
	var err error
	if mode == "retail" {
		products, err = nf.client.GetRetail(start)
	} else if mode == "buylist" {
		products, err = nf.client.GetBuylist(start)
	}
	if err != nil {
		return nil
	}

	for _, product := range products {
		if product.Quantity == 0 || product.Price <= 0 {
			continue
		}

		theCard, err := preprocess(&product)
		if err != nil {
			continue
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			// Both sides are present, ignore errors from doubles
			if product.Card.Layout != "normal" {
				continue
			}

			// No easy way to tell duplicates apart
			var alias *mtgmatcher.AliasingError
			if product.Card.Number == 0 && (errors.As(err, &alias) || theCard.Variation == "") {
				continue
			}

			// Ignore errors from known incorrect cards (wrong cn)
			if theCard.Edition == "Collectors' Edition" {
				continue
			}

			nf.printf("%v", err)
			nf.printf("%q", theCard)
			nf.printf("%q", product)
			if alias != nil {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					nf.printf("- %s", card)
				}
			}
			continue
		}

		if mode == "retail" {
			cond := product.Condition
			if cond == "DMG" {
				cond = "PO"
			}
			slug := product.Set.Slug
			if slug == "" {
				slug = product.Card.Set.Slug
			}

			price := float64(product.Price) / 100 * nf.exchangeRate
			link := "https://95mtg.com/singles/" + slug + "/" + product.Card.Slug
			channel <- respChan{
				cardId: cardId,
				invEntry: &mtgban.InventoryEntry{
					Conditions: cond,
					Price:      price,
					Quantity:   product.Quantity,
					URL:        link,
				},
			}
		} else if mode == "buylist" {
			link := "https://95mtg.com/buylist/search/?name=" + url.QueryEscape(product.Card.Name)
			var priceRatio, sellPrice float64

			invCards := nf.inventory[cardId]
			for _, invCard := range invCards {
				sellPrice = invCard.Price
				break
			}

			for _, cond := range product.Conditions {
				if cond == "DMG" {
					cond = "PO"
				}

				price := float64(product.Price) / 100 * nf.exchangeRate
				if cond != "NM" {
					price *= map[string]float64{
						"SP": 0.8,
						"MP": 0.65,
						"HP": 0.6,
						"PO": 0.55,
					}[cond]
				}

				if sellPrice > 0 {
					priceRatio = price / sellPrice * 100
				}

				channel <- respChan{
					cardId: cardId,
					buyEntry: &mtgban.BuylistEntry{
						Conditions: cond,
						BuyPrice:   price,
						PriceRatio: priceRatio,
						URL:        link,
					},
				}
			}
		}
	}

	return nil
}

func (nf *Ninetyfive) scrape(mode string) error {
	var totalProducts int
	var err error
	if mode == "retail" {
		totalProducts, err = nf.client.RetailTotals()
	} else if mode == "buylist" {
		totalProducts, err = nf.client.BuylistTotals()
	} else {
		err = errors.New("unknown mode")
	}
	if err != nil {
		return err
	}

	pages := make(chan int)
	channel := make(chan respChan)
	var wg sync.WaitGroup

	nf.printf("Parsing %d items", totalProducts)
	for i := 0; i < nf.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for start := range pages {
				err := nf.processPage(channel, start, mode)
				if err != nil {
					nf.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 1; i <= totalProducts/NFDefaultResultsPerPage+1; i++ {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for record := range channel {
		if record.invEntry != nil {
			err := nf.inventory.Add(record.cardId, record.invEntry)
			if err != nil {
				nf.printf("%v", err)
				continue
			}
		} else if record.buyEntry != nil {
			err := nf.buylist.Add(record.cardId, record.buyEntry)
			if err != nil {
				nf.printf("%v", err)
				continue
			}
		}
	}

	if mode == "retail" {
		nf.inventoryDate = time.Now()
	} else if mode == "buylist" {
		nf.buylistDate = time.Now()
	}

	return nil
}

func (nf *Ninetyfive) Inventory() (mtgban.InventoryRecord, error) {
	if len(nf.inventory) > 0 {
		return nf.inventory, nil
	}

	err := nf.scrape("retail")
	if err != nil {
		return nil, err
	}

	return nf.inventory, nil
}

func (nf *Ninetyfive) Buylist() (mtgban.BuylistRecord, error) {
	if len(nf.buylist) > 0 {
		return nf.buylist, nil
	}

	err := nf.scrape("buylist")
	if err != nil {
		return nil, err
	}

	return nf.buylist, nil
}

func (nf *Ninetyfive) Info() (info mtgban.ScraperInfo) {
	info.Name = "95mtg"
	info.Shorthand = "95"
	info.InventoryTimestamp = nf.inventoryDate
	info.BuylistTimestamp = nf.buylistDate
	info.MultiCondBuylist = true
	info.NoCredit = true
	return
}
