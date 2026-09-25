package mtgban

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestArbitUsesTheBackendPassedIn pins that Arbit and Mismatch resolve every
// card against the backend they were handed, not any other backend that
// happens to exist: a second backend is built for the same ids, naming the
// cards differently, and the callback proves which one a run actually
// consulted.
func TestArbitUsesTheBackendPassedIn(t *testing.T) {
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

			var calls int
			seen := map[string]bool{}
			opts := &ArbitOpts{
				CustomCardFilter: func(co *mtgmatcher.CardObject) (float64, bool) {
					calls++
					seen[co.Name] = true
					return 1, false
				},
			}
			run := func(b *mtgmatcher.Backend) []ArbitEntry {
				calls = 0
				seen = map[string]bool{}
				if report == "arbit" {
					return Arbit(b, opts, vendorOf(buy), sellerOf(inv, ScraperInfo{}))
				}
				return Mismatch(b, opts, sellerOf(reference, ScraperInfo{}), sellerOf(inv, ScraperInfo{}))
			}

			rows := run(wantedBackend)
			if len(rows) != 2 || calls != 2 || !seen["wanted"] || seen["other"] {
				t.Fatalf("report did not resolve against the backend it was passed: %d rows, %d lookups, saw %v", len(rows), calls, seen)
			}

			// Handing over the other backend is what changes what the
			// callback sees; the first one, built for the same ids but no
			// longer passed anywhere, is not consulted at all.
			rows = run(otherBackend)
			if len(rows) != 2 || calls != 2 || !seen["other"] || seen["wanted"] {
				t.Fatalf("report did not follow the second backend: %d rows, %d lookups, saw %v", len(rows), calls, seen)
			}
		})
	}
}

// A report handed no datastore has nothing to resolve its ids against, so it
// names nothing rather than nil-dereferencing on the first lookup. The
// options are not a way around it: they filter, they do not resolve.
func TestReportsRefuseANilBackend(t *testing.T) {
	inv := InventoryRecord{"card": {{Conditions: "NM", Price: 10, Quantity: 1}}}

	for _, tt := range []struct {
		desc string
		run  func() []ArbitEntry
	}{
		{"arbit", func() []ArbitEntry {
			return Arbit(nil, nil,
				vendorOf(BuylistRecord{"card": {{Conditions: "NM", BuyPrice: 15}}}),
				sellerOf(inv, ScraperInfo{Name: "seller"}))
		}},
		{"mismatch", func() []ArbitEntry {
			return Mismatch(nil, nil,
				sellerOf(InventoryRecord{"card": {{Conditions: "NM", Price: 20, Quantity: 1}}},
					ScraperInfo{Name: "reference"}),
				sellerOf(inv, ScraperInfo{Name: "probe"}))
		}},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := tt.run(); got != nil {
				t.Errorf("a nil backend named %d entries, want nil", len(got))
			}
		})
	}
}
