package purplemana

import (
	"context"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"

	"golang.org/x/oauth2/google"
	spreadsheet "gopkg.in/Iwark/spreadsheet.v2"
)

type Purplemana struct {
	LogCallback mtgban.LogCallbackFunc
	buylistDate time.Time

	client  *http.Client
	buylist mtgban.BuylistRecord
}

func NewScraper(credentials string) (*Purplemana, error) {
	pm := Purplemana{}
	pm.buylist = mtgban.BuylistRecord{}

	data, err := ioutil.ReadFile(credentials)
	if err != nil {
		return nil, err
	}

	conf, err := google.JWTConfigFromJSON(data, spreadsheet.Scope)
	if err != nil {
		return nil, err
	}
	pm.client = conf.Client(context.TODO())

	return &pm, nil
}

func (pm *Purplemana) printf(format string, a ...interface{}) {
	if pm.LogCallback != nil {
		pm.LogCallback("[PM] "+format, a...)
	}
}

func (pm *Purplemana) parseBL() error {
	service := spreadsheet.NewServiceWithClient(pm.client)
	spreadsheet, err := service.FetchSpreadsheet("1Z-wDWITH5s7tSigWnZqf8GmKJtNXj4aBpHT8OsY2lCg")
	if err != nil {
		return err
	}

	sheet, err := spreadsheet.SheetByIndex(0)
	if err != nil {
		return err
	}

	for _, row := range sheet.Rows {
		if len(row) < 7 {
			continue
		}

		cardName := row[2].Value

		if cardName == "Ring of Ma'rÃ»f" {
			cardName = "Ring of Ma'rûf"
		}

		edition := row[3].Value

		if strings.Contains(edition, "Artist Proof") {
			continue
		}

		price, _ := mtgmatcher.ParsePrice(row[7].Value)
		credit, _ := mtgmatcher.ParsePrice(row[6].Value)

		if cardName == "" || edition == "" || price == 0 {
			continue
		}

		variant := ""
		if strings.Contains(cardName, "(") {
			vars := mtgmatcher.SplitVariants(cardName)
			cardName = vars[0]
			if len(vars) > 1 {
				variant = vars[1]
			}
		}

		theCard := &mtgmatcher.Card{
			Name:      cardName,
			Edition:   edition,
			Variation: variant,
		}

		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			alias, ok := err.(*mtgmatcher.AliasingError)
			if ok && edition == "Antiquities" {
				continue
			}

			pm.printf("%v", err)
			pm.printf("%q", theCard)
			pm.printf("%q", row)
			if ok {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					pm.printf("- %s", card)
				}
			}
			continue
		}

		out := &mtgban.BuylistEntry{
			BuyPrice:   price,
			TradePrice: credit,
			URL:        "https://www.purplemana.com/trading/",
		}
		err = pm.buylist.Add(cardId, out)
		if err != nil {
			pm.printf("%v", err)
		}
	}

	pm.buylistDate = time.Now()

	return nil
}

func (pm *Purplemana) Buylist() (mtgban.BuylistRecord, error) {
	if len(pm.buylist) > 0 {
		return pm.buylist, nil
	}

	err := pm.parseBL()
	if err != nil {
		return nil, err
	}

	return pm.buylist, nil
}

func (pm *Purplemana) Info() (info mtgban.ScraperInfo) {
	info.Name = "Purplemana"
	info.Shorthand = "PM"
	info.BuylistTimestamp = pm.buylistDate
	info.Grading = mtgban.DefaultGrading
	return
}
