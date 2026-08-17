package merlion

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/hashicorp/go-cleanhttp"
)

const buylistURL = "https://www.merlion.gg/api/buylist/riftbound/csv"

// The product link spells the TCGplayer id ahead of a slug that is free to
// change, and that id is the only key the feed shares with the datastore.
var productLink = regexp.MustCompile(`/product/(\d+)`)

// The condition column carries the finish alongside the grade rather than
// filling a column of its own, so a foil Near Mint reads "Near Mint Foil".
// Both are read by substring: the column is one free-text field and the
// storefront is free to punctuate it differently.
const (
	foilTag  = "Foil"
	nearMint = "Near Mint"
)

type Card struct {
	Name    string
	Edition string
	Number  string

	// Condition verbatim, finish included, e.g. "Near Mint Foil".
	Condition string
	Foil      bool

	TCGplayerID string
	BuyPrice    float64
	Quantity    int
	URL         string
}

func DownloadBuylistCSV(ctx context.Context) ([]Card, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, buylistURL, http.NoBody)
	if err != nil {
		return nil, err
	}

	resp, err := cleanhttp.DefaultClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("buylist request returned %s", resp.Status)
	}

	return parseBuylist(resp.Body)
}

func parseBuylist(r io.Reader) ([]Card, error) {
	// Discard a possible UTF-8 BOM before it trips up the csv parser
	buf := bufio.NewReader(r)
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
		column[strings.TrimSpace(name)] = i
	}
	for _, name := range []string{"Product Name", "Condition", "Buy Price", "TCGplayer Link"} {
		_, found := column[name]
		if !found {
			return nil, fmt.Errorf("csv header is missing %q", name)
		}
	}

	// A record is only worth returning once it carries the id that resolves
	// it, so the field lookups below stay total.
	field := func(record []string, name string) string {
		i, found := column[name]
		if !found || i >= len(record) {
			return ""
		}
		return strings.TrimSpace(record[i])
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

		link := field(record, "TCGplayer Link")
		id := productLink.FindStringSubmatch(link)
		if id == nil {
			continue
		}

		price, err := strconv.ParseFloat(field(record, "Buy Price"), 64)
		if err != nil {
			continue
		}
		quantity, _ := strconv.Atoi(field(record, "Buy Quantity"))

		condition := field(record, "Condition")

		cards = append(cards, Card{
			Name:        field(record, "Product Name"),
			Edition:     field(record, "Set Name"),
			Number:      field(record, "Number"),
			Condition:   condition,
			Foil:        strings.Contains(condition, foilTag),
			TCGplayerID: id[1],
			BuyPrice:    price,
			Quantity:    quantity,
			URL:         field(record, "Merlion Buylist Link"),
		})
	}

	return cards, nil
}
