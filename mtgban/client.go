package mtgban

import (
	"errors"
)

var ErrScraperNotFound = errors.New("scraper not found")

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
	// Reset state
	bc.sellerDisabled[scraper.Info().Shorthand] = false
	bc.vendorDisabled[scraper.Info().Shorthand] = false

	market, isMarket := scraper.(Market)
	if isMarket {
		for _, name := range market.MarketNames() {
			bc.RegisterMarket(market, name)
		}
	}
	trader, isTrader := scraper.(Trader)
	if isTrader {
		for _, name := range trader.TraderNames() {
			bc.RegisterTrader(trader, name)
		}
	}

	// Register
	bc.scrapers[scraper.Info().Shorthand] = scraper
}

// Add a Scraper to the client, enable the seller side only (if any)
// If the added scraper is a market, it will be split into its subsellers
func (bc *BanClient) RegisterSeller(scraper Scraper) {
	market, isMarket := scraper.(Market)
	if isMarket {
		for _, name := range market.MarketNames() {
			bc.RegisterMarket(market, name)
		}
		return
	}

	bc.scrapers[scraper.Info().Shorthand] = scraper
	bc.sellerDisabled[scraper.Info().Shorthand] = false
	_, found := bc.vendorDisabled[scraper.Info().Shorthand]
	if !found {
		bc.vendorDisabled[scraper.Info().Shorthand] = true
	}
}

// Add a Scraper to the client, enable the Market with the given name
func (bc *BanClient) RegisterMarket(scraper Market, name string) {
	market := &BaseMarket{}
	market.scraper = scraper
	market.info = scraper.InfoForScraper(name)

	// Disable the market itself from providing seller data
	bc.sellerDisabled[scraper.Info().Shorthand] = true
	// Disable any vendor side of the split market (not the market itself)
	_, found := bc.vendorDisabled[market.info.Shorthand]
	if !found {
		bc.vendorDisabled[market.info.Shorthand] = true
	}

	// Register
	bc.scrapers[market.info.Shorthand] = market
}

// Add a Scraper to the client, enable the vendor side only (if any)
// If the added scraper is a trader, it will be split into its subvendors
func (bc *BanClient) RegisterVendor(scraper Scraper) {
	trader, isTrader := scraper.(Trader)
	if isTrader {
		for _, name := range trader.TraderNames() {
			bc.RegisterTrader(trader, name)
		}
		return
	}

	bc.scrapers[scraper.Info().Shorthand] = scraper
	_, found := bc.sellerDisabled[scraper.Info().Shorthand]
	if !found {
		bc.sellerDisabled[scraper.Info().Shorthand] = true
	}
	bc.vendorDisabled[scraper.Info().Shorthand] = false
}

// Add a Scraper to the client, enable the Trader with the given name
func (bc *BanClient) RegisterTrader(scraper Trader, name string) {
	trader := &BaseTrader{}
	trader.scraper = scraper
	trader.info = scraper.InfoForScraper(name)

	// Disable the trader itself from providing vendor data
	bc.vendorDisabled[scraper.Info().Shorthand] = true
	// Disable any seller side of the split trader (not the trader itself)
	_, found := bc.sellerDisabled[trader.info.Shorthand]
	if !found {
		bc.sellerDisabled[trader.info.Shorthand] = true
	}

	// Register
	bc.scrapers[trader.info.Shorthand] = trader
}

// Load inventory and buylist content for each scraper registered in the client
func (bc *BanClient) Load() error {
	for _, scraper := range bc.scrapers {
		seller, ok := scraper.(Seller)
		if ok && !bc.sellerDisabled[scraper.Info().Shorthand] {
			inv, err := seller.Inventory()
			if err != nil {
				return err
			}
			if len(inv) == 0 {
				return errors.New("empty inventory")
			}
		}
		vendor, ok := scraper.(Vendor)
		if ok && !bc.vendorDisabled[scraper.Info().Shorthand] {
			bl, err := vendor.Buylist()
			if err != nil {
				return err
			}
			if len(bl) == 0 {
				return errors.New("empty buylist")
			}
		}
	}
	return nil
}

// Return the scraper with a matching name from the ones registered in the client
func (bc *BanClient) ScraperByName(shorthand string) (Scraper, error) {
	scraper, found := bc.scrapers[shorthand]
	if !found {
		return nil, ErrScraperNotFound
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
