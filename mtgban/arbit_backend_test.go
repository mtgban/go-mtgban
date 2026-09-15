package mtgban

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestArbitUsesTheBackendNamedInOpts pins that Arbit and Mismatch resolve
// every card against the backend ArbitOpts.Backend names, not any other
// backend that happens to exist: a second backend is built for the same ids,
// naming the cards differently, and the callback proves which one a run
// actually consulted.
func TestArbitUsesTheBackendNamedInOpts(t *testing.T) {
	for _, report := range []string{"arbit", "mismatch"} {
		t.Run(report, func(t *testing.T) {
			wanted := map[string]*mtgmatcher.CardObject{}
			other := map[string]*mtgmatcher.CardObject{}
			inv, reference, buy := InventoryRecord{}, InventoryRecord{}, BuylistRecord{}
			for _, id := range []string{"first", "second"} {
				wanted[id] = &mtgmatcher.CardObject{Card: mtgmatcher.Card{UUID: id, Name: "wanted"}}
				other[id] = &mtgmatcher.CardObject{Card: mtgmatcher.Card{UUID: id, Name: "other"}}
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
			wantedBackend := backendFor(wanted)
			otherBackend := backendFor(other)

			run := func(opts *ArbitOpts) []ArbitEntry {
				if report == "arbit" {
					return Arbit(opts, vendorOf(buy), sellerOf(inv, ScraperInfo{}))
				}
				return Mismatch(opts, sellerOf(reference, ScraperInfo{}), sellerOf(inv, ScraperInfo{}))
			}

			var calls int
			seen := map[string]bool{}
			opts := &ArbitOpts{
				Backend: wantedBackend,
				CustomCardFilter: func(co *mtgmatcher.CardObject) (float64, bool) {
					calls++
					seen[co.Name] = true
					return 1, false
				},
			}
			rows := run(opts)
			if len(rows) != 2 || calls != 2 || !seen["wanted"] || seen["other"] {
				t.Fatalf("report did not resolve against the named backend: %d rows, %d lookups, saw %v", len(rows), calls, seen)
			}

			// Swapping the backend the opts name is what changes what the
			// callback sees; the other backend, built for the same ids but
			// never named in opts, is not consulted at all.
			opts.Backend = otherBackend
			calls = 0
			seen = map[string]bool{}
			rows = run(opts)
			if len(rows) != 2 || calls != 2 || !seen["other"] || seen["wanted"] {
				t.Fatalf("report did not follow the swapped backend: %d rows, %d lookups, saw %v", len(rows), calls, seen)
			}
		})
	}
}

// TestArbitCapturesTheGlobalOnce pins that a nil ArbitOpts.Backend falls back
// to the global datastore captured once at entry, not read again as the
// report runs: a callback that republishes the global mid-report is still
// answered by the backend the report captured first.
func TestArbitCapturesTheGlobalOnce(t *testing.T) {
	for _, report := range []string{"arbit", "mismatch"} {
		t.Run(report, func(t *testing.T) {
			wanted := map[string]*mtgmatcher.CardObject{}
			other := map[string]*mtgmatcher.CardObject{}
			inv, reference, buy := InventoryRecord{}, InventoryRecord{}, BuylistRecord{}
			for _, id := range []string{"first", "second"} {
				wanted[id] = &mtgmatcher.CardObject{Card: mtgmatcher.Card{UUID: id, Name: "wanted"}}
				other[id] = &mtgmatcher.CardObject{Card: mtgmatcher.Card{UUID: id, Name: "other"}}
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
			wantedBackend := backendFor(wanted)
			otherBackend := backendFor(other)

			previous := mtgmatcher.GlobalDatastore()
			t.Cleanup(func() { mtgmatcher.SetGlobalDatastore(previous) })
			mtgmatcher.SetGlobalDatastore(wantedBackend)

			var calls int
			seen := map[string]bool{}
			opts := &ArbitOpts{
				CustomCardFilter: func(co *mtgmatcher.CardObject) (float64, bool) {
					calls++
					seen[co.Name] = true
					mtgmatcher.SetGlobalDatastore(otherBackend)
					return 1, false
				},
			}

			var rows []ArbitEntry
			if report == "arbit" {
				rows = Arbit(opts, vendorOf(buy), sellerOf(inv, ScraperInfo{}))
			} else {
				rows = Mismatch(opts, sellerOf(reference, ScraperInfo{}), sellerOf(inv, ScraperInfo{}))
			}

			if len(rows) != 2 || calls != 2 || !seen["wanted"] || seen["other"] {
				t.Fatalf("report did not capture the global once: %d rows, %d lookups, saw %v", len(rows), calls, seen)
			}
		})
	}
}
