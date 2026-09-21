package coolstuffinc

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// pagingRow is a product row, so a test page can be told from an empty
// one. Only the count matters here, never what it says.
const pagingRow = `<div class="row product-search-row main-container"></div>`

// pagingServer answers a search and the pages after it from a table of
// URIs, recording what was asked for. A URI the table does not name
// answers a page with no rows, which is what the storefront does when it
// declines a request.
func pagingServer(t *testing.T, pages func(srvURL, uri string) (rows int, next string)) (*httptest.Server, *[]string) {
	t.Helper()

	var asked []string
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uri := r.URL.RequestURI()
		asked = append(asked, uri)

		rows, next := pages(srv.URL, uri)
		fmt.Fprint(w, "<html>")
		for range rows {
			fmt.Fprint(w, pagingRow)
		}
		if next != "" {
			fmt.Fprintf(w, `<span id="nextLink"><a href=%q></a></span>`, next)
		}
		fmt.Fprint(w, "</html>")
	}))
	t.Cleanup(srv.Close)

	saved := csiSearchURL
	csiSearchURL = srv.URL
	t.Cleanup(func() { csiSearchURL = saved })

	return srv, &asked
}

func walkSearch(t *testing.T) {
	t.Helper()
	csi := Coolstuffinc{client: newCSIHTTPClient(), shelf: GameMagic}
	results := make(chan responseChan, 1)
	err := csi.processSearch(context.Background(), results, "Coldsnap", nil)
	if err != nil {
		t.Fatalf("processSearch(): %v", err)
	}
}

func wantAsked(t *testing.T, asked, want []string) {
	t.Helper()
	if len(asked) != len(want) {
		t.Fatalf("asked for %v, want %v", asked, want)
	}
	for i := range want {
		if asked[i] != want[i] {
			t.Errorf("request %d was %q, want %q", i, asked[i], want[i])
		}
	}
}

// TestPagingAsksForTheLargerPage pins that the walk reads the first page
// again at the size the storefront's links honour, and follows its own
// links from there. The page number goes back to 1 with the size: page 2
// of a 50-row page is rows 51-100, so keeping the storefront's own page
// number would step over rows 26-50 without a word.
func TestPagingAsksForTheLargerPage(t *testing.T) {
	_, asked := pagingServer(t, func(srvURL, uri string) (int, string) {
		switch uri {
		case "/":
			return 25, srvURL + "/sq/12345&page=2"
		case "/sq/12345&resultsPerPage=50&page=1":
			return 50, srvURL + "/sq/12345&resultsPerPage=50&page=2"
		case "/sq/12345&resultsPerPage=50&page=2":
			return 20, ""
		}
		return 0, ""
	})

	walkSearch(t)

	wantAsked(t, *asked, []string{
		"/",
		"/sq/12345&resultsPerPage=50&page=1",
		"/sq/12345&resultsPerPage=50&page=2",
	})
}

// TestPagingFollowsTheStorefrontsLink pins that the pages are asked for
// as the storefront links them once it declines the larger page. It
// writes the page number onto the saved query with an "&" where a "?"
// belongs, and a link rebuilt from the id rather than followed is a
// guess at that spelling.
//
// A widened page with no rows on it is the storefront declining the
// size, not the shelf ending - the link being widened is the one it
// wrote to say there is more - so the walk falls back to it rather than
// stopping with the shelf half read.
func TestPagingFollowsTheStorefrontsLink(t *testing.T) {
	_, asked := pagingServer(t, func(srvURL, uri string) (int, string) {
		switch uri {
		case "/":
			return 25, srvURL + "/sq/12345&page=2"
		case "/sq/12345&page=2":
			return 25, srvURL + "/sq/12345&page=3"
		case "/sq/12345&page=3":
			return 10, ""
		}
		// Including the widened page, which this storefront declines.
		return 0, ""
	})

	walkSearch(t)

	wantAsked(t, *asked, []string{
		"/",
		"/sq/12345&resultsPerPage=50&page=1",
		"/sq/12345&page=2",
		"/sq/12345&page=3",
	})
}

// TestPagingKeepsItsOwnFirstPage pins the refusal that keeps the widening
// honest: a next-page link carrying no page number cannot be moved back to
// the first page, and following it unchanged would read the second page as
// the first and lose every row before it. The walk keeps the page it has.
func TestPagingKeepsItsOwnFirstPage(t *testing.T) {
	_, asked := pagingServer(t, func(srvURL, uri string) (int, string) {
		switch uri {
		case "/":
			return 25, srvURL + "/sq/12345&start=26"
		case "/sq/12345&start=26":
			return 10, ""
		}
		return 0, ""
	})

	walkSearch(t)

	wantAsked(t, *asked, []string{"/", "/sq/12345&start=26"})
}

// TestPagingReportsARefusedPage pins that a page answered with anything
// but success is said out loud. A refusal's body parses as a page holding
// no rows and linking nowhere, which reads as the shelf ending rather
// than as the error it is. The server errors were loud already - the
// client retries those and gives up with an error of its own - so the
// silent half-read shelf was the statuses it hands straight back.
func TestPagingReportsARefusedPage(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RequestURI() == "/" {
			fmt.Fprintf(w, `<html>%s<span id="nextLink"><a href=%q></a></span></html>`,
				pagingRow, srv.URL+"/sq/12345&page=2")
			return
		}
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `<html></html>`)
	}))
	t.Cleanup(srv.Close)

	saved := csiSearchURL
	csiSearchURL = srv.URL
	t.Cleanup(func() { csiSearchURL = saved })

	csi := Coolstuffinc{client: newCSIHTTPClient(), shelf: GameMagic}
	results := make(chan responseChan, 1)
	err := csi.processSearch(context.Background(), results, "Coldsnap", nil)
	if err == nil {
		t.Fatal("processSearch() said nothing about a refused page")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("processSearch() = %v, want the status in it", err)
	}
}

// TestPagingStopsOnASinglePage pins that a shelf whose results end on the
// first page costs one request. There is no link to follow, so there is
// nothing to widen either.
func TestPagingStopsOnASinglePage(t *testing.T) {
	_, asked := pagingServer(t, func(_, uri string) (int, string) {
		if uri == "/" {
			return 12, ""
		}
		return 0, ""
	})

	walkSearch(t)

	wantAsked(t, *asked, []string{"/"})
}
