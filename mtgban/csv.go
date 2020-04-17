package mtgban

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"

	"github.com/kodabb/go-mtgban/mtgdb"
)

var (
	// The canonical header that will be present in all market files
	MarketHeader = []string{
		"Key", "Name", "Edition", "F/NF", "Conditions", "Price", "Quantity", "Seller",
	}
	// The canonical header that will be present in all inventory files
	InventoryHeader = []string{
		"Key", "Name", "Edition", "F/NF", "Conditions", "Price", "Quantity",
	}
	// The canonical header that will be present in all buylist files
	BuylistHeader = []string{
		"Key", "Name", "Edition", "F/NF", "Conditions", "Buy Price", "Trade Price", "Quantity", "Price Ratio", "Quantity Ratio",
	}

	ArbitHeader = []string{
		"Key", "Name", "Edition", "F/NF", "Conditions", "Sell Price", "Buy Price", "Trade Price", "Difference", "Spread", "Price Ratio", "Quantity Ratio",
	}
	MismatchHeader = []string{
		"Key", "Name", "Edition", "F/NF", "Conditions", "Price", "Difference", "Spread",
	}

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
	seller.name = "Base Seller"
	seller.shorthand = "BS"

	return &seller, nil
}

func NewVendorFromCSV(r io.Reader, grade map[string]float64) (Vendor, error) {
	buylist, err := LoadBuylistFromCSV(r)
	if err != nil {
		return nil, err
	}

	vendor := BaseVendor{}
	vendor.grade = grade
	vendor.buylist = buylist
	vendor.name = "Base Vendor"
	vendor.shorthand = "BV"

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

		foil := record[3] == "FOIL"
		price, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			return nil, fmt.Errorf("Error reading record: %v", err)
		}
		qty, err := strconv.Atoi(record[6])
		if err != nil {
			return nil, fmt.Errorf("Error reading record: %v", err)
		}

		sellerName := ""
		if len(record) > 7 {
			sellerName = record[7]
		}

		card := &mtgdb.Card{
			Id:      record[0],
			Name:    record[1],
			Edition: record[2],
			Foil:    foil,
		}

		entry := &InventoryEntry{
			Conditions: record[4],
			Price:      price,
			Quantity:   qty,
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

		foil := record[3] == "FOIL"
		buyPrice, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			return nil, fmt.Errorf("Error reading record %s: %v", record[5], err)
		}
		tradePrice, err := strconv.ParseFloat(record[6], 64)
		if err != nil {
			return nil, fmt.Errorf("Error reading record %s: %v", record[6], err)
		}
		qty, err := strconv.Atoi(record[7])
		if err != nil {
			return nil, fmt.Errorf("Error reading record %s: %v", record[7], err)
		}
		priceRatio, err := strconv.ParseFloat(record[8], 64)
		if err != nil {
			return nil, fmt.Errorf("Error reading record %s: %v", record[8], err)
		}
		qtyRatio, err := strconv.ParseFloat(record[9], 64)
		if err != nil {
			return nil, fmt.Errorf("Error reading record %s: %v", record[9], err)
		}

		card := &mtgdb.Card{
			Id:      record[0],
			Name:    record[1],
			Edition: record[2],
			Foil:    foil,
		}
		entry := &BuylistEntry{
			Conditions:    record[4],
			BuyPrice:      buyPrice,
			TradePrice:    tradePrice,
			Quantity:      qty,
			PriceRatio:    priceRatio,
			QuantityRatio: qtyRatio,
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
				entry.Conditions,
				fmt.Sprintf("%0.2f", entry.Price),
				fmt.Sprint(entry.Quantity),
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
			entry.Conditions,
			fmt.Sprintf("%0.2f", entry.BuyPrice),
			fmt.Sprintf("%0.2f", entry.TradePrice),
			fmt.Sprint(entry.Quantity),
			fmt.Sprintf("%0.2f%%", entry.PriceRatio),
			fmt.Sprintf("%0.2f%%", entry.QuantityRatio),
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

	err := csvWriter.Write(ArbitHeader)
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

		err = csvWriter.Write([]string{
			card.Id,
			card.Name,
			card.Edition,
			foil,
			inv.Conditions,
			fmt.Sprintf("%0.2f", inv.Price),
			fmt.Sprintf("%0.2f", bl.BuyPrice),
			fmt.Sprintf("%0.2f", bl.TradePrice),
			fmt.Sprintf("%0.2f", entry.Difference),
			fmt.Sprintf("%0.2f%%", entry.Spread),
			fmt.Sprintf("%0.2f%%", bl.PriceRatio),
			fmt.Sprintf("%0.2f%%", bl.QuantityRatio),
		})
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

	header := []string{
		"Id", "Card Name", "Edition", "F/NF",
	}
	header = append(header, root.Names...)
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
			card.Id, card.Name, card.Edition, foil,
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
