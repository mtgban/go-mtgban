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

// CollapsedRatioThreshold is the ratio between the most and the least a
// vendor pays for one card at one grade above which the pair is worth
// reporting. A shop moves a price for honest reasons - a quantity tier, a
// credit multiplier folded into the same vendor name - and those stay within
// a factor of two. Past it the two prices are describing cards that are not
// the same one.
const CollapsedRatioThreshold = 2.0

// CollapsedPricing is one card a vendor buys at more than one price at the
// same grade.
type CollapsedPricing struct {
	CardID     string
	Conditions string
	VendorName string

	// High and Low are the most and the least the vendor pays at this
	// grade, and Ratio is the first over the second.
	High  float64
	Low   float64
	Ratio float64

	// HighURL and LowURL are the two listings. They are the point of the
	// report: a storefront publishes the wording that tells the printings
	// apart, and it is that wording the match dropped.
	HighURL string
	LowURL  string

	// Count is how many prices the vendor holds for this card at this
	// grade, which is how many of its products landed on one id.
	Count int
}

// CollapsedPricings names the cards a vendor buys at several prices at one
// grade, widest spread first.
//
// A shop pays one price for one card at one grade, so a second price is not
// a better offer but a second product: the feed named two printings and the
// match folded them onto one id. Unlike SuspectPricings this needs only the
// one side, and it is certain where that one is a guess - a buy price above
// what the same shop asks has honest explanations, two buy prices at one
// grade have none.
//
// It reports rather than removes, for the same reason SuspectPricings does:
// the entry that should go is whichever one the match got wrong, and which
// that is cannot be read off the prices. What the list is for is the match
// behind it. A scraper whose feed does state one price per printing can
// refuse the second outright with BuylistRecord.AddUnique.
func CollapsedPricings(bl BuylistRecord, threshold float64) []CollapsedPricing {
	var out []CollapsedPricing

	for cardID, entries := range bl {
		// One vendor's prices at one grade, keyed the way a duplicate is:
		// a record holds several vendors, and a shop quoting cash beside
		// store credit publishes them under names of their own.
		type key struct {
			condition string
			vendor    string
		}
		grouped := map[key][]BuylistEntry{}
		for _, entry := range entries {
			k := key{entry.Conditions, entry.VendorName}
			grouped[k] = append(grouped[k], entry)
		}

		for k, group := range grouped {
			if len(group) < 2 {
				continue
			}
			high, low := group[0], group[0]
			for _, entry := range group {
				if entry.BuyPrice > high.BuyPrice {
					high = entry
				}
				if entry.BuyPrice < low.BuyPrice {
					low = entry
				}
			}
			if low.BuyPrice <= 0 {
				continue
			}
			ratio := high.BuyPrice / low.BuyPrice
			if ratio < threshold {
				continue
			}

			out = append(out, CollapsedPricing{
				CardID:     cardID,
				Conditions: k.condition,
				VendorName: k.vendor,
				High:       high.BuyPrice,
				Low:        low.BuyPrice,
				Ratio:      ratio,
				HighURL:    high.URL,
				LowURL:     low.URL,
				Count:      len(group),
			})
		}
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Ratio == out[j].Ratio {
			if out[i].CardID == out[j].CardID {
				return out[i].Conditions < out[j].Conditions
			}
			return out[i].CardID < out[j].CardID
		}
		return out[i].Ratio > out[j].Ratio
	})

	return out
}
