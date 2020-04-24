package mtgban

import "github.com/kodabb/go-mtgban/mtgdb"

const (
	DefaultArbitMinDiff        = 0.2
	DefaultArbitMinSpread      = 25.0
	DefaultMismatchMinDiff     = 1.0
	DefaultMismatchMinSpread   = 100.0
	DefaultMultiArbitMinDiff   = 0.5
	DefaultMultiArbitMinSpread = 50.0
)

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

		buylistPrice := blEntry.BuyPrice
		if useTrades {
			buylistPrice = blEntry.TradePrice
		}

		grade := vendor.Grading(card, blEntry)
		for _, invEntry := range invEntries {
			price := invEntry.Price * rate
			blPrice := buylistPrice

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

type MultiArbitEntry struct {
	SellerName string
	VendorName string

	Quantity int
	Entries  []ArbitEntry

	Price        float64
	BuylistPrice float64

	Difference float64
	Spread     float64
}

type MultiArbitOpts struct {
	Options *ArbitOpts
	Extra   float64

	MinDifference float64
	MinSpread     float64
}

func MultiArbit(opts *MultiArbitOpts, vendor Vendor, market Market) (result []MultiArbitEntry, err error) {
	sellers, err := Seller2Sellers(market)
	if err != nil {
		return
	}

	minDiff := DefaultMultiArbitMinDiff
	minSpread := DefaultMultiArbitMinSpread

	extra := 0.0
	var arbitOpts *ArbitOpts
	if opts != nil {
		arbitOpts = opts.Options
		extra = opts.Extra
		if opts.MinDifference != 0 {
			minDiff = opts.MinDifference
		}
		if opts.MinSpread != 0 {
			minSpread = opts.MinSpread
		}
	}

	for _, seller := range sellers {
		arbit, err := Arbit(arbitOpts, vendor, seller)
		if err != nil {
			return nil, err
		}

		if len(arbit) == 0 {
			continue
		}

		quantity := 0
		totalPrice := extra
		totalBuylistPrice := 0.0
		for _, entry := range arbit {
			grade := vendor.Grading(entry.Card, entry.BuylistEntry)

			blPrice := entry.BuylistEntry.BuyPrice
			cond := entry.InventoryEntry.Conditions
			if cond != "NM" {
				blPrice *= grade[cond]
			}

			quantity += entry.InventoryEntry.Quantity
			totalPrice += entry.InventoryEntry.Price * float64(entry.InventoryEntry.Quantity)
			totalBuylistPrice += blPrice * float64(entry.InventoryEntry.Quantity)
		}

		spread := 100 * (totalBuylistPrice - totalPrice) / totalPrice
		difference := totalBuylistPrice - totalPrice

		if difference > minDiff && spread > minSpread {
			res := MultiArbitEntry{
				SellerName:   seller.Info().Name,
				VendorName:   vendor.Info().Name,
				Entries:      arbit,
				Quantity:     quantity,
				Price:        totalPrice,
				BuylistPrice: totalBuylistPrice,
				Spread:       spread,
				Difference:   difference,
			}
			result = append(result, res)
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
				if refEntry.Conditions != invEntry.Conditions {
					continue
				}
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
