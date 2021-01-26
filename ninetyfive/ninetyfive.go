package ninetyfive

import (
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

	buylistDate time.Time
	buylist     mtgban.BuylistRecord
}

func NewScraper() *Ninetyfive {
	nf := Ninetyfive{}
	nf.buylist = mtgban.BuylistRecord{}
	nf.client = NewNFClient()
	nf.MaxConcurrency = defaultConcurrency
	return &nf
}

type respChan struct {
	cardId   string
	buyEntry *mtgban.BuylistEntry
}

func (nf *Ninetyfive) printf(format string, a ...interface{}) {
	if nf.LogCallback != nil {
		nf.LogCallback("[95] "+format, a...)
	}
}

func (nf *Ninetyfive) processPage(channel chan<- respChan, start int) error {
	products, err := nf.client.GetBuylist(start)
	if err != nil {
		return nil
	}

	for _, product := range products {
		if product.Quantity == 0 || product.Price <= 0 {
			continue
		}

		theCard, err := preprocess(product.Card, product.Foil == 1)
		if err != nil {
			continue
		}

		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			// Both sides are present, ignore errors from doubles
			if product.Card.Layout != "normal" {
				continue
			}

			// No easy way to tell duplicates apart
			alias, ok := err.(*mtgmatcher.AliasingError)
			if product.Card.Number == 0 && (ok || theCard.Variation == "") {
				continue
			}

			nf.printf("%v", err)
			nf.printf("%q", theCard)
			nf.printf("%q", product)
			if ok {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					nf.printf("- %s", card)
				}
			}
			continue
		}

		link := "https://95mtg.com/buylist/search/?name=" + url.QueryEscape(product.Card.Name)
		for _, cond := range product.Conditions {
			if cond == "DMG" {
				cond = "PO"
			}

			price := float64(product.Price) / 100
			if cond != "NM" {
				price *= map[string]float64{
					"SP": 0.8,
					"MP": 0.65,
					"HP": 0.6,
					"PO": 0.55,
				}[cond]
			}

			channel <- respChan{
				cardId: cardId,
				buyEntry: &mtgban.BuylistEntry{
					Conditions: cond,
					BuyPrice:   price,
					Quantity:   product.Quantity,
					URL:        link,
				},
			}
		}
	}

	return nil
}

func (nf *Ninetyfive) parseBL() error {
	totalProducts, err := nf.client.BuylistTotals()
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
				err := nf.processPage(channel, start)
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
		err := nf.buylist.AddRelaxed(record.cardId, record.buyEntry)
		if err != nil {
			co, _ := mtgmatcher.GetUUID(record.cardId)
			// Try adjusting for the lack of variants
			if co.Edition == "Arabian Nights" {
				secondId, err := mtgmatcher.Match(&mtgmatcher.Card{
					Name:      co.Name,
					Edition:   co.Edition,
					Variation: "light",
				})
				if err == nil {
					err = nf.buylist.AddRelaxed(secondId, record.buyEntry)
					if err == nil {
						continue
					}
				}
			}
			nf.printf("%v", err)
			continue
		}
	}

	nf.buylistDate = time.Now()

	return nil
}

func (nf *Ninetyfive) Buylist() (mtgban.BuylistRecord, error) {
	if len(nf.buylist) > 0 {
		return nf.buylist, nil
	}

	err := nf.parseBL()
	if err != nil {
		return nil, err
	}

	return nf.buylist, nil
}

func (nf *Ninetyfive) Info() (info mtgban.ScraperInfo) {
	info.Name = "95mtg"
	info.Shorthand = "95"
	info.BuylistTimestamp = nf.buylistDate
	info.MultiCondBuylist = true
	info.NoCredit = true
	return
}
