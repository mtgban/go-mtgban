package bazaarofmagic

import (
	"os"
	"slices"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestParseListing(t *testing.T) {
	f, err := os.Open("testdata/listing.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatal(err)
	}

	total := parseTotal(doc)
	if total != 4174 {
		t.Errorf("total: got %d, want 4174", total)
	}

	want := []Listing{
		{ID: "9174849", Title: "A Good Day to Die (#438) (Klingon) (foil) (showcase)", Finish: "1", Price: 10},
		{ID: "9169717", Title: "Forest (#325)", Finish: "0", Price: 0},
		{ID: "4000001", Title: "Black Lotus", Finish: "0", Price: 40001},
		{ID: "9145362", Title: "Magic: The Gathering - Secret Lair Drop Series: Secret Lair x Sonic: Turbo Gear (foil)", Price: 71.96},
		// The product page's order, sale first.
		{ID: "9145927", Title: "Magic: The Gathering - Teenage Mutant Ninja Turtles Collector Booster Box (12 boosters)", Price: 299.70},
		{ID: "9166852", Title: "Magic: The Gathering - Star Trek Bundle", Price: 49.95},
	}
	got := parseRows(doc)
	for i := range got {
		got[i].URL = ""
	}
	if !slices.Equal(got, want) {
		t.Errorf("rows:\n got %+v\nwant %+v", got, want)
	}
}
