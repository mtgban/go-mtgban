package cardtrader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// listingsTransport answers every request with one expansion's listings.
type listingsTransport []byte

func (l listingsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader(l)),
		Request:    req,
	}, nil
}

// TestSealedPricedLine drives Load's pricing over the fixture blueprints and
// pins the line it ends on. The premium pack sells twice (one blueprint, two
// entries), the bridged pack and the starter deck once each; the box has
// nothing in stock and the case only an opened copy, so both are placed and
// never priced, and the blueprint no product holds is not read at all.
func TestSealedPricedLine(t *testing.T) {
	ct, err := newSealedMatcher(sealedMatchBackend())
	if err != nil {
		t.Fatal(err)
	}
	listing := func(id, bp, qty int, zero bool, description string) Product {
		p := Product{ID: id, BlueprintID: bp, Quantity: qty, Description: description}
		p.Properties.GundamLanguage = "en"
		p.Price = CTPrice{Cents: 1000 + id, Currency: "USD"}
		p.User.Name = fmt.Sprintf("seller%d", id)
		p.User.SealedZero = zero
		return p
	}
	listings, err := json.Marshal(map[int][]Product{
		100: {listing(1, 100, 0, false, "")},
		101: {listing(2, 101, 1, false, ""), listing(3, 101, 2, true, "")},
		200: {listing(4, 200, 1, false, "")},
		300: {listing(5, 300, 1, false, "")},
		400: {listing(6, 400, 1, false, "opened")},
		800: {listing(7, 800, 1, false, "")},
	})
	if err != nil {
		t.Fatal(err)
	}
	ct.client = &CTAuthClient{client: &http.Client{Transport: listingsTransport(listings)}}
	ct.inventory = mtgban.InventoryRecord{}

	placed := ct.placeBlueprints(sealedMatchBlueprints())
	var logs []string
	ct.logCallback = func(format string, a ...any) {
		logs = append(logs, fmt.Sprintf(format, a...))
	}
	ct.priceSealed(context.Background(), placed, map[int]string{1: "Freedom Ascension"})

	if len(ct.inventory["gd05-premium"]) != 2 {
		t.Fatalf("the premium pack holds %v, want both listings", ct.inventory["gd05-premium"])
	}
	want := "[CTSealed] Priced 1 products by id, 1 through a bridge, 1 by name"
	var said int
	for _, line := range logs {
		if line == want {
			said++
		}
	}
	if said != 1 || logs[len(logs)-1] != want {
		t.Errorf("logged %q, want it to end on %q, once", logs, want)
	}
}
