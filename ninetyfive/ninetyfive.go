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
	"github.com/kodabb/go-mtgban/mtgjson"
)

const (
	maxConcurrency = 8

	nfBuylistURL = "http://95mtg.com/buylist/print_list.txt"
)

// NinetyfiveBuylist is the Scraper for the Ninetyfive Online vendor.
type Ninetyfive struct {
	LogCallback   mtgban.LogCallbackFunc
	InventoryDate time.Time
	BuylistDate   time.Time

	db        mtgjson.MTGDB
	inventory map[string][]mtgban.InventoryEntry
	buylist   map[string]mtgban.BuylistEntry

	norm *mtgban.Normalizer
}

// NewBuylist initializes a Scraper for retriving buylist information, using
// the passed-in client to make http connections.
func NewScraper(db mtgjson.MTGDB) *Ninetyfive {
	nf := Ninetyfive{}
	nf.db = db
	nf.inventory = map[string][]mtgban.InventoryEntry{}
	nf.buylist = map[string]mtgban.BuylistEntry{}
	nf.norm = mtgban.NewNormalizer()
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

		card := &nfCard{
			Name:   cardName,
			Code:   setCode,
			IsFoil: isFoil,
		}

		cc, err := nf.convert(card)
		if err != nil {
			nf.printf("%v", err)
			continue
		}

		if quantity > 0 && price > 0 {
			out := mtgban.BuylistEntry{
				Card:       *cc,
				Conditions: "NM",
				BuyPrice:   price,
				TradePrice: 0,
				Quantity:   quantity,
			}
			err := mtgban.BuylistAdd(nf.buylist, out)
			if err != nil {
				nf.printf("%v", err)
			}
		}
	}

	nf.BuylistDate = time.Now()

	return nil
}

func (nf *Ninetyfive) Buylist() (map[string]mtgban.BuylistEntry, error) {
	if len(nf.buylist) > 0 {
		return nf.buylist, nil
	}

	start := time.Now()
	nf.printf("Buylist scraping started at %s", start)

	err := nf.parseBL()
	if err != nil {
		return nil, err
	}
	nf.printf("Buylist scraping took %s", time.Since(start))

	return nf.buylist, nil
}

func (nf *Ninetyfive) Grading(entry mtgban.BuylistEntry) (grade map[string]float64) {
	return nil
}

func (nf *Ninetyfive) Info() (info mtgban.ScraperInfo) {
	info.Name = "95mtg"
	info.Shorthand = "95"
	info.BuylistTimestamp = nf.BuylistDate
	return
}
