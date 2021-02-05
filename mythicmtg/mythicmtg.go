package mythicmtg

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	mmtgBuylistURL = "https://mythicmtg.com/public-buylist"
)

var cardTable = map[string]string{
	"Archangel Avacyn | Avacyn, the Purifer": "Archangel Avacyn // Avacyn, the Purifier",
}

type Mythicmtg struct {
	LogCallback mtgban.LogCallbackFunc
	buylistDate time.Time

	buylist mtgban.BuylistRecord
}

func NewScraper() *Mythicmtg {
	mmtg := Mythicmtg{}
	mmtg.buylist = mtgban.BuylistRecord{}
	return &mmtg
}

func (mmtg *Mythicmtg) printf(format string, a ...interface{}) {
	if mmtg.LogCallback != nil {
		mmtg.LogCallback("[MMTG] "+format, a...)
	}
}

func (mmtg *Mythicmtg) parseBL() error {
	resp, err := http.Get(mmtgBuylistURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return err
	}

	doc.Find(`table[class="ty-table"]`).Each(func(i int, s *goquery.Selection) {
		editionName := s.Find("thead").Find("tr:nth-child(1)").Find("th").Text()

		s.Find("tbody").Find("tr").Each(func(_ int, rows *goquery.Selection) {
			cardName := rows.Find("td:nth-child(1)").Text()
			priceStr := rows.Find("td:nth-child(3)").Text()
			creditStr := rows.Find("td:nth-child(4)").Text()
			qtyStr := rows.Find("td:nth-child(5)").Text()

			qty, err := strconv.Atoi(qtyStr)
			if err != nil || qty == 0 {
				return
			}

			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil || price == 0 {
				return
			}
			credit, err := strconv.ParseFloat(creditStr, 64)
			if err != nil {
				return
			}

			variant := ""
			variants := strings.Split(cardName, " - ")
			cardName = variants[0]
			if len(variants) > 1 {
				variant = variants[1]
			}

			if variant == "Extended" && editionName == "Kaldheim" {
				switch cardName {
				case "Alrund's Epiphany",
					"Battle Mammoth",
					"Haunting Voyage",
					"Quakebringer",
					"Starnheim Unleashed":
					variant = "Borderless"
				default:
					variant = "Extended Art"
				}
			}

			isFoil := false
			variants = mtgmatcher.SplitVariants(cardName)
			cardName = variants[0]
			if len(variants) > 1 && variants[1] == "Foil" {
				isFoil = true
			}

			lutName, found := cardTable[cardName]
			if found {
				cardName = lutName
			}

			theCard := &mtgmatcher.Card{
				Name:      cardName,
				Edition:   editionName,
				Variation: variant,
				Foil:      isFoil,
			}
			cardId, err := mtgmatcher.Match(theCard)
			if err != nil {
				mmtg.printf("%v", err)
				mmtg.printf("%q", theCard)
				alias, ok := err.(*mtgmatcher.AliasingError)
				if ok {
					probes := alias.Probe()
					for _, probe := range probes {
						card, _ := mtgmatcher.GetUUID(probe)
						mmtg.printf("- %s", card)
					}
				}
				return
			}

			out := &mtgban.BuylistEntry{
				BuyPrice:   price,
				TradePrice: credit,
				Quantity:   qty,
				URL:        mmtgBuylistURL,
			}
			err = mmtg.buylist.Add(cardId, out)
			if err != nil {
				mmtg.printf("%v", err)
			}
		})
	})

	return nil
}

func (mmtg *Mythicmtg) Buylist() (mtgban.BuylistRecord, error) {
	if len(mmtg.buylist) > 0 {
		return mmtg.buylist, nil
	}

	err := mmtg.parseBL()
	if err != nil {
		return nil, err
	}

	return mmtg.buylist, nil
}

func grading(cardId string, entry mtgban.BuylistEntry) (grade map[string]float64) {
	if entry.BuyPrice < 30 {
		return nil
	}

	grade = map[string]float64{
		"SP": 0.8, "MP": 0.6, "HP": 0,
	}

	return
}

func (mmtg *Mythicmtg) Info() (info mtgban.ScraperInfo) {
	info.Name = "Mythic MTG"
	info.Shorthand = "MMTG"
	info.BuylistTimestamp = mmtg.buylistDate
	info.Grading = grading
	return
}
