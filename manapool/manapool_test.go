package manapool

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestMain loads the datastore when one is configured; the unit test below
// reads no cards, so a checkout without it still runs that.
func TestMain(m *testing.M) {
	if path := os.Getenv("ALLPRINTINGS5_PATH"); path != "" {
		if err := datastore.Load(path); err != nil {
			log.Fatalln(err)
		}
	}
	os.Exit(m.Run())
}

// TestAddCheapestKeepsTheLowerPrice pins that a printing the store files under
// more than one product is priced once per grade, at the lower of the prices
// it arrives with, whichever product arrives first.
func TestAddCheapestKeepsTheLowerPrice(t *testing.T) {
	mp := NewScraper()
	mp.addCheapest("card", &mtgban.InventoryEntry{Conditions: "NM", Price: 25, URL: "first"})
	mp.addCheapest("card", &mtgban.InventoryEntry{Conditions: "NM", Price: 15, URL: "cheaper"})
	mp.addCheapest("card", &mtgban.InventoryEntry{Conditions: "NM", Price: 40, URL: "dearer"})
	mp.addCheapest("card", &mtgban.InventoryEntry{Conditions: "SP", Price: 9, URL: "other-grade"})

	got := map[string]mtgban.InventoryEntry{}
	for _, e := range mp.Inventory()["card"] {
		got[e.Conditions] = e
	}
	if len(got) != 2 {
		t.Fatalf("published %d grades, want NM and SP: %+v", len(got), got)
	}
	if got["NM"].Price != 15 || got["NM"].URL != "cheaper" {
		t.Errorf("NM is %.0f from %q, want 15 from the cheaper product", got["NM"].Price, got["NM"].URL)
	}
	if got["SP"].Price != 9 {
		t.Errorf("SP is %.0f, want 9", got["SP"].Price)
	}
}

// TestReplayCapturedVariants runs a captured price list through the same
// path Load takes, and pins that no row is refused as a duplicate. It needs
// the datastore and a capture of https://manapool.com/api/v1/prices/variants
// named by MANAPOOL_VARIANTS_PATH; it reports how many grades were repriced
// to a lower product, which is the change this buys.
func TestReplayCapturedVariants(t *testing.T) {
	path := os.Getenv("MANAPOOL_VARIANTS_PATH")
	if path == "" || len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("MANAPOOL_VARIANTS_PATH or ALLPRINTINGS5_PATH not set")
	}
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var doc struct {
		Data []Product `json:"data"`
	}
	if err := json.NewDecoder(f).Decode(&doc); err != nil {
		t.Fatal(err)
	}

	var logged []string
	mp := NewScraper()
	mp.LogCallback = func(format string, a ...any) { logged = append(logged, fmt.Sprintf(format, a...)) }
	mp.price(doc.Data)

	var dupes int
	for _, l := range logged {
		if strings.Contains(l, "duplicate") {
			dupes++
		}
	}
	if dupes != 0 {
		t.Errorf("%d rows still refused as duplicates; first: %s", dupes, logged[0])
	}
	var rows int
	for _, entries := range mp.Inventory() {
		rows += len(entries)
	}
	t.Logf("%d list rows -> %d inventory rows, %d log lines, %d duplicate refusals", len(doc.Data), rows, len(logged), dupes)
	for _, l := range logged {
		t.Logf("log: %s", l)
	}
}
