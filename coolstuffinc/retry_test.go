package coolstuffinc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestSearchSurvivesTransientFailures pins that a search whose first
// request fails is retried rather than abandoned. The first page of a
// search stands for a whole edition: abandoning it drops every row of
// that edition silently, which a run of the storefront's 445 Magic
// editions did twice in one evening.
func TestSearchSurvivesTransientFailures(t *testing.T) {
	const page = `<html><span id="nextLink"><a href="/sq/12345&page=2"></a></span></html>`

	for _, tt := range []struct {
		desc string
		fail func(w http.ResponseWriter)
	}{
		{"the storefront answers with a server error", func(w http.ResponseWriter) {
			w.WriteHeader(http.StatusInternalServerError)
		}},
		{"the storefront drops the connection", func(w http.ResponseWriter) {
			hijacker, ok := w.(http.Hijacker)
			if !ok {
				return
			}
			conn, _, err := hijacker.Hijack()
			if err == nil {
				_ = conn.Close()
			}
		}},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			var attempts int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts++
				if attempts < 3 {
					tt.fail(w)
					return
				}
				_, _ = w.Write([]byte(page))
			}))
			defer srv.Close()

			saved := csiSearchURL
			csiSearchURL = srv.URL
			defer func() { csiSearchURL = saved }()

			result, err := Search(context.Background(), GameMagic, "Coldsnap", false, nil)
			if err != nil {
				t.Fatalf("Search() after %d attempts: %v", attempts, err)
			}
			if result.PageID != "12345" {
				t.Errorf("PageID = %q, want %q", result.PageID, "12345")
			}
			if attempts != 3 {
				t.Errorf("served %d attempts, want 3", attempts)
			}
		})
	}
}
