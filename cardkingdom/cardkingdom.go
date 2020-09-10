package cardkingdom

import (
	"net/url"
	"strconv"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgdb"
)

type Cardkingdom struct {
	LogCallback   mtgban.LogCallbackFunc
	Partner       string
	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraper() *Cardkingdom {
	ck := Cardkingdom{}
	ck.inventory = mtgban.InventoryRecord{}
	ck.buylist = mtgban.BuylistRecord{}
	return &ck
}

func (ck *Cardkingdom) printf(format string, a ...interface{}) {
	if ck.LogCallback != nil {
		ck.LogCallback("[CK] "+format, a...)
	}
}

func (ck *Cardkingdom) scrape() error {
	ckClient := NewCKClient()
	pricelist, err := ckClient.GetPriceList()
	if err != nil {
		return err
	}

	for _, card := range pricelist.Data {
		theCard, err := preprocess(card)
		if err != nil {
			continue
		}

		cc, err := theCard.Match()
		if err != nil {
			ck.printf("%q", theCard)
			ck.printf("%q", card)
			ck.printf("%v", err)
			continue
		}

		var sellPrice float64
		u, _ := url.Parse("https://www.cardkingdom.com/")

		sellPrice, err = strconv.ParseFloat(card.SellPrice, 64)
		if err != nil {
			ck.printf("%v", err)
		}
		if card.SellQuantity > 0 && sellPrice > 0 {
			u.Path = card.URL
			if ck.Partner != "" {
				q := u.Query()
				q.Set("partner", ck.Partner)
				q.Set("utm_source", ck.Partner)
				q.Set("utm_medium", "affiliate")
				q.Set("utm_campaign", ck.Partner)
				u.RawQuery = q.Encode()
			}

			out := &mtgban.InventoryEntry{
				Conditions: "NM",
				Price:      sellPrice,
				Quantity:   card.SellQuantity,
				URL:        u.String(),
			}
			err = ck.inventory.Add(cc.Id, out)
			if err != nil {
				ck.printf("%v", err)
			}
		}

		u, _ = url.Parse("https://www.cardkingdom.com/purchasing/mtg_singles")
		if card.BuyQuantity > 0 {
			price, err := strconv.ParseFloat(card.BuyPrice, 64)
			if err != nil {
				ck.printf("%v", err)
			}
			if price > 0 {
				var priceRatio float64

				if sellPrice > 0 {
					priceRatio = price / sellPrice * 100
				}

				q := u.Query()
				q.Set("filter[search]", "mtg_advanced")
				q.Set("filter[name]", card.Name)
				if ck.Partner != "" {
					q.Set("partner", ck.Partner)
					q.Set("utm_source", ck.Partner)
					q.Set("utm_medium", "affiliate")
					q.Set("utm_campaign", ck.Partner)
				}
				u.RawQuery = q.Encode()

				out := &mtgban.BuylistEntry{
					BuyPrice:   price,
					TradePrice: price * 1.3,
					Quantity:   card.BuyQuantity,
					PriceRatio: priceRatio,
					URL:        u.String(),
				}
				err = ck.buylist.Add(cc.Id, out)
				if err != nil {
					ck.printf("%v", err)
				}
			}
		}
	}

	ck.inventoryDate = time.Now()
	ck.buylistDate = time.Now()

	return nil
}

func (ck *Cardkingdom) Inventory() (mtgban.InventoryRecord, error) {
	if len(ck.inventory) > 0 {
		return ck.inventory, nil
	}

	err := ck.scrape()
	if err != nil {
		return nil, err
	}

	return ck.inventory, nil

}

func (ck *Cardkingdom) Buylist() (mtgban.BuylistRecord, error) {
	if len(ck.buylist) > 0 {
		return ck.buylist, nil
	}

	err := ck.scrape()
	if err != nil {
		return nil, err
	}

	return ck.buylist, nil
}

func grading(cardId string, entry mtgban.BuylistEntry) (grade map[string]float64) {
	card, _ := mtgdb.ID2Card(cardId)
	switch {
	case card.Foil:
		grade = map[string]float64{
			"SP": 0.75, "MP": 0.5, "HP": 0.3,
		}
	case entry.BuyPrice < 15:
		grade = map[string]float64{
			"SP": 0.8, "MP": 0.7, "HP": 0.5,
		}
	case entry.BuyPrice >= 15 && entry.BuyPrice < 25:
		grade = map[string]float64{
			"SP": 0.85, "MP": 0.7, "HP": 0.5,
		}
	case entry.BuyPrice >= 25 && entry.BuyPrice < 100:
		grade = map[string]float64{
			"SP": 0.85, "MP": 0.75, "HP": 0.65,
		}
	case entry.BuyPrice >= 100:
		grade = map[string]float64{
			"SP": 0.9, "MP": 0.8, "HP": 0.7,
		}
	}

	switch card.Edition {
	case "Limited Edition Alpha",
		"Limited Edition Beta",
		"Unlimited Edition":
		grade = map[string]float64{
			"SP": 0.8, "MP": 0.6, "HP": 0.4,
		}
	}

	return
}

func (ck *Cardkingdom) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Kingdom"
	info.Shorthand = "CK"
	info.InventoryTimestamp = ck.inventoryDate
	info.BuylistTimestamp = ck.buylistDate
	info.Grading = grading
	return
}
