// Package mtgban defines interfaces for scrapers and utility functions
// to obtain pricing information from various vendors.
package mtgban

import (
	"github.com/kodabb/go-mtgban/mtgjson"
)

type LogCallbackFunc func(format string, a ...interface{})

// Card is a generic card representation using fields defined by the MTGJSON project.
// This does not hold true for ancillary cards, such as tokens and emblems, in which
// the vendor custom data is returned as-is.
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

// Entry is a generic association of a specific card with its pricing and
// quantity data as available from the scraped website.
type Entry interface {
	// CanonicalCard returns a generic Card representation.
	// The db argument is provided by mtgjson.LoadAllPrintings function.
	//
	// If the card cannot be matched with a MTGJSON card entry, an error is returned.
	// Users may still access the underlying data type with type casting.
	CanonicalCard(db mtgjson.MTGDB) (*Card, error)

	// Price returns the pricing information associated to a card.
	Price() float64

	// TradePrice returns the trade-in credit associated with a card.
	TradePrice() float64

	// Quantity return the amount of the card available or required.
	Quantity() int

	Market() string
	Conditions() string
}

// A Scraper is used to efficiently retrieve information from a vendor website.
type Scraper interface {
	// Scrape returns an array of Entry, containing pricing and card information
	// from the given vendor website.
	Scrape() ([]Entry, error)

	//TODO
	//LoadCSV(io.Reader) ([]Entry, error)
	//Write(CSV(io.Writer) error
}
