package mtgban

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

var (
	// The canonical header that will be present in all inventory files
	InventoryHeader = []string{
		"Key", "Name", "Set", "F/NF", "Conditions", "Price", "Quantity",
	}
	// The canonical header that will be present in all buylist files
	BuylistHeader = []string{
		"Key", "Name", "Set", "F/NF", "Conditions", "Buy Price", "Trade Price", "Quantity", "Price Ratio", "Quantity Ratio",
	}

	ArbitHeader = []string{
		"Key", "Name", "Edition", "F/NF", "Conditions", "Sell Price", "Buy Price", "Trade Price", "Difference", "Spread", "Price Ratio", "Quantity Ratio",
	}
	MismatchHeader = []string{
		"Key", "Name", "Edition", "F/NF", "Conditions", "Price", "Difference", "Spread",
	}
)

func NewVendorFromCSV(r io.Reader, grade map[string]float64) (Vendor, error) {
	vendor := BaseBuylist{}
	vendor.buylist = BuylistRecord{}
	vendor.grade = grade

	buylist, err := LoadBuylistFromCSV(r)
	if err != nil {
		return nil, err
	}
	for _, entry := range buylist {
		err = BuylistAdd(vendor.buylist, entry)
		if err != nil {
			return nil, err
		}
	}

	return &vendor, nil
}

func NewSellerFromCSV(r io.Reader) (Seller, error) {
	seller := BaseInventory{}
	seller.inventory = InventoryRecord{}

	inventory, err := LoadInventoryFromCSV(r)
	if err != nil {
		return nil, err
	}

	for _, entry := range inventory {
		err = InventoryAdd(seller.inventory, entry)
		if err != nil {
			return nil, err
		}
	}

	return &seller, nil
}

func LoadInventoryFromCSV(r io.Reader) ([]InventoryEntry, error) {
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
		for i, tag := range InventoryHeader {
			if tag != first[i] {
				okHeader = false
				break
			}
		}
	}
	if !okHeader {
		return nil, fmt.Errorf("Malformed inventory file")
	}

	inventory := []InventoryEntry{}
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

		card := InventoryEntry{
			Card: Card{
				Id:   record[0],
				Name: record[1],
				Set:  record[2],
				Foil: foil,
			},
			Conditions: record[4],
			Price:      price,
			Quantity:   qty,
		}

		inventory = append(inventory, card)
	}

	return inventory, nil
}

func LoadBuylistFromCSV(r io.Reader) ([]BuylistEntry, error) {
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

	buylist := []BuylistEntry{}
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

		card := BuylistEntry{
			Card: Card{
				Id:   record[0],
				Name: record[1],
				Set:  record[2],
				Foil: foil,
			},
			Conditions:    record[4],
			BuyPrice:      buyPrice,
			TradePrice:    tradePrice,
			Quantity:      qty,
			PriceRatio:    priceRatio,
			QuantityRatio: qtyRatio,
		}

		buylist = append(buylist, card)
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

	err = csvWriter.Write(InventoryHeader)
	if err != nil {
		return err
	}

	for id, cards := range inventory {
		for _, card := range cards {
			foil := ""
			if card.Foil {
				foil = "FOIL"
			}

			err = csvWriter.Write([]string{
				id,
				card.Name,
				card.Set,
				foil,
				card.Conditions,
				fmt.Sprintf("%0.2f", card.Price),
				fmt.Sprint(card.Quantity),
			})
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

	for id, card := range buylist {
		foil := ""
		if card.Foil {
			foil = "FOIL"
		}

		err = csvWriter.Write([]string{
			id,
			card.Name,
			card.Set,
			foil,
			card.Conditions,
			fmt.Sprintf("%0.2f", card.BuyPrice),
			fmt.Sprintf("%0.2f", card.TradePrice),
			fmt.Sprint(card.Quantity),
			fmt.Sprintf("%0.2f%%", card.PriceRatio),
			fmt.Sprintf("%0.2f%%", card.QuantityRatio),
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
		card := bl.Card
		foil := ""
		if card.Foil {
			foil = "FOIL"
		}

		err = csvWriter.Write([]string{
			card.Id,
			card.Name,
			card.Set,
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
		card := inv.Card
		foil := ""
		if card.Foil {
			foil = "FOIL"
		}

		err = csvWriter.Write([]string{
			card.Id,
			card.Name,
			card.Set,
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
	header = append(header, "Best Offer")
	err := csvWriter.Write(header)
	if err != nil {
		return err
	}

	for card, entries := range root.Entries {
		foil := ""
		if card.Foil {
			foil = "FOIL"
		}

		out := []string{card.Id, card.Name, card.Set, foil}
		for _, entry := range entries {
			out = append(out, fmt.Sprintf("%0.2f", entry.Price))
		}
		out = append(out, root.BestOffer[card].ScraperName)

		err = csvWriter.Write(out)
		if err != nil {
			return err
		}

		csvWriter.Flush()
	}

	return nil
}
