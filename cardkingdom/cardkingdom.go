package cardkingdom

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

type Cardkingdom struct {
	LogCallback   mtgban.LogCallbackFunc
	Partner       string
	InventoryDate time.Time
	BuylistDate   time.Time

	db        mtgjson.MTGDB
	inventory map[string][]mtgban.InventoryEntry
	buylist   map[string]mtgban.BuylistEntry

	norm *mtgban.Normalizer
}

func NewScraper(db mtgjson.MTGDB) *Cardkingdom {
	ck := Cardkingdom{}
	ck.db = db
	ck.inventory = map[string][]mtgban.InventoryEntry{}
	ck.buylist = map[string]mtgban.BuylistEntry{}
	ck.norm = mtgban.NewNormalizer()
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
		if strings.Contains(card.Name, "Token") ||
			strings.Contains(card.Name, "Emblem") ||
			strings.Contains(card.Name, "Checklist") ||
			strings.Contains(card.Variation, "Misprint") ||
			strings.Contains(card.Variation, "Oversized") ||
			card.Name == "Blank Card" ||
			card.Edition == "Art Series" ||
			card.Variation == "MagicFest Non-Foil - 2020" ||
			card.Variation == "Urza's Saga Arena Foil NO SYMBOL" ||
			card.SKU == "OVERSIZ" ||
			card.SKU == "PRES-005A" {
			continue
		}

		setCode := ""
		number := ""
		isFoil := card.IsFoil == "true"

		sku := card.SKU
		fixup, found := skuFixupTable[sku]
		if found {
			sku = fixup
		}

		fields := strings.Split(sku, "-")
		if len(fields) > 1 {
			setCode = fields[0]
			if len(setCode) > 3 && isFoil && strings.HasPrefix(setCode, "F") {
				setCode = setCode[1:]
			}
			if len(setCode) == 4 && strings.HasPrefix(setCode, "T") {
				continue
			}
			number = strings.Join(fields[1:], "")
			number = strings.TrimLeft(number, "0")
			number = strings.ToLower(number)
		}

		cardName := card.Name
		name, found := cardTable[cardName]
		if found {
			cardName = name
		}

		ckCard := ckCard{
			Name:      cardName,
			Edition:   card.Edition,
			Foil:      isFoil,
			SetCode:   setCode,
			Variation: card.Variation,
			Number:    number,
		}

		cc, err := ck.convert(&ckCard)
		if err != nil {
			ck.printf("%v", err)
			continue
		}

		var sellPrice float64
		u, _ := url.Parse("https://www.cardkingdom.com/")
		if card.SellQuantity > 0 {
			sellPrice, err = strconv.ParseFloat(card.SellPrice, 64)
			if err != nil {
				ck.printf("%v", err)
			}
			if sellPrice > 0 {
				u.Path = card.URL
				if ck.Partner != "" {
					q := u.Query()
					q.Set("partner", ck.Partner)
					q.Set("utm_source", ck.Partner)
					q.Set("utm_medium", "affiliate")
					q.Set("utm_campaign", ck.Partner)
					u.RawQuery = q.Encode()
				}

				out := mtgban.InventoryEntry{
					Card:       *cc,
					Conditions: "NM",
					Price:      sellPrice,
					Quantity:   card.SellQuantity,
					Notes:      u.String(),
				}
				err = mtgban.InventoryAdd(ck.inventory, out)
				if err != nil {
					ck.printf("%v", err)
				}
			}
		}

		u, _ = url.Parse("https://www.cardkingdom.com/purchasing/mtg_singles")
		if card.BuyQuantity > 0 {
			price, err := strconv.ParseFloat(card.BuyPrice, 64)
			if err != nil {
				ck.printf("%v", err)
			}
			if price > 0 {
				var priceRatio, qtyRatio float64

				if sellPrice > 0 {
					priceRatio = price / sellPrice * 100
				}
				if card.SellQuantity > 0 {
					qtyRatio = float64(card.BuyQuantity) / float64(card.SellQuantity) * 100
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
				if isFoil {
					q.Set("filter[foil]", "1")
				} else {
					q.Set("filter[nonfoil]", "1")
				}
				u.RawQuery = q.Encode()

				out := mtgban.BuylistEntry{
					Card:          *cc,
					Conditions:    "NM",
					BuyPrice:      price,
					TradePrice:    price * 1.3,
					Quantity:      card.BuyQuantity,
					PriceRatio:    priceRatio,
					QuantityRatio: qtyRatio,
					Notes:         u.String(),
				}
				err = mtgban.BuylistAdd(ck.buylist, out)
				if err != nil {
					ck.printf("%v", err)
				}
			}
		}
	}

	ck.InventoryDate = time.Now()
	ck.BuylistDate = time.Now()

	return nil
}

func (ck *Cardkingdom) Inventory() (map[string][]mtgban.InventoryEntry, error) {
	if len(ck.inventory) > 0 {
		return ck.inventory, nil
	}

	start := time.Now()
	ck.printf("Inventory scraping started at %s", start)

	err := ck.scrape()
	if err != nil {
		return nil, err
	}
	ck.printf("Inventory scraping took %s", time.Since(start))

	return ck.inventory, nil

}

func (ck *Cardkingdom) Buylist() (map[string]mtgban.BuylistEntry, error) {
	if len(ck.buylist) > 0 {
		return ck.buylist, nil
	}

	start := time.Now()
	ck.printf("Buylist scraping started at %s", start)

	err := ck.scrape()
	if err != nil {
		return nil, err
	}
	ck.printf("Buylist scraping took %s", time.Since(start))

	return ck.buylist, nil
}

func (ck *Cardkingdom) Grading(entry mtgban.BuylistEntry) (grade map[string]float64) {
	switch {
	case entry.Card.Foil:
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

	switch entry.Card.Set {
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
	info.InventoryTimestamp = ck.InventoryDate
	info.BuylistTimestamp = ck.BuylistDate
	return
}
