package vegassingles

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// storefront answers pages of a catalog of total products laid out size to a
// page, and fails the page failAt whenever that is not zero. Pages past the
// catalog come back empty, which is how the real storefront ends a feed.
type storefront struct {
	total  int
	size   int
	failAt int
}

func (s *storefront) handler(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := strconv.Atoi(r.URL.Query().Get("page"))
		if err != nil {
			t.Errorf("page %q: %v", r.URL.Query().Get("page"), err)
			return
		}
		if page == s.failAt {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var products []VSProduct
		for i := (page - 1) * s.size; i < page*s.size && i < s.total; i++ {
			products = append(products, VSProduct{ID: fmt.Sprint(i)})
		}
		_ = json.NewEncoder(w).Encode(VSResponse{Products: products})
	}
}

func (s *storefront) scraper(t *testing.T) (*Vegassingles, func()) {
	t.Helper()
	srv := httptest.NewServer(s.handler(t))
	vs := NewScraper(GameRiftbound)
	client := *vs.client
	client.baseURL = srv.URL
	vs.client = &client
	// One page at a time, so the pages a crawl walks are the pages it needed
	// rather than whatever a fan-out happened to reach first.
	vs.MaxConcurrency = 1
	return vs, srv.Close
}

// TestCrawlReadsTheCutOffTheServedPageSize covers how a crawl tells a catalog
// that ended from one the storefront's result window cut off. The page size
// used to be written here as a constant, so a storefront serving anything else
// would have made every last page look ragged: no cut is ever reported, the
// passes that widen the crawl stop running, and nothing says so. The size the
// storefront is actually serving is in the pages it served.
func TestCrawlReadsTheCutOffTheServedPageSize(t *testing.T) {
	for _, tt := range []struct {
		name        string
		total, size int
		want        bool
	}{
		// The layout the storefront serves today, ending ragged and ending
		// flush.
		{"a ragged last page", 100, 24, false},
		{"an exact multiple of the page size", 96, 24, true},
		// The same two catalogs behind a storefront that resized its page.
		{"a ragged last page of a resized feed", 100, 12, false},
		{"an exact multiple of a resized page", 96, 12, true},
		// A catalog small enough to be one page says nothing either way,
		// and must not be read as cut off every single run.
		{"a single page", 7, 24, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := &storefront{total: tt.total, size: tt.size}
			vs, done := store.scraper(t)
			defer done()

			cut := vs.crawl(context.Background(), sortForward, "", 1,
				map[string]bool{}, map[string]bool{})
			if cut != tt.want {
				t.Errorf("crawl reported cut=%v, want %v", cut, tt.want)
			}
		})
	}
}

// TestCrawlSurvivesAFailedTailPage covers the walk past the fanned-out range,
// where a page that would not load used to end the whole scrape: the error
// travelled up through Load and threw away every product the run had already
// priced, while the very same failure inside the fanned range was logged and
// the page skipped. The walk cannot skip a page and stay finite - only an
// empty page ends it - so it stops instead, and what it collected stands.
func TestCrawlSurvivesAFailedTailPage(t *testing.T) {
	store := &storefront{total: 100, size: 24, failAt: 3}
	vs, done := store.scraper(t)
	defer done()

	// The bare products the storefront hands back are turned down one by
	// one, which is not what this is about; keep what the crawl says about
	// the pages themselves.
	var reported []string
	vs.LogCallback = func(format string, a ...any) {
		line := fmt.Sprintf(format, a...)
		if strings.Contains(line, "process error") {
			return
		}
		reported = append(reported, line)
	}

	seen := map[string]bool{}
	cut := vs.crawl(context.Background(), sortForward, "", 2, seen, map[string]bool{})
	if len(seen) != 48 {
		t.Errorf("the crawl kept %d products, want the 48 it read", len(seen))
	}
	if !cut {
		t.Error("a feed with a page nobody could read was reported as finished")
	}
	want := "[VS] page 3: unexpected status code: 404"
	if len(reported) != 1 || reported[0] != want {
		t.Errorf("the crawl said %q, want just %q", reported, want)
	}
}
