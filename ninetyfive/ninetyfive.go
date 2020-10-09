package ninetyfive

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	nfBuylistURL = "http://95mtg.com/buylist/print_list.txt"
)

type Ninetyfive struct {
	LogCallback mtgban.LogCallbackFunc
	buylistDate time.Time

	buylist mtgban.BuylistRecord
}

func NewScraper() *Ninetyfive {
	nf := Ninetyfive{}
	nf.buylist = mtgban.BuylistRecord{}
	return &nf
}

func (nf *Ninetyfive) printf(format string, a ...interface{}) {
	if nf.LogCallback != nil {
		nf.LogCallback("[95] "+format, a...)
	}
}

func (nf *Ninetyfive) parseBL() error {
	resp, err := http.Get(nfBuylistURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	r := csv.NewReader(resp.Body)
	r.Comma = '~'

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if len(record) < 3 {
			return fmt.Errorf("Unsupported buylist format (%d)", len(record))
		}

		fields := strings.Split(record[2], "_")
		if len(fields) < 5 {
			return fmt.Errorf("Unsupported buylist data format (%d)", len(fields))
		}

		cardName := strings.TrimSpace(record[0])

		priceStr := strings.TrimSpace(record[1])
		priceStr = strings.Replace(priceStr, "$", "", 1)
		priceStr = strings.Replace(priceStr, ",", "", 1)
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return err
		}

		setCode := fields[0]
		quantity, err := strconv.Atoi(fields[1])
		if err != nil {
			return err
		}
		condition := fields[2]
		if condition != "NM" {
			continue
		}
		lang := fields[3]
		if lang != "EN" {
			continue
		}
		isFoil, _ := strconv.ParseBool(fields[4])

		theCard := &mtgmatcher.Card{
			Name:    cardName,
			Edition: setCode,
			Foil:    isFoil,
		}

		cardId, err := mtgmatcher.Match(theCard)
		if err != nil {
			nf.printf("%v", err)
			nf.printf("%q", theCard)
			nf.printf("%q", record)
			alias, ok := err.(*mtgmatcher.AliasingError)
			if ok {
				probes := alias.Probe()
				for _, probe := range probes {
					card, _ := mtgmatcher.GetUUID(probe)
					nf.printf("- %s", card)
				}
			}
			continue
		}

		if quantity > 0 && price > 0 {
			out := &mtgban.BuylistEntry{
				BuyPrice: price,
				Quantity: quantity,
				URL:      "http://www.95mtg.com/buylist/",
			}
			err := nf.buylist.Add(cardId, out)
			if err != nil {
				nf.printf("%v", err)
			}
		}
	}

	nf.buylistDate = time.Now()

	return nil
}

func (nf *Ninetyfive) Buylist() (mtgban.BuylistRecord, error) {
	if len(nf.buylist) > 0 {
		return nf.buylist, nil
	}

	err := nf.parseBL()
	if err != nil {
		return nil, err
	}

	return nf.buylist, nil
}

func (nf *Ninetyfive) Info() (info mtgban.ScraperInfo) {
	info.Name = "95mtg"
	info.Shorthand = "95"
	info.BuylistTimestamp = nf.buylistDate
	info.Grading = mtgban.DefaultGrading
	info.NoCredit = true
	return
}
