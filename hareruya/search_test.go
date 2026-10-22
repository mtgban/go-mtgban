package hareruya

import (
	"context"
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
	}{
		{name: "every listing served", numFound: 2*60 + 1},
		{name: "listings past the last page", numFound: 4 * 60, fails: true},
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
		})
	}
}
