package starcitygames

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// A catalog export that breaks partway through returns a reader that has
// already answered 200, so the client's request-level retry never sees it.
// Serve a truncated array first and the whole one after, and check the second
// pass is the one that counts.
func TestStreamCatalogResumesAfterBrokenStream(t *testing.T) {
	const full = `[{"id":1,"sku":"a"},{"id":2,"sku":"b"},{"id":3,"sku":"c"}]`

	var served int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served++
		w.Header().Set("Content-Type", "application/json")
		if served == 1 {
			// Cut the array mid-product and hang up, the way a dropped
			// connection leaves it.
			_, _ = w.Write([]byte(`[{"id":1,"sku":"a"},{"id":2,"s`))
			w.(http.Flusher).Flush()
			if hijacker, ok := w.(http.Hijacker); ok {
				conn, _, err := hijacker.Hijack()
				if err == nil {
					_ = conn.Close()
				}
			}
			return
		}
		_, _ = w.Write([]byte(full))
	}))
	defer srv.Close()

	scg := NewSCGClient("key")
	client := *scg
	client.catalogURL = srv.URL

	var resets int
	var got []int
	err := client.StreamCatalog(context.Background(),
		func() { resets++; got = nil },
		func(p CatalogProduct) error {
			got = append(got, p.ID)
			return nil
		})
	if err != nil {
		t.Fatalf("StreamCatalog: %v", err)
	}
	if served != 2 {
		t.Errorf("served %d catalogs, want 2", served)
	}
	if resets != 1 {
		t.Errorf("reset %d times, want 1", resets)
	}
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Errorf("got %v, want the three products of the whole export", got)
	}
}

// A cancelled context is the caller giving up, not a flaky peer, so it must
// come straight back rather than spend the replay budget.
func TestStreamCatalogHonoursCancellation(t *testing.T) {
	var served int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served++
	}))
	defer srv.Close()

	scg := NewSCGClient("key")
	client := *scg
	client.catalogURL = srv.URL

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.StreamCatalog(ctx, func() {}, func(CatalogProduct) error {
		t.Error("no product should reach the callback")
		return nil
	})
	if err == nil {
		t.Fatal("StreamCatalog returned no error for a cancelled context")
	}
	if served != 0 {
		t.Errorf("served %d catalogs for a cancelled context, want 0", served)
	}
}
