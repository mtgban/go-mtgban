package blueprint

import (
	"errors"
	"strconv"
	"strings"
	"time"

	excelize "github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/PuerkitoBio/goquery"
	"github.com/hashicorp/go-cleanhttp"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	buylistURL = "http://www.mtgblueprint.com"
)

type Blueprint struct {
	LogCallback mtgban.LogCallbackFunc
	buylistDate time.Time

	buylist mtgban.BuylistRecord
}

func NewScraper() *Blueprint {
	bp := Blueprint{}
	bp.buylist = mtgban.BuylistRecord{}
	return &bp
}

func (bp *Blueprint) printf(format string, a ...interface{}) {
	if bp.LogCallback != nil {
		bp.LogCallback("[BP] "+format, a...)
	}
}

func getSpreadsheetURL() (string, error) {
	resp, err := cleanhttp.DefaultClient().Get(buylistURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	link, found := doc.Find(`a`).First().Attr("href")
	if !found {
		return "", errors.New("spreadsheet anchor tag not found")
	}

	return buylistURL + link, nil
}

func (bp *Blueprint) parseBL() error {
	blURL, err := getSpreadsheetURL()
	if err != nil {
		return err
	}
	bp.printf("Using %s as input spreadsheet", blURL)

	resp, err := cleanhttp.DefaultClient().Get(blURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	f, err := excelize.OpenReader(resp.Body)
	if err != nil {
		return err
	}

	sheets := f.GetSheetList()
	var i int
	for i = range sheets {
		if sheets[i] != "Instructions" {
			break
		}
	}

	// Get all the rows in the Sheet1.
	rows, err := f.GetRows(sheets[i])
	if err != nil {
		return err
	}
	for _, row := range rows {
		if len(row) < 5 {
			continue
		}
		cardName := row[4]
		edition := row[5]
		price, _ := strconv.ParseFloat(row[2], 64)

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
		if errors.Is(err, mtgmatcher.ErrUnsupported) {
			continue
		} else if err != nil {
			bp.printf("%v", err)
			bp.printf("%q", theCard)
			bp.printf("%q", row)

			var alias *mtgmatcher.AliasingError
			if errors.As(err, &alias) {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					bp.printf("- %s", card)
				}
			}
			continue
		}

		out := &mtgban.BuylistEntry{
			BuyPrice: price,
			URL:      "http://www.mtgblueprint.com/",
		}
		err = bp.buylist.Add(cardId, out)
		if err != nil {
			bp.printf("%v", err)
		}
	}

	bp.buylistDate = time.Now()

	return nil
}

func (bp *Blueprint) Buylist() (mtgban.BuylistRecord, error) {
	if len(bp.buylist) > 0 {
		return bp.buylist, nil
	}

	err := bp.parseBL()
	if err != nil {
		return nil, err
	}

	return bp.buylist, nil
}

func (bp *Blueprint) Info() (info mtgban.ScraperInfo) {
	info.Name = "Blueprint"
	info.Shorthand = "BP"
	info.BuylistTimestamp = &bp.buylistDate
	info.NoCredit = true
	return
}
