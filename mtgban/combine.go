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

func CombineInventories(sellers []Seller) (*CombineRoot, error) {
	root := &CombineRoot{
		Names:   []string{},
		Entries: map[string]map[string]CombineEntry{},
	}

	for _, seller := range sellers {
		sellerName := seller.(Scraper).Info().Name
		root.Names = append(root.Names, sellerName)

		inv, err := seller.Inventory()
		if err != nil {
			return nil, err
		}

		for card, entries := range inv {
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

	return root, nil
}

func CombineBuylists(vendors []Vendor, useCredit bool) (*CombineRoot, error) {
	root := &CombineRoot{
		Names:   []string{},
		Entries: map[string]map[string]CombineEntry{},
	}

	for _, vendor := range vendors {
		vendorName := vendor.(Scraper).Info().Name
		root.Names = append(root.Names, vendorName)

		bl, err := vendor.Buylist()
		if err != nil {
			return nil, err
		}

		for card, entries := range bl {
			_, found := root.Entries[card]
			if !found {
				root.Entries[card] = map[string]CombineEntry{}
			}

			// aka NM
			entry := entries[0]

			price := entry.BuyPrice
			if useCredit {
				price = entry.TradePrice
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

	return root, nil
}
