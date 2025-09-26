package mtgban

type CombineRoot struct {
	Names   []string
	Entries map[string]map[string]CombineEntry
}

type CombineEntry struct {
	ScraperName string
	Price       float64
	Ratio       float64
	Quantity    int
	URL         string
}

func CombineInventories(sellers []Seller) *CombineRoot {
	root := &CombineRoot{
		Names:   []string{},
		Entries: map[string]map[string]CombineEntry{},
	}

	for _, seller := range sellers {
		sellerName := seller.Info().Name
		root.Names = append(root.Names, sellerName)

		for card, entries := range seller.Inventory() {
			for _, entry := range entries {
				if entry.Conditions != "NM" {
					continue
				}
				if entry.Price == 0 {
					continue
				}

				_, found := root.Entries[card]
				if !found {
					root.Entries[card] = map[string]CombineEntry{}
				}

				price := entry.Price
				res := CombineEntry{
					ScraperName: sellerName,
					Price:       price,
					Quantity:    entry.Quantity,
					URL:         entry.URL,
				}

				root.Entries[card][sellerName] = res
			}
		}
	}

	return root
}

func CombineBuylists(vendors []Vendor, useCredit bool) *CombineRoot {
	root := &CombineRoot{
		Names:   []string{},
		Entries: map[string]map[string]CombineEntry{},
	}

	for _, vendor := range vendors {
		vendorName := vendor.Info().Name
		root.Names = append(root.Names, vendorName)

		for card, entries := range vendor.Buylist() {
			_, found := root.Entries[card]
			if !found {
				root.Entries[card] = map[string]CombineEntry{}
			}

			// aka NM
			entry := entries[0]

			price := entry.BuyPrice
			if useCredit {
				price *= vendor.Info().CreditMultiplier
			}

			res := CombineEntry{
				ScraperName: vendorName,
				Price:       price,
				Ratio:       entry.PriceRatio,
				Quantity:    entry.Quantity,
				URL:         entry.URL,
			}

			root.Entries[card][vendorName] = res
		}
	}

	return root
}
