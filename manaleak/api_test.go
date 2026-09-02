package manaleak

import (
	"os"
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

	products := parseListing(doc)
	if len(products) != 3 {
		t.Fatalf("parsed %d products, want 3", len(products))
	}

	first := products[0]
	if first.Name != "Hadoken - Lightning Bolt" ||
		first.SetName != "Secret Lair" ||
		first.TCGProductID != "272554" ||
		first.MultiverseID != "" ||
		first.Price != 7.51 ||
		!first.OutOfStock {
		t.Errorf("first row parsed as %+v", first)
	}

	if m := showingTotal.FindStringSubmatch("Showing 1 to 58 of 58 (1 Pages)"); m == nil || m[1] != "58" {
		t.Errorf("showing total parsed as %v", m)
	}
}
