package mtgban

import (
	"fmt"
)

// BanClient abstracts some common operations that can be performed on any
// Scraper type types, as well as offering a way to retrieve a single or
// multiple Scapers.
type BanClient struct {
	scrapers map[string]Scraper
}

// Return an empty BanClient
func NewClient() *BanClient {
	bc := BanClient{}
	bc.scrapers = map[string]Scraper{}
	return &bc
}

// Add a Scraper to the client
func (bc *BanClient) Register(scraper Scraper) {
	bc.scrapers[scraper.Info().Shorthand] = scraper
}

// Load inventory and buylist content for each scraper registered in the client
func (bc *BanClient) Load() error {
	for _, scraper := range bc.scrapers {
		seller, ok := scraper.(Seller)
		if ok {
			_, err := seller.Inventory()
			if err != nil {
				return err
			}
		}
		vendor, ok := scraper.(Vendor)
		if ok {
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
		if !ok {
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
		if !ok {
			continue
		}
		vendors = append(vendors, vendor)
	}
	return
}
