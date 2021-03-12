package blueprint

import (
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	excelize "github.com/360EntSecGroup-Skylar/excelize/v2"
	"github.com/hashicorp/go-cleanhttp"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	BlueprintURL = "http://www.mtgblueprint.com/s/Blueprint_%s.xlsx"
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

func (bp *Blueprint) parseBL() error {
	var reader io.ReadCloser
	for i := 0; i < 12; i++ {
		t := time.Now().AddDate(0, -i, 1)
		blURL := fmt.Sprintf(BlueprintURL, t.Format("January_2006"))
		bp.printf("Trying %s", blURL)
		resp, err := cleanhttp.DefaultClient().Get(blURL)
		if err != nil {
			bp.printf("not found, continuing")
			continue
		}
		if resp.StatusCode == 200 {
			reader = resp.Body
			break
		}
		defer resp.Body.Close()
		bp.printf("url found, but with status %d, continuing", resp.StatusCode)
	}
	if reader == nil {
		bp.printf("no updates over a year")
		return nil
	}

	f, err := excelize.OpenReader(reader)
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
		if err != nil {
			bp.printf("%v", err)
			bp.printf("%q", theCard)
			bp.printf("%q", row)
			alias, ok := err.(*mtgmatcher.AliasingError)
			if ok {
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

func grading(_ string, entry mtgban.BuylistEntry) (grade map[string]float64) {
	return nil
}

func (bp *Blueprint) Info() (info mtgban.ScraperInfo) {
	info.Name = "Blueprint"
	info.Shorthand = "BP"
	info.BuylistTimestamp = bp.buylistDate
	info.Grading = grading
	info.NoCredit = true
	return
}
