package cardkingdom

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
	defaultQueries     = 12
)

type CardkingdomHotBuylist struct {
	LogCallback  mtgban.LogCallbackFunc
	Concurrency  int
	TotalQueries int

	buylistDate time.Time
	buylist     mtgban.BuylistRecord
}

func NewHotScraper() *CardkingdomHotBuylist {
	ck := CardkingdomHotBuylist{}
	ck.buylist = mtgban.BuylistRecord{}
	ck.Concurrency = defaultConcurrency
	ck.TotalQueries = defaultQueries
	return &ck
}

func (ck *CardkingdomHotBuylist) printf(format string, a ...interface{}) {
	if ck.LogCallback != nil {
		ck.LogCallback("[CKHot] "+format, a...)
	}
}

func (ck *CardkingdomHotBuylist) processPage(channel chan<- respChan) error {
	ckClient := NewCKClient()

	pricelist, err := ckClient.GetHotBuylist()
	if err != nil {
		return err
	}

	for _, card := range pricelist {
		vars := mtgmatcher.SplitVariants(card.ShortName)
		variant := ""
		cardName := vars[0]
		if len(vars) > 1 {
			variant = vars[1]
		}
		edition := strings.TrimSuffix(card.Name, ": "+card.ShortName)

		theCard := &mtgmatcher.Card{
			Name:      cardName,
			Edition:   edition,
			Variation: variant,
			Foil:      strings.Contains(variant, "Foil"),
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			ck.printf("%v", err)
			ck.printf("%q", theCard)
			ck.printf("%q", card)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					ck.printf("- %s", card)
				}
			}
			continue
		}

		sellPrice, err := strconv.ParseFloat(card.HotPrice, 64)
		if err != nil {
			ck.printf("%v %v", err, card)
		}

		price, err := strconv.ParseFloat(card.BuyPrice, 64)
		if err != nil {
			ck.printf("%v %v", err, card)
		}

		gradeMap := grading(cardId, price, card.Edition == "Promotional")
		for _, grade := range mtgban.DefaultGradeTags {
			factor := gradeMap[grade]
			var priceRatio float64

			if sellPrice > 0 {
				priceRatio = price / sellPrice * 100
			}

			out := respChan{
				cardId: cardId,
				buyEntry: &mtgban.BuylistEntry{
					BuyPrice:   price * factor,
					TradePrice: price * factor * 1.3,
					Quantity:   card.BuyQuantity,
					PriceRatio: priceRatio,
				},
			}
			channel <- out
		}
	}

	return nil
}

type respChan struct {
	cardId   string
	buyEntry *mtgban.BuylistEntry
}

func (ck *CardkingdomHotBuylist) scrape() error {
	pages := make(chan int)
	channel := make(chan respChan)
	var wg sync.WaitGroup

	for i := 0; i < ck.Concurrency; i++ {
		wg.Add(1)
		go func() {
			for range pages {
				err := ck.processPage(channel)
				if err != nil {
					ck.printf("%s", err.Error())
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for i := 0; i < ck.TotalQueries; i++ {
			pages <- i
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for record := range channel {
		err := ck.buylist.AddRelaxed(record.cardId, record.buyEntry)
		if err != nil {
			ck.printf("%v", err)
			continue
		}
	}

	ck.buylistDate = time.Now()

	return nil
}

func (ck *CardkingdomHotBuylist) Buylist() (mtgban.BuylistRecord, error) {
	if len(ck.buylist) > 0 {
		return ck.buylist, nil
	}

	err := ck.scrape()
	if err != nil {
		return nil, err
	}

	return ck.buylist, nil
}

func (ck *CardkingdomHotBuylist) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Kingdom Hot Buylist"
	info.Shorthand = "CKHot"
	info.BuylistTimestamp = &ck.buylistDate
	return
}
