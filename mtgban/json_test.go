package mtgban

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func testScraperInfo() ScraperInfo {
	ts := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	return ScraperInfo{
		Name:               "Test Store",
		Shorthand:          "TS",
		CountryFlag:        "EU",
		InventoryTimestamp: &ts,
		BuylistTimestamp:   &ts,
	}
}

func testInventory(t *testing.T) InventoryRecord {
	t.Helper()
	inv := InventoryRecord{}
	add := func(cardID string, entry *InventoryEntry) {
		err := inv.Add(cardID, entry)
		if err != nil {
			t.Fatalf("add %s: %v", cardID, err)
		}
	}
	add("uuid-a", &InventoryEntry{Conditions: "NM", Price: 10.5, Quantity: 2, URL: "u1"})
	add("uuid-a", &InventoryEntry{Conditions: "SP", Price: 8, Quantity: 1, URL: "u2"})
	add("uuid-b", &InventoryEntry{Conditions: "NM", Price: 3.25, Quantity: 7, URL: "u3"})
	return inv
}

func testBuylist(t *testing.T) BuylistRecord {
	t.Helper()
	bl := BuylistRecord{}
	add := func(cardID string, entry *BuylistEntry) {
		err := bl.Add(cardID, entry)
		if err != nil {
			t.Fatalf("add %s: %v", cardID, err)
		}
	}
	add("uuid-a", &BuylistEntry{Conditions: "NM", BuyPrice: 5, Quantity: 4, URL: "b1"})
	add("uuid-c", &BuylistEntry{Conditions: "NM", BuyPrice: 1.75, Quantity: 9, URL: "b2"})
	return bl
}

// A dump written by either writer has to come back through the reader
// exactly as it went in, one card at a time being an implementation
// detail of the decode and nothing the caller should see.
func TestReadSellerFromJSONRoundTrip(t *testing.T) {
	want := testInventory(t)
	var buf bytes.Buffer
	err := WriteSellerToJSON(NewSellerFromInventory(want, testScraperInfo()), &buf)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	seller, err := ReadSellerFromJSON(&buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	got := seller.Inventory()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("inventory did not round-trip:\n got %v\nwant %v", got, want)
	}
	if seller.Info().Shorthand != "TS" {
		t.Errorf("info did not round-trip: %+v", seller.Info())
	}
}

func TestReadVendorFromJSONRoundTrip(t *testing.T) {
	want := testBuylist(t)
	var buf bytes.Buffer
	err := WriteVendorToJSON(NewVendorFromBuylist(want, testScraperInfo()), &buf)
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	vendor, err := ReadVendorFromJSON(&buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	got := vendor.Buylist()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("buylist did not round-trip:\n got %v\nwant %v", got, want)
	}
}

// WriteScraperToJSON emits both sides for a Market, and each reader has
// to pick out its own without tripping over the other.
func TestReadFromJSONBothSidesPresent(t *testing.T) {
	inv, bl := testInventory(t), testBuylist(t)
	encoded, err := json.Marshal(scraperJSON{
		Info:      testScraperInfo(),
		Inventory: inv,
		Buylist:   bl,
	})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	raw := string(encoded)

	seller, err := ReadSellerFromJSON(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("read seller: %v", err)
	}
	if !reflect.DeepEqual(seller.Inventory(), inv) {
		t.Error("seller side did not round-trip from a both-sided dump")
	}

	vendor, err := ReadVendorFromJSON(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("read vendor: %v", err)
	}
	if !reflect.DeepEqual(vendor.Buylist(), bl) {
		t.Error("vendor side did not round-trip from a both-sided dump")
	}
}

// A side the writer left out stays absent rather than becoming empty,
// and a key a later writer adds is read past rather than failing.
func TestReadFromJSONSkipsWhatItDoesNotKnow(t *testing.T) {
	raw := `{"info":{"shorthand":"TS"},"future_field":{"a":[1,2,3]},` +
		`"inventory":{"uuid-a":[{"conditions":"NM","price":1.5,"quantity":1}]},` +
		`"another":"ignored"}`

	seller, err := ReadSellerFromJSON(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(seller.Inventory()) != 1 {
		t.Errorf("inventory = %v, want one card", seller.Inventory())
	}

	vendor, err := ReadVendorFromJSON(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if vendor.Buylist() != nil {
		t.Errorf("buylist = %v, want nil for a dump that carries none", vendor.Buylist())
	}
}

func TestReadFromJSONRejectsMalformed(t *testing.T) {
	for _, raw := range []string{``, `{`, `{"inventory":`, `{"inventory":{"a":[}}`, `not json`} {
		_, err := ReadSellerFromJSON(strings.NewReader(raw))
		if err == nil {
			t.Errorf("%q: want an error, got none", raw)
		}
	}
}

// WriteScraperToJSON writes whichever sides the scraper has, so a store that
// both sells and buys lands in one file the readers each take their half of.
func TestWriteScraperToJSONWritesBothSides(t *testing.T) {
	scraper := &bothSides{inventory: testInventory(t), buylist: testBuylist(t)}

	var buf bytes.Buffer
	err := WriteScraperToJSON(scraper, &buf)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	raw := buf.String()

	seller, err := ReadSellerFromJSON(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("read seller: %v", err)
	}
	if !reflect.DeepEqual(seller.Inventory(), scraper.inventory) {
		t.Error("the inventory side did not round-trip")
	}

	vendor, err := ReadVendorFromJSON(strings.NewReader(raw))
	if err != nil {
		t.Fatalf("read vendor: %v", err)
	}
	if !reflect.DeepEqual(vendor.Buylist(), scraper.buylist) {
		t.Error("the buylist side did not round-trip")
	}
}

// A scraper stamps both timestamps whatever it collected, so the stamp of a
// side that came back empty is dropped on the way out: that is what lets a
// reader tell a side nobody collected from one collected and found empty.
func TestWriteScraperToJSONDropsTheTimestampOfAnEmptySide(t *testing.T) {
	for _, tc := range []struct {
		name         string
		scraper      *bothSides
		wantSellerTS bool
		wantVendorTS bool
	}{
		{"sells and buys", &bothSides{inventory: testInventory(t), buylist: testBuylist(t)}, true, true},
		{"sells only", &bothSides{inventory: testInventory(t)}, true, false},
		{"buys only", &bothSides{buylist: testBuylist(t)}, false, true},
		{"collected nothing", &bothSides{}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := WriteScraperToJSON(tc.scraper, &buf)
			if err != nil {
				t.Fatalf("write: %v", err)
			}

			var got scraperJSON
			err = json.Unmarshal(buf.Bytes(), &got)
			if err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if (got.Info.InventoryTimestamp != nil) != tc.wantSellerTS {
				t.Errorf("inventory timestamp present = %v, want %v", got.Info.InventoryTimestamp != nil, tc.wantSellerTS)
			}
			if (got.Info.BuylistTimestamp != nil) != tc.wantVendorTS {
				t.Errorf("buylist timestamp present = %v, want %v", got.Info.BuylistTimestamp != nil, tc.wantVendorTS)
			}
		})
	}
}

// A scraper that only sells is written without a buylist key at all, and the
// timestamp of the side it does not have goes with it.
func TestWriteScraperToJSONWritesOnlyTheSidesTheScraperHas(t *testing.T) {
	var buf bytes.Buffer
	err := WriteScraperToJSON(NewSellerFromInventory(testInventory(t), testScraperInfo()), &buf)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if strings.Contains(buf.String(), `"buylist"`) {
		t.Errorf("a seller was written with a buylist key:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), `"buylist_ts"`) {
		t.Errorf("a seller was written with a buylist timestamp:\n%s", buf.String())
	}

	buf.Reset()
	err = WriteScraperToJSON(NewVendorFromBuylist(testBuylist(t), testScraperInfo()), &buf)
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if strings.Contains(buf.String(), `"inventory"`) {
		t.Errorf("a vendor was written with an inventory key:\n%s", buf.String())
	}
	if strings.Contains(buf.String(), `"inventory_ts"`) {
		t.Errorf("a vendor was written with an inventory timestamp:\n%s", buf.String())
	}
}

// The decode walks the dump token by token rather than handing it all to
// Decode, so a shape that is valid JSON but not a dump has to be refused
// where the walk reaches it instead of read as far as it parses.
func TestReadFromJSONRejectsWhatIsNotADump(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want string
	}{
		{`[1,2]`, "field name"},
		{`{"inventory":[1,2]}`, "card id"},
		{`{"buylist":[1,2]}`, "card id"},
	} {
		_, err := ReadSellerFromJSON(strings.NewReader(tc.raw))
		if err == nil {
			t.Errorf("%q: want an error, got none", tc.raw)
			continue
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%q: error %q does not say where %s was due", tc.raw, err, tc.want)
		}
	}
}

func TestReadVendorFromJSONRejectsMalformed(t *testing.T) {
	for _, raw := range []string{``, `{`, `{"buylist":`, `{"buylist":{"a":[}}`, `not json`} {
		_, err := ReadVendorFromJSON(strings.NewReader(raw))
		if err == nil {
			t.Errorf("%q: want an error, got none", raw)
		}
	}
}

// A dump that stops mid-token is a download that did not finish, and the walk
// has to report it where it runs out rather than keep what it read.
func TestReadFromJSONRejectsATruncatedDump(t *testing.T) {
	for _, raw := range []string{
		`{"info`,
		`{"inventory":{"uuid`,
		`{"buylist":{"uuid`,
	} {
		_, err := ReadSellerFromJSON(strings.NewReader(raw))
		if err == nil {
			t.Errorf("%q: want an error, got none", raw)
		}
	}
}
