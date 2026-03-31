package mtgstocks

import (
	"context"
	"errors"
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

func (stks *MTGStocks) Load(ctx context.Context) error {
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

	var items []requestChan
	for _, interest := range averagesFoil {
		items = append(items, requestChan{name: "Average", interest: interest})
	}
	for _, interest := range averagesRegular {
		items = append(items, requestChan{name: "Average", interest: interest})
	}
	for _, interest := range marketsFoil {
		items = append(items, requestChan{name: "Market", interest: interest})
	}
	for _, interest := range marketsRegular {
		items = append(items, requestChan{name: "Market", interest: interest})
	}

	mtgban.WorkerPool(ctx, stks.MaxConcurrency, items,
		func(_ context.Context, page requestChan, channel chan<- responseChan) error {
			return stks.processEntry(channel, page)
		},
		func(result responseChan) {
			err := stks.inventory.Add(result.cardId, &result.entry)
			if err != nil {
				stks.printf("%s", err.Error())
			}
		},
		stks.printf,
	)

	stks.inventoryDate = time.Now()

	return nil
}

func (stks *MTGStocks) Inventory() mtgban.InventoryRecord {
	return stks.inventory
}

func (stks *MTGStocks) Info() (info mtgban.ScraperInfo) {
	info.Name = "MTGStocks"
	info.Shorthand = "STKS"
	info.InventoryTimestamp = &stks.inventoryDate
	info.MetadataOnly = true
	return
}
