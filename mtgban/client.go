package mtgban

import (
	"fmt"
)

// BanClient abstracts some common operations that can be performed on any
// Scraper type types, as well as offering a way to retrieve a single or
// multiple Scapers.
type BanClient struct {
	scrapers       map[string]Scraper
	sellerDisabled map[string]bool
	vendorDisabled map[string]bool
}

// Return an empty BanClient
func NewClient() *BanClient {
	bc := BanClient{}
	bc.scrapers = map[string]Scraper{}
	bc.sellerDisabled = map[string]bool{}
	bc.vendorDisabled = map[string]bool{}
	return &bc
}

// Add a Scraper to the client
func (bc *BanClient) Register(scraper Scraper) {
	bc.scrapers[scraper.Info().Shorthand] = scraper
	bc.sellerDisabled[scraper.Info().Shorthand] = false
	bc.vendorDisabled[scraper.Info().Shorthand] = false
}

// Add a Scraper to the client, enable the seller side only (if any)
func (bc *BanClient) RegisterSeller(scraper Scraper) {
	bc.scrapers[scraper.Info().Shorthand] = scraper
	bc.vendorDisabled[scraper.Info().Shorthand] = true
}

// Add a Scraper to the client, enable the vendor side only (if any)
func (bc *BanClient) RegisterVendor(scraper Scraper) {
	bc.scrapers[scraper.Info().Shorthand] = scraper
	bc.sellerDisabled[scraper.Info().Shorthand] = true
}

// Load inventory and buylist content for each scraper registered in the client
func (bc *BanClient) Load() error {
	for _, scraper := range bc.scrapers {
		seller, ok := scraper.(Seller)
		if ok && !bc.sellerDisabled[scraper.Info().Shorthand] {
			_, err := seller.Inventory()
			if err != nil {
				return err
			}
		}
		vendor, ok := scraper.(Vendor)
		if ok && !bc.vendorDisabled[scraper.Info().Shorthand] {
			_, err := vendor.Buylist()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Return the scraper with a matching name from the ones registered in the client
func (bc *BanClient) ScraperByName(shorthand string) (Scraper, error) {
	scraper, found := bc.scrapers[shorthand]
	if !found {
		return nil, fmt.Errorf("Scraper %s not found", shorthand)
	}
	return scraper, nil
}

// Return a new slice containing all the scrapers registered in the client
func (bc *BanClient) Scrapers() (scrapers []Scraper) {
	for _, scraper := range bc.scrapers {
		scrapers = append(scrapers, scraper)
	}
	return
}

// Return a new slice containing all the sellers registered in the client
func (bc *BanClient) Sellers() (sellers []Seller) {
	for _, maybeSeller := range bc.scrapers {
		seller, ok := maybeSeller.(Seller)
		if !ok || bc.sellerDisabled[maybeSeller.Info().Shorthand] {
			continue
		}
		sellers = append(sellers, seller)
	}
	return
}

// Return a new slice containing all the vendors registered in the client
func (bc *BanClient) Vendors() (vendors []Vendor) {
	for _, maybeVendor := range bc.scrapers {
		vendor, ok := maybeVendor.(Vendor)
		if !ok || bc.vendorDisabled[maybeVendor.Info().Shorthand] {
			continue
		}
		vendors = append(vendors, vendor)
	}
	return
}

// Return a new slice containing all the markets registered in the client
func (bc *BanClient) Markets() (markets []Market) {
	for _, maybeMarket := range bc.scrapers {
		market, ok := maybeMarket.(Market)
		if !ok || bc.sellerDisabled[maybeMarket.Info().Shorthand] {
			continue
		}
		markets = append(markets, market)
	}
	return
}
