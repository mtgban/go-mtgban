package mtgban

import (
	"encoding/json"
	"fmt"
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

// readScraperJSON decodes a dump one card at a time. Handing the whole
// file to Decode makes it buffer every byte before parsing any of it,
// which on a gigabyte-sized dump costs about twice the file in buffer
// growth alone and keeps the raw text resident alongside the prices it
// decodes into. Walking the object instead leaves the decoder holding
// one card's entries at a time.
func readScraperJSON(r io.Reader) (*scraperJSON, error) {
	var data scraperJSON
	dec := json.NewDecoder(r)

	// Opening brace of the dump
	_, err := dec.Token()
	if err != nil {
		return nil, err
	}

	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, ok := keyToken.(string)
		if !ok {
			return nil, fmt.Errorf("dump holds %v where a field name was due", keyToken)
		}

		switch key {
		case "info":
			err = dec.Decode(&data.Info)
		case "inventory":
			data.Inventory = InventoryRecord{}
			err = decodeRecord(dec, data.Inventory)
		case "buylist":
			data.Buylist = BuylistRecord{}
			err = decodeRecord(dec, data.Buylist)
		default:
			// Anything a later writer adds, read past and drop
			var skip json.RawMessage
			err = dec.Decode(&skip)
		}
		if err != nil {
			return nil, err
		}
	}

	// Closing brace of the dump
	_, err = dec.Token()
	if err != nil {
		return nil, err
	}
	return &data, nil
}

// decodeRecord fills one price side, decoding the entries of a card as
// they are reached rather than the side as a whole.
func decodeRecord[T any](dec *json.Decoder, out map[string][]T) error {
	// Opening brace of the record
	_, err := dec.Token()
	if err != nil {
		return err
	}

	for dec.More() {
		keyToken, err := dec.Token()
		if err != nil {
			return err
		}
		cardID, ok := keyToken.(string)
		if !ok {
			return fmt.Errorf("record holds %v where a card id was due", keyToken)
		}

		var entries []T
		err = dec.Decode(&entries)
		if err != nil {
			return err
		}
		out[cardID] = entries
	}

	// Closing brace of the record
	_, err = dec.Token()
	return err
}

// ReadSellerFromJSON rebuilds a seller from what the Write functions emit.
// The result carries prices and the info it was written with, not the scraper
// that produced them: it never reaches the network and Load is a no-op.
func ReadSellerFromJSON(r io.Reader) (Seller, error) {
	data, err := readScraperJSON(r)
	if err != nil {
		return nil, err
	}

	return NewSellerFromInventory(data.Inventory, data.Info), nil
}

// ReadVendorFromJSON rebuilds a vendor from what the Write functions emit,
// with the same caveat as ReadSellerFromJSON: prices only, no scraper behind
// them.
func ReadVendorFromJSON(r io.Reader) (Vendor, error) {
	data, err := readScraperJSON(r)
	if err != nil {
		return nil, err
	}

	return NewVendorFromBuylist(data.Buylist, data.Info), nil
}
