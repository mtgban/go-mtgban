package purplemana

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/go-cleanhttp"
	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type Purplemana struct {
	LogCallback mtgban.LogCallbackFunc
	buylistDate time.Time

	buylist mtgban.BuylistRecord
}

const (
	buylistURL = "https://www.purplemana.com/buylist_csv"
)

func NewScraper() *Purplemana {
	pm := Purplemana{}
	pm.buylist = mtgban.BuylistRecord{}
	return &pm
}

func (pm *Purplemana) printf(format string, a ...interface{}) {
	if pm.LogCallback != nil {
		pm.LogCallback("[PM] "+format, a...)
	}
}

func (pm *Purplemana) parseBL() error {
	resp, err := cleanhttp.DefaultClient().Get(buylistURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	r := csv.NewReader(resp.Body)

	// Remove the title
	_, err = r.Read()
	if err != nil {
		return fmt.Errorf("unable to open csv: " + err.Error())
	}

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// CARD NAME - SERIES,CARD_INDEX,CARD_NAME,SET,RETAIL,TRADELIST,SCRYFALLID,IMAGE_URL,LP_CASH,NM_CASH,TCGURL,MKMURL,SCRYIMAGE,
		if len(record) < 9 {
			continue
		}

		id := record[6]
		if id == "" {
			continue
		}

		cardId, err := mtgmatcher.Match(&mtgmatcher.Card{
			Id: id,
		})
		if err != nil {
			pm.printf("%v", err)
			pm.printf("%q", id)
			pm.printf("%q", record)
			alias, ok := err.(*mtgmatcher.AliasingError)
			if ok {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					pm.printf("- %s", card)
				}
			}
			continue
		}

		credit, _ := mtgmatcher.ParsePrice(record[5])
		price, err := mtgmatcher.ParsePrice(record[9])
		if err != nil {
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
