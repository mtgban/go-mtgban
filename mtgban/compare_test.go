package mtgban

import (
	"maps"
	"slices"
)

// These stop compiling when either entry gains a field, so the comparisons
// below cannot fall behind one without anyone noticing.
var (
	_ = InventoryEntry{0, "", 0, "", "", false, "", "", nil, nil}
	_ = BuylistEntry{0, "", 0, 0, "", "", "", "", nil}
)

// sameInventoryEntry compares every field of two inventory entries.
func sameInventoryEntry(a, b InventoryEntry) bool {
	return a.Quantity == b.Quantity && a.Conditions == b.Conditions && a.Price == b.Price &&
		a.URL == b.URL && a.SellerName == b.SellerName && a.Bundle == b.Bundle &&
		a.OriginalID == b.OriginalID && a.InstanceID == b.InstanceID &&
		maps.Equal(a.CustomFields, b.CustomFields) && maps.Equal(a.ExtraValues, b.ExtraValues)
}

// sameBuylistEntry compares every field of two buylist entries.
func sameBuylistEntry(a, b BuylistEntry) bool {
	return a.Quantity == b.Quantity && a.Conditions == b.Conditions && a.BuyPrice == b.BuyPrice &&
		a.PriceRatio == b.PriceRatio && a.URL == b.URL && a.VendorName == b.VendorName &&
		a.OriginalID == b.OriginalID && a.InstanceID == b.InstanceID &&
		maps.Equal(a.CustomFields, b.CustomFields)
}

// sameInventory compares two inventories card by card, entry by entry.
func sameInventory(a, b InventoryRecord) bool {
	return maps.EqualFunc(a, b, func(x, y []InventoryEntry) bool {
		return slices.EqualFunc(x, y, sameInventoryEntry)
	})
}

// sameBuylist compares two buylists card by card, entry by entry.
func sameBuylist(a, b BuylistRecord) bool {
	return maps.EqualFunc(a, b, func(x, y []BuylistEntry) bool {
		return slices.EqualFunc(x, y, sameBuylistEntry)
	})
}
