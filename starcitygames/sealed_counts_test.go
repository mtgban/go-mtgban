package starcitygames

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
)

// TestSealedPricedLine runs Load over the fixture catalog and pins the line
// it ends on. The box is placed by id and listed twice, each listing sold
// and bought (one product, several entries); the case is placed by name; the
// gift set is placed by name with nothing in stock and no buy price, so it
// is never priced.
func TestSealedPricedLine(t *testing.T) {
	catalog := sealedMatchCatalog()
	for i := range catalog {
		if catalog[i].SKU == "SLD-LOR-GFT-011-EN" {
			catalog[i].Variants = []CatalogVariant{{SKU: "SLD-LOR-GFT-011-EN-NM", Qty: 0, Price: "40.00"}}
		}
	}
	body, err := json.Marshal(catalog)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	}))
	defer srv.Close()
	sets := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"hits":[]}`))
	}))
	defer sets.Close()

	scg, err := NewScraperSealed(sealedMatchBackend(t), "")
	if err != nil {
		t.Fatal(err)
	}
	client := *scg.client
	client.catalogURL = srv.URL
	client.setsURL = sets.URL
	scg.client = &client
	var logs []string
	scg.logCallback = func(format string, a ...any) {
		logs = append(logs, fmt.Sprintf(format, a...))
	}

	err = scg.Load(context.Background())
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if len(scg.inventory["wsp-box"]) == 0 || len(scg.buylist["wsp-box"]) == 0 {
		t.Fatalf("the box is not both sold and bought: %v, %v", scg.inventory, scg.buylist)
	}
	want := "[SCGSealed] Priced 1 products by id, 0 through a bridge, 1 by name"
	if len(logs) == 0 || logs[len(logs)-1] != want || slices.Index(logs, want) != len(logs)-1 {
		t.Errorf("logged %q, want it to end on %q, once", logs, want)
	}
	if !slices.Contains(logs, "[SCGSealed] Priced 2 sealed products") {
		t.Errorf("logged %q, want the uuid count beside it unchanged", logs)
	}
}
