package coolstuffinc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
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
			if result.NextLink != "/sq/12345&page=2" {
				t.Errorf("NextLink = %q, want %q", result.NextLink, "/sq/12345&page=2")
			}
			if attempts != 3 {
				t.Errorf("served %d attempts, want 3", attempts)
			}
		})
	}
}

// TestRetriesReachTheScraperLog pins that a retried request is reported once,
// through the log callback the scraper is registered with, tagged and without
// its query, and that a request answered first time reports nothing.
func TestRetriesReachTheScraperLog(t *testing.T) {
	var served atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if served.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		fmt.Fprint(w, "<html></html>")
	}))
	defer srv.Close()

	saved := csiSearchURL
	csiSearchURL = srv.URL + "/sq/?sig=secret"
	defer func() { csiSearchURL = saved }()

	var logged []string
	scraper, err := mtgban.NewScraper(&mtgmatcher.Backend{Game: mtgmatcher.GameMagic}, "coolstuffinc",
		mtgban.WithLogCallback(func(format string, a ...any) {
			logged = append(logged, fmt.Sprintf(format, a...))
		}))
	if err != nil {
		t.Fatal(err)
	}
	csi, ok := scraper.(*Coolstuffinc)
	if !ok {
		t.Fatalf("built a %T", scraper)
	}

	// The first search is retried once, the second is answered first time.
	for range 2 {
		err = csi.processSearch(context.Background(), make(chan responseChan), "Coldsnap", nil)
		if err != nil {
			t.Fatal(err)
		}
	}
	if served.Load() != 3 {
		t.Errorf("served %d requests, want 3", served.Load())
	}
	want := "[CSI] POST " + srv.URL + "/sq/: retry 1 of "
	if len(logged) != 1 || !strings.HasPrefix(logged[0], want) || strings.Contains(logged[0], "secret") {
		t.Errorf("logged %q, want one line starting %q", logged, want)
	}
}
