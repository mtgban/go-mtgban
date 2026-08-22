// Package arcanafrisia scrapes Arcana Frisia.
package arcanafrisia

import (
	"context"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Arcanafrisia prices what Arcana Frisia buys; they publish no sale prices.
type Arcanafrisia struct {
	LogCallback mtgban.LogCallbackFunc

	buylistDate time.Time
	buylist     mtgban.BuylistRecord
}

// NewScraper returns a buylist scraper.
func NewScraper() *Arcanafrisia {
	af := Arcanafrisia{}
	af.buylist = mtgban.BuylistRecord{}
	return &af
}

func (af *Arcanafrisia) printf(format string, a ...any) {
	if af.LogCallback != nil {
		af.LogCallback("[AF] "+format, a...)
	}
}

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (af *Arcanafrisia) Load(ctx context.Context) error {
	rate, err := mtgban.GetExchangeRate(ctx, "EUR")
	if err != nil {
		return err
	}

	cards, err := GetBuylist(ctx)
	if err != nil {
		return err
	}
	af.printf("Found %d buylist entries", len(cards))

	for _, card := range cards {
		cardID, err := mtgmatcher.MatchID(card.ScryfallID, card.Finish == "foil")
		if err != nil {
			if !mtgmatcher.IsToken(card.Name) {
				af.printf("%v: %s %s (%s)", err, card.ScryfallID, card.Name, card.SetCode)
			}
			continue
		}

		// Map the store's grades onto the ones used internally
		cond := map[string]string{
			"NM": "NM",
			"EX": "SP",
			"GD": "MP",
		}[card.Condition]
		if cond == "" {
			af.printf("Unknown condition %q for %s (%s)", card.Condition, card.Name, card.SetCode)
			continue
		}

		out := &mtgban.BuylistEntry{
			Conditions: cond,
			BuyPrice:   card.PriceEUR * rate,
			Quantity:   card.BuyLimit,
			URL:        card.URL,
		}
		err = af.buylist.AddRelaxed(cardID, out)
		if err != nil {
			af.printf("%v", err)
		}
	}

	af.buylistDate = time.Now()

	return nil
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (af *Arcanafrisia) Buylist() mtgban.BuylistRecord {
	return af.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (af *Arcanafrisia) Info() (info mtgban.ScraperInfo) {
	info.Name = "Arcana Frisia"
	info.Shorthand = "AF"
	info.CountryFlag = "EU"
	info.BuylistTimestamp = &af.buylistDate
	return
}
