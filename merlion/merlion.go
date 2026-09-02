// Package merlion scrapes Merlion Games.
package merlion

import (
	"context"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Merlion prices what Merlion Games buys.
type Merlion struct {
	LogCallback mtgban.LogCallbackFunc

	buylistDate time.Time
	buylist     mtgban.BuylistRecord
}

// NewScraper returns a buylist scraper.
func NewScraper() *Merlion {
	mg := Merlion{}
	mg.buylist = mtgban.BuylistRecord{}
	return &mg
}

func (mg *Merlion) printf(format string, a ...any) {
	if mg.LogCallback != nil {
		mg.LogCallback("[MG] "+format, a...)
	}
}

// Merlion quotes one price per printing and buys played copies at a set
// discount off it, so the feed's price is the Near Mint one and the rest of
// the ladder comes from these factors. A grade left out is one they do not
// buy at all.
var gradeFactors = map[string]float64{
	"NM": 1,
	"SP": 0.8,
}

// Played copies are only worth quoting on the expensive printings; below this
// the discount lands within rounding of the Near Mint price and would just
// double the entries for nothing.
const playedPriceFloor = 100

// Load fetches everything this scraper offers. See mtgban.Scraper.
func (mg *Merlion) Load(ctx context.Context) error {
	cards, err := DownloadBuylistCSV(ctx)
	if err != nil {
		return err
	}
	mg.printf("Found %d buylist entries", len(cards))

	for _, card := range cards {
		// The feed names a printing by its TCGplayer id: ConvertID crosses
		// into the matcher's uuids and MatchID applies the finish. So
		// neither the name nor the edition has to survive the round trip.
		uuid := mtgmatcher.ConvertID(mtgmatcher.IDSpaceTCGplayer, card.TCGplayerID)
		cardID, err := mtgmatcher.MatchID(uuid, card.Foil)
		if err != nil {
			mg.printf("%v: %s %s (%s)", err, card.TCGplayerID, card.Name, card.Edition)
			continue
		}

		// The quoted price belongs to a Near Mint copy; any other grade would
		// be priced as though it were one.
		if !strings.Contains(card.Condition, nearMint) {
			mg.printf("skipping %s (%s): grade %q is not %s",
				card.Name, card.Edition, card.Condition, nearMint)
			continue
		}

		for _, grade := range mtgban.DefaultGradeTags {
			factor, found := gradeFactors[grade]
			if !found {
				continue
			}
			if grade != "NM" && card.BuyPrice <= playedPriceFloor {
				continue
			}

			// The feed counts how many copies they want, not how many per
			// grade, so only the quoted grade carries the number.
			var quantity int
			if grade == "NM" {
				quantity = card.Quantity
			}

			out := &mtgban.BuylistEntry{
				Conditions: grade,
				BuyPrice:   card.BuyPrice * factor,
				Quantity:   quantity,
				URL:        card.URL,
			}
			err = mg.buylist.AddRelaxed(cardID, out)
			if err != nil {
				mg.printf("%v", err)
			}
		}
	}

	mg.buylistDate = time.Now()

	return nil
}

// Buylist returns what Load collected. See mtgban.Vendor.
func (mg *Merlion) Buylist() mtgban.BuylistRecord {
	return mg.buylist
}

// Info describes this scraper. See mtgban.Scraper.
func (mg *Merlion) Info() (info mtgban.ScraperInfo) {
	info.Name = "Merlion Games"
	info.Shorthand = "MG"
	info.Game = mtgban.GameRiftbound
	info.BuylistTimestamp = &mg.buylistDate
	return
}
