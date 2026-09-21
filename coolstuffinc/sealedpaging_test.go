package coolstuffinc

import (
	"context"
	"testing"
)

// walkSealedSearch drives the sealed name search over whatever
// pagingServer is answering. The rows it reads are empty, so the walk
// never reaches the resolver and needs no datastore: what these pin is
// which pages get asked for.
func walkSealedSearch(t *testing.T) {
	t.Helper()
	csi := Sealed{client: newCSIHTTPClient(), shelf: GameMagic}
	channel := make(chan responseChan, 1)
	err := csi.processSealedSearch(context.Background(), channel, "Booster Box")
	if err != nil {
		t.Fatalf("processSealedSearch(): %v", err)
	}
}

// TestSealedPagingAsksForTheLargerPage pins that the sealed walk reads
// its first page again at the larger size, the same way the singles walk
// does. The two share one helper and one shape, and only the singles one
// was pinned.
func TestSealedPagingAsksForTheLargerPage(t *testing.T) {
	_, asked := pagingServer(t, func(srvURL, uri string) (int, string) {
		switch uri {
		case "/":
			return 25, srvURL + "/sq/12345&page=2"
		case "/sq/12345&resultsPerPage=50&page=1":
			return 50, srvURL + "/sq/12345&resultsPerPage=50&page=2"
		case "/sq/12345&resultsPerPage=50&page=2":
			return 14, ""
		}
		return 0, ""
	})

	walkSealedSearch(t)

	wantAsked(t, *asked, []string{
		"/",
		"/sq/12345&resultsPerPage=50&page=1",
		"/sq/12345&resultsPerPage=50&page=2",
	})
}

// TestSealedPagingFollowsTheStorefrontsLink pins the fallback. A sealed
// walk that stopped at the widened page would price a query's first 25
// products and say nothing about the rest, which is the failure this
// whole shape is guarded against.
func TestSealedPagingFollowsTheStorefrontsLink(t *testing.T) {
	_, asked := pagingServer(t, func(srvURL, uri string) (int, string) {
		switch uri {
		case "/":
			return 25, srvURL + "/sq/12345&page=2"
		case "/sq/12345&page=2":
			return 10, ""
		}
		// Including the widened page, which this storefront declines.
		return 0, ""
	})

	walkSealedSearch(t)

	wantAsked(t, *asked, []string{
		"/",
		"/sq/12345&resultsPerPage=50&page=1",
		"/sq/12345&page=2",
	})
}

// TestSealedPagingStopsOnASinglePage pins that a query whose products end
// on the first page costs one request, with no link to widen.
func TestSealedPagingStopsOnASinglePage(t *testing.T) {
	_, asked := pagingServer(t, func(_, uri string) (int, string) {
		if uri == "/" {
			return 8, ""
		}
		return 0, ""
	})

	walkSealedSearch(t)

	wantAsked(t, *asked, []string{"/"})
}
