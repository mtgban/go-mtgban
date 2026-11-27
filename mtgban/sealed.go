package mtgban

// SealedVia says how a store's sealed product reached a datastore product, or
// why it did not.
type SealedVia int

const (
	// SealedByID means the datastore carries the store's own id.
	SealedByID SealedVia = iota
	// SealedByBridge means it was placed through another store's id.
	SealedByBridge
	// SealedByName means it was placed by the name resolver.
	SealedByName
	// SealedSkipped means a rule says the datastore does not carry it.
	SealedSkipped
	// SealedRefused means the resolver could not place it.
	SealedRefused
)

// String names the value, for logs and test failures.
func (via SealedVia) String() string {
	switch via {
	case SealedByID:
		return "by id"
	case SealedByBridge:
		return "by bridge"
	case SealedByName:
		return "by name"
	case SealedSkipped:
		return "skipped"
	case SealedRefused:
		return "refused"
	}
	return "unknown"
}

// SealedMatch is what a sealed scraper's matching decided for one product of
// a store's catalog, before any stock or price was looked at.
type SealedMatch struct {
	// StoreID is the store's id for the product, as the store writes it.
	StoreID string
	// Name is the product's name, as the store writes it.
	Name string
	// UUID is the datastore product it was placed on, set for SealedByID,
	// SealedByBridge and SealedByName. Where the store maps one id to
	// several, it is the one a plain nonfoil listing is priced onto.
	UUID string
	// Via says how it was placed, or that it was not.
	Via SealedVia
	// Reason says why, set for SealedSkipped and SealedRefused.
	Reason string
}
