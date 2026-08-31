package starcitygames

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
)

// TestFabVariantMarked pins which set codes are read as carrying the marker.
// The trim is what guards it: a code is only marked where taking the digit off
// still leaves one, so the print-run codes and the plain ones are left alone.
func TestFabVariantMarked(t *testing.T) {
	for _, tt := range []struct {
		sku  string
		want bool
	}{
		{"SGL-FAB-EVO2-056-ENR", true},
		{"SGL-FAB-AAZ2-005-ENR", true},
		// The first-edition sets mark the run and the treatment at once.
		{"SGL-FAB-CRU12-082-ENR", true},
		{"SGL-FAB-EVO-056-ENR", false},
		// A print run is not a treatment.
		{"SGL-FAB-CRU1-068-ENN", false},
		{"SGL-FAB-CRUU-107-ENR", false},
		// Codes that end in a marker letter of their own stay whole.
		{"SGL-FAB-KSU-021-ENN", false},
		{"", false},
	} {
		if got := fabVariantMarked(tt.sku); got != tt.want {
			t.Errorf("fabVariantMarked(%q) = %v, want %v", tt.sku, got, tt.want)
		}
	}
}

// TestCatalogFabMarkedPrinting pins the printing a marked sku reaches. Star
// City Games gives the second printing of a collector number a sku of its own
// and says nothing else about it: the set, the number, the finish and the
// rarity are the same words on both, so the marker is the whole of the
// difference and the datastore is what names the treatment it stands for.
//
// Read as the plain printing, the listing quoted the wrong card's price: the
// Extended Art War Machine and the ordinary rainbow foil beside it were one
// entry with two Star City Games rows under it.
func TestCatalogFabMarkedPrinting(t *testing.T) {
	withGameDatastore(t, "FLESHANDBLOOD_PATH", fleshandblood.Load)

	for _, tt := range []struct {
		desc    string
		product CatalogProduct
		want    string
	}{
		{
			desc: "the marker reaches the extended art",
			product: CatalogProduct{
				SKU: "SGL-FAB-EVO2-056-ENR", Name: "War Machine", Set: "Bright Lights",
				CollectorNumber: "056", Finish: "Rainbow Foil", Rarity: "Majestic", Language: "English",
			},
			want: "extendedart",
		},
		{
			desc: "and the plain sku still reaches the plain printing",
			product: CatalogProduct{
				SKU: "SGL-FAB-EVO-056-ENR", Name: "War Machine", Set: "Bright Lights",
				CollectorNumber: "056", Finish: "Rainbow Foil", Rarity: "Majestic", Language: "English",
			},
			want: "",
		},
		{
			desc: "as does the plain sku in another finish",
			product: CatalogProduct{
				SKU: "SGL-FAB-EVO-056-ENN", Name: "War Machine", Set: "Bright Lights",
				CollectorNumber: "056", Finish: "Non-foil", Rarity: "Majestic", Language: "English",
			},
			want: "",
		},
		{
			// The marker says a second printing and not which: the treatment
			// it stands for is whatever the datastore holds beside the plain
			// one, and it is not always the extended art.
			desc: "the treatment is read rather than assumed",
			product: CatalogProduct{
				SKU: "SGL-FAB-AAZ2-005-ENR", Name: "Hidden Agenda", Set: "Armory Deck: Azalea",
				CollectorNumber: "005", Finish: "Rainbow Foil", Rarity: "Majestic", Language: "English",
			},
			want: "purple",
		},
		{
			// A treatment the rarity names is already reached without the
			// marker, and reading the marker must not move it.
			desc: "a marvel is left to the rarity that names it",
			product: CatalogProduct{
				SKU: "SGL-FAB-OUT2-159-ENC", Name: "Codex of Bloodrot", Set: "Outsiders",
				CollectorNumber: "159", Finish: "Cold Foil", Rarity: "Marvel", Language: "English",
			},
			want: "marvel",
		},
		{
			// The first-edition sets carry both marks at once, and the run
			// comes off the set name while the treatment comes off the sku.
			desc: "a first edition keeps its run and gains its treatment",
			product: CatalogProduct{
				SKU: "SGL-FAB-CRU12-082-ENR", Name: "Twinning Blade", Set: "Crucible of War (1st Edition)",
				CollectorNumber: "082", Finish: "Rainbow Foil", Rarity: "Majestic", Language: "English",
			},
			want: "extendedart",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			id, err := resolveProduct(GameFleshAndBlood, tt.product)
			if err != nil {
				t.Fatalf("resolveProduct(%s) = %v", tt.product.SKU, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if tt.want == "" {
				if len(co.PromoTypes) > 0 {
					t.Errorf("%s resolved to %s, which is the %v printing", tt.product.SKU, id, co.PromoTypes)
				}
				return
			}
			if !slices.Contains(co.PromoTypes, tt.want) {
				t.Errorf("%s resolved to %s with %v, want the %s printing",
					tt.product.SKU, id, co.PromoTypes, tt.want)
			}
		})
	}
}
