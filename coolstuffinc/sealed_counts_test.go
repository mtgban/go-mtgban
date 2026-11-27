package coolstuffinc

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"
)

// buylistOrSearch answers the buylist download with the fixture and hands
// every other request, the searches, to the fixture server.
type buylistOrSearch struct {
	buylist fixtureTransport
}

func (f buylistOrSearch) RoundTrip(req *http.Request) (*http.Response, error) {
	_, found := f.buylist[req.URL.String()]
	if found {
		return f.buylist.RoundTrip(req)
	}
	return http.DefaultTransport.RoundTrip(req)
}

// TestSealedPricedLine runs Load over the fixture feeds and pins the line it
// ends on. The box sells and is bought (one product, two entries), the tins
// and the disputed id sell or are bought by name; the deck is out of stock,
// placed and never priced, and every search answering the same rows prices
// nothing twice.
func TestSealedPricedLine(t *testing.T) {
	csi, err := NewScraperSealed(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	searchFixture(t, "testdata/sealed_match_search.html")
	csi.client = &http.Client{Transport: buylistOrSearch{fixtureTransport{
		fmt.Sprintf(csiBuylistURL, csi.shelf): "testdata/sealed_match_buylist.json",
	}}}
	var logs []string
	csi.logCallback = func(format string, a ...any) {
		logs = append(logs, fmt.Sprintf(format, a...))
	}

	err = csi.Load(context.Background())
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if len(csi.inventory["ygo-box"]) == 0 || len(csi.buylist["ygo-box"]) == 0 {
		t.Fatalf("the box is not both sold and bought: %v, %v", csi.inventory, csi.buylist)
	}
	want := "[CSISealed] Priced 1 products by id, 0 through a bridge, 3 by name"
	if len(logs) == 0 || logs[len(logs)-1] != want || slices.Index(logs, want) != len(logs)-1 {
		t.Errorf("logged %q, want it to end on %q, once", logs, want)
	}
}
