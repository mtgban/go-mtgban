package mtgban

import (
	"encoding/json"
	"io"
)

type scraperJSON struct {
	Info      ScraperInfo     `json:"info"`
	Inventory InventoryRecord `json:"inventory,omitempty"`
	Buylist   BuylistRecord   `json:"buylist,omitempty"`
}

func WriteScraperToJSON(scraper Scraper, w io.Writer) error {
	var data scraperJSON

	seller, isSeller := scraper.(Seller)
	vendor, isVendor := scraper.(Vendor)
	if isSeller {
		data.Inventory = seller.Inventory()
	}
	if isVendor {
		data.Buylist = vendor.Buylist()
	}
	data.Info = scraper.Info()

	if len(data.Inventory) == 0 {
		data.Info.InventoryTimestamp = nil
	}
	if len(data.Buylist) == 0 {
		data.Info.BuylistTimestamp = nil
	}

	return json.NewEncoder(w).Encode(&data)
}

func WriteSellerToJSON(seller Seller, w io.Writer) error {
	var data scraperJSON

	data.Inventory = seller.Inventory()
	data.Info = seller.Info()
	data.Info.BuylistTimestamp = nil

	return json.NewEncoder(w).Encode(&data)
}

func WriteVendorToJSON(vendor Vendor, w io.Writer) error {
	var data scraperJSON

	data.Buylist = vendor.Buylist()
	data.Info = vendor.Info()
	data.Info.InventoryTimestamp = nil

	return json.NewEncoder(w).Encode(&data)
}

func ReadSellerFromJSON(r io.Reader) (Seller, error) {
	var data scraperJSON

	err := json.NewDecoder(r).Decode(&data)
	if err != nil {
		return nil, err
	}

	return NewSellerFromInventory(data.Inventory, data.Info), nil
}

func ReadVendorFromJSON(r io.Reader) (Vendor, error) {
	var data scraperJSON

	err := json.NewDecoder(r).Decode(&data)
	if err != nil {
		return nil, err
	}

	return NewVendorFromBuylist(data.Buylist, data.Info), nil
}
