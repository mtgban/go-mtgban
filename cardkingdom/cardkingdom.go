package cardkingdom

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
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

type retailDetail struct {
	condition string
	quantity  int
	price     float64
}

var CKErrUnsupported = errors.New("unsupported")

/* For newly added cards the embedded nmPrice may not be initialized, so we
 * use the fallbackPrice that is known to be always valid. */
func parseConditions(values conditionValues, fallbackPrice float64) (out []retailDetail) {
	nmPrice, _ := strconv.ParseFloat(values.NMPrice, 64)
	if values.NMQty > 0 {
		if nmPrice == 0 {
			nmPrice = fallbackPrice
		}
		out = append(out, retailDetail{
			condition: "NM",
			quantity:  values.NMQty,
			price:     nmPrice,
		})
	}

	exPrice, _ := strconv.ParseFloat(values.EXPrice, 64)
	if exPrice > 0 && values.EXQty > 0 {
		out = append(out, retailDetail{
			condition: "SP",
			quantity:  values.EXQty,
			price:     exPrice,
		})
	}

	vgPrice, _ := strconv.ParseFloat(values.VGPrice, 64)
	if vgPrice > 0 && values.VGQty > 0 {
		out = append(out, retailDetail{
			condition: "MP",
			quantity:  values.VGQty,
			price:     vgPrice,
		})
	}

	goPrice, _ := strconv.ParseFloat(values.GOPrice, 64)
	if goPrice > 0 && values.GOQty > 0 {
		out = append(out, retailDetail{
			condition: "HP",
			quantity:  values.GOQty,
			price:     goPrice,
		})
	}

	return
}

func (ck *Cardkingdom) scrape() error {
	ckClient := NewCKClient()
	pricelist, err := ckClient.GetPriceList()
	if err != nil {
		return err
	}

	for _, card := range pricelist {
		theCard, err := Preprocess(card)
		if err != nil {
			if err != CKErrUnsupported {
				ck.printf("%v", err)
			}
			continue
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

			details := parseConditions(card.ConditionValues, sellPrice)
			for _, detail := range details {
				out := &mtgban.InventoryEntry{
					Conditions: detail.condition,
					Price:      detail.price,
					Quantity:   detail.quantity,
					URL:        u.String(),
				}
				err = ck.inventory.Add(cardId, out)
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

			q := u.Query()
			q.Set("filter[search]", "mtg_advanced")

			// Include as much detail as possible in the name
			cardName := card.Name
			if card.Variation != "" {
				cardName = fmt.Sprintf("%s (%s)", card.Name, card.Variation)
			}
			q.Set("filter[name]", cardName)

			// Always show both non-foil and foil cards, the filtering
			// on the website accurate enough (ie for Prerelease)
			q.Set("filter[singles]", "1")
			q.Set("filter[foils]", "1")

			// Edition is derived from the url itself, not the edition field
			urlPaths := strings.Split(card.URL, "/")
			if len(urlPaths) > 2 {
				q.Set("filter[edition]", urlPaths[1])
			}
			if ck.Partner != "" {
				q.Set("partner", ck.Partner)
				q.Set("utm_source", ck.Partner)
				q.Set("utm_medium", "affiliate")
				q.Set("utm_campaign", ck.Partner)
			}
			u.RawQuery = q.Encode()

			gradeMap := grading(cardId, price)
			for _, grade := range mtgban.DefaultGradeTags {
				factor := gradeMap[grade]
				var priceRatio float64

				if sellPrice > 0 {
					priceRatio = price / sellPrice * 100
				}

				out := &mtgban.BuylistEntry{
					Conditions: grade,
					BuyPrice:   price * factor,
					TradePrice: price * factor * 1.3,
					Quantity:   card.BuyQuantity,
					PriceRatio: priceRatio,
					URL:        u.String(),
				}
				// Add the line entry as needed by the csv import
				if grade == "NM" {
					out.CustomFields = map[string]string{
						"CKTitle":   cardName,
						"CKEdition": card.Edition,
						"CKFoil":    card.IsFoil,
						"CKSKU":     card.SKU,
						"CKID":      fmt.Sprint(card.Id),
					}
				}
				err = ck.buylist.Add(cardId, out)
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

func grading(cardId string, price float64) (grade map[string]float64) {
	co, err := mtgmatcher.GetUUID(cardId)
	if err != nil {
		return
	}
	set, err := mtgmatcher.GetSet(co.SetCode)
	if err != nil {
		return
	}

	switch {
	case co.Foil && !set.IsFoilOnly:
		grade = map[string]float64{
			"NM": 1, "SP": 0.75, "MP": 0.5, "HP": 0.3,
		}
	case price < 15:
		grade = map[string]float64{
			"NM": 1, "SP": 0.8, "MP": 0.7, "HP": 0.5,
		}
	case price >= 15 && price < 25:
		grade = map[string]float64{
			"NM": 1, "SP": 0.85, "MP": 0.7, "HP": 0.5,
		}
	case price >= 25 && price < 100:
		grade = map[string]float64{
			"NM": 1, "SP": 0.85, "MP": 0.75, "HP": 0.65,
		}
	case price >= 100:
		grade = map[string]float64{
			"NM": 1, "SP": 0.9, "MP": 0.8, "HP": 0.7,
		}
	}

	switch co.Edition {
	case "Limited Edition Alpha",
		"Limited Edition Beta",
		"Unlimited Edition":
		grade = map[string]float64{
			"NM": 1, "SP": 0.8, "MP": 0.6, "HP": 0.4,
		}
	}

	return
}

func (ck *Cardkingdom) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Kingdom"
	info.Shorthand = "CK"
	info.InventoryTimestamp = &ck.inventoryDate
	info.BuylistTimestamp = &ck.buylistDate
	return
}
