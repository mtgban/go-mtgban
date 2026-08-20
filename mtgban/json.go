package mtgban

import (
	"encoding/json"
	"io"
)

// scraperJSON is the on-disk shape every Write function emits and every Read
// function accepts. Both price sides are optional so one file can hold either
// or both.
type scraperJSON struct {
	Info      ScraperInfo     `json:"info"`
	Inventory InventoryRecord `json:"inventory,omitempty"`
	Buylist   BuylistRecord   `json:"buylist,omitempty"`
}

// WriteScraperToJSON writes whichever sides of a scraper exist: a Market gets
// both its inventory and its buylist, a plain Seller or Vendor only its own.
// The timestamp of a side that came back empty is dropped, so a reader can
// tell "not collected" from "collected and found nothing".
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

// WriteSellerToJSON writes the inventory side alone, in the same shape
// WriteScraperToJSON produces, so ReadSellerFromJSON reads either.
func WriteSellerToJSON(seller Seller, w io.Writer) error {
	var data scraperJSON

	data.Inventory = seller.Inventory()
	data.Info = seller.Info()
	data.Info.BuylistTimestamp = nil

	return json.NewEncoder(w).Encode(&data)
}

// WriteVendorToJSON writes the buylist side alone, in the same shape
// WriteScraperToJSON produces, so ReadVendorFromJSON reads either.
func WriteVendorToJSON(vendor Vendor, w io.Writer) error {
	var data scraperJSON

	data.Buylist = vendor.Buylist()
	data.Info = vendor.Info()
	data.Info.InventoryTimestamp = nil

	return json.NewEncoder(w).Encode(&data)
}

// ReadSellerFromJSON rebuilds a seller from what the Write functions emit.
// The result carries prices and the info it was written with, not the scraper
// that produced them: it never reaches the network and Load is a no-op.
func ReadSellerFromJSON(r io.Reader) (Seller, error) {
	var data scraperJSON

	err := json.NewDecoder(r).Decode(&data)
	if err != nil {
		return nil, err
	}

	return NewSellerFromInventory(data.Inventory, data.Info), nil
}

// ReadVendorFromJSON rebuilds a vendor from what the Write functions emit,
// with the same caveat as ReadSellerFromJSON: prices only, no scraper behind
// them.
func ReadVendorFromJSON(r io.Reader) (Vendor, error) {
	var data scraperJSON

	err := json.NewDecoder(r).Decode(&data)
	if err != nil {
		return nil, err
	}

	return NewVendorFromBuylist(data.Buylist, data.Info), nil
}
