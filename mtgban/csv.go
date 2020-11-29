package mtgban

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

var (
	// The base Card fields for the canonical headers
	CardHeader = []string{
		"Key", "Name", "Edition", "F/NF", "Number", "Rarity",
	}

	// The canonical header that will be present in all inventory files
	InventoryHeader = append(CardHeader, "Conditions", "Price", "Quantity", "URL")

	// The canonical header that will be present in all market files
	MarketHeader = append(InventoryHeader, "Seller", "Bundle")

	// The canonical header that will be present in all buylist files
	BuylistHeader = append(CardHeader, "Buy Price", "Trade Price", "Quantity", "Price Ratio", "URL", "Conditions", "Vendor")

	ArbitHeader = append(CardHeader, "Conditions", "Available", "Sell Price", "Buy Price", "Trade Price", "Difference", "Spread", "Abs Difference", "Price Ratio")

	MismatchHeader = append(CardHeader, "Conditions", "Price", "Reference", "Difference", "Spread")

	MultiArbitHeader = []string{
		"Seller", "Cards", "Listings", "Total Prices", "Total Buylist", "Difference", "Spread",
	}

	MarketTotalsHeader = append(CardHeader, "Listings", "Total Quantity", "Lowest Price", "Average", "Spread")
)

func LoadInventoryFromCSV(r io.Reader, flags ...bool) (InventoryRecord, error) {
	strict := true
	if len(flags) > 0 {
		strict = flags[0]
	}
	csvReader := csv.NewReader(r)
	first, err := csvReader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("Empty input file")
	}
	if err != nil {
		return nil, fmt.Errorf("Error reading header: %v", err)
	}

	okHeader := true
	if len(first) < len(InventoryHeader) {
		okHeader = false
	} else {
		header := InventoryHeader
		if first[len(first)-1] == MarketHeader[len(MarketHeader)-1] {
			header = MarketHeader
		}
		for i, tag := range header {
			if tag != first[i] {
				okHeader = false
				break
			}
		}
	}
	if !okHeader {
		return nil, fmt.Errorf("Malformed inventory file")
	}

	inventory := InventoryRecord{}
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if strict {
				return nil, fmt.Errorf("Error reading record: %v", err)
			}
			continue
		}

		index := len(CardHeader)
		cardId := record[0]
		_, err = mtgmatcher.GetUUID(cardId)
		if err != nil {
			if strict {
				return nil, fmt.Errorf("Error reading record: %v (%v)", err, record)
			}
			continue
		}

		conditions := record[index]
		index++
		price, err := strconv.ParseFloat(record[index], 64)
		if err != nil {
			if strict {
				return nil, fmt.Errorf("Error reading record: %v", err)
			}
			continue
		}
		index++
		qty, err := strconv.Atoi(record[index])
		if err != nil {
			if strict {
				return nil, fmt.Errorf("Error reading record: %v", err)
			}
			continue
		}
		index++

		URL := record[index]
		index++

		sellerName := ""
		if len(record) > index {
			sellerName = record[index]
			index++
		}
		bundle := false
		if len(record) > index {
			bundle = record[index] == "Y"
			index++
		}

		entry := &InventoryEntry{
			Conditions: conditions,
			Price:      price,
			Quantity:   qty,
			URL:        URL,
			SellerName: sellerName,
			Bundle:     bundle,
		}

		inventory.Add(cardId, entry)
	}

	return inventory, nil
}

func LoadBuylistFromCSV(r io.Reader, flags ...bool) (BuylistRecord, error) {
	strict := true
	if len(flags) > 0 {
		strict = flags[0]
	}
	csvReader := csv.NewReader(r)
	first, err := csvReader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("Empty input file")
	}
	if err != nil {
		return nil, fmt.Errorf("Error reading header: %v", err)
	}

	okHeader := true
	if len(first) < len(BuylistHeader) {
		okHeader = false
	} else {
		for i, tag := range BuylistHeader {
			if tag != first[i] {
				okHeader = false
				break
			}
		}
	}
	if !okHeader {
		return nil, fmt.Errorf("Malformed buylist file")
	}

	buylist := BuylistRecord{}
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if strict {
				return nil, fmt.Errorf("Error reading record: %v", err)
			}
			continue
		}

		index := len(CardHeader)
		cardId := record[0]
		_, err = mtgmatcher.GetUUID(cardId)
		if err != nil {
			if strict {
				return nil, fmt.Errorf("Error reading record: %v (%v)", err, record)
			}
			continue
		}

		buyPrice, err := strconv.ParseFloat(record[index], 64)
		if err != nil {
			if strict {
				return nil, fmt.Errorf("Error reading record %s: %v", record[index], err)
			}
			continue
		}
		index++
		tradePrice, err := strconv.ParseFloat(record[index], 64)
		if err != nil {
			if strict {
				return nil, fmt.Errorf("Error reading record %s: %v", record[index], err)
			}
			continue
		}
		index++
		qty, err := strconv.Atoi(record[index])
		if err != nil {
			if strict {
				return nil, fmt.Errorf("Error reading record %s: %v", record[index], err)
			}
			continue
		}
		index++
		priceRatio, err := strconv.ParseFloat(strings.TrimSuffix(record[index], "%"), 64)
		if err != nil {
			if strict {
				return nil, fmt.Errorf("Error reading record %s: %v", record[index], err)
			}
			continue
		}
		index++

		URL := record[index]
		index++

		cond := record[index]
		index++

		vendorName := ""
		if len(record) > index {
			vendorName = record[index]
			index++
		}

		entry := &BuylistEntry{
			Conditions: cond,
			BuyPrice:   buyPrice,
			TradePrice: tradePrice,
			Quantity:   qty,
			PriceRatio: priceRatio,
			URL:        URL,
			VendorName: vendorName,
		}

		buylist.Add(cardId, entry)
	}

	return buylist, nil
}

func cardId2record(cardId string) ([]string, error) {
	co, err := mtgmatcher.GetUUID(cardId)
	if err != nil {
		return nil, err
	}

	foil := ""
	if co.Foil {
		foil = "FOIL"
	}

	record := []string{
		cardId,
		co.Card.Name,
		co.Edition,
		foil,
		co.Card.Number,
		strings.ToUpper(string(co.Card.Rarity[0])),
	}
	return record, nil
}

func WriteInventoryToCSV(seller Seller, w io.Writer) error {
	inventory, err := seller.Inventory()
	if err != nil {
		return err
	}
	if len(inventory) == 0 {
		return nil
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	header := InventoryHeader
	_, isMarket := seller.(Scraper).(Market)
	if isMarket {
		header = MarketHeader
	}

	err = csvWriter.Write(header)
	if err != nil {
		return err
	}

	for cardId, entries := range inventory {
		cardHeader, err := cardId2record(cardId)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			record := append(cardHeader,
				entry.Conditions,
				fmt.Sprintf("%0.2f", entry.Price),
				fmt.Sprint(entry.Quantity),
				entry.URL,
			)
			if isMarket {
				record = append(record, entry.SellerName)
				bundle := ""
				if entry.Bundle {
					bundle = "Y"
				}
				record = append(record, bundle)
			}

			err = csvWriter.Write(record)
			if err != nil {
				return err
			}
		}
		csvWriter.Flush()
	}

	return nil
}

func WriteBuylistToCSV(vendor Vendor, w io.Writer) error {
	buylist, err := vendor.Buylist()
	if err != nil {
		return err
	}
	if len(buylist) == 0 {
		return nil
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	err = csvWriter.Write(BuylistHeader)
	if err != nil {
		return err
	}

	for cardId, entries := range buylist {
		for _, entry := range entries {
			record, err := cardId2record(cardId)
			if err != nil {
				continue
			}

			record = append(record,
				fmt.Sprintf("%0.2f", entry.BuyPrice),
				fmt.Sprintf("%0.2f", entry.TradePrice),
				fmt.Sprint(entry.Quantity),
				fmt.Sprintf("%0.2f", entry.PriceRatio),
				entry.URL,
				entry.Conditions,
				entry.VendorName,
			)

			err = csvWriter.Write(record)
			if err != nil {
				return err
			}
			csvWriter.Flush()
		}
	}

	return nil
}

func WriteArbitrageToCSV(arbitrage []ArbitEntry, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	hasExtraSeller := false
	header := ArbitHeader
	if len(arbitrage) > 0 && arbitrage[0].InventoryEntry.SellerName != "" {
		header = append(header, "Seller")
		header = append(header, "Bundle")
		hasExtraSeller = true
	}
	err := csvWriter.Write(header)
	if err != nil {
		return err
	}

	for _, entry := range arbitrage {
		bl := entry.BuylistEntry
		inv := entry.InventoryEntry

		record, err := cardId2record(entry.CardId)
		if err != nil {
			continue
		}

		record = append(record,
			inv.Conditions,
			fmt.Sprintf("%d", inv.Quantity),
			fmt.Sprintf("%0.2f", inv.Price),
			fmt.Sprintf("%0.2f", bl.BuyPrice),
			fmt.Sprintf("%0.2f", bl.TradePrice),
			fmt.Sprintf("%0.2f", entry.Difference),
			fmt.Sprintf("%0.2f", entry.Spread),
			fmt.Sprintf("%0.2f", entry.AbsoluteDifference),
			fmt.Sprintf("%0.2f", bl.PriceRatio),
		)
		if hasExtraSeller {
			record = append(record, inv.SellerName)
			bundle := ""
			if inv.Bundle {
				bundle = "Y"
			}
			record = append(record, bundle)
		}
		err = csvWriter.Write(record)
		if err != nil {
			return err
		}

		csvWriter.Flush()
	}

	return nil
}

func WriteMismatchToCSV(mismatch []MismatchEntry, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	hasExtraSeller := false
	header := MismatchHeader
	if len(mismatch) > 0 && mismatch[0].Inventory.SellerName != "" {
		header = append(header, "Seller")
		header = append(header, "Bundle")
		hasExtraSeller = true
	}

	err := csvWriter.Write(header)
	if err != nil {
		return err
	}

	for _, entry := range mismatch {
		inv := entry.Inventory
		ref := entry.Reference

		record, err := cardId2record(entry.CardId)
		if err != nil {
			continue
		}

		record = append(record,
			inv.Conditions,
			fmt.Sprintf("%0.2f", inv.Price),
			fmt.Sprintf("%0.2f", ref.Price),
			fmt.Sprintf("%0.2f", entry.Difference),
			fmt.Sprintf("%0.2f", entry.Spread),
		)
		if hasExtraSeller {
			record = append(record, inv.SellerName)
			bundle := ""
			if inv.Bundle {
				bundle = "Y"
			}
			record = append(record, bundle)
		}
		err = csvWriter.Write(record)
		if err != nil {
			return err
		}

		csvWriter.Flush()
	}

	return nil
}

func WriteCombineToCSV(root *CombineRoot, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	header := append(CardHeader, root.Names...)
	err := csvWriter.Write(header)
	if err != nil {
		return err
	}

	for cardId, entries := range root.Entries {
		out, err := cardId2record(cardId)
		if err != nil {
			continue
		}
		for _, vendorName := range root.Names {
			out = append(out, fmt.Sprintf("%0.2f", entries[vendorName].Price))
		}

		err = csvWriter.Write(out)
		if err != nil {
			return err
		}

		csvWriter.Flush()
	}

	return nil
}

func WriteMultiArbitrageToCSV(multi []MultiArbitEntry, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	err := csvWriter.Write(MultiArbitHeader)
	if err != nil {
		return err
	}

	for _, entry := range multi {
		err = csvWriter.Write([]string{
			entry.SellerName,
			fmt.Sprintf("%d", entry.Quantity),
			fmt.Sprintf("%d", len(entry.Entries)),
			fmt.Sprintf("%0.2f", entry.Price),
			fmt.Sprintf("%0.2f", entry.BuylistPrice),
			fmt.Sprintf("%0.2f", entry.Difference),
			fmt.Sprintf("%0.2f", entry.Spread),
		})
		if err != nil {
			return err
		}

		csvWriter.Flush()
	}

	return nil
}

func WritePennyToCSV(penny []PennystockEntry, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	hasExtraSeller := false
	header := InventoryHeader
	if len(penny) > 0 && penny[0].InventoryEntry.SellerName != "" {
		header = append(header, "Seller")
		header = append(header, "Bundle")
		hasExtraSeller = true
	}
	err := csvWriter.Write(header)
	if err != nil {
		return err
	}

	for _, entry := range penny {
		inv := entry.InventoryEntry

		record, err := cardId2record(entry.CardId)
		if err != nil {
			continue
		}

		record = append(record,
			inv.Conditions,
			fmt.Sprintf("%0.2f", inv.Price),
			fmt.Sprintf("%d", inv.Quantity),
			"",
		)
		if hasExtraSeller {
			record = append(record, inv.SellerName)
			bundle := ""
			if inv.Bundle {
				bundle = "Y"
			}
			record = append(record, bundle)
		}
		err = csvWriter.Write(record)
		if err != nil {
			return err
		}

		csvWriter.Flush()
	}

	return nil
}

func WriteTotalsToCSV(totals []MarketTotalsEntry, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	header := MarketTotalsHeader
	err := csvWriter.Write(header)
	if err != nil {
		return err
	}

	for _, entry := range totals {
		record, err := cardId2record(entry.CardId)
		if err != nil {
			continue
		}

		record = append(record,
			fmt.Sprintf("%d", entry.SingleListings),
			fmt.Sprintf("%d", entry.TotalQuantities),
			fmt.Sprintf("%0.2f", entry.Lowest),
			fmt.Sprintf("%0.2f", entry.Average),
			fmt.Sprintf("%0.2f", entry.Spread),
		)
		err = csvWriter.Write(record)
		if err != nil {
			return err
		}

		csvWriter.Flush()
	}

	return nil
}
