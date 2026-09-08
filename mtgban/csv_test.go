package mtgban

import (
	"bytes"
	"encoding/csv"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// csvCards is the datastore the writers read to fill the card columns. A card
// the datastore does not hold is dropped from the file, so every id used below
// needs an entry here or the row it should have written never appears.
func csvCards() map[string]*mtgmatcher.CardObject {
	return map[string]*mtgmatcher.CardObject{
		"plain": {
			Card:    mtgmatcher.Card{Name: "Plain Card", Rarity: "rare", Number: "10"},
			Edition: "Alpha Set",
		},
		"shiny": {
			Card:    mtgmatcher.Card{Name: "Shiny Card", Rarity: "mythic", Number: "11"},
			Edition: "Alpha Set",
			Foil:    true,
		},
		"scratched": {
			Card:    mtgmatcher.Card{Name: "Etched Card", Rarity: "rare", Number: "12"},
			Edition: "Alpha Set",
			Etched:  true,
		},
		"box": {
			Card:    mtgmatcher.Card{Name: "Booster Box", Number: "13"},
			Edition: "Alpha Set",
			Sealed:  true,
		},
	}
}

// writeCSV runs a writer and hands back what it produced, parsed.
func writeCSV(t *testing.T, write func(w *bytes.Buffer) error) [][]string {
	t.Helper()
	var buf bytes.Buffer
	err := write(&buf)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	records, err := csv.NewReader(&buf).ReadAll()
	if err != nil {
		t.Fatalf("the writer produced unreadable csv: %v", err)
	}
	return records
}

// rowFor returns the single row written for a card id, which is column zero.
func rowFor(t *testing.T, records [][]string, cardID string) []string {
	t.Helper()
	for _, record := range records[1:] {
		if record[0] == cardID {
			return record
		}
	}
	t.Fatalf("no row for %s in %v", cardID, records)
	return nil
}

// An inventory has to come back from its own file as it went in, which is
// what the nightly dumps and everything reading them depend on.
func TestInventoryRoundTripsThroughCSV(t *testing.T) {
	installCards(t, csvCards())

	want := InventoryRecord{}
	add := func(cardID string, entry *InventoryEntry) {
		t.Helper()
		err := want.Add(cardID, entry)
		if err != nil {
			t.Fatalf("add %s: %v", cardID, err)
		}
	}
	add("plain", &InventoryEntry{Conditions: "NM", Price: 10.50, Quantity: 2, URL: "u1"})
	add("plain", &InventoryEntry{Conditions: "SP", Price: 8.25, Quantity: 1, URL: "u2"})
	add("shiny", &InventoryEntry{Conditions: "NM", Price: 3.75, Quantity: 7, URL: "u3"})

	var buf bytes.Buffer
	err := WriteInventoryToCSV(want, &buf)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := LoadInventoryFromCSV(&buf)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loaded %v, want %v", got, want)
	}
}

// A market file carries the seller each price belongs to, and the loader has
// to pick the wider header up from the file rather than being told about it.
func TestInventoryRoundTripsThroughCSVWithItsSellers(t *testing.T) {
	installCards(t, csvCards())

	want := InventoryRecord{
		"plain": {{Conditions: "NM", Price: 1.50, Quantity: 3, URL: "u1", SellerName: "Store A", Bundle: true}},
		"shiny": {{Conditions: "NM", Price: 2.50, Quantity: 1, URL: "u2", SellerName: "Store B"}},
	}

	var buf bytes.Buffer
	err := WriteInventoryToCSV(want, &buf)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := LoadInventoryFromCSV(&buf)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loaded %v, want %v", got, want)
	}
}

// The cart header carries the two ids an order needs, which no other header
// holds and which the loader detects the same way.
func TestInventoryRoundTripsThroughCSVWithTheCartIds(t *testing.T) {
	installCards(t, csvCards())

	want := InventoryRecord{
		"plain": {{
			Conditions: "NM", Price: 1.50, Quantity: 3, URL: "u1",
			SellerName: "Store A", OriginalID: "prod-1", InstanceID: "sku-1",
		}},
	}

	var buf bytes.Buffer
	err := WriteInventoryToCSV(want, &buf)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(buf.String(), "Instance Id") {
		t.Fatalf("an entry carrying the cart ids was written without them:\n%s", buf.String())
	}

	got, err := LoadInventoryFromCSV(&buf)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loaded %v, want %v", got, want)
	}
}

// A buylist file has a trade price column the loader does not read back: it
// is the cash price times the store's credit multiplier, derivable from what
// is kept, and a reader that took it for the buy price would overpay.
func TestBuylistRoundTripsThroughCSVWithoutTheTradePrice(t *testing.T) {
	installCards(t, csvCards())

	want := BuylistRecord{
		"plain": {{Conditions: "NM", BuyPrice: 4.00, Quantity: 2, PriceRatio: 40.00, URL: "b1", VendorName: "Store A"}},
	}

	var buf bytes.Buffer
	err := WriteBuylistToCSV(want, 1.3, &buf)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(buf.String(), "5.20") {
		t.Fatalf("the trade price was not written at 1.3 times the cash price:\n%s", buf.String())
	}

	got, err := LoadBuylistFromCSV(&buf)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("loaded %v, want %v", got, want)
	}
}

// The price ratio is a percentage, and a file that spells it with the sign
// reads the same as one that leaves it off, which is all this writer emits.
func TestLoadBuylistFromCSVReadsARatioWithItsSign(t *testing.T) {
	installCards(t, csvCards())

	file := strings.Join(BuylistHeader, ",") + "\n" +
		"plain,Plain Card,Alpha Set,nonfoil,10,rare,NM,4.00,4.00,2,40.00%,b1,Store A\n"

	got, err := LoadBuylistFromCSV(strings.NewReader(file))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got["plain"][0].PriceRatio != 40 {
		t.Errorf("PriceRatio = %v, want 40", got["plain"][0].PriceRatio)
	}
}

// An empty file is a store that collected nothing, not a broken file.
func TestLoadFromCSVAcceptsAnEmptyFile(t *testing.T) {
	inv, err := LoadInventoryFromCSV(strings.NewReader(""))
	if err != nil {
		t.Errorf("inventory: %v", err)
	}
	if len(inv) != 0 {
		t.Errorf("inventory = %v, want empty", inv)
	}

	bl, err := LoadBuylistFromCSV(strings.NewReader(""))
	if err != nil {
		t.Errorf("buylist: %v", err)
	}
	if len(bl) != 0 {
		t.Errorf("buylist = %v, want empty", bl)
	}
}

// A header that is not one this package writes is refused outright rather
// than read column by column into whatever it happens to line up with.
func TestLoadFromCSVRejectsAMalformedHeader(t *testing.T) {
	for _, header := range []string{
		"UUID,Name,Edition",
		"UUID,Name,Edition,Finish,Number,Rarity,Conditions,Price,Quantity,Link",
		"Price,Quantity,UUID,Name,Edition,Finish,Number,Rarity,Conditions,URL",
	} {
		_, err := LoadInventoryFromCSV(strings.NewReader(header + "\n"))
		if err == nil {
			t.Errorf("inventory %q: want an error, got none", header)
		}
		_, err = LoadBuylistFromCSV(strings.NewReader(header + "\n"))
		if err == nil {
			t.Errorf("buylist %q: want an error, got none", header)
		}
	}
}

// Strictness is the difference between a file this package wrote, where a
// bad row means the writer is broken, and a file from elsewhere, where it
// means one row is and the rest are still worth having.
func TestLoadInventoryFromCSVIsStrictUnlessToldOtherwise(t *testing.T) {
	installCards(t, csvCards())

	file := strings.Join(InventoryHeader, ",") + "\n" +
		"plain,Plain Card,Alpha Set,nonfoil,10,rare,NM,not a price,1,u1\n" +
		"shiny,Shiny Card,Alpha Set,foil,11,mythic,NM,2.50,1,u2\n"

	_, err := LoadInventoryFromCSV(strings.NewReader(file))
	if err == nil {
		t.Error("a bad row was read without an error")
	}

	got, err := LoadInventoryFromCSV(strings.NewReader(file), false)
	if err != nil {
		t.Fatalf("lenient load: %v", err)
	}
	if len(got) != 1 || got["shiny"][0].Price != 2.50 {
		t.Errorf("lenient load = %v, want only the readable row", got)
	}
}

func TestLoadBuylistFromCSVIsStrictUnlessToldOtherwise(t *testing.T) {
	installCards(t, csvCards())

	file := strings.Join(BuylistHeader, ",") + "\n" +
		"plain,Plain Card,Alpha Set,nonfoil,10,rare,NM,not a price,0,1,40.00,b1,Store A\n" +
		"shiny,Shiny Card,Alpha Set,foil,11,mythic,NM,2.50,2.50,1,40.00,b2,Store A\n"

	_, err := LoadBuylistFromCSV(strings.NewReader(file))
	if err == nil {
		t.Error("a bad row was read without an error")
	}

	got, err := LoadBuylistFromCSV(strings.NewReader(file), false)
	if err != nil {
		t.Fatalf("lenient load: %v", err)
	}
	if len(got) != 1 || got["shiny"][0].BuyPrice != 2.50 {
		t.Errorf("lenient load = %v, want only the readable row", got)
	}
}

// An id the datastore does not hold stops a buylist row but not an inventory
// one, which keeps reading a market file whose ids are the seller's own.
func TestLoadFromCSVDisagreesOnAnUnknownCard(t *testing.T) {
	installCards(t, csvCards())

	inv, err := LoadInventoryFromCSV(strings.NewReader(
		strings.Join(InventoryHeader, ",") + "\n" +
			"ghost|Ghost Card|SET|1,Ghost Card,SET,nonfoil,1,rare,NM,1.00,1,u1\n"))
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}
	if len(inv) != 1 {
		t.Errorf("inventory = %v, want the row kept under its own id", inv)
	}

	_, err = LoadBuylistFromCSV(strings.NewReader(
		strings.Join(BuylistHeader, ",") + "\n" +
			"ghost,Ghost Card,SET,nonfoil,1,rare,NM,1.00,1.00,1,40.00,b1,Store A\n"))
	if err == nil {
		t.Error("a buylist row for an unknown card was read without an error")
	}
}

// The finish column is what tells four prices for the same card apart.
func TestWriteInventoryToCSVNamesTheFinish(t *testing.T) {
	installCards(t, csvCards())

	inv := InventoryRecord{}
	for _, cardID := range []string{"plain", "shiny", "scratched", "box"} {
		inv[cardID] = []InventoryEntry{{Conditions: "NM", Price: 1, Quantity: 1}}
	}

	records := writeCSV(t, func(w *bytes.Buffer) error {
		return WriteInventoryToCSV(inv, w)
	})

	finish := slices.Index(InventoryHeader, "Finish")
	for cardID, want := range map[string]string{
		"plain": "nonfoil", "shiny": "foil", "scratched": "etched", "box": "sealed",
	} {
		got := rowFor(t, records, cardID)[finish]
		if got != want {
			t.Errorf("%s finish = %q, want %q", cardID, got, want)
		}
	}
}

// A card the datastore lost between the scrape and the dump is dropped from
// the file rather than failing the whole write.
func TestWriteToCSVSkipsACardItCannotName(t *testing.T) {
	installCards(t, csvCards())

	inv := InventoryRecord{
		"plain":   {{Conditions: "NM", Price: 1, Quantity: 1}},
		"missing": {{Conditions: "NM", Price: 1, Quantity: 1}},
		// A pipe id has to carry all four fields to be written
		"short|Card": {{Conditions: "NM", Price: 1, Quantity: 1}},
	}
	records := writeCSV(t, func(w *bytes.Buffer) error {
		return WriteInventoryToCSV(inv, w)
	})
	if len(records) != 2 {
		t.Errorf("wrote %d rows, want the header and the one nameable card: %v", len(records), records)
	}

	bl := BuylistRecord{
		"plain":   {{Conditions: "NM", BuyPrice: 1, Quantity: 1}},
		"missing": {{Conditions: "NM", BuyPrice: 1, Quantity: 1}},
	}
	records = writeCSV(t, func(w *bytes.Buffer) error {
		return WriteBuylistToCSV(bl, 1, w)
	})
	if len(records) != 2 {
		t.Errorf("wrote %d rows, want the header and the one nameable card: %v", len(records), records)
	}
}

// A pipe id stands in for a card with no datastore entry, spelling the name,
// edition and number the columns would have been filled from.
func TestWriteInventoryToCSVReadsThePipeID(t *testing.T) {
	installCards(t, csvCards())

	inv := InventoryRecord{
		"ghost|Ghost Card|SET|42": {{Conditions: "NM", Price: 1, Quantity: 1}},
	}
	records := writeCSV(t, func(w *bytes.Buffer) error {
		return WriteInventoryToCSV(inv, w)
	})

	want := []string{"ghost|Ghost Card|SET|42", "Ghost Card", "SET", "42", "", ""}
	got := records[1][:len(want)]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("card columns = %v, want %v", got, want)
	}
}

func arbitEntry() ArbitEntry {
	return ArbitEntry{
		CardID: "plain",
		InventoryEntry: InventoryEntry{
			Conditions: "NM", Price: 10.00, Quantity: 4, URL: "sell-link",
		},
		BuylistEntry: BuylistEntry{
			Conditions: "NM", BuyPrice: 15.00, Quantity: 3, URL: "buy-link",
		},
		ReferenceEntry: InventoryEntry{
			Conditions: "NM", Price: 12.00, Quantity: 1, URL: "ref-link",
		},
		Difference:         5.00,
		Spread:             50.00,
		AbsoluteDifference: 15.00,
		Profitability:      1.23,
	}
}

// The arbitrage report carries both prices and every number derived from
// them, in the order the header names.
func TestWriteArbitrageToCSVReportsTheTrade(t *testing.T) {
	installCards(t, csvCards())

	records := writeCSV(t, func(w *bytes.Buffer) error {
		return WriteArbitrageToCSV([]ArbitEntry{arbitEntry()}, w)
	})
	if !reflect.DeepEqual(records[0], ArbitHeader) {
		t.Errorf("header = %v, want %v", records[0], ArbitHeader)
	}

	want := []string{"NM", "4", "10.00", "15.00", "5.00", "50.00", "15.00", "1.23", "sell-link", "buy-link"}
	got := records[1][len(CardHeader):]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("row = %v, want %v", got, want)
	}
}

// The mismatch report is the same row against a reference price rather than
// a buy price, so the reference is what has to land in the second column.
func TestWriteMismatchToCSVReportsTheReferencePrice(t *testing.T) {
	installCards(t, csvCards())

	records := writeCSV(t, func(w *bytes.Buffer) error {
		return WriteMismatchToCSV([]ArbitEntry{arbitEntry()}, w)
	})
	if !reflect.DeepEqual(records[0], MismatchHeader) {
		t.Errorf("header = %v, want %v", records[0], MismatchHeader)
	}

	want := []string{"NM", "10.00", "12.00", "5.00", "50.00"}
	got := records[1][len(CardHeader):]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("row = %v, want %v", got, want)
	}
}

// The penny report is a shopping list rather than a trade, so it is written
// under the inventory header with nothing in the link column.
func TestWritePennyToCSVReportsTheShelf(t *testing.T) {
	installCards(t, csvCards())

	records := writeCSV(t, func(w *bytes.Buffer) error {
		return WritePennyToCSV([]ArbitEntry{arbitEntry()}, w)
	})
	if !reflect.DeepEqual(records[0], InventoryHeader) {
		t.Errorf("header = %v, want %v", records[0], InventoryHeader)
	}

	want := []string{"NM", "10.00", "4", ""}
	got := records[1][len(CardHeader):]
	if !reflect.DeepEqual(got, want) {
		t.Errorf("row = %v, want %v", got, want)
	}
}

// A report on a market names the seller each row came from, which is the
// difference between a row someone can act on and one they cannot.
func TestReportsNameTheSellerWhenThereIsOne(t *testing.T) {
	installCards(t, csvCards())

	entry := arbitEntry()
	entry.InventoryEntry.SellerName = "Store A"
	entry.InventoryEntry.Bundle = true

	for _, tc := range []struct {
		name  string
		write func(entries []ArbitEntry, w *bytes.Buffer) error
	}{
		{"arbitrage", func(e []ArbitEntry, w *bytes.Buffer) error { return WriteArbitrageToCSV(e, w) }},
		{"mismatch", func(e []ArbitEntry, w *bytes.Buffer) error { return WriteMismatchToCSV(e, w) }},
		{"penny", func(e []ArbitEntry, w *bytes.Buffer) error { return WritePennyToCSV(e, w) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			records := writeCSV(t, func(w *bytes.Buffer) error {
				return tc.write([]ArbitEntry{entry}, w)
			})
			header := records[0]
			if header[len(header)-2] != "Seller" || header[len(header)-1] != "Bundle" {
				t.Fatalf("header = %v, want it to end in Seller and Bundle", header)
			}
			row := records[1]
			if row[len(row)-2] != "Store A" || row[len(row)-1] != "Y" {
				t.Errorf("row = %v, want it to end in the seller and its bundle flag", row)
			}
		})
	}
}

// Every report header is built by appending to the one before it, so they
// share a backing array; a writer that widens its own copy for a market must
// not widen the package's, or the next report is written under the wrong one.
func TestReportsLeaveTheSharedHeadersAlone(t *testing.T) {
	installCards(t, csvCards())

	headers := map[string][]string{
		"CardHeader":      slices.Clone(CardHeader),
		"InventoryHeader": slices.Clone(InventoryHeader),
		"MarketHeader":    slices.Clone(MarketHeader),
		"CartHeader":      slices.Clone(CartHeader),
		"BuylistHeader":   slices.Clone(BuylistHeader),
		"ArbitHeader":     slices.Clone(ArbitHeader),
		"MismatchHeader":  slices.Clone(MismatchHeader),
	}

	entry := arbitEntry()
	entry.InventoryEntry.SellerName = "Store A"
	entries := []ArbitEntry{entry}
	var buf bytes.Buffer
	for _, write := range []func([]ArbitEntry, *bytes.Buffer) error{
		func(e []ArbitEntry, w *bytes.Buffer) error { return WriteArbitrageToCSV(e, w) },
		func(e []ArbitEntry, w *bytes.Buffer) error { return WriteMismatchToCSV(e, w) },
		func(e []ArbitEntry, w *bytes.Buffer) error { return WritePennyToCSV(e, w) },
	} {
		buf.Reset()
		err := write(entries, &buf)
		if err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	for name, want := range map[string][]string{
		"CardHeader": CardHeader, "InventoryHeader": InventoryHeader,
		"MarketHeader": MarketHeader, "CartHeader": CartHeader,
		"BuylistHeader": BuylistHeader, "ArbitHeader": ArbitHeader,
		"MismatchHeader": MismatchHeader,
	} {
		if !reflect.DeepEqual(want, headers[name]) {
			t.Errorf("%s = %v, want %v", name, want, headers[name])
		}
	}
}

// A report with nothing in it is still a file with a header, which is what
// tells a reader the run happened and found nothing.
func TestReportsWriteAHeaderWithNoRows(t *testing.T) {
	for _, tc := range []struct {
		name  string
		write func(w *bytes.Buffer) error
	}{
		{"arbitrage", func(w *bytes.Buffer) error { return WriteArbitrageToCSV(nil, w) }},
		{"mismatch", func(w *bytes.Buffer) error { return WriteMismatchToCSV(nil, w) }},
		{"penny", func(w *bytes.Buffer) error { return WritePennyToCSV(nil, w) }},
		{"inventory", func(w *bytes.Buffer) error { return WriteInventoryToCSV(InventoryRecord{}, w) }},
		{"buylist", func(w *bytes.Buffer) error { return WriteBuylistToCSV(BuylistRecord{}, 1, w) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			records := writeCSV(t, tc.write)
			if len(records) != 1 {
				t.Errorf("wrote %d rows, want the header alone", len(records))
			}
		})
	}
}

// A destination that dies takes the report down with it rather than being
// reported as a written file that is missing most of its rows.
func TestReportsReportAFailedDestination(t *testing.T) {
	installCards(t, csvCards())

	entries := []ArbitEntry{arbitEntry(), arbitEntry(), arbitEntry()}
	for _, tc := range []struct {
		name  string
		write func(w *failAfter) error
	}{
		{"arbitrage", func(w *failAfter) error { return WriteArbitrageToCSV(entries, w) }},
		{"mismatch", func(w *failAfter) error { return WriteMismatchToCSV(entries, w) }},
		{"penny", func(w *failAfter) error { return WritePennyToCSV(entries, w) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.write(&failAfter{budget: 40})
			if err == nil {
				t.Error("a destination that failed mid-write reported success")
			}
		})
	}
}

// A row with the wrong number of columns is the file being ragged rather
// than one field being unreadable, and the csv reader reports it before any
// of the fields are looked at.
func TestLoadFromCSVHandlesARaggedFile(t *testing.T) {
	installCards(t, csvCards())

	inventory := strings.Join(InventoryHeader, ",") + "\n" +
		"plain,Plain Card,Alpha Set,nonfoil,10,rare,NM\n" +
		"shiny,Shiny Card,Alpha Set,foil,11,mythic,NM,2.50,1,u2\n"
	buylist := strings.Join(BuylistHeader, ",") + "\n" +
		"plain,Plain Card,Alpha Set,nonfoil,10,rare,NM\n" +
		"shiny,Shiny Card,Alpha Set,foil,11,mythic,NM,2.50,2.50,1,40.00,b2,Store A\n"

	_, err := LoadInventoryFromCSV(strings.NewReader(inventory))
	if err == nil {
		t.Error("inventory: a ragged file was read without an error")
	}
	_, err = LoadBuylistFromCSV(strings.NewReader(buylist))
	if err == nil {
		t.Error("buylist: a ragged file was read without an error")
	}

	inv, err := LoadInventoryFromCSV(strings.NewReader(inventory), false)
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}
	if len(inv) != 1 {
		t.Errorf("inventory = %v, want only the whole row", inv)
	}
	bl, err := LoadBuylistFromCSV(strings.NewReader(buylist), false)
	if err != nil {
		t.Fatalf("buylist: %v", err)
	}
	if len(bl) != 1 {
		t.Errorf("buylist = %v, want only the whole row", bl)
	}
}

// A header that is not readable as csv at all fails as a header rather than
// as the missing columns it would otherwise look like.
func TestLoadFromCSVRejectsAnUnreadableHeader(t *testing.T) {
	_, err := LoadInventoryFromCSV(strings.NewReader("\"unclosed\n"))
	if err == nil {
		t.Error("inventory: an unreadable header was read without an error")
	}
	_, err = LoadBuylistFromCSV(strings.NewReader("\"unclosed\n"))
	if err == nil {
		t.Error("buylist: an unreadable header was read without an error")
	}
}

// A buylist header the right width but naming something else is refused on
// the names, not on the count.
func TestLoadBuylistFromCSVRejectsARenamedColumn(t *testing.T) {
	header := slices.Clone(BuylistHeader)
	header[len(header)-1] = "Store"

	_, err := LoadBuylistFromCSV(strings.NewReader(strings.Join(header, ",") + "\n"))
	if err == nil {
		t.Error("a header naming a column it does not have was read without an error")
	}
}

// Each field the loader parses is its own reason to drop a row, and a lenient
// read has to get past all of them to the rows that are whole.
func TestLoadBuylistFromCSVSkipsEveryUnreadableField(t *testing.T) {
	installCards(t, csvCards())

	file := strings.Join(BuylistHeader, ",") + "\n" +
		"missing,Gone Card,Alpha Set,nonfoil,9,rare,NM,1.00,1.00,1,40.00,b0,Store A\n" +
		"plain,Plain Card,Alpha Set,nonfoil,10,rare,NM,not a price,0,1,40.00,b1,Store A\n" +
		"plain,Plain Card,Alpha Set,nonfoil,10,rare,NM,1.00,1.00,many,40.00,b1,Store A\n" +
		"plain,Plain Card,Alpha Set,nonfoil,10,rare,NM,1.00,1.00,1,a lot,b1,Store A\n" +
		"shiny,Shiny Card,Alpha Set,foil,11,mythic,NM,2.50,2.50,1,40.00,b2,Store A\n"

	got, err := LoadBuylistFromCSV(strings.NewReader(file), false)
	if err != nil {
		t.Fatalf("lenient load: %v", err)
	}
	if len(got) != 1 || got["shiny"][0].BuyPrice != 2.50 {
		t.Errorf("lenient load = %v, want only the whole row", got)
	}
}

// The inventory loader reads a card the datastore does not hold only when the
// id spells the card itself; a bare unknown id is a row it cannot place.
func TestLoadInventoryFromCSVNeedsToPlaceTheCard(t *testing.T) {
	installCards(t, csvCards())

	file := strings.Join(InventoryHeader, ",") + "\n" +
		"missing,Gone Card,Alpha Set,nonfoil,9,rare,NM,1.00,1,u0\n" +
		"plain,Plain Card,Alpha Set,nonfoil,10,rare,NM,1.00,not a number,u1\n" +
		"shiny,Shiny Card,Alpha Set,foil,11,mythic,NM,2.50,1,u2\n"

	_, err := LoadInventoryFromCSV(strings.NewReader(file))
	if err == nil {
		t.Error("a row for an unknown card was read without an error")
	}

	got, err := LoadInventoryFromCSV(strings.NewReader(file), false)
	if err != nil {
		t.Fatalf("lenient load: %v", err)
	}
	if len(got) != 1 || got["shiny"][0].Price != 2.50 {
		t.Errorf("lenient load = %v, want only the placeable row", got)
	}
}

// A file holding the same row twice is a dump written twice over, not a store
// with two of a card at one price, and a strict read says so.
func TestLoadFromCSVRefusesARepeatedRow(t *testing.T) {
	installCards(t, csvCards())

	row := "plain,Plain Card,Alpha Set,nonfoil,10,rare,NM,1.00,1,u1\n"
	_, err := LoadInventoryFromCSV(strings.NewReader(strings.Join(InventoryHeader, ",") + "\n" + row + row))
	if err == nil {
		t.Error("inventory: a repeated row was read without an error")
	}

	row = "plain,Plain Card,Alpha Set,nonfoil,10,rare,NM,1.00,1.00,1,40.00,b1,Store A\n"
	_, err = LoadBuylistFromCSV(strings.NewReader(strings.Join(BuylistHeader, ",") + "\n" + row + row))
	if err == nil {
		t.Error("buylist: a repeated row was read without an error")
	}
}

// A destination that was already gone fails on the header, before any price
// is written, rather than reporting an empty file as a written one.
func TestWriteToCSVReportsADestinationThatWasNeverThere(t *testing.T) {
	installCards(t, csvCards())

	entries := []ArbitEntry{arbitEntry()}
	for _, tc := range []struct {
		name  string
		write func(w *failAfter) error
	}{
		{"inventory", func(w *failAfter) error {
			return WriteInventoryToCSV(InventoryRecord{"plain": {{Conditions: "NM", Price: 1, Quantity: 1}}}, w)
		}},
		{"buylist", func(w *failAfter) error {
			return WriteBuylistToCSV(BuylistRecord{"plain": {{Conditions: "NM", BuyPrice: 1, Quantity: 1}}}, 1, w)
		}},
		{"arbitrage", func(w *failAfter) error { return WriteArbitrageToCSV(entries, w) }},
		{"mismatch", func(w *failAfter) error { return WriteMismatchToCSV(entries, w) }},
		{"penny", func(w *failAfter) error { return WritePennyToCSV(entries, w) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.write(&failAfter{})
			if err == nil {
				t.Error("a destination that was never there reported success")
			}
		})
	}
}

// The quantity and the ratio are read after the price, so a file broken in
// one of them reaches a strict read further in than the tests above do.
func TestLoadBuylistFromCSVStopsAtAnyUnreadableField(t *testing.T) {
	installCards(t, csvCards())

	for _, row := range []string{
		"plain,Plain Card,Alpha Set,nonfoil,10,rare,NM,1.00,1.00,many,40.00,b1,Store A\n",
		"plain,Plain Card,Alpha Set,nonfoil,10,rare,NM,1.00,1.00,1,a lot,b1,Store A\n",
	} {
		_, err := LoadBuylistFromCSV(strings.NewReader(strings.Join(BuylistHeader, ",") + "\n" + row))
		if err == nil {
			t.Errorf("%q was read without an error", row)
		}
	}
}

// A buylist stops at the row the destination died on rather than carrying on
// writing into a file nothing is reaching.
func TestWriteBuylistToCSVStopsAtAFailedDestination(t *testing.T) {
	bl := BuylistRecord{}
	for _, id := range []string{"a|Card A|SET|1", "b|Card B|SET|2", "c|Card C|SET|3"} {
		bl[id] = []BuylistEntry{{Conditions: "NM", BuyPrice: 1, Quantity: 1}}
	}
	err := WriteBuylistToCSV(bl, 1, &failAfter{budget: 80})
	if err == nil {
		t.Error("a destination that failed mid-write reported success")
	}
}

// A report row for a card the datastore no longer holds is dropped, the same
// way the price files drop it, rather than failing the report.
func TestReportsSkipACardTheyCannotName(t *testing.T) {
	installCards(t, csvCards())

	known := arbitEntry()
	unknown := arbitEntry()
	unknown.CardID = "missing"
	entries := []ArbitEntry{unknown, known}

	for _, tc := range []struct {
		name  string
		write func(w *bytes.Buffer) error
	}{
		{"arbitrage", func(w *bytes.Buffer) error { return WriteArbitrageToCSV(entries, w) }},
		{"mismatch", func(w *bytes.Buffer) error { return WriteMismatchToCSV(entries, w) }},
		{"penny", func(w *bytes.Buffer) error { return WritePennyToCSV(entries, w) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			records := writeCSV(t, tc.write)
			if len(records) != 2 || records[1][0] != "plain" {
				t.Errorf("wrote %v, want the header and the one nameable row", records)
			}
		})
	}
}
