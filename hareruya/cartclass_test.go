package hareruya

import (
	"bytes"
	"os"
	"slices"
	"testing"

	"github.com/PuerkitoBio/goquery"

	"github.com/mtgban/go-mtgban/mtgban"
)

func fixture(t *testing.T, name string) *goquery.Document {
	t.Helper()
	page, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(page))
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

// Each row keeps its own class; the second lot's main row has no button,
// and must not take its extra row's
func TestLazyRowClasses(t *testing.T) {
	got := (&Hareruya{}).lazyResults(fixture(t, "lazy_classes.html"))
	want := [][]Row{
		{
			{Quantity: 4, Condition: mtgban.NM, Price: 1000, Class: "27947"},
			{Quantity: 1, Condition: mtgban.SP, Price: 700, Class: "27948"},
		},
		{
			{Quantity: 1, Condition: mtgban.NM, Price: 300},
			{Quantity: 27, Condition: mtgban.SP, Price: 250, Class: "18808"},
		},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d blocks, want %d", len(got), len(want))
	}
	for i := range want {
		if !slices.Equal(got[i].Rows, want[i]) {
			t.Errorf("block %d rows = %+v, want %+v", i, got[i].Rows, want[i])
		}
	}
}

func TestPurchaseCartClass(t *testing.T) {
	got := cartClass(fixture(t, "purchase_class.html").Find(".itemList"))
	if got != "452235" {
		t.Errorf("cartClass = %q, want 452235", got)
	}
}
