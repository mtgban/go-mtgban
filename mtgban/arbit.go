package mtgban

import "github.com/kodabb/go-mtgban/mtgdb"

const DefaultArbitMinDiff = 0.2
const DefaultArbitMinSpread = 25.0
const DefaultMismatchMinDiff = 1.0
const DefaultMismatchMinSpread = 100.0

type ArbitOpts struct {
	Rate float64

	MinDiff   float64
	MinSpread float64

	UseTrades bool
}

type ArbitEntry struct {
	Card mtgdb.Card

	BuylistEntry
	InventoryEntry

	Difference float64
	Spread     float64
}

func Arbit(opts *ArbitOpts, vendor Vendor, seller Seller) (result []ArbitEntry, err error) {
	minDiff := DefaultArbitMinDiff
	minSpread := DefaultArbitMinSpread
	useTrades := false
	rate := 1.0
	if opts != nil {
		if opts.MinDiff != 0 {
			minDiff = opts.MinDiff
		}
		if opts.MinSpread != 0 {
			minSpread = opts.MinSpread
		}
		if opts.Rate != 0 {
			rate = opts.Rate
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

	for card, blEntry := range buylist {
		invEntries, found := inventory[card]
		if !found {
			continue
		}

		blPrice := blEntry.BuyPrice
		if useTrades {
			blPrice = blEntry.TradePrice
		}

		grade := vendor.Grading(card, blEntry)
		for _, invEntry := range invEntries {
			price := invEntry.Price * rate

			if invEntry.Conditions != "NM" {
				blPrice *= grade[invEntry.Conditions]
			}

			spread := 100 * (blPrice - price) / price
			difference := blPrice - price

			if difference > minDiff && spread > minSpread {
				res := ArbitEntry{
					Card:           card,
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
	Card mtgdb.Card

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

	for card, refEntries := range referenceInv {
		invEntries, found := probeInv[card]
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
						Card:           card,
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
