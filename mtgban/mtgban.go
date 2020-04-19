// Package mtgban defines interfaces for scrapers and utility functions
// to obtain pricing information from various vendors.
package mtgban

import (
	"time"

	"github.com/kodabb/go-mtgban/mtgdb"
)

// InventoryEntry represents an entry for selling a particular Card
type InventoryEntry struct {
	Quantity   int
	Conditions string
	Price      float64

	URL string
}

// BuylistEntry represents an entry for buying a particular Card
type BuylistEntry struct {
	Quantity   int
	Conditions string
	BuyPrice   float64
	TradePrice float64

	PriceRatio    float64
	QuantityRatio float64

	URL string
}

// ScraperInfo contains
type ScraperInfo struct {
	Name               string
	Shorthand          string
	InventoryTimestamp time.Time
	BuylistTimestamp   time.Time
}

// Scraper is the interface both Sellers and Vendors need to implement
type Scraper interface {
	Info() ScraperInfo
}

type InventoryRecord map[mtgdb.Card][]InventoryEntry

// Seller is the interface describing actions to be performed on a seller inventory
type Seller interface {
	// Return the inventory for a Seller. If not already loaded, it will start
	// scraping the seller gathering the necessary data.
	Inventory() (InventoryRecord, error)

	// Return some information about the seller
	Info() ScraperInfo
}

type BuylistRecord map[mtgdb.Card]BuylistEntry

// Vendor is the interface describing actions to be performed on a vendor buylist
type Vendor interface {
	// Return the buylist for a Vendor. If not already loaded, it will start
	// scraping the vendor gathering the necessary data.
	Buylist() (BuylistRecord, error)

	// Return the grading scale for adjusting prices according to conditions
	Grading(mtgdb.Card, BuylistEntry) map[string]float64

	// Return some information about the vendor
	Info() ScraperInfo
}
