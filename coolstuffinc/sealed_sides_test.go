package coolstuffinc

import (
	"testing"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
)

// TestSealedSidesReportWhatLoaded pins that a side never loaded is reported as
// absent rather than as empty. UnfoldScrapers reads the timestamps to decide
// whether a scraper is a seller, a vendor, or both, and a stamped-but-empty
// buylist made a vendor of the games Cool Stuff Inc sells sealed for and buys
// none in. The dump then refused that vendor for holding no data, which cost
// the run a non-fatal error and left the buylist file the website reloads
// missing - the worker went red on a store simply not buying something.
func TestSealedSidesReportWhatLoaded(t *testing.T) {
	for _, tt := range []struct {
		desc                   string
		inventory, buylist     bool
		wantSeller, wantVendor bool
	}{
		{"a scraper that loaded neither side is neither", false, false, false, false},
		{"one that sold and did not buy is a seller alone", true, false, true, false},
		{"one that bought and did not sell is a vendor alone", false, true, false, true},
		{"and one that did both is both", true, true, true, true},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			csi := &Sealed{game: GameLorcana}
			if tt.inventory {
				csi.inventoryDate = time.Now()
			}
			if tt.buylist {
				csi.buylistDate = time.Now()
			}
			info := csi.Info()
			if got := info.InventoryTimestamp != nil; got != tt.wantSeller {
				t.Errorf("InventoryTimestamp set = %v, want %v", got, tt.wantSeller)
			}
			if got := info.BuylistTimestamp != nil; got != tt.wantVendor {
				t.Errorf("BuylistTimestamp set = %v, want %v", got, tt.wantVendor)
			}

			sellers, vendors := mtgban.UnfoldScrapers([]mtgban.Scraper{csi})
			if got := len(sellers) > 0; got != tt.wantSeller {
				t.Errorf("unfolded as a seller = %v, want %v", got, tt.wantSeller)
			}
			if got := len(vendors) > 0; got != tt.wantVendor {
				t.Errorf("unfolded as a vendor = %v, want %v", got, tt.wantVendor)
			}
		})
	}
}
