package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
)

// TestSecondBucketMerges covers the Armory Deck singles Star City Games splits
// across two product records. Both records price the same card at the same
// price and differ only in how many copies each holds, so the second used to
// read as a duplicate and its copies were discarded; they belong in the
// first's count.
//
// Which record the catalog streams first is not fixed - most of the Armory
// Deck: Pleiades listings lead with the marked one - so the pair has to fold
// either way round, and both orders are run here.
//
// The quantities are the ones the catalog carried on 2026-08-24: eleven Near
// Mint and five Played under the plain sku, seven and two under the second.
func TestSecondBucketMerges(t *testing.T) {
	withGameDatastore(t, "FLESHANDBLOOD_PATH", fleshandblood.Load)

	product := func(sku string, nm, sp int) CatalogProduct {
		return CatalogProduct{
			SKU: sku, Name: "Anka, Drag Under", Game: "Flesh and Blood",
			Set: "Armory Deck: Gravy Bones", ProductType: ProductTypeSingles,
			CollectorNumber: "014", Finish: "Non-foil", FinishGroup: "Non-foil",
			URL: "/anka-drag-under-" + sku + "/",
			Variants: []CatalogVariant{
				{SKU: sku + "1", Condition: "Near Mint", Qty: nm, Price: "0.49"},
				{SKU: sku + "2", Condition: "Played", Qty: sp, Price: "0.39"},
			},
		}
	}
	plain := product("SGL-FAB-AGB-014-ENN", 11, 5)
	marked := product("SGL-FAB-AGB-014_CC-ENN", 7, 2)

	for _, tt := range []struct {
		name  string
		order []CatalogProduct
	}{
		{"the plain record first", []CatalogProduct{plain, marked}},
		{"the marked record first", []CatalogProduct{marked, plain}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			scg := NewScraper(GameFleshAndBlood, "")
			var logs int
			scg.LogCallback = func(format string, a ...any) { logs++ }
			for _, p := range tt.order {
				scg.processProduct(p)
			}

			if logs != 0 {
				t.Errorf("the pair was reported %d times, want it folded in silently", logs)
			}
			if len(scg.inventory) != 1 {
				t.Fatalf("got %d priced cards, want 1", len(scg.inventory))
			}
			for _, entries := range scg.inventory {
				if len(entries) != 2 {
					t.Fatalf("got %d entries, want one per grade: %v", len(entries), entries)
				}
				want := map[string]int{"NM": 18, "SP": 7}
				for _, entry := range entries {
					if entry.Quantity != want[entry.Conditions] {
						t.Errorf("%s holds %d copies, want %d", entry.Conditions, entry.Quantity, want[entry.Conditions])
					}
				}
			}
		})
	}
}

// TestStreamRetryForgetsBuckets covers the reset a broken catalog stream runs
// before reading the whole export again. The listings a second stock record
// has been seen for are part of what a run collected, and they go back with
// the rest of it: every record is about to arrive a second time, and a first
// arrival must not be taken for a second one.
func TestStreamRetryForgetsBuckets(t *testing.T) {
	scg := NewScraper(GameFleshAndBlood, "")
	scg.buckets["SGL-FAB-AGB-014-ENN"] = struct{}{}
	scg.reset()

	if len(scg.buckets) != 0 {
		t.Errorf("the retry kept %v", scg.buckets)
	}
	if len(scg.inventory) != 0 || len(scg.buylist) != 0 {
		t.Errorf("the retry kept %d inventory and %d buylist keys",
			len(scg.inventory), len(scg.buylist))
	}
}

// TestBucketKey pins what makes two records one listing's. Both records of a
// pair have to answer with the same key and nothing else may join them, since
// the key is what relaxes the add that would otherwise keep them apart.
func TestBucketKey(t *testing.T) {
	for _, tt := range []struct{ sku, want string }{
		{"SGL-FAB-AGB-014_CC-ENN", "SGL-FAB-AGB-014-ENN"},
		{"SGL-FAB-AGB-014-ENN", "SGL-FAB-AGB-014-ENN"},
		{"SGL-FAB-APS-017_CC-ENN", "SGL-FAB-APS-017-ENN"},
		{"SGL-FAB-AGB-014-ENR", "SGL-FAB-AGB-014-ENR"},
		{"SGL-FAB-AGB-015-ENN", "SGL-FAB-AGB-015-ENN"},
		// The marker only counts where the number is, not anywhere it happens
		// to be spelled, and a sku short of a number segment has no listing to
		// name beyond itself.
		{"SGL-LOR-PRM-P3_031-ENC", "SGL-LOR-PRM-P3_031-ENC"},
		{"SGL-FAB", "SGL-FAB"},
	} {
		t.Run(tt.sku, func(t *testing.T) {
			if got := bucketKey(tt.sku); got != tt.want {
				t.Errorf("bucketKey(%q) = %q, want %q", tt.sku, got, tt.want)
			}
		})
	}
}
