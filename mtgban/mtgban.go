// Package mtgban defines interfaces for scrapers and utility functions
// to obtain pricing information from various vendors.
package mtgban

// Card is a generic card representation using fields defined by the MTGJSON project.
type Card struct {
	// The unique identifier of a card. When the UUID can be used to associate
	// two versions of the same card (for example because one is foil), `_f`
	// suffix is appended to it.
	Id string `json:"id"`

	// The official name of the card
	Name string `json:"name"`

	// The set the card comes from
	Set string `json:"set"`

	// Whether the card is foil or not
	Foil bool `json:"foil"`
}

// InventoryEntry represents an entry for selling a particular Card
type InventoryEntry struct {
	Card

	Quantity   int
	Conditions string
	Price      float64

	Notes string
}

// BuylistEntry represents an entry for buying a particular Card
type BuylistEntry struct {
	Card

	Quantity   int
	Conditions string
	BuyPrice   float64
	TradePrice float64

	PriceRatio    float64
	QuantityRatio float64

	Notes string
}

// ScraperInfo contains
type ScraperInfo struct {
	Name      string
	Shorthand string
}

// Scraper is the interface both Sellers and Vendors need to implement
type Scraper interface {
	Info() ScraperInfo
}

// Seller is the interface describing actions to be performed on an seller inventory
type Seller interface {
	// Return the inventory for a Seller. If not already loaded, it will start
	// scraping the seller gathering the necessary data.
	Inventory() (map[string][]InventoryEntry, error)

	// Add an Entry to the Seller's inventory, validating input.
	InventoryAdd(InventoryEntry) error
}

// Vendor is the interface describing actions to be performed on an vendor buylist
type Vendor interface {
	// Return the buylist for a Vendor. If not already loaded, it will start
	// scraping the vendor gathering the necessary data.
	Buylist() (map[string][]BuylistEntry, error)

	// Add an Entry to the Vendor's buylist, validating input.
	BuylistAdd(BuylistEntry) error
}
