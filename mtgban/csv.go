package mtgban

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var (
	// CardHeader is the set of card fields every other header builds on
	CardHeader = []string{
		"UUID", "Name", "Edition", "Finish", "Number", "Rarity",
	}

	// InventoryHeader is the header written to every inventory file
	InventoryHeader = append(CardHeader, "Conditions", "Price", "Quantity", "URL")

	// MarketHeader is the header written to a market's per-seller files
	MarketHeader = append(InventoryHeader, "Seller", "Bundle")

	// CartHeader is MarketHeader plus the ids a carter needs to place an order
	CartHeader = append(MarketHeader, "Original Id", "Instance Id")

	// BuylistHeader is the header written to every buylist file
	BuylistHeader = append(CardHeader, "Conditions", "Buy Price", "Trade Price", "Quantity", "Price Ratio", "URL", "Vendor")

	// ArbitHeader is the header for the arbitrage reports, carrying both
	// prices and the numbers derived from them
	ArbitHeader = append(CardHeader, "Conditions", "Available", "Sell Price", "Buy Price", "Difference", "Spread", "Abs Difference", "Profitability", "Buy Link", "Sell Link")

	// MismatchHeader is the header for the mismatch reports
	MismatchHeader = append(CardHeader, "Conditions", "Price", "Reference", "Difference", "Spread")
)

func record2entry(record []string) (*InventoryEntry, error) {
	index := len(CardHeader)
	cardID := record[0]
	_, err := mtgmatcher.GetUUID(cardID)
	if err != nil && !strings.Contains(cardID, "|") {
		return nil, fmt.Errorf("error reading record: %v (%v)", err, record)
	}

	conditions := record[index]
	index++
	price, err := strconv.ParseFloat(record[index], 64)
	if err != nil {
		return nil, fmt.Errorf("error reading record: %v", err)
	}
	index++
	qty, err := strconv.Atoi(record[index])
	if err != nil {
		return nil, fmt.Errorf("error reading record: %v", err)
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
	ogID := ""
	if len(record) > index {
		ogID = record[index]
		index++
	}
	instanceID := ""
	if len(record) > index {
		instanceID = record[index]
		index++
	}

	return &InventoryEntry{
		Conditions: conditions,
		Price:      price,
		Quantity:   qty,
		URL:        URL,
		SellerName: sellerName,
		Bundle:     bundle,
		OriginalId: ogID,
		InstanceId: instanceID,
	}, nil
}

// LoadInventoryFromCSV reads what WriteInventoryToCSV produced. It stops at
// the first unreadable row unless the optional flag is false, in which case
// bad rows are skipped and whatever parsed is returned; use that for a file
// from somewhere else, not for one this package wrote.
func LoadInventoryFromCSV(r io.Reader, flags ...bool) (InventoryRecord, error) {
	strict := true
	if len(flags) > 0 {
		strict = flags[0]
	}
	csvReader := csv.NewReader(r)
	csvReader.ReuseRecord = true

	first, err := csvReader.Read()
	if err == io.EOF {
		return InventoryRecord{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading header: %v", err)
	}

	okHeader := true
	if len(first) < len(InventoryHeader) {
		okHeader = false
	} else {
		// Assume a normal header, then check what's the last element,
		// and adjust accordingly to what is detected
		header := InventoryHeader
		if first[len(first)-1] == MarketHeader[len(MarketHeader)-1] {
			header = MarketHeader
		} else if first[len(first)-1] == CartHeader[len(CartHeader)-1] {
			header = CartHeader
		}
		for i, tag := range header {
			if tag != first[i] {
				okHeader = false
				break
			}
		}
	}
	if !okHeader {
		return nil, errors.New("malformed inventory file")
	}

	inventory := InventoryRecord{}
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if strict {
				return nil, fmt.Errorf("error reading record: %v", err)
			}
			continue
		}

		entry, err := record2entry(record)
		if err != nil {
			if strict {
				return nil, err
			}
			continue
		}

		inventory.Add(record[0], entry)
	}

	return inventory, nil
}

// LoadBuylistFromCSV reads what WriteBuylistToCSV produced, with the same
// optional leniency as LoadInventoryFromCSV.
func LoadBuylistFromCSV(r io.Reader, flags ...bool) (BuylistRecord, error) {
	strict := true
	if len(flags) > 0 {
		strict = flags[0]
	}
	csvReader := csv.NewReader(r)
	csvReader.ReuseRecord = true

	first, err := csvReader.Read()
	if err == io.EOF {
		return BuylistRecord{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error reading header: %v", err)
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
		return nil, errors.New("malformed buylist file")
	}

	buylist := BuylistRecord{}
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			if strict {
				return nil, fmt.Errorf("error reading record: %v", err)
			}
			continue
		}

		index := len(CardHeader)
		cardID := record[0]
		_, err = mtgmatcher.GetUUID(cardID)
		if err != nil {
			if strict {
				return nil, fmt.Errorf("error reading record: %v (%v)", err, record)
			}
			continue
		}

		cond := record[index]
		index++

		buyPrice, err := strconv.ParseFloat(record[index], 64)
		if err != nil {
			if strict {
				return nil, fmt.Errorf("error reading record %s: %v", record[index], err)
			}
			continue
		}
		index++

		// Skip TradePrice
		index++

		qty, err := strconv.Atoi(record[index])
		if err != nil {
			if strict {
				return nil, fmt.Errorf("error reading record %s: %v", record[index], err)
			}
			continue
		}
		index++
		priceRatio, err := strconv.ParseFloat(strings.TrimSuffix(record[index], "%"), 64)
		if err != nil {
			if strict {
				return nil, fmt.Errorf("error reading record %s: %v", record[index], err)
			}
			continue
		}
		index++

		URL := record[index]
		index++

		vendorName := ""
		if len(record) > index {
			vendorName = record[index]
			index++
		}

		entry := &BuylistEntry{
			Conditions: cond,
			BuyPrice:   buyPrice,
			Quantity:   qty,
			PriceRatio: priceRatio,
			URL:        URL,
			VendorName: vendorName,
		}

		buylist.Add(cardID, entry)
	}

	return buylist, nil
}

func cardID2record(cardID string) ([]string, error) {
	if strings.Contains(cardID, "|") {
		fields := strings.Split(cardID, "|")
		if len(fields) != 4 {
			return nil, fmt.Errorf("unsupported id format %s", cardID)
		}
		record := []string{
			cardID,
			fields[1],
			fields[2],
			fields[3],
			"",
			"",
		}
		return record, nil
	}

	co, err := mtgmatcher.GetUUID(cardID)
	if err != nil {
		return nil, err
	}

	finish := "nonfoil"
	if co.Sealed {
		finish = "sealed"
	} else if co.Foil {
		finish = "foil"
	} else if co.Etched {
		finish = "etched"
	}

	record := []string{
		cardID,
		co.Card.Name,
		co.Edition,
		finish,
		co.Card.Number,
		co.Card.Rarity,
	}
	return record, nil
}

// WriteInventoryToCSV writes an inventory under CardHeader, or under the
// Market headers when the entries name a seller.
func WriteInventoryToCSV(inventory InventoryRecord, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	header := InventoryHeader
	for _, entries := range inventory {
		if entries[0].SellerName != "" {
			header = MarketHeader
		}
		if entries[0].OriginalId != "" || entries[0].InstanceId != "" {
			header = CartHeader
		}
		break
	}

	err := csvWriter.Write(header)
	if err != nil {
		return err
	}

	for cardID, entries := range inventory {
		cardHeader, err := cardID2record(cardID)
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
			if len(header) >= len(MarketHeader) {
				record = append(record, entry.SellerName)
				bundle := ""
				if entry.Bundle {
					bundle = "Y"
				}
				record = append(record, bundle)

				if len(header) >= len(CartHeader) {
					record = append(record, entry.OriginalId)
					record = append(record, entry.InstanceId)
				}
			}

			err = csvWriter.Write(record)
			if err != nil {
				return err
			}
		}
		csvWriter.Flush()
	}

	// Every record above was flushed as it was written, so this reports
	// what the last write to w returned rather than nil
	return csvWriter.Error()
}

// WriteBuylistToCSV writes a buylist, adding a trade-price column worth
// creditMuliplier times the cash price, for vendors who pay more in credit.
// Pass 1 to make the two columns agree.
func WriteBuylistToCSV(buylist BuylistRecord, creditMuliplier float64, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	err := csvWriter.Write(BuylistHeader)
	if err != nil {
		return err
	}

	for cardID, entries := range buylist {
		for _, entry := range entries {
			record, err := cardID2record(cardID)
			if err != nil {
				continue
			}

			record = append(record,
				entry.Conditions,
				fmt.Sprintf("%0.2f", entry.BuyPrice),
				fmt.Sprintf("%0.2f", entry.BuyPrice*creditMuliplier),
				fmt.Sprint(entry.Quantity),
				fmt.Sprintf("%0.2f", entry.PriceRatio),
				entry.URL,
				entry.VendorName,
			)

			err = csvWriter.Write(record)
			if err != nil {
				return err
			}
			csvWriter.Flush()
		}
	}

	return csvWriter.Error()
}

// WriteArbitrageToCSV writes what Arbit returned, both prices and the spread
// and profitability derived from them.
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

		record, err := cardID2record(entry.CardId)
		if err != nil {
			continue
		}

		record = append(record,
			inv.Conditions,
			fmt.Sprintf("%d", inv.Quantity),
			fmt.Sprintf("%0.2f", inv.Price),
			fmt.Sprintf("%0.2f", bl.BuyPrice),
			fmt.Sprintf("%0.2f", entry.Difference),
			fmt.Sprintf("%0.2f", entry.Spread),
			fmt.Sprintf("%0.2f", entry.AbsoluteDifference),
			fmt.Sprintf("%0.2f", entry.Profitability),
			inv.URL,
			bl.URL,
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

	return csvWriter.Error()
}

// WriteMismatchToCSV writes what Mismatch returned, the probed price beside
// the reference it was compared against.
func WriteMismatchToCSV(mismatch []ArbitEntry, w io.Writer) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	hasExtraSeller := false
	header := MismatchHeader
	if len(mismatch) > 0 && mismatch[0].InventoryEntry.SellerName != "" {
		header = append(header, "Seller")
		header = append(header, "Bundle")
		hasExtraSeller = true
	}

	err := csvWriter.Write(header)
	if err != nil {
		return err
	}

	for _, entry := range mismatch {
		inv := entry.InventoryEntry
		ref := entry.ReferenceEntry

		record, err := cardID2record(entry.CardId)
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

	return csvWriter.Error()
}

// WritePennyToCSV writes what Pennystock returned.
func WritePennyToCSV(penny []ArbitEntry, w io.Writer) error {
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

		record, err := cardID2record(entry.CardId)
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

	return csvWriter.Error()
}
