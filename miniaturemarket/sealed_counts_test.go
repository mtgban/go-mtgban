package miniaturemarket

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"testing"
)

// secondPackListing lists the booster pack again at another price.
const secondPackListing = `<div class="product-info">
<a class="product-name" href="https://www.miniaturemarket.com/p/200">Flesh &amp; Blood TCG: High Seas - Booster Pack</a>
<input type="hidden" name="product-id" value="200">
<span class="product-price">$5.49</span>
</div>
`

// doubledTransport answers every request with the fixture page and the pack
// listed a second time.
type doubledTransport struct{}

func (doubledTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	data, err := os.ReadFile("testdata/sealed_match_page.html")
	if err != nil {
		return nil, err
	}
	data = append(data, secondPackListing...)
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       io.NopCloser(bytes.NewReader(data)),
		Request:    req,
	}, nil
}

// TestSealedPricedLine runs Load over the fixture page and pins the line it
// ends on. The box is placed by id, the pack, the blitz deck and the Silver
// Age deck by name, the pack listed twice (one product, two entries); the
// deck whose price does not parse is placed and never priced.
func TestSealedPricedLine(t *testing.T) {
	mm, err := NewScraperSealed(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	mm.client = &http.Client{Transport: doubledTransport{}}
	var logs []string
	mm.logCallback = func(format string, a ...any) {
		logs = append(logs, fmt.Sprintf(format, a...))
	}

	err = mm.Load(context.Background())
	if err != nil {
		t.Fatalf("Load(): %v", err)
	}
	if len(mm.inventory["sea-pack"]) != 2 {
		t.Fatalf("the pack holds %v, want both listings", mm.inventory["sea-pack"])
	}
	want := "[MMSealed] Priced 1 products by id, 0 through a bridge, 3 by name"
	if len(logs) == 0 || logs[len(logs)-1] != want || slices.Index(logs, want) != len(logs)-1 {
		t.Errorf("logged %q, want it to end on %q, once", logs, want)
	}
}
