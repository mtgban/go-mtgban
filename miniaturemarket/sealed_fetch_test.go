package miniaturemarket

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// walkPages are the three pages of a catalog the widget paginates, the
// first carrying the pagination link to the last.
var walkPages = map[string]string{
	"1": `<a class="page-link" href="?p=2">2</a><a class="page-link" href="?p=3">3</a>
<div class="product-info">
<a class="product-name" href="https://www.miniaturemarket.com/p/100">Mystery Assortment</a>
<input type="hidden" name="product-id" value="100">
</div>`,
	"2": `<div class="product-info">
<a class="product-name" href="https://www.miniaturemarket.com/p/200">Flesh &amp; Blood TCG: High Seas - Booster Pack</a>
<input type="hidden" name="product-id" value="200">
</div>`,
	"3": `<div class="product-info">
<a class="product-name" href="https://www.miniaturemarket.com/p/700">Flesh &amp; Blood TCG: Mystery Bundle</a>
<input type="hidden" name="product-id" value="700">
</div>`,
}

// walkTransport answers each page of walkPages, and the status given for
// the pages named in failing.
type walkTransport struct {
	failing map[string]int
}

func (wt walkTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	page := req.URL.Query().Get("p")
	status := http.StatusOK
	body := walkPages[page]
	if code, found := wt.failing[page]; found {
		status = code
		body = "<html><body>Access denied</body></html>"
	}
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

// walkExport runs the export's fetch and match over the transport.
func walkExport(t *testing.T, rt http.RoundTripper) ([]mtgban.SealedMatch, error) {
	t.Helper()
	mm, err := newSealedMatcher(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	mm.client = &http.Client{Transport: rt}
	return mm.matchSealedCatalog(context.Background())
}

// TestMatchSealedCatalogWalksEveryPage pins the export reading every page
// of a multi-page catalog and matching what it found.
func TestMatchSealedCatalogWalksEveryPage(t *testing.T) {
	got, err := walkExport(t, walkTransport{})
	if err != nil {
		t.Fatalf("matchSealedCatalog(): %v", err)
	}
	want := []mtgban.SealedMatch{
		{StoreID: "100", Name: "Mystery Assortment", UUID: "sea-box", Via: mtgban.SealedByID},
		{StoreID: "200", Name: "Flesh & Blood TCG: High Seas - Booster Pack", UUID: "sea-pack", Via: mtgban.SealedByName},
		{StoreID: "700", Name: "Flesh & Blood TCG: Mystery Bundle", Via: mtgban.SealedRefused, Reason: matcher.ErrCardDoesNotExist.Error()},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestMatchSealedCatalogLostPage pins a page answered with an error status
// failing the whole export rather than leaving its listings out.
func TestMatchSealedCatalogLostPage(t *testing.T) {
	got, err := walkExport(t, walkTransport{failing: map[string]int{"2": http.StatusInternalServerError}})
	if err == nil {
		t.Fatalf("got %d matches, want an error", len(got))
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error %q does not name the status", err)
	}
}

// TestMatchSealedCatalogBlockedStore pins a store answering every request
// with a refusal page failing the export instead of reporting nothing.
func TestMatchSealedCatalogBlockedStore(t *testing.T) {
	blocked := map[string]int{"1": http.StatusForbidden}
	got, err := walkExport(t, walkTransport{failing: blocked})
	if err == nil {
		t.Fatalf("got %d matches, want an error", len(got))
	}
}

// TestMatchSealedCatalogNoListings pins a walk that reads its pages and finds
// no listing on any failing the export: the widget always lists something.
func TestMatchSealedCatalogNoListings(t *testing.T) {
	got, err := walkExport(t, emptyTransport{})
	if err == nil {
		t.Fatalf("got %d matches, want an error", len(got))
	}
}

// emptyTransport answers every request with a page that parses and lists
// nothing.
type emptyTransport struct{}

func (emptyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("<html><body></body></html>")),
		Request:    req,
	}, nil
}

// countFailsTransport answers the first request, the page count, with a
// 500, and every later one with the page walkPages holds.
type countFailsTransport struct {
	served *atomic.Int64
}

func (ct countFailsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if ct.served.Add(1) == 1 {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Header:     http.Header{},
			Body:       io.NopCloser(strings.NewReader("<html><body>Try again later</body></html>")),
			Request:    req,
		}, nil
	}
	return walkTransport{}.RoundTrip(req)
}

// TestMatchSealedCatalogPageCountFails pins the page count's own status
// check: a refused count read as a one-page catalog would walk page 1 alone,
// which loads fine, and report a short answer.
func TestMatchSealedCatalogPageCountFails(t *testing.T) {
	var served atomic.Int64
	got, err := walkExport(t, countFailsTransport{served: &served})
	if err == nil {
		t.Fatalf("got %d matches, want an error", len(got))
	}
	if served.Load() != 1 {
		t.Errorf("made %d requests, want the page count alone", served.Load())
	}
}
