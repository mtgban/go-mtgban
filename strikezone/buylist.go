package strikezone

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	http "github.com/hashicorp/go-retryablehttp"
	"github.com/kodabb/go-mtgban/mtgban"
)

const szBuylistURL = "http://shop.strikezoneonline.com/List/MagicBuyList.txt"

// StrikezoneBuylist is the Scraper for the Strikezone Online vendor.
type StrikezoneBuylist struct{}

// NewBuylist initializes a Scraper for retriving buylist information, using
// the passed-in client to make http connections.
func NewBuylist() mtgban.Scraper {
	return &StrikezoneBuylist{}
}

// Scrape returns an array of Entry, containing pricing and card information
func (sz *StrikezoneBuylist) Scrape() ([]mtgban.Entry, error) {
	resp, err := http.Get(szBuylistURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	r := csv.NewReader(resp.Body)
	r.Comma = '	'
	r.LazyQuotes = true

	db := []mtgban.Entry{}
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(record) < 5 {
			return nil, fmt.Errorf("Unsupported buylist format (%d)", len(record))
		}

		cardName := strings.TrimSpace(record[1])
		cardSet := strings.TrimSpace(record[0])

		notes := strings.TrimSpace(record[2])

		quantity, err := strconv.Atoi(strings.TrimSpace(record[3]))
		if err != nil {
			return nil, err
		}

		priceStr := strings.TrimSpace(record[4])
		priceStr = strings.Replace(priceStr, ",", "", 1)
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return nil, err
		}

		// skip invalid offers
		if price <= 0 {
			continue
		}

		// skip duplicates, with less than NM conditions
		if strings.Contains(notes, "Play") {
			continue
		}

		// skip tokens, too many variations
		if strings.Contains(cardName, "Token") {
			continue
		}

		isFoil := notes == "Foil"

		db = append(db, &SZCard{
			Name:    cardName,
			Set:     cardSet,
			Foil:    isFoil,
			Pricing: price,
			Qty:     quantity,
		})
	}

	return db, nil
}
