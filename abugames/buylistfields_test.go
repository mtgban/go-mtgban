package abugames

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

// TestBuylistKeepsABUNames pins the edition and title ABU's buylist invoice
// sorts by on every buylist entry of Revised Goblin King, as ABU's catalog
// listed it on 2026-11-20: cash and credit, and the grades flooring fills.
func TestBuylistKeepsABUNames(t *testing.T) {
	b := realDatastore(t)

	doc := `{"product_id":"151168","display_title":"Goblin King","simple_title":"Goblin King","magic_edition_sort":"Revised","layout":"Normal","rarity":"Rare","language":["English"],"title":"English","card_number":"155","scryfall_id":["e3094187-d666-414b-a1fd-ae0ef55c3fcb"],"tcgplayer_id":[1435],"multiverseid":[1296]`
	page := `{"grouped":{"product_id":{"ngroups":1,"groups":[{"groupValue":"151168","doclist":{"docs":[` +
		doc + `,"id":"269725","condition":"NM","price":8.65,"buy_list_quantity":15,"buy_price":2.96,"trade_price":5.46},` +
		doc + `,"id":"269726","condition":"PLD","price":5.09,"buy_list_quantity":38,"buy_price":2.17,"trade_price":4.2},` +
		doc + `,"id":"283492","condition":"HP","price":4.59,"buy_list_quantity":37,"buy_price":1.97,"trade_price":3.82}` +
		`]}}]}}}`
	abu := NewScraper(b)
	abu.client.client = &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		body := page
		if r.URL.Query().Get("start") != "0" {
			body = `{}`
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}
	err := abu.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	var entries int
	for _, offers := range abu.Buylist() {
		for _, entry := range offers {
			entries++
			if entry.CustomFields["ABUEdition"] != "Revised" || entry.CustomFields["ABUTitle"] != "Goblin King" {
				t.Errorf("%s %s entry keeps %q", entry.VendorName, entry.Conditions, entry.CustomFields)
			}
		}
	}
	// NM, PLD and HP for cash and credit, plus the SP and MP flooring fills
	if entries != 8 {
		t.Errorf("%d buylist entries, want 8", entries)
	}
}
