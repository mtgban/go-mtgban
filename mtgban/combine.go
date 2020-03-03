package mtgban

import "math"

type CombineRoot struct {
	Names     []string
	Entries   map[Card][]CombineEntry
	BestOffer map[Card]CombineEntry
}

type CombineEntry struct {
	ScraperName string
	Price       float64
	Ratio       float64
	Quantity    int
	Notes       string
}

func CombineInventories(sellers []Seller) (*CombineRoot, error) {
	root := &CombineRoot{
		Names:     []string{},
		Entries:   map[Card][]CombineEntry{},
		BestOffer: map[Card]CombineEntry{},
	}

	result := map[Card]map[string]InventoryEntry{}

	for _, seller := range sellers {
		sellerName := seller.(Scraper).Info().Name
		root.Names = append(root.Names, sellerName)

		inv, err := seller.Inventory()
		if err != nil {
			return nil, err
		}

		for _, entries := range inv {
			for _, entry := range entries {
				if entry.Conditions != "NM" {
					continue
				}

				_, found := result[entry.Card]
				if !found {
					result[entry.Card] = map[string]InventoryEntry{}
				}
				result[entry.Card][sellerName] = entry
			}
		}
	}

	for card, entries := range result {
		_, found := root.Entries[card]
		if !found {
			root.Entries[card] = []CombineEntry{}
		}
		minPrice := math.MaxFloat64
		var bestEntry CombineEntry

		for _, sellerName := range root.Names {
			price := entries[sellerName].Price

			entry := CombineEntry{
				ScraperName: sellerName,
				Price:       price,
				Quantity:    entries[sellerName].Quantity,
				Notes:       entries[sellerName].Notes,
			}
			root.Entries[card] = append(root.Entries[card], entry)

			if price != 0 && minPrice > price {
				minPrice = price
				bestEntry = entry
			}
		}
		root.BestOffer[card] = bestEntry
	}

	return root, nil
}

func CombineBuylists(vendors []Vendor, useCredit bool) (*CombineRoot, error) {
	root := &CombineRoot{
		Names:     []string{},
		Entries:   map[Card][]CombineEntry{},
		BestOffer: map[Card]CombineEntry{},
	}

	result := map[Card]map[string]BuylistEntry{}

	for _, vendor := range vendors {
		vendorName := vendor.(Scraper).Info().Name
		root.Names = append(root.Names, vendorName)

		bl, err := vendor.Buylist()
		if err != nil {
			return nil, err
		}

		for _, entry := range bl {
			_, found := result[entry.Card]
			if !found {
				result[entry.Card] = map[string]BuylistEntry{}
			}
			result[entry.Card][vendorName] = entry
		}
	}

	for card, entries := range result {
		_, found := root.Entries[card]
		if !found {
			root.Entries[card] = []CombineEntry{}
		}
		maxPrice := 0.0
		var bestEntry CombineEntry

		for _, vendorName := range root.Names {
			price := entries[vendorName].BuyPrice
			if useCredit {
				price = entries[vendorName].TradePrice
			}

			entry := CombineEntry{
				ScraperName: vendorName,
				Price:       price,
				Ratio:       entries[vendorName].PriceRatio,
				Quantity:    entries[vendorName].Quantity,
				Notes:       entries[vendorName].Notes,
			}
			root.Entries[card] = append(root.Entries[card], entry)

			if maxPrice < price {
				maxPrice = price
				bestEntry = entry
			}
		}
		root.BestOffer[card] = bestEntry
	}

	return root, nil
}
