package gamenerdz

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)

// readLorcanaDatastore reads the Lorcana datastore, or skips where the run
// carries none: this package's other tests read Magic's own.
func readLorcanaDatastore(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	path := os.Getenv("LORCANA_PATH")
	if path == "" {
		t.Skip("Need LORCANA_PATH set to run this test")
	}
	b, err := datastore.Read("lorcana", path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestResolveProductLorcanaHolofoilOnly pins the skip: a "(N)" promo number
// names its printing without its finish, so a plain listing can land on a
// printing that was only ever made foil. Pegasus - Gift for Hercules (1) is
// sold Holofoil only, so a Normal-finish listing for it must be skipped
// rather than collide with the real Holofoil listing on the same id.
func TestResolveProductLorcanaHolofoilOnly(t *testing.T) {
	b := readLorcanaDatastore(t)
	gn, err := NewScraper(b)
	if err != nil {
		t.Fatal(err)
	}
	product := GNProduct{
		ID:          "pegasus-normal",
		DisplayName: "Pegasus - Gift for Hercules (1) - Disney Lorcana Promo Cards",
		ProductData: GNProductData{SetName: "Disney Lorcana Promo Cards"},
	}
	got, err := gn.resolveProduct(modeBuylist, product)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		co, _ := b.GetUUID(got)
		t.Errorf("resolveProduct(%q) = %s (%v), want skipped: no Normal printing exists",
			product.DisplayName, got, co)
	}
}
