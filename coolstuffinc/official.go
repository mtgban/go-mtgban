package coolstuffinc

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type CoolstuffincOfficial struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	// false = load buylist from main api
	// true  = load buylist from buylist endpoint
	IgnoreOfficialBuylist bool

	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord

	client *CSIClient

	edition2id map[string]string
}

func NewScraperOfficial(key string) *CoolstuffincOfficial {
	csi := CoolstuffincOfficial{}
	csi.inventory = mtgban.InventoryRecord{}
	csi.buylist = mtgban.BuylistRecord{}
	csi.client = NewCSIClient(key)
	csi.IgnoreOfficialBuylist = true
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

		if card.PriceBuy > 0 && !csi.IgnoreOfficialBuylist {
			var priceRatio float64

			if card.PriceRetail > 0 {
				priceRatio = card.PriceBuy / card.PriceRetail * 100
			}

			for i, deduction := range deductions {
				out := mtgban.BuylistEntry{
					Conditions: mtgban.DefaultGradeTags[i],
					BuyPrice:   card.PriceBuy * deduction,
					PriceRatio: priceRatio,
					URL:        defaultBuylistPage,
				}
				err = csi.buylist.Add(cardId, &out)
				if err != nil && i == 0 {
					csi.printf("%v", err)
					csi.printf("%q", theCard)
					csi.printf("%q", card)
				}
			}
		}
	}

	csi.inventoryDate = time.Now()
	if !csi.IgnoreOfficialBuylist {
		csi.buylistDate = time.Now()
	}

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

func (csi *CoolstuffincOfficial) parseBL() error {
	edition2id, err := LoadBuylistEditions()
	if err != nil {
		return err
	}
	csi.printf("Loaded %d editions", len(edition2id))

	products, err := GetBuylist()
	if err != nil {
		return err
	}
	csi.printf("Found %d products", len(products))

	for _, product := range products {
		if product.RarityName == "Box" {
			continue
		}

		// Build link early to help debug
		u, _ := url.Parse(csiBuylistLink)
		v := url.Values{}
		v.Set("s", "mtg")
		v.Set("a", "1")
		v.Set("name", product.Name)
		v.Set("f[]", fmt.Sprint(product.IsFoil))

		id, found := edition2id[product.ItemSet]
		if found {
			v.Set("is[]", id)
		}
		u.RawQuery = v.Encode()
		link := u.String()

		theCard, err := PreprocessBuylist(product)
		if err != nil {
			continue
		}

		cardId, err := mtgmatcher.Match(theCard)
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			csi.printf("error: %v", err)
			csi.printf("original: %q", product)
			csi.printf("preprocessed: %q", theCard)
			csi.printf("link: %q", link)

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

		buyPrice, err := mtgmatcher.ParsePrice(product.Price)
		if err != nil {
			csi.printf("%s error: %s", product.Name, err.Error())
			continue
		}

		var priceRatio, sellPrice float64

		invCards := csi.inventory[cardId]
		for _, invCard := range invCards {
			sellPrice = invCard.Price
			break
		}
		if sellPrice > 0 {
			priceRatio = buyPrice / sellPrice * 100
		}

		for i, deduction := range deductions {
			buyEntry := mtgban.BuylistEntry{
				Conditions: mtgban.DefaultGradeTags[i],
				BuyPrice:   buyPrice * deduction,
				PriceRatio: priceRatio,
				URL:        link,
			}

			err := csi.buylist.Add(cardId, &buyEntry)
			if err != nil {
				csi.printf("%s", err.Error())
				continue
			}
		}
	}

	csi.buylistDate = time.Now()

	return nil
}

func (csi *CoolstuffincOfficial) Buylist() (mtgban.BuylistRecord, error) {
	if len(csi.buylist) > 0 {
		return csi.buylist, nil
	}

	err := csi.parseBL()
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
	info.CreditMultiplier = 1.25
	return
}
