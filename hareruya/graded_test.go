package hareruya

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

type fixtureTransport map[string]string

func (f fixtureTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	page := []byte(`{"response":{"page":2,"docs":[]}}`)
	name := f[req.URL.Path+"?page="+req.URL.Query().Get("page")]
	if name != "" {
		var err error
		page, err = os.ReadFile("testdata/" + name)
		if err != nil {
			return nil, err
		}
	}
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(bytes.NewReader(page)), Request: req}, nil
}

// TestGradedSellerCarriesSlab walks a set holding a slab, a slab whose label
// does not read, and a raw lot: every graded entry carries its Slab, the
// unread one as NM, and no raw entry carries one.
func TestGradedSellerCarriesSlab(t *testing.T) {
	ha := NewScraper(realDatastore(t))
	ha.exchangeRate = 1
	ha.client = &http.Client{Transport: fixtureTransport{
		"/en/products/search/unisearch_api?page=1": "graded_search.json",
		"/en/products/search/unisearch/lazy?page=": "graded_lazy.html",
	}}

	results := make(chan responseChan)
	var walkErr error
	go func() {
		walkErr = ha.walkSet(context.Background(), results, "1", "", map[string]bool{})
		close(results)
	}()

	want := map[string]struct {
		seller string
		cond   mtgban.Condition
		slab   *mtgban.Slab
	}{
		"1281153": {"Hareruya Graded", mtgban.SP, &mtgban.Slab{Grader: "CGC", Score: 7.5}},
		"1057787": {"Hareruya Graded", mtgban.NM, &mtgban.Slab{Grader: "PSA", Label: "PSA11"}},
		"452467":  {"Hareruya", mtgban.NM, nil},
		"452468":  {"Hareruya", mtgban.SP, nil},
	}
	got := map[string]bool{}
	for result := range results {
		entry := result.invEntry
		w, found := want[entry.InstanceID]
		if !found {
			t.Errorf("unexpected entry %+v", entry)
			continue
		}
		got[entry.InstanceID] = true
		if entry.SellerName != w.seller || entry.Conditions != w.cond || !sameSlab(entry.Slab, w.slab) {
			t.Errorf("%s = %s %s %+v, want %s %s %+v", entry.InstanceID, entry.SellerName, entry.Conditions, entry.Slab, w.seller, w.cond, w.slab)
		}
		if (entry.Slab != nil) != strings.HasSuffix(entry.SellerName, "Graded") {
			t.Errorf("%s in %s with slab %+v", entry.InstanceID, entry.SellerName, entry.Slab)
		}
	}
	if walkErr != nil {
		t.Fatal(walkErr)
	}
	if len(got) != len(want) {
		t.Errorf("got entries %v, want %d", got, len(want))
	}
}

// TestGradedMode pins that only the graded sub-seller lists slabs.
func TestGradedMode(t *testing.T) {
	ha := NewScraper(&matcher.Backend{})
	for _, name := range ha.MarketNames() {
		info := ha.InfoForScraper(name)
		graded := name == "Hareruya Graded"
		if info.GradedMode != graded {
			t.Errorf("InfoForScraper(%q).GradedMode = %v", name, info.GradedMode)
		}
		wantName := name
		if graded {
			wantName = "Hareruya"
		}
		if info.Name != wantName {
			t.Errorf("InfoForScraper(%q).Name = %q, want %q", name, info.Name, wantName)
		}
	}
}
