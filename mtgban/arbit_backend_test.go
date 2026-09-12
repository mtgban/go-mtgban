package mtgban

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func TestAnalysisPinsBackendAcrossReload(t *testing.T) {
	for _, report := range []string{"arbit", "mismatch"} {
		t.Run(report, func(t *testing.T) {
			cards := map[string]*mtgmatcher.CardObject{}
			inv, reference, buy := InventoryRecord{}, InventoryRecord{}, BuylistRecord{}
			for _, id := range []string{"first", "second"} {
				cards[id] = &mtgmatcher.CardObject{Card: mtgmatcher.Card{UUID: id}}
				if err := inv.Add(id, &InventoryEntry{Price: 1}); err != nil {
					t.Fatal(err)
				}
				if err := reference.Add(id, &InventoryEntry{Price: 2}); err != nil {
					t.Fatal(err)
				}
				if err := buy.Add(id, &BuylistEntry{BuyPrice: 2}); err != nil {
					t.Fatal(err)
				}
			}
			installCards(t, cards)
			var calls int
			opts := &ArbitOpts{CustomCardFilter: func(co *mtgmatcher.CardObject) (float64, bool) {
				calls++
				// Whichever card the map visits first reloads the global. The
				// second card must still resolve against this report's snapshot.
				mtgmatcher.SetGlobalDatastore(&mtgmatcher.Backend{})
				return 1, false
			}}
			var rows []ArbitEntry
			if report == "arbit" {
				rows = Arbit(opts, vendorOf(buy), sellerOf(inv, ScraperInfo{}))
			} else {
				rows = Mismatch(opts, sellerOf(reference, ScraperInfo{}), sellerOf(inv, ScraperInfo{}))
			}
			if len(rows) != 2 || calls != 2 {
				t.Fatalf("report crossed snapshots: %d rows, %d card lookups", len(rows), calls)
			}
			// Explicit backends also work when nothing is installed globally.
			opts.Backend = &mtgmatcher.Backend{UUIDs: cards}
			opts.CustomCardFilter = nil
			if report == "arbit" {
				rows = Arbit(opts, vendorOf(buy), sellerOf(inv, ScraperInfo{}))
			} else {
				rows = Mismatch(opts, sellerOf(reference, ScraperInfo{}), sellerOf(inv, ScraperInfo{}))
			}
			if len(rows) != 2 {
				t.Fatalf("explicit backend returned %d rows", len(rows))
			}
		})
	}
}
