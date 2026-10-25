package hareruya

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

// The search answers every page past its last one as the last one,
// renumbered. Reading stops there, and fails when listings remain past it
// or the answer carries no page number.
func TestSearchStopsAtRenumberedPage(t *testing.T) {
	for _, tt := range []struct {
		name     string
		numFound int
		noPage   bool
		fails    bool
		capped   bool
	}{
		{name: "every listing served", numFound: 2*searchRows + 1},
		{name: "listings past the last page", numFound: 4 * searchRows, fails: true, capped: true},
		{name: "no page number", noPage: true, fails: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			const lastPage = 3
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				page, _ := strconv.Atoi(r.URL.Query().Get("page"))
				page = min(page, lastPage)
				if tt.noPage {
					page = 0
				}
				fmt.Fprintf(w, `{"response":{"numFound":%d,"page":%d,"docs":[{"product":"%d"}]}}`, tt.numFound, page, page)
			}))
			defer srv.Close()

			start := lastPage + 1
			if tt.noPage {
				start = 1
			}
			for page := 1; page < start; page++ {
				docs, err := search(context.Background(), srv.Client(), fmt.Sprintf("%s?page=%d", srv.URL, page), page)
				if err != nil || len(docs) != 1 {
					t.Fatalf("page %d: got %d docs, %v", page, len(docs), err)
				}
			}
			docs, err := search(context.Background(), srv.Client(), fmt.Sprintf("%s?page=%d", srv.URL, start), start)
			if (err != nil) != tt.fails || docs != nil {
				t.Errorf("page %d: got %d docs, %v, want failure %v", start, len(docs), err, tt.fails)
			}
			var capErr *SearchCapError
			capped := errors.As(err, &capErr)
			if capped != tt.capped || (capped && capErr.NumFound != tt.numFound) {
				t.Errorf("page %d: cap error %v for %v", start, capped, err)
			}
		})
	}
}

// A set read again once per color sends each listing once, a multicolored
// card included.
func TestUnsent(t *testing.T) {
	sent := map[string]bool{}
	first := []Product{{Product: "1"}, {Product: "2", ProductClass: "a"}, {Product: "2", ProductClass: "b"}}
	kept := unsent(first, sent)
	if len(kept) != len(first) {
		t.Errorf("first reading kept %d of %d", len(kept), len(first))
	}
	second := []Product{{Product: "2", ProductClass: "b"}, {Product: "3"}}
	got := unsent(second, sent)
	if len(got) != 1 || got[0].Product != "3" {
		t.Errorf("second reading kept %v", got)
	}
}
