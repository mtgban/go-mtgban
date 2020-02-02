package strikezone

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

const (
	maxConcurrency = 8

	szInventoryURL = "http://shop.strikezoneonline.com/Category/Magic_the_Gathering_Singles.html"
	szBuylistURL   = "http://shop.strikezoneonline.com/List/MagicBuyList.txt"
)

var untaggedTags = []string{
	"2011 Holiday",
	"2015 Judge Promo",
	"BIBB",
	"Convention Foil M19",
	"Draft Weekend",
	"FNM",
	"Full Box Promo",
	"GP Promo",
	"Grand Prix 2018",
	"Holiday Promo",
	"Judge Promo",
	"Judge 2020",
	"League Promo",
	"MagicFest 2019",
	"MagicFest 2020",
	"MCQ Promo",
	"Media Promo",
	"Players Tour Qualifier PTQ Promo",
	"Prerelease",
	"SDCC 2015",
	"Shooting Star Promo",
	"Standard Showdown 2017",
	"Store Champ",
	"Store Championship",
}

// StrikezoneBuylist is the Scraper for the Strikezone Online vendor.
type Strikezone struct {
	LogCallback mtgban.LogCallbackFunc

	db        mtgjson.MTGDB
	inventory map[string][]mtgban.InventoryEntry
	buylist   map[string][]mtgban.BuylistEntry

	norm *mtgban.Normalizer
}

// NewBuylist initializes a Scraper for retriving buylist information, using
// the passed-in client to make http connections.
func NewScraper(db mtgjson.MTGDB) *Strikezone {
	sz := Strikezone{}
	sz.db = db
	sz.inventory = map[string][]mtgban.InventoryEntry{}
	sz.buylist = map[string][]mtgban.BuylistEntry{}
	sz.norm = mtgban.NewNormalizer()
	return &sz
}

func (sz *Strikezone) printf(format string, a ...interface{}) {
	if sz.LogCallback != nil {
		sz.LogCallback(format, a...)
	}
}

func (sz *Strikezone) parseBL() error {
	resp, err := http.Get(szBuylistURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	r := csv.NewReader(resp.Body)
	r.Comma = '	'
	r.LazyQuotes = true

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if len(record) < 5 {
			return fmt.Errorf("Unsupported buylist format (%d)", len(record))
		}

		cardName := strings.TrimSpace(record[1])
		cardSet := strings.TrimSpace(record[0])

		notes := strings.TrimSpace(record[2])

		quantity, err := strconv.Atoi(strings.TrimSpace(record[3]))
		if err != nil {
			return err
		}

		priceStr := strings.TrimSpace(record[4])
		priceStr = strings.Replace(priceStr, ",", "", 1)
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			return err
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

		isFoil := strings.Contains(notes, "Foil")

		cn, found := cardTable[cardName]
		if found {
			cardName = cn
		}

		// Sometimes the buylist specifies tags at the end of the card name,
		// but without parenthesis, so make sure they are present.
		for _, tag := range untaggedTags {
			if strings.HasSuffix(cardName, tag) {
				cardName = strings.Replace(cardName, tag, "("+tag+")", 1)
				break
			}
		}

		switch {
		case strings.HasPrefix(cardName, "Snow-Cover "):
			cardName = strings.Replace(cardName, "Snow-Cover ", "Snow-Covered ", 1)
		}

		card := &szCard{
			Name:    cardName,
			Edition: cardSet,
			IsFoil:  isFoil,
		}

		cc, err := sz.convert(card)
		if err != nil {
			sz.printf("%v", err)
			continue
		}

		if quantity > 0 && price > 0 {
			var sellPrice, priceRatio, qtyRatio float64
			sellQty := 0

			invCards := sz.inventory[cc.Id]
			for _, invCard := range invCards {
				if invCard.Conditions == "NM" {
					sellPrice = invCard.Price
					sellQty = invCard.Quantity
					break
				}
			}

			if sellPrice > 0 {
				priceRatio = price / sellPrice * 100
			}
			if sellQty > 0 {
				qtyRatio = float64(quantity) / float64(sellQty) * 100
			}

			out := mtgban.BuylistEntry{
				Card:          *cc,
				Conditions:    "NM",
				BuyPrice:      price,
				TradePrice:    price * 1.3,
				Quantity:      quantity,
				PriceRatio:    priceRatio,
				QuantityRatio: qtyRatio,
			}
			err := sz.BuylistAdd(out)
			if err != nil {
				switch cc.Name {
				// Ignore errors coming from lands for now
				case "Plains", "Island", "Swamp", "Mountain", "Forest":
				default:
					sz.printf("%v", err)
				}
				continue
			}
		}
	}

	return nil
}

func (sz *Strikezone) BuylistAdd(card mtgban.BuylistEntry) error {
	entries, found := sz.buylist[card.Id]
	if found {
		for _, entry := range entries {
			if entry.BuyPrice == card.BuyPrice {
				return fmt.Errorf("Attempted to add a duplicate buylist card:\n-new: %v\n-old: %v", card, entry)
			}
		}
	}

	sz.buylist[card.Id] = append(sz.buylist[card.Id], card)
	return nil
}

func (sz *Strikezone) Buylist() (map[string][]mtgban.BuylistEntry, error) {
	if len(sz.buylist) > 0 {
		return sz.buylist, nil
	}

	sz.printf("Empty buylist, scraping started")
	err := sz.parseBL()
	if err != nil {
		return nil, err
	}
	return sz.buylist, nil
}

func (sz *Strikezone) Info() (info mtgban.ScraperInfo) {
	info.Name = "Strike Zone"
	info.Shorthand = "SZ"
	return
}
