package mtgban

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/kodabb/go-mtgban/mtgdb"
)

var (
	// The base Card fields for the canonical headers
	CardHeader = []string{
		"Key", "Name", "Edition", "F/NF", "Number", "Rarity",
	}

	// The canonical header that will be present in all inventory files
	InventoryHeader = append(CardHeader, "Conditions", "Price", "Quantity", "URL")

	// The canonical header that will be present in all market files
	MarketHeader = append(InventoryHeader, "Seller")

	// The canonical header that will be present in all buylist files
	BuylistHeader = append(CardHeader, "Buy Price", "Trade Price", "Quantity", "Price Ratio", "URL")

	ArbitHeader = append(CardHeader, "Conditions", "Available", "Sell Price", "Buy Price", "Trade Price", "Difference", "Spread", "Abs Difference", "Price Ratio")

	MismatchHeader = append(CardHeader, "Conditions", "Price", "Difference", "Spread")

	MultiArbitHeader = []string{
		"Seller", "Cards", "Listings", "Total Prices", "Total Buylist", "Difference", "Spread",
	}
)

func NewSellerFromCSV(r io.Reader) (Seller, error) {
	inventory, err := LoadInventoryFromCSV(r)
	if err != nil {
		return nil, err
	}

	seller := BaseSeller{}
	seller.inventory = inventory
	seller.info.Name = "Base Seller"
	seller.info.Shorthand = "BS"

	return &seller, nil
}

func NewVendorFromCSV(r io.Reader) (Vendor, error) {
	buylist, err := LoadBuylistFromCSV(r)
	if err != nil {
		return nil, err
	}

	vendor := BaseVendor{}
	vendor.buylist = buylist
	vendor.info.Name = "Base Vendor"
	vendor.info.Shorthand = "BV"
	vendor.info.Grading = DefaultGrading

	return &vendor, nil
}

func LoadInventoryFromCSV(r io.Reader) (InventoryRecord, error) {
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
			return nil, fmt.Errorf("Error reading record: %v", err)
		}

		index := 3
		foil := record[index] == "FOIL"
		index++
		number := record[index]
		index++
		rarity := record[index]
		index++
		conditions := record[index]
		index++
		price, err := strconv.ParseFloat(record[index], 64)
		if err != nil {
			return nil, fmt.Errorf("Error reading record: %v", err)
		}
		index++
		qty, err := strconv.Atoi(record[index])
		if err != nil {
			return nil, fmt.Errorf("Error reading record: %v", err)
		}
		index++

		URL := record[index]
		index++

		sellerName := ""
		if len(record) > index {
			sellerName = record[index]
			index++
		}

		card := &mtgdb.Card{
			Id:      record[0],
			Name:    record[1],
			Edition: record[2],
			Foil:    foil,
			Number:  number,
			Rarity:  rarity,
		}

		entry := &InventoryEntry{
			Conditions: conditions,
			Price:      price,
			Quantity:   qty,
			URL:        URL,
			SellerName: sellerName,
		}

		inventory.Add(card, entry)
	}

	return inventory, nil
}

func LoadBuylistFromCSV(r io.Reader) (BuylistRecord, error) {
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
			return nil, fmt.Errorf("Error reading record: %v", err)
		}

		index := 3
		foil := record[index] == "FOIL"
		index++
		number := record[index]
		index++
		rarity := record[index]
		index++
		buyPrice, err := strconv.ParseFloat(record[index], 64)
		if err != nil {
			return nil, fmt.Errorf("Error reading record %s: %v", record[index], err)
		}
		index++
		tradePrice, err := strconv.ParseFloat(record[index], 64)
		if err != nil {
			return nil, fmt.Errorf("Error reading record %s: %v", record[index], err)
		}
		index++
		qty, err := strconv.Atoi(record[index])
		if err != nil {
			return nil, fmt.Errorf("Error reading record %s: %v", record[index], err)
		}
		index++
		priceRatio, err := strconv.ParseFloat(strings.TrimSuffix(record[index], "%"), 64)
		if err != nil {
			return nil, fmt.Errorf("Error reading record %s: %v", record[index], err)
		}
		index++

		URL := record[index]
		index++

		card := &mtgdb.Card{
			Id:      record[0],
			Name:    record[1],
			Edition: record[2],
			Foil:    foil,
			Number:  number,
			Rarity:  rarity,
		}
		entry := &BuylistEntry{
			BuyPrice:   buyPrice,
			TradePrice: tradePrice,
			Quantity:   qty,
			PriceRatio: priceRatio,
			URL:        URL,
		}

		buylist.Add(card, entry)
	}

	return buylist, nil
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

	for card, entries := range inventory {
		for _, entry := range entries {
			foil := ""
			if card.Foil {
				foil = "FOIL"
			}

			record := []string{
				card.Id,
				card.Name,
				card.Edition,
				foil,
				card.Number,
				card.Rarity,
				entry.Conditions,
				fmt.Sprintf("%0.2f", entry.Price),
				fmt.Sprint(entry.Quantity),
				entry.URL,
			}
			if isMarket {
				record = append(record, entry.SellerName)
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

	for card, entry := range buylist {
		foil := ""
		if card.Foil {
			foil = "FOIL"
		}

		err = csvWriter.Write([]string{
			card.Id,
			card.Name,
			card.Edition,
			foil,
			card.Number,
			card.Rarity,
			fmt.Sprintf("%0.2f", entry.BuyPrice),
			fmt.Sprintf("%0.2f", entry.TradePrice),
			fmt.Sprint(entry.Quantity),
			fmt.Sprintf("%0.2f%%", entry.PriceRatio),
			entry.URL,
		})
		if err != nil {
			return err
		}
		csvWriter.Flush()
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
		hasExtraSeller = true
	}
	err := csvWriter.Write(header)
	if err != nil {
		return err
	}

	for _, entry := range arbitrage {
		bl := entry.BuylistEntry
		inv := entry.InventoryEntry
		card := entry.Card
		foil := ""
		if card.Foil {
			foil = "FOIL"
		}

		record := []string{
			card.Id,
			card.Name,
			card.Edition,
			foil,
			card.Number,
			card.Rarity,
			inv.Conditions,
			fmt.Sprintf("%d", inv.Quantity),
			fmt.Sprintf("%0.2f", inv.Price),
			fmt.Sprintf("%0.2f", bl.BuyPrice),
			fmt.Sprintf("%0.2f", bl.TradePrice),
			fmt.Sprintf("%0.2f", entry.Difference),
			fmt.Sprintf("%0.2f%%", entry.Spread),
			fmt.Sprintf("%0.2f", entry.AbsoluteDifference),
			fmt.Sprintf("%0.2f%%", bl.PriceRatio),
		}
		if hasExtraSeller {
			record = append(record, inv.SellerName)
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

	err := csvWriter.Write(MismatchHeader)
	if err != nil {
		return err
	}

	for _, entry := range mismatch {
		inv := entry.InventoryEntry
		card := entry.Card
		foil := ""
		if card.Foil {
			foil = "FOIL"
		}

		err = csvWriter.Write([]string{
			card.Id,
			card.Name,
			card.Edition,
			foil,
			card.Number,
			card.Rarity,
			inv.Conditions,
			fmt.Sprintf("%0.2f", inv.Price),
			fmt.Sprintf("%0.2f", entry.Difference),
			fmt.Sprintf("%0.2f%%", entry.Spread),
		})
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

	for card, entries := range root.Entries {
		foil := ""
		if card.Foil {
			foil = "FOIL"
		}

		out := []string{
			card.Id,
			card.Name,
			card.Edition,
			foil,
			card.Number,
			card.Rarity,
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
			fmt.Sprintf("%0.2f%%", entry.Spread),
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
		hasExtraSeller = true
	}
	err := csvWriter.Write(header)
	if err != nil {
		return err
	}

	for _, entry := range penny {
		card := entry.Card
		inv := entry.InventoryEntry
		foil := ""
		if card.Foil {
			foil = "FOIL"
		}

		record := []string{
			card.Id,
			card.Name,
			card.Edition,
			foil,
			card.Number,
			card.Rarity,
			inv.Conditions,
			fmt.Sprintf("%0.2f", inv.Price),
			fmt.Sprintf("%d", inv.Quantity),
		}
		if hasExtraSeller {
			record = append(record, inv.SellerName)
		}
		err = csvWriter.Write(record)
		if err != nil {
			return err
		}

		csvWriter.Flush()
	}

	return nil
}
