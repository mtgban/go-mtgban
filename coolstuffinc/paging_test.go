package coolstuffinc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestPagingFollowsTheStorefrontsLink pins that the pages after the first
// are asked for as the storefront links them. It writes the page number
// onto the saved query with an "&" where a "?" belongs, and a link
// rebuilt from the id rather than followed is a guess at that spelling.
func TestPagingFollowsTheStorefrontsLink(t *testing.T) {
	var asked []string

	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, r.URL.RequestURI())

		// The rows themselves are beside the point here: what is
		// pinned is which pages get asked for, and the last one
		// carries no link on.
		next := ""
		switch r.URL.RequestURI() {
		case "/":
			next = srv.URL + "/sq/12345&page=2"
		case "/sq/12345&page=2":
			next = srv.URL + "/sq/12345&page=3"
		}
		if next == "" {
			fmt.Fprint(w, `<html></html>`)
			return
		}
		fmt.Fprintf(w, `<html><span id="nextLink"><a href=%q></a></span></html>`, next)
	}))
	defer srv.Close()

	saved := csiSearchURL
	csiSearchURL = srv.URL
	defer func() { csiSearchURL = saved }()

	csi := Coolstuffinc{client: newCSIHTTPClient(), shelf: GameMagic}
	results := make(chan responseChan, 1)
	err := csi.processSearch(context.Background(), results, "Coldsnap", nil)
	if err != nil {
		t.Fatalf("processSearch(): %v", err)
	}

	want := []string{"/", "/sq/12345&page=2", "/sq/12345&page=3"}
	if len(asked) != len(want) {
		t.Fatalf("asked for %v, want %v", asked, want)
	}
	for i := range want {
		if asked[i] != want[i] {
			t.Errorf("request %d was %q, want %q", i, asked[i], want[i])
		}
	}
}
