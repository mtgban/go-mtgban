package mtgban

func Seller2Sellers(market Market) ([]Seller, error) {
	inventory, err := market.Inventory()
	if err != nil {
		return nil, err
	}

	listSellers := map[string]bool{}
	for _, entries := range inventory {
		for _, entry := range entries {
			listSellers[entry.SellerName] = true
		}
	}

	sellers := make([]Seller, 0, len(listSellers))
	for sellerName, _ := range listSellers {
		inventory, err := market.InventoryForSeller(sellerName)
		if err != nil {
			return nil, err
		}
		seller := &BaseSeller{}
		seller.inventory = inventory
		seller.name = sellerName
		seller.shorthand = sellerName
		sellers = append(sellers, seller)
	}
	return sellers, nil
}
