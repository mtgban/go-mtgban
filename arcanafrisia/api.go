package arcanafrisia

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/go-cleanhttp"
)

const buylistURL = "https://buylist.arcanafrisia.com/buylist.csv"

// Card is one entry of the published buylist.
type Card struct {
	Name            string
	SetCode         string
	CollectorNumber string
	ScryfallID      string
	TCGplayerID     string
	Condition       string
	Finish          string
	PriceEUR        float64
	BuyLimit        int
	Language        string
	URL             string
}

// GetBuylist downloads the whole buylist in one call.
func GetBuylist(ctx context.Context) ([]Card, error) {
	// Bust any cache in front of the feed so the daily scrape gets a fresh copy
	url := buylistURL + "?v=" + strconv.FormatInt(time.Now().Unix(), 10)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Discard a possible UTF-8 BOM before it trips up the csv parser
	buf := bufio.NewReader(resp.Body)
	bom, err := buf.Peek(3)
	if err == nil && string(bom) == "\ufeff" {
		buf.Discard(3)
	}

	reader := csv.NewReader(buf)

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("could not read csv header: %w", err)
	}
	column := map[string]int{}
	for i, name := range header {
		column[name] = i
	}

	var cards []Card
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		price, err := strconv.ParseFloat(record[column["unit_price_eur"]], 64)
		if err != nil {
			continue
		}
		limit, _ := strconv.Atoi(record[column["buy_limit"]])

		cards = append(cards, Card{
			Name:            record[column["card_name"]],
			SetCode:         record[column["set_code"]],
			CollectorNumber: record[column["collector_number"]],
			ScryfallID:      record[column["scryfall_id"]],
			TCGplayerID:     record[column["tcgplayer_id"]],
			Condition:       record[column["condition"]],
			Finish:          record[column["finish"]],
			PriceEUR:        price,
			BuyLimit:        limit,
			Language:        record[column["language"]],
			URL:             record[column["card_url"]],
		})
	}

	return cards, nil
}
