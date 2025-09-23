package mtgstocks

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

const (
	defaultConcurrency = 8
)

type MTGStocks struct {
	LogCallback    mtgban.LogCallbackFunc
	inventoryDate  time.Time
	MaxConcurrency int

	client    *STKSClient
	inventory mtgban.InventoryRecord
}

type requestChan struct {
	name     string
	interest StocksInterest
}

type responseChan struct {
	cardId string
	entry  mtgban.InventoryEntry
}

func (stks *MTGStocks) printf(format string, a ...interface{}) {
	if stks.LogCallback != nil {
		stks.LogCallback("[STKS] "+format, a...)
	}
}

func NewScraper() *MTGStocks {
	stks := MTGStocks{}
	stks.client = NewClient()
	stks.inventory = mtgban.InventoryRecord{}
	stks.MaxConcurrency = defaultConcurrency
	return &stks
}

func (stks *MTGStocks) processEntry(channel chan<- responseChan, req requestChan) error {
	if req.interest.Percentage < 0 {
		return nil
	}

	theCard, err := preprocess(req.interest.Print.Name, req.interest.Print.SetName, req.interest.Foil)
	if err != nil {
		return nil
	}

	cardId, err := mtgmatcher.Match(theCard)
	if errors.Is(err, mtgmatcher.ErrUnsupported) {
		return nil
	} else if err != nil {
		switch theCard.Edition {
		case "Alliances",
			"Fallen Empires",
			"Homelands",
			"World Championship Decks":
			return nil
		default:
			if mtgmatcher.IsBasicLand(theCard.Name) {
				return nil
			}
		}

		stks.printf("%q", theCard)
		stks.printf("%s | %s | %v", req.interest.Print.Name, req.interest.Print.SetName, req.interest.Foil)

		var alias *mtgmatcher.AliasingError
		if errors.As(err, &alias) {
			probes := alias.Probe()
			for _, probe := range probes {
				card, _ := mtgmatcher.GetUUID(probe)
				stks.printf("- %s", card)
			}
		}
		return err
	}

	link, err := getLink(req.interest.Print.Slug)
	if err != nil {
		stks.printf("invalid data type used for %s", req.interest.Print.Name)
	}
	out := responseChan{
		cardId: cardId,
		entry: mtgban.InventoryEntry{
			Price:      req.interest.PresentPrice,
			Quantity:   1,
			URL:        link,
			SellerName: req.name + " " + mtgmatcher.Title(req.interest.InterestType),
		},
	}

	channel <- out

	return nil
}

func (stks *MTGStocks) scrape(ctx context.Context) error {
	averagesRegular, err := stks.client.AverageInterests(ctx, false)
	if err != nil {
		stks.printf("averages regular " + err.Error())
	}
	averagesFoil, err := stks.client.AverageInterests(ctx, true)
	if err != nil {
		stks.printf("averages foil " + err.Error())
	}
	marketsRegular, err := stks.client.MarketInterests(ctx, false)
	if err != nil {
		stks.printf("market regular " + err.Error())
	}
	marketsFoil, err := stks.client.MarketInterests(ctx, true)
	if err != nil {
		stks.printf("market foil" + err.Error())
	}
	if len(averagesRegular) == 0 && len(averagesFoil) == 0 &&
		len(marketsRegular) == 0 && len(marketsFoil) == 0 {
		return errors.New("nothing was loaded from mtgstocks")
	}

	pages := make(chan requestChan)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < stks.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for page := range pages {
				err := stks.processEntry(channel, page)
				if err != nil {
					stks.printf("%s", err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		for _, interest := range averagesFoil {
			pages <- requestChan{
				name:     "Average",
				interest: interest,
			}
		}
		for _, interest := range averagesRegular {
			pages <- requestChan{
				name:     "Average",
				interest: interest,
			}
		}
		for _, interest := range marketsFoil {
			pages <- requestChan{
				name:     "Market",
				interest: interest,
			}
		}
		for _, interest := range marketsRegular {
			pages <- requestChan{
				name:     "Market",
				interest: interest,
			}
		}
		close(pages)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		err := stks.inventory.Add(result.cardId, &result.entry)
		if err != nil {
			stks.printf("%s", err.Error())
			continue
		}
	}

	stks.inventoryDate = time.Now()

	return nil
}

func (stks *MTGStocks) Inventory() (mtgban.InventoryRecord, error) {
	if len(stks.inventory) > 0 {
		return stks.inventory, nil
	}

	err := stks.scrape(context.TODO())
	if err != nil {
		return nil, err
	}

	return stks.inventory, nil
}

func (stks *MTGStocks) Info() (info mtgban.ScraperInfo) {
	info.Name = "MTGStocks"
	info.Shorthand = "STKS"
	info.InventoryTimestamp = &stks.inventoryDate
	info.MetadataOnly = true
	return
}
