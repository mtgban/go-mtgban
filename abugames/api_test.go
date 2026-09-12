package abugames

import (
	"context"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"
)

type catalogTransport func(*http.Request) (*http.Response, error)

func (f catalogTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestGetProductReadsIdentifiers(t *testing.T) {
	client := &ABUClient{client: &http.Client{Transport: catalogTransport(func(r *http.Request) (*http.Response, error) {
		fields := strings.Split(r.URL.Query().Get("fl"), ",")
		for _, key := range []string{"scryfall_id", "tcgplayer_id", "multiverseid"} {
			if !slices.Contains(fields, key) {
				t.Errorf("catalog request omits %s", key)
			}
		}
		// Actual Solr field shapes: strings for Scryfall, integers for TCGplayer,
		// each wrapped in an array. Older products can omit either field.
		body := `{"grouped":{"product_id":{"groups":[{"groupValue":"8118507","doclist":{"docs":[{"id":"2655495","display_title":"Counterspell (NYCC 2024) - FOIL","scryfall_id":["f2a7042f-a6f0-4e77-86a2-5eb0d2587363"],"tcgplayer_id":[589737],"multiverseid":[74476]},{"id":"missing"}]}}]}}}`
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}}
	product, err := client.GetProduct(context.Background(), "", 0)
	if err != nil {
		t.Fatal(err)
	}
	docs := product.Grouped.ProductID.Groups[0].Doclist.Docs
	if !slices.Equal(docs[0].ScryfallIDs, []string{"f2a7042f-a6f0-4e77-86a2-5eb0d2587363"}) || !slices.Equal(docs[0].TCGplayerIDs, []int64{589737}) || !slices.Equal(docs[0].MultiverseIDs, []int64{74476}) {
		t.Fatalf("identifiers not decoded: %+v", docs[0])
	}
	if len(docs[1].ScryfallIDs) != 0 || len(docs[1].TCGplayerIDs) != 0 || len(docs[1].MultiverseIDs) != 0 {
		t.Fatal("missing fields should decode empty")
	}
}
