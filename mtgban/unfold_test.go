package mtgban

import (
	"context"
	"testing"
	"time"
)

// bothSides is a scraper that sells and buys, with either record allowed to
// be empty - the shape of a store that stocks a category it does not buy.
type bothSides struct {
	inventory InventoryRecord
	buylist   BuylistRecord
}

func (s *bothSides) Load(context.Context) error { return nil }
func (s *bothSides) Inventory() InventoryRecord { return s.inventory }
func (s *bothSides) Buylist() BuylistRecord     { return s.buylist }

func (s *bothSides) Info() ScraperInfo {
	now := time.Now()
	// Stamped either way, the way every scraper in the tree stamps them.
	return ScraperInfo{
		Name:               "Fixture",
		Shorthand:          "FIX",
		InventoryTimestamp: &now,
		BuylistTimestamp:   &now,
	}
}

func entry() *InventoryEntry  { return &InventoryEntry{Conditions: "NM", Price: 1, Quantity: 1} }
func bidEntry() *BuylistEntry { return &BuylistEntry{Conditions: "NM", BuyPrice: 1} }

// TestUnfoldReadsTheRecords pins that a side holding nothing is not unfolded.
// What a scraper implements says what it is able to do, not what the store
// does today, and reading the interface alone made vendors of the games Cool
// Stuff Inc sells sealed for and buys none in - which the caller then refused
// for holding no data, failing the run over a store not buying something.
func TestUnfoldReadsTheRecords(t *testing.T) {
	for _, tt := range []struct {
		desc                     string
		inventory, buylist       bool
		wantSellers, wantVendors int
	}{
		{"a scraper with both records is both", true, true, 1, 1},
		{"one that sells what it does not buy is a seller alone", true, false, 1, 0},
		{"one that buys what it does not sell is a vendor alone", false, true, 0, 1},
		{"and one holding nothing is neither", false, false, 0, 0},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			s := &bothSides{inventory: InventoryRecord{}, buylist: BuylistRecord{}}
			if tt.inventory {
				if err := s.inventory.Add("uuid", entry()); err != nil {
					t.Fatal(err)
				}
			}
			if tt.buylist {
				if err := s.buylist.Add("uuid", bidEntry()); err != nil {
					t.Fatal(err)
				}
			}

			sellers, vendors := UnfoldScrapers([]Scraper{s})
			if len(sellers) != tt.wantSellers {
				t.Errorf("unfolded %d sellers, want %d", len(sellers), tt.wantSellers)
			}
			if len(vendors) != tt.wantVendors {
				t.Errorf("unfolded %d vendors, want %d", len(vendors), tt.wantVendors)
			}
		})
	}
}
