package mtgban

import "sort"

// SuspectRatioThreshold is the buy-to-retail percentage at which a pairing is
// worth reporting. A shop buys to resell, so its buy price sits well under the
// price it asks: across a Yu-Gi-Oh run the median pairing is 25% and the 99th
// percentile 86%. Above 90% the two prices are describing cards that are
// probably not the same one - a textured foil bought at the plain card's id,
// a Secret Rare bought at the id its common shares - and above 100% the shop
// would be paying more than it charges, which no shop does.
const SuspectRatioThreshold = 90.0

// SuspectPricing is one card a seller both buys and sells at prices too close
// together to be the same printing.
type SuspectPricing struct {
	CardID     string
	Conditions string

	BuyPrice float64
	Price    float64

	// Ratio is BuyPrice over Price as a percentage, spelled the way the
	// buylist entries already spell theirs.
	Ratio float64

	BuyURL    string
	RetailURL string
}

// SuspectPricings names the cards a scraper prices on both sides at a ratio of
// at least threshold, worst first. It compares a condition against its own -
// an NM buy price against an NM asking price - because a shop's grades carry
// their own deductions and reading one against another says nothing.
//
// It reports rather than removes. A store credit multiplier, a bulk shelf
// where both prices round to a cent, and a card a shop is short of all push a
// pairing up honestly, and dropping an entry on suspicion loses a real price
// as surely as a bad match publishes a wrong one. What the list is for is the
// match behind it.
func SuspectPricings(inv InventoryRecord, bl BuylistRecord, threshold float64) []SuspectPricing {
	var out []SuspectPricing

	for cardID, buyEntries := range bl {
		invEntries, found := inv[cardID]
		if !found {
			continue
		}

		for _, condition := range FullGradeTags {
			buy, buyURL := bestBuy(buyEntries, condition)
			ask, askURL := lowestAsk(invEntries, condition)
			if buy <= 0 || ask <= 0 {
				continue
			}

			ratio := buy / ask * 100
			if ratio < threshold {
				continue
			}

			out = append(out, SuspectPricing{
				CardID:     cardID,
				Conditions: condition,
				BuyPrice:   buy,
				Price:      ask,
				Ratio:      ratio,
				BuyURL:     buyURL,
				RetailURL:  askURL,
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Ratio == out[j].Ratio {
			return out[i].CardID < out[j].CardID
		}
		return out[i].Ratio > out[j].Ratio
	})

	return out
}

// bestBuy answers the most a seller pays for a card at one grade, which is the
// figure a listing is judged against.
func bestBuy(entries []BuylistEntry, condition string) (float64, string) {
	var price float64
	var link string
	for _, entry := range entries {
		if entry.Conditions != condition || entry.BuyPrice <= price {
			continue
		}
		price, link = entry.BuyPrice, entry.URL
	}
	return price, link
}

// lowestAsk answers the least a seller charges for a card at one grade.
func lowestAsk(entries []InventoryEntry, condition string) (float64, string) {
	var price float64
	var link string
	for _, entry := range entries {
		if entry.Conditions != condition || entry.Price <= 0 {
			continue
		}
		if price == 0 || entry.Price < price {
			price, link = entry.Price, entry.URL
		}
	}
	return price, link
}
