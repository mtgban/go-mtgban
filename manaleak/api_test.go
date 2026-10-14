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
		first.ProductID != "112651" ||
		first.MultiverseID != "" ||
		first.Price != 7.51 ||
		!first.OutOfStock {
		t.Errorf("first row parsed as %+v", first)
	}

	if m := showingTotal.FindStringSubmatch("Showing 1 to 58 of 58 (1 Pages)"); m == nil || m[1] != "58" {
		t.Errorf("showing total parsed as %v", m)
	}
}

// TestParseBuylistListing pins a buylist row, copied verbatim from the
// buylist category listing, whose link carries the product id the buylist
// cart takes, as a retail row's does.
func TestParseBuylistListing(t *testing.T) {
	f, err := os.Open("testdata/buylist_row.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatal(err)
	}

	products := parseListing(doc)
	if len(products) != 1 {
		t.Fatalf("parsed %d products, want 1", len(products))
	}
	row := products[0]
	if row.Name != "Aboshan's Desire" || row.ProductID != "13424" || row.MultiverseID != "29728" || row.Price != 0.01 {
		t.Errorf("buylist row parsed as %+v", row)
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

// TestParseProduct pins the category and code read off a product page, copied
// verbatim from manaleak.com.
func TestParseProduct(t *testing.T) {
	f, err := os.Open("testdata/product.html")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	doc, err := goquery.NewDocumentFromReader(f)
	if err != nil {
		t.Fatal(err)
	}

	category, code := parseProduct(doc)
	if category != "The List" || code != "mtgTLP269131" {
		t.Errorf("product page parsed as %q, %q", category, code)
	}
}

// TestReadPage pins every product code shape manaleak.com serves, copied
// verbatim from its pages: an id behind a set code, a bare id, a set code and
// collector number, the storefront's own index, and the "MTG" codes of
// sealed product, which name no card. A "-f" ending states the finish, and
// the letters a set code opens with name the shelf.
func TestReadPage(t *testing.T) {
	for _, tt := range []struct {
		code                    string
		tcg, bare, index, shelf string
		card, foil, ambiguous   bool
	}{
		{code: "mtgTLP269131", tcg: "269131", shelf: "TLP", card: true, ambiguous: true},
		{code: "mtg2X2276647-f", tcg: "276647", card: true, foil: true, ambiguous: true},
		{code: "mtgMBP125841-f", tcg: "125841", shelf: "MBP", card: true, foil: true, ambiguous: true},
		{code: "mtgSCD456950", tcg: "456950", shelf: "SCD", card: true, ambiguous: true},
		{code: "mtgCMB1246964", tcg: "246964", shelf: "CMB", card: true, ambiguous: true},
		{code: "mtgSTA235515-f", tcg: "235515", shelf: "STA", card: true, foil: true, ambiguous: true},
		{code: "mtgSLD213821-jp-f", tcg: "213821", shelf: "SLD", card: true, foil: true, ambiguous: true},
		{code: "mtg401991", bare: "401991", card: true},
		{code: "mtg473052-f", bare: "473052", card: true, foil: true},
		{code: "mtg2916", bare: "2916", card: true},
		{code: "mtgELD290-f", index: "290", shelf: "ELD", card: true, foil: true},
		{code: "mtgGRC_12-f", index: "12", shelf: "GRC", card: true, foil: true},
		{code: "mtgDM_1", index: "1", shelf: "DM", card: true},
		{code: "mtgFNMP_180-f", index: "180", shelf: "FNMP", card: true, foil: true},
		{code: "mtgMBP_132-f", index: "132", shelf: "MBP", card: true, foil: true},
		{code: "mtg439392a", card: true},
		{code: "MTGFDN"},
		{code: ""},
	} {
		var p MLProduct
		p.readPage("Promos", tt.code)
		if p.TCGProductID != tt.tcg || p.CodeID != tt.bare || p.Number != tt.index || p.Shelf != tt.shelf ||
			p.FromPage != tt.card || p.Foil != tt.foil || p.AmbiguousID != tt.ambiguous {
			t.Errorf("%s: got tcg=%q bare=%q index=%q shelf=%q card=%v foil=%v ambiguous=%v, want tcg=%q bare=%q index=%q shelf=%q card=%v foil=%v ambiguous=%v",
				tt.code, p.TCGProductID, p.CodeID, p.Number, p.Shelf, p.FromPage, p.Foil, p.AmbiguousID,
				tt.tcg, tt.bare, tt.index, tt.shelf, tt.card, tt.foil, tt.ambiguous)
		}
	}
}
