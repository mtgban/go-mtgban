package mtgban

import (
	"fmt"
)

const DefaultArbitMinDiff = 0.2
const DefaultArbitMinSpread = 25.0
const DefaultMismatchMinDiff = 1.0
const DefaultMismatchMinSpread = 100.0

type ArbitOpts struct {
	MinDiff   float64
	MinSpread float64

	UseTrades bool
}

type ArbitEntry struct {
	BuylistEntry
	InventoryEntry

	Difference float64
	Spread     float64
}

func Arbit(opts *ArbitOpts, vendor Vendor, seller Seller) (result []ArbitEntry, err error) {
	minDiff := DefaultArbitMinDiff
	minSpread := DefaultArbitMinSpread
	useTrades := false
	if opts != nil {
		if opts.MinDiff != 0 {
			minDiff = opts.MinDiff
		}
		if opts.MinSpread != 0 {
			minSpread = opts.MinSpread
		}
		useTrades = opts.UseTrades
	}

	buylist, err := vendor.Buylist()
	if err != nil {
		return nil, err
	}
	inventory, err := seller.Inventory()
	if err != nil {
		return nil, err
	}

	for key, blEntry := range buylist {
		invEntries, found := inventory[key]
		if !found {
			continue
		}

		blPrice := blEntry.BuyPrice
		if useTrades {
			blPrice = blEntry.TradePrice
		}

		for _, invEntry := range invEntries {
			price := invEntry.Price

			cond := invEntry.Conditions
			if cond != blEntry.Conditions {
				adjust := 1.0
				switch cond {
				case "NM":
				case "SP":
					adjust = 0.75
				case "MP":
					adjust = 0.66
				case "HP":
					adjust = 0.50
				case "PO":
					adjust = 0.33
				default:
					return nil, fmt.Errorf("Unknown %s condition for %q", cond, invEntry)
				}
				blPrice *= adjust
			}

			spread := 100 * (blPrice - price) / price
			difference := blPrice - price

			if difference > minDiff && spread > minSpread {
				res := ArbitEntry{
					BuylistEntry:   blEntry,
					InventoryEntry: invEntry,
					Difference:     difference,
					Spread:         spread,
				}
				result = append(result, res)
			}
		}
	}

	return
}

type MismatchEntry struct {
	InventoryEntry

	Difference float64
	Spread     float64
}

func Mismatch(opts *ArbitOpts, reference Seller, probe Seller) (result []MismatchEntry, err error) {
	minDiff := DefaultMismatchMinDiff
	minSpread := DefaultMismatchMinSpread
	if opts != nil {
		if opts.MinDiff != 0 {
			minDiff = opts.MinDiff
		}
		if opts.MinSpread != 0 {
			minSpread = opts.MinSpread
		}
	}

	referenceInv, err := reference.Inventory()
	if err != nil {
		return nil, err
	}
	probeInv, err := probe.Inventory()
	if err != nil {
		return nil, err
	}

	for key, refEntries := range referenceInv {
		invEntries, found := probeInv[key]
		if !found {
			continue
		}
		for _, refEntry := range refEntries {
			for _, invEntry := range invEntries {
				refPrice := refEntry.Price
				price := invEntry.Price
				spread := 100 * (refPrice - price) / price
				difference := refPrice - price

				if difference > minDiff && spread > minSpread {
					res := MismatchEntry{
						InventoryEntry: invEntry,
						Difference:     difference,
						Spread:         spread,
					}
					result = append(result, res)
				}
			}
		}
	}

	return
}
