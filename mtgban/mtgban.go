// Package mtgban defines interfaces for scrapers and utility functions
// to obtain pricing information from various vendors.
package mtgban

import (
	"io"
	"time"

	"github.com/kodabb/go-mtgban/mtgdb"
)

// InventoryEntry represents an entry for selling a particular Card
type InventoryEntry struct {
	Quantity   int
	Conditions string
	Price      float64

	URL string

	// Only used for a Marketplace inventory
	SellerName string
}

// BuylistEntry represents an entry for buying a particular Card
type BuylistEntry struct {
	Quantity   int
	BuyPrice   float64
	TradePrice float64

	PriceRatio float64

	URL string
}

// ScraperInfo contains
type ScraperInfo struct {
	Name               string
	Shorthand          string
	CountryFlag        string
	InventoryTimestamp time.Time
	BuylistTimestamp   time.Time
	MetadataOnly       bool
	NoCredit           bool

	// Return the grading scale for adjusting prices according to conditions
	Grading func(mtgdb.Card, BuylistEntry) map[string]float64
}

// A generic grading function that estimates deductions when not available
func DefaultGrading(card mtgdb.Card, entry BuylistEntry) (grade map[string]float64) {
	grade = map[string]float64{
		"SP": 0.8, "MP": 0.6, "HP": 0.4,
	}
	return
}

// Scraper is the interface both Sellers and Vendors need to implement
type Scraper interface {
	Info() ScraperInfo
}

// Initializer is the inteface used to identify scrapers that can have
// data loaded offline.
type Initializer interface {
	// Initialize an inventory.
	IntializeInventory(io.Reader) error
}

type InventoryRecord map[mtgdb.Card][]InventoryEntry

// Market is the interface describing actions to be performed on the
// inventory available on a platform, usually combining different sellers
type Market interface {
	// Return the whole inventory for a Market. If not already loaded,
	// it will start scraping the seller gathering the necessary data.
	Inventory() (InventoryRecord, error)

	// Return the inventory for any given seller present in the market.
	// If possible, it will use the Inventory() call to populate data.
	InventoryForSeller(string) (InventoryRecord, error)

	// Return some information about the market
	Info() ScraperInfo
}

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

	// Return some information about the vendor
	Info() ScraperInfo
}
