package gamenerdz

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// A listing has to name a finish the printing was sold in. This storefront
// mints a "-F-" sku beside the plain one whether or not the set ever printed
// a foil, and where it did not both listings answer with the single printing
// there is - the minted one carrying a price of its own, six dollars against
// four cents on Iconic Shield.
func TestFinishPrinted(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct {
		desc                 string
		product              GNProduct
		wantSet, wantNum     string
		wantFoil, wantEtched bool
		printed              bool
	}{
		{
			desc: "the minted foil of a card the set printed in nonfoil only",
			product: GNProduct{
				DisplayName:    "Iconic Shield (MSC-520) - Commander: Marvel Super Heroes Foil",
				SelectedFinish: "foil",
				ProductData:    GNProductData{Set: "msc", SetName: "Commander: Marvel Super Heroes"},
			},
			wantSet: "MSC", wantNum: "520", wantFoil: true, printed: false,
		},
		{
			desc: "the plain listing beside it, which is the card",
			product: GNProduct{
				DisplayName:    "Iconic Shield (MSC-520) - Commander: Marvel Super Heroes",
				SelectedFinish: "nonfoil",
				ProductData:    GNProductData{Set: "msc", SetName: "Commander: Marvel Super Heroes"},
			},
			wantSet: "MSC", wantNum: "520", printed: true,
		},
		{
			desc: "an etched printing asked for as etched",
			product: GNProduct{
				DisplayName:    "Aeromoeba (Retro Frame) (Foil Etched) (MH2-389) - Modern Horizons 2 Foil",
				SelectedFinish: "foil",
				ProductData:    GNProductData{Set: "mh2", SetName: "Modern Horizons 2"},
			},
			wantSet: "MH2", wantNum: "389", wantFoil: true, wantEtched: true, printed: true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			card, err := preprocess(b, tt.product, mtgmatcher.GameMagic)
			if err != nil {
				t.Fatalf("preprocess: %v", err)
			}
			foil, etched := card.Foil, saysEtched(tt.product)
			if foil != tt.wantFoil || etched != tt.wantEtched {
				t.Errorf("asked foil=%v etched=%v, want foil=%v etched=%v",
					foil, etched, tt.wantFoil, tt.wantEtched)
			}
			id, err := b.Match(card)
			if err != nil {
				t.Fatalf("Match: %v", err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("resolved to %s #%s, want %s #%s", co.SetCode, co.Number, tt.wantSet, tt.wantNum)
			}
			if got := finishPrinted(b, id, foil, etched); got != tt.printed {
				t.Errorf("finishPrinted = %v, want %v (printing carries %v)", got, tt.printed, co.Finishes)
			}
		})
	}
}

// The storefront also mints a "-F-" sku on Modern Horizons 3 Commander cards
// printed plain only, and a plain foil request is answered with the ripple
// foil star twin of the number. That foil is listed under its own name.
func TestMintedRipple(t *testing.T) {
	gn, err := NewScraper(realDatastore(t))
	if err != nil {
		t.Fatal(err)
	}
	setData := GNProductData{Set: "m3c", SetName: "Modern Horizons 3 Commander"}
	for _, tt := range []struct {
		display string
		finish  string
		landed  bool
	}{
		{"Magus of the Candelabra (M3C-236) - Modern Horizons 3 Commander Foil", "foil", false},
		{"Magus of the Candelabra (Ripple Foil) (M3C-236) - Modern Horizons 3 Commander Foil", "foil", true},
		{"Magus of the Candelabra (M3C-236) - Modern Horizons 3 Commander", "nonfoil", true},
	} {
		product := GNProduct{ID: tt.display, DisplayName: tt.display, SelectedFinish: tt.finish, ProductData: setData}
		got, err := gn.resolveProduct(modeBuylist, product)
		if err != nil {
			t.Fatalf("%q: %v", tt.display, err)
		}
		if (got != "") != tt.landed {
			t.Errorf("resolveProduct(%q) = %q, want landed=%v", tt.display, got, tt.landed)
		}
	}
}
