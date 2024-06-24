// Package mtgban defines interfaces for scrapers and utility functions
// to obtain pricing information from various vendors.
package mtgban

import (
	"time"
)

// Interface describing common operations on entries
type GenericEntry interface {
	Pricing() float64
	Condition() string
}

// InventoryEntry represents an entry for selling a particular Card
type InventoryEntry struct {
	// Quantity of this entry
	Quantity int `json:"quantity"`

	// The grade of the current entry
	// Only supported values are listed in FullGradeTags
	Conditions string `json:"conditions"`

	// The price of this entry, in USD
	Price float64 `json:"price"`

	// The link for this entry on the scraper website (if available)
	URL string `json:"url"`

	// Only used for a Marketplace inventory
	SellerName string `json:"seller_name,omitempty"`

	// Part of a hub of sellers that can ship directly
	Bundle bool `json:"bundle,omitempty"`

	// Original identifier as available from the scraper
	// This is usually the "product id".
	OriginalId string `json:"original_id,omitempty"`

	// Original instance identifier as available from the scraper
	// This is usually the "SKU", or the id of the entry taking into
	// account different properties, such as conditions, language etc
	InstanceId string `json:"instance_id,omitempty"`

	// Any additional custom fields set by the scraper
	CustomFields map[string]string `json:"custom_fields,omitempty"`

	// SKU id, composed of ScryfallId, language, finish, and condition
	SKUID string `json:"sku_id,omitempty"`
}

func (ie InventoryEntry) Pricing() float64 {
	return ie.Price
}

func (ie InventoryEntry) Condition() string {
	return ie.Conditions
}

// BuylistEntry represents an entry for buying a particular Card
type BuylistEntry struct {
	// Quantity of this entry
	Quantity int `json:"quantity"`

	// The grade of the current entry
	// Only supported values are listed in FullGradeTags
	// If empty it is considered "NM".
	Conditions string `json:"conditions"`

	// The price at which this entry is bought, in USD
	BuyPrice float64 `json:"buy_price"`

	// The ratio between the sale and buy prices, indicating desiderability
	// of the entry by the provider
	PriceRatio float64 `json:"price_ratio,omitempty"`

	// The link for this entry on the scraper website (if available)
	URL string `json:"url"`

	// Name of the vendor providing the entry
	VendorName string `json:"vendor_name,omitempty"`

	// Original identifier as available from the scraper
	OriginalId string `json:"original_id,omitempty"`

	// Original instance identifier as available from the scraper
	// This is usually the "SKU", or the id of the entry taking into
	// account different properties, such as conditions, language etc
	InstanceId string `json:"instance_id,omitempty"`

	// Any additional custom fields set by the scraper
	CustomFields map[string]string `json:"custom_fields,omitempty"`

	// SKU id, composed of ScryfallId, language, finish, and condition
	SKUID string `json:"sku_id,omitempty"`
}

func (be BuylistEntry) Pricing() float64 {
	return be.BuyPrice
}

func (be BuylistEntry) Condition() string {
	return be.Conditions
}

// ScraperInfo contains
type ScraperInfo struct {
	// Full name of the store
	Name string `json:"name"`

	// Shorthand or ID of the store
	Shorthand string `json:"shorthand"`

	// Symbol for worldwide stores
	CountryFlag string `json:"country,omitempty"`

	// Timestamp of the last Inventory() execution
	InventoryTimestamp *time.Time `json:"inventory_ts,omitempty"`

	// Timestamp of the last Buylist() execution
	BuylistTimestamp *time.Time `json:"buylist_ts,omitempty"`

	// Only index-style data is available, no quantities or conditions
	MetadataOnly bool `json:"metadata,omitempty"`

	// Percentage multiplier for the store credit
	CreditMultiplier float64 `json:"credit_multiplier,omitempty"`

	// Inventory quantities are not available
	NoQuantityInventory bool `json:"no_qty_inventory,omitempty"`

	// Scraper contains sealed information instead of singles
	SealedMode bool `json:"sealed,omitempty"`

	// Any additional custom fields set by the user
	CustomFields map[string]string `json:"custom_fields,omitempty"`
}

// The default list of conditions most scrapers output
var DefaultGradeTags = []string{
	"NM", "SP", "MP", "HP",
}

// The full list of conditions supported
var FullGradeTags = []string{
	"NM", "SP", "MP", "HP", "PO",
}

// Scraper is the interface both Sellers and Vendors need to implement
type Scraper interface {
	Info() ScraperInfo
}

// Carter is the inteface used to identify Seller scrapers that can
// add entries to the online cart of the provider.
type Carter interface {
	// Enable the cart interface (loading the existing cart for example).
	Activate(user, pass string) error

	// Add an InventoryEntry to the online cart.
	Add(entry InventoryEntry) error
}

// The base map for Seller containing a uuid pointing to an array of InventoryEntry
type InventoryRecord map[string][]InventoryEntry

// Market is the interface describing actions to be performed on the
// inventory available on a platform, usually combining different sellers
type Market interface {
	// Return all names for the sellers present in the Market
	MarketNames() []string

	// Market implements the Seller interface
	Seller
}

// Trader is the interface describing actions to be performed on the
// buylist available on a platform, usually combining different vendors
type Trader interface {
	// Return all names for the sellers present in the Trader
	TraderNames() []string

	// Trader implements the Vendor interface
	Vendor
}

// Seller is the interface describing actions to be performed on a seller inventory
type Seller interface {
	// Return the inventory for a Seller. If not already loaded, it will start
	// scraping the seller gathering the necessary data.
	Inventory() (InventoryRecord, error)

	// Return some information about the seller
	Info() ScraperInfo
}

// The base map for Vendor containing a uuid pointing to an array of BuylistEntry
type BuylistRecord map[string][]BuylistEntry

// Vendor is the interface describing actions to be performed on a vendor buylist
type Vendor interface {
	// Return the buylist for a Vendor. If not already loaded, it will start
	// scraping the vendor gathering the necessary data.
	Buylist() (BuylistRecord, error)

	// Return some information about the vendor
	Info() ScraperInfo
}
