package cardkingdom

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
	"github.com/kodabb/go-mtgban/mtgban"
)

const ckBuylistBaseURL = "https://www.cardkingdom.com/purchasing/mtg_singles?filter%5Bipp%5D=1000&filter%5Bsort%5D=edition&filter%5Bsearch%5D=mtg_advanced&page="

// CardkingdomBuylist is the Scraper for the Strikezone Online vendor.
type CardkingdomBuylist struct {
	LogCallback mtgban.LogCallbackFunc
}

// NewBuylist initializes a Scraper for retriving buylist information, using
// the passed-in client to make http connections.
func NewBuylist() mtgban.Scraper {
	ck := CardkingdomBuylist{}
	return &ck
}

func (ck *CardkingdomBuylist) printf(format string, a ...interface{}) {
	if ck.LogCallback != nil {
		ck.LogCallback(format, a...)
	}
}

// Scrape returns an array of Entry, containing pricing and card information
func (ck *CardkingdomBuylist) Scrape() ([]mtgban.Entry, error) {
	db := []mtgban.Entry{}

	c := colly.NewCollector(
		colly.AllowedDomains("www.cardkingdom.com"),

		// Cache responses to prevent multiple download of pages
		// even if the collector is restarted - daily
		colly.CacheDir(fmt.Sprintf(".cache/%d", time.Now().YearDay())),
	)

	// Callback for when a scraped page contains a list element
	c.OnHTML("li[class]", func(e *colly.HTMLElement) {
		// The interesting part is in this class only, discard the rest
		if !strings.Contains(e.Attr("class"), "productItemWrapper") {
			return
		}

		// Retrieve text properties
		cardName := e.ChildText(`span[class="productDetailTitle"]`)
		meta := e.ChildText(`div[class="productDetailSet"]`)
		m := strings.Split(meta, " (")
		cardSet := m[0]
		// m[1] contains rarity
		isFoil := strings.Contains(meta, "FOIL")

		typeLine := e.ChildText(`div[class="productDetailType"]`)

		// Prices are split into variables, combine them
		usd := e.ChildText(`div[class="usdSellPrice"] > span[class="sellDollarAmount"]`)
		cents := e.ChildText(`div[class="usdSellPrice"] > span[class="sellCentsAmount"]`)
		if usd == "" {
			usd = "0"
		}
		if cents == "" {
			cents = "0"
		}
		digits, err := strconv.Atoi(strings.Replace(usd, ",", "", 1))
		if err != nil {
			ck.printf("%s (price) %s", cardName, err.Error())
			return
		}
		price, err := strconv.ParseFloat(fmt.Sprintf("%d.%s", digits, cents), 64)
		if err != nil {
			ck.printf(err.Error())
			return
		}
		quantity := e.ChildAttr(`input[class="maxQty"]`, "value")
		if quantity == "" {
			quantity = "0"
		}
		qty, err := strconv.Atoi(quantity)
		if err != nil {
			ck.printf("%s (qty) %s", cardName, err.Error())
			return
		}
		if price <= 0 || qty <= 0 {
			return
		}

		// Skip cards with too many variations
		if cardSet == "Art Series" ||
			strings.Contains(cardName, "Blank Card") ||
			strings.Contains(cardName, "Checklist") ||
			strings.Contains(cardName, "Emblem") ||
			strings.Contains(cardName, "Oversized") ||
			strings.Contains(cardName, "Promo Plane") ||
			strings.Contains(cardName, "Token") ||
			(cardSet == "Unglued" && (cardName == "Goblin" ||
				cardName == "Pegasus" ||
				cardName == "Sheep" ||
				cardName == "Soldier" ||
				cardName == "Squirrel" ||
				cardName == "Zombie")) {
			return
		}

		// Skip cards that don't officially exist
		switch cardName {
		case "Terra Stomper (Resale Foil)", // does not exist
			"Island (Urza's Saga Arena Foil NO SYMBOL)", // misprint
			"Drudge Skeletons (German Misprint)":        // misprint
			return
		}
		if cardName == "In Oketra's Name" && cardSet == "Mystery Booster" {
			return
		}

		cc := CKCard{
			Name:    cardName,
			Set:     cardSet,
			Foil:    isFoil,
			Pricing: price,
			Qty:     qty,
			Type:    typeLine,
		}
		db = append(db, &cc)
	})

	// Callback for when a scraped page contains an anchor element
	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")

		// Do not follow links from disabled navigation buttons
		// Only follow search page links
		if e.Attr("disabled") != "disabled" && strings.HasPrefix(link, ckBuylistBaseURL) {
			c.Visit(e.Request.AbsoluteURL(link))
		}
	})

	c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		RandomDelay: 1 * time.Second,
	})

	c.OnRequest(func(r *colly.Request) {
		ck.printf("Visiting %s\n", r.URL.String())
	})

	// TODO: start from a random page
	return db, c.Visit(ckBuylistBaseURL + "1")
}
