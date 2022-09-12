package coolstuffinc

import (
	"errors"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type CoolstuffincOfficial struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client *CSIClient
}

func NewScraperOfficial(key string) *CoolstuffincOfficial {
	csi := CoolstuffincOfficial{}
	csi.inventory = mtgban.InventoryRecord{}
	csi.buylist = mtgban.BuylistRecord{}
	csi.client = NewCSIClient(key)
	return &csi
}

func (csi *CoolstuffincOfficial) printf(format string, a ...interface{}) {
	if csi.LogCallback != nil {
		csi.LogCallback("[CSI] "+format, a...)
	}
}

func (csi *CoolstuffincOfficial) scrape() error {
	pricelist, err := csi.client.GetPriceList()
	if err != nil {
		return err
	}

	for _, card := range pricelist {
		theCard, err := Preprocess(card)
		if err != nil {
			continue
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			switch theCard.Edition {
			case "World Championship Decks":
				continue
			case "Homelands", "Fallen Empires", "Alliances", "Deckmasters":
				continue
			case "Ravnica Allegiance", "Guilds of Ravnica", "Unstable", "Commander Anthology Volume II":
				continue
			case "Mystical Archive", "Strixhaven: School of Mages", "Strixhaven: School of Mages - Variants":
				continue
			default:
				if theCard.IsBasicLand() {
					continue
				}
			}
			csi.printf("%v", err)
			csi.printf("%q", theCard)
			csi.printf("%q", card)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					csi.printf("- %s", card)
				}
			}
			continue
		}

		link := card.URL
		if csi.Partner != "" {
			link += "?utm_referrer=" + csi.Partner + "&utm_source=" + csi.Partner
		}
		if card.QuantityRetail > 0 && card.PriceRetail > 0 {
			out := &mtgban.InventoryEntry{
				Conditions: "NM",
				Price:      card.PriceRetail,
				Quantity:   card.QuantityRetail,
				URL:        link,
			}
			err = csi.inventory.Add(cardId, out)
			if err != nil {
				csi.printf("%v", err)
				csi.printf("%q", theCard)
				csi.printf("%q", card)
			}
		}

		if card.PriceBuy > 0 {
			var priceRatio float64

			if card.PriceRetail > 0 {
				priceRatio = card.PriceBuy / card.PriceRetail * 100
			}

			out := &mtgban.BuylistEntry{
				BuyPrice:   card.PriceBuy,
				TradePrice: card.PriceBuy * 1.3,
				PriceRatio: priceRatio,
				URL:        "https://www.coolstuffinc.com/main_buylist_display.php",
			}
			err = csi.buylist.Add(cardId, out)
			if err != nil {
				csi.printf("%v", err)
				csi.printf("%q", theCard)
				csi.printf("%q", card)
			}
		}
	}

	csi.inventoryDate = time.Now()
	csi.buylistDate = time.Now()

	return nil
}

func (csi *CoolstuffincOfficial) Inventory() (mtgban.InventoryRecord, error) {
	if len(csi.inventory) > 0 {
		return csi.inventory, nil
	}

	err := csi.scrape()
	if err != nil {
		return nil, err
	}

	return csi.inventory, nil

}

func (csi *CoolstuffincOfficial) Buylist() (mtgban.BuylistRecord, error) {
	if len(csi.buylist) > 0 {
		return csi.buylist, nil
	}

	err := csi.scrape()
	if err != nil {
		return nil, err
	}

	return csi.buylist, nil
}

func (csi *CoolstuffincOfficial) Info() (info mtgban.ScraperInfo) {
	info.Name = "Coolstuffinc"
	info.Shorthand = "CSI"
	info.InventoryTimestamp = &csi.inventoryDate
	info.BuylistTimestamp = &csi.buylistDate
	info.Grading = mtgban.DefaultGrading
	return
}
