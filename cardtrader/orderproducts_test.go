package cardtrader

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type stubTransport string

func (s stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(string(s))),
		Request:    req,
	}, nil
}

// TestGetOrderProductsProperties pins the key an order item carries its
// properties under, which is not the one a listing uses.
func TestGetOrderProductsProperties(t *testing.T) {
	body := `{"order_items":[{"id":116567480,"blueprint_id":217203,"quantity":1,` +
		`"properties":{"condition":"Slightly Played","collector_number":"089","mtg_foil":true}}]}`
	ct := &CTAuthClient{client: &http.Client{Transport: stubTransport(body)}}

	products, err := ct.GetOrderProducts(context.Background(), 40683607)
	if err != nil {
		t.Fatal(err)
	}
	if len(products) != 1 {
		t.Fatalf("got %d products, want 1", len(products))
	}
	props := products[0].Properties
	if !props.MTGFoil || props.Condition != "Slightly Played" || props.Number != "089" {
		t.Errorf("got %+v, want a foil Slightly Played 089", props)
	}
}
