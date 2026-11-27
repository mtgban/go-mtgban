package coolstuffinc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// searchServer points the search at a server answering each query with
// page, except the query named in failing, answered with a 500.
func searchServer(t *testing.T, page []byte, failing string) *atomic.Int64 {
	t.Helper()
	var asked atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked.Add(1)
		if failing != "" && r.FormValue("name") == failing {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(page)
	}))
	t.Cleanup(srv.Close)
	saved := csiSearchURL
	csiSearchURL = srv.URL
	t.Cleanup(func() { csiSearchURL = saved })
	return &asked
}

// fetchExport runs the export's fetch and match over the search server and
// the fixture buylist.
func fetchExport(t *testing.T) ([]mtgban.SealedMatch, error) {
	t.Helper()
	csi, err := newSealedMatcher(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	csi.client = &http.Client{Transport: buylistOrSearch{fixtureTransport{
		fmt.Sprintf(csiBuylistURL, csi.shelf): "testdata/sealed_match_buylist.json",
	}}}
	return csi.matchSealedCatalog(context.Background())
}

// TestMatchSealedCatalogFetches drives the export's fetch: every query of
// sealedQueries answers the same rows, merged to one match per id with the
// buylist's, as the rows test pins them.
func TestMatchSealedCatalogFetches(t *testing.T) {
	page, err := os.ReadFile("testdata/sealed_match_search.html")
	if err != nil {
		t.Fatal(err)
	}
	asked := searchServer(t, page, "")
	got, err := fetchExport(t)
	if err != nil {
		t.Fatalf("matchSealedCatalog(): %v", err)
	}
	if asked.Load() != int64(len(sealedQueries)) {
		t.Errorf("asked %d searches, want %d", asked.Load(), len(sealedQueries))
	}
	want := sealedMatchExport(t, sealedMatchBackend(t))
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestMatchSealedCatalogLostSearch pins one search failing the export rather
// than leaving its rows out.
func TestMatchSealedCatalogLostSearch(t *testing.T) {
	page, err := os.ReadFile("testdata/sealed_match_search.html")
	if err != nil {
		t.Fatal(err)
	}
	searchServer(t, page, "kit")
	got, err := fetchExport(t)
	if err == nil {
		t.Fatalf("got %d matches, want an error", len(got))
	}
}

// TestMatchSealedCatalogNoRows pins searches answering no row at all failing
// the export: a refused search answers its form with none.
func TestMatchSealedCatalogNoRows(t *testing.T) {
	searchServer(t, []byte("<html><body></body></html>"), "")
	got, err := fetchExport(t)
	if err == nil {
		t.Fatalf("got %d matches, want an error", len(got))
	}
}

// TestMatchSealedCatalogNoID pins a row of either feed carrying no id left
// out, where it would otherwise report a product under an empty id.
func TestMatchSealedCatalogNoID(t *testing.T) {
	csi, err := newSealedMatcher(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	search := []sealedRow{
		{name: "Structure Deck Albaz Strike"},
		{pid: "30", name: "Dragons of Legend Tin"},
	}
	buylist := []CSIPriceEntry{
		{Name: "Dragons of Legend Tin", RarityName: "Box"},
		{Name: "Mystery Bundle", RarityName: "Box"},
	}
	got := csi.matchCatalog(search, buylist)
	want := []mtgban.SealedMatch{
		{StoreID: "30", Name: "Dragons of Legend Tin", UUID: "ygo-tin", Via: mtgban.SealedByName},
	}
	if len(got) != len(want) || got[0] != want[0] {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// TestMatchSealedCatalogNoPIDs pins searches answering rows none of which
// carries a product id failing the export: the store moved the id in its
// markup, and reporting the buylist alone would read every search-only id
// as gone.
func TestMatchSealedCatalogNoPIDs(t *testing.T) {
	page := []byte(`<div class="row product-search-row main-container">
<span itemprop="name">Structure Deck Albaz Strike</span>
<span class="rating-display moved" data-pid="300"></span>
<link itemprop="availability" content="https://schema.org/InStock">
</div>`)
	searchServer(t, page, "")
	got, err := fetchExport(t)
	if err == nil {
		t.Fatalf("got %d matches, want an error", len(got))
	}
	if !strings.Contains(err.Error(), "markup") {
		t.Errorf("error %q does not say the markup may have changed", err)
	}
}

// TestMatchSealedCatalogNonePlaced pins searches whose rows carry ids and
// place nothing as a legitimate answer: the buylist's boxes alone.
func TestMatchSealedCatalogNonePlaced(t *testing.T) {
	page := []byte(`<div class="row product-search-row main-container">
<span itemprop="name">Blue-Eyes White Dragon</span>
<span class="rating-display " data-pid="500"></span>
<link itemprop="availability" content="https://schema.org/InStock">
</div>`)
	searchServer(t, page, "")
	got, err := fetchExport(t)
	if err != nil {
		t.Fatalf("matchSealedCatalog(): %v", err)
	}
	for _, match := range got {
		if match.StoreID == "500" {
			t.Errorf("reported the card row: %+v", match)
		}
	}
	if len(got) == 0 {
		t.Error("got no matches, want the buylist's boxes")
	}
}
