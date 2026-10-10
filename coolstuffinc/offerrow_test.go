package coolstuffinc

import (
	"bytes"
	"os"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestOfferRowID(t *testing.T) {
	page, err := os.ReadFile("testdata/offer_rows.html")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	doc.Find(`div[itemprop="offers"]`).Each(func(_ int, offer *goquery.Selection) {
		got = append(got, offerRowID(offer))
	})
	want := []string{"10753983", "10753987", ""}
	if len(got) != len(want) {
		t.Fatalf("got %d rows, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d id = %q, want %q", i, got[i], want[i])
		}
	}
}
