package mtgban

import (
	"sort"

	"golang.org/x/exp/maps"
)

// Separate a Market into multiple Seller objects
func Seller2Sellers(market Market) ([]Seller, error) {
	// Make sure inventory is loaded
	_, err := market.Inventory()
	if err != nil {
		return nil, err
	}

	// Retrieve the list of unique sellers, and create a single seller
	listSellers := market.MarketNames()
	sellers := make([]Seller, 0, len(listSellers))
	for _, sellerName := range listSellers {
		inventory, err := market.InventoryForSeller(sellerName)
		if err != nil {
			return nil, err
		}
		seller := &BaseSeller{}
		seller.inventory = inventory
		seller.info = market.Info()
		seller.info.Name = sellerName
		seller.info.Shorthand = sellerName
		seller.info.CustomFields = maps.Clone(market.Info().CustomFields)
		sellers = append(sellers, seller)
	}
	return sellers, nil
}

type MarketTotalsEntry struct {
	CardId string

	SingleListings  int
	TotalQuantities int

	Lowest  float64
	Average float64
	Spread  float64
}

type MarketTotalsOptions struct {
	FilterFunc func(cardId string) bool
}

func MarketTotals(opts *MarketTotalsOptions, market Market) (result []MarketTotalsEntry, err error) {
	inventory, err := market.Inventory()
	if err != nil {
		return nil, err
	}

	for cardId, entries := range inventory {
		if opts != nil && opts.FilterFunc != nil {
			if opts.FilterFunc(cardId) {
				continue
			}
		}

		qty := 0
		var prices []float64
		for _, entry := range entries {
			if entry.Conditions == "PO" {
				continue
			}

			qty += entry.Quantity
			prices = append(prices, entry.Price)
		}
		if len(prices) == 0 {
			continue
		}

		// Sort in increasing order, so that it's easier
		// to find the other properties
		sort.Float64s(prices)

		lowest := prices[0]
		if lowest == 0 {
			continue
		}
		spread := 100 * (prices[len(prices)-1] - lowest) / lowest
		average := average(prices)
		listings := len(prices)

		result = append(result, MarketTotalsEntry{
			CardId: cardId,

			SingleListings:  listings,
			TotalQuantities: qty,

			Lowest:  lowest,
			Average: average,
			Spread:  spread,
		})
	}

	return
}

func average(slice []float64) float64 {
	total := 0.0
	for _, v := range slice {
		total += v
	}
	return total / float64(len(slice))
}
