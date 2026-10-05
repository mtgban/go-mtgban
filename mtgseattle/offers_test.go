package mtgseattle

import (
	"os"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"

	"github.com/PuerkitoBio/goquery"
)

// A product stocked in two grades lists only the NM copy in its grid block;
// both must come out, each at its own price less the site discount.
func TestInventoryOffersReadsEveryVariant(t *testing.T) {
	f, err := os.Open("testdata/multi_variant.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatal(err)
	}

	var ms MTGSeattle
	var got []offer
	doc.Find(`ul[class="products"] li[class="product"] div[class="inner"] div[class="meta"]`).Each(func(_ int, s *goquery.Selection) {
		got = append(got, ms.inventoryOffers(s)...)
	})

	want := []offer{
		{conditions: mtgban.NM, price: 69.99 * 0.95, qty: 1},
		{conditions: mtgban.SP, price: 59.99 * 0.95, qty: 1},
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
