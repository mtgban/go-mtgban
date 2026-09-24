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

// TestCardImage pins every image-path shape manaleak.com serves through
// readImage: three that carry an id (TCGplayer, multiverse, and an
// unconfirmed TCGplayer id from the bare "-WxH" form), and two that carry
// none - a promo-code filename and a sealed product. Paths copied verbatim
// from manaleak.com.
func TestCardImage(t *testing.T) {
	for _, tt := range []struct {
		src       string
		tcg, mv   string
		ambiguous bool
	}{
		// TCGplayer-id era, two-segment folder.
		{src: "https://cdn.ionsuite.com/image/ion-echo/data/mtg/sld/272554_200w-250x250.jpg", tcg: "272554"},
		// Multiverse-id era, two-segment folder.
		{src: "https://cdn.ionsuite.com/image/ion-echo/data/mtg/KTK/386701.full-250x250.jpg", mv: "386701"},
		// Multiverse-id era, folder glued onto "mtg" with no slash.
		{src: "https://cdn.ionsuite.com/image/ion-echo/data/mtghourofdevastation/430768.full-250x250.jpg", mv: "430768"},
		// Bare "-WxH" form, neither marker: read as an unconfirmed TCGplayer id.
		{src: "https://cdn.ionsuite.com/image/ion-echo/data/mtg/eternal-masters/118431-250x250.jpg", tcg: "118431", ambiguous: true},
		// A promo-code filename names no numeric id.
		{src: "https://cdn.ionsuite.com/image/ion-echo/data/mtg/DM/DM_9.full-250x250.jpg"},
		// Sealed product and repacks live outside any "mtg" folder.
		{src: "https://cdn.ionsuite.com/image/manaleak/data/Sealed Products/Reality Fracture/FRA Bundle-250x250.jpg"},
	} {
		var p MLProduct
		p.readImage(tt.src)
		if p.TCGProductID != tt.tcg || p.MultiverseID != tt.mv || p.AmbiguousID != tt.ambiguous {
			t.Errorf("%s: got tcg=%q mv=%q ambiguous=%v, want tcg=%q mv=%q ambiguous=%v",
				tt.src, p.TCGProductID, p.MultiverseID, p.AmbiguousID, tt.tcg, tt.mv, tt.ambiguous)
		}
	}
}
