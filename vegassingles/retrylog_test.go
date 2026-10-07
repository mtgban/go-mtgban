package vegassingles

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestRetryLinesFollowLogRetries pins that a scraper built through the
// registry reports a retried request only when the run asked for it.
func TestRetryLinesFollowLogRetries(t *testing.T) {
	for _, logRetries := range []bool{false, true} {
		t.Run(fmt.Sprint("logRetries=", logRetries), func(t *testing.T) {
			served := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				served++
				if served == 1 {
					w.Header().Set("Retry-After", "0")
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				_ = json.NewEncoder(w).Encode(VSResponse{})
			}))
			defer srv.Close()

			var lines []string
			opts := []mtgban.Option{mtgban.WithLogCallback(func(format string, a ...any) {
				lines = append(lines, fmt.Sprintf(format, a...))
			})}
			if logRetries {
				opts = append(opts, mtgban.WithLogRetries())
			}
			scraper, err := mtgban.NewScraper(&mtgmatcher.Backend{Game: "riftbound"}, "vegassingles", opts...)
			if err != nil {
				t.Fatal(err)
			}
			vs, ok := scraper.(*Vegassingles)
			if !ok {
				t.Fatalf("got %T", scraper)
			}
			vs.client.baseURL = srv.URL

			_, err = vs.client.getCount(context.Background(), "")
			if err != nil {
				t.Fatal(err)
			}
			if served != 2 {
				t.Fatalf("served %d requests, want a retry", served)
			}
			retried := len(lines) == 1 && strings.HasPrefix(lines[0], "[VS] GET http://") &&
				strings.HasSuffix(lines[0], ": retry 1 of 4")
			if retried != logRetries || len(lines) > 1 {
				t.Errorf("logged %q with logRetries %v", lines, logRetries)
			}
		})
	}
}
