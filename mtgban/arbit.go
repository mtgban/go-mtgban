package mtgban

import (
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

const (
	DefaultArbitMinDiff      = 0.2
	DefaultArbitMinSpread    = 25.0
	DefaultMismatchMinDiff   = 1.0
	DefaultMismatchMinSpread = 100.0
)

type ArbitOpts struct {
	Rate float64

	MinDiff   float64
	MinSpread float64

	UseTrades bool
}

type ArbitEntry struct {
	CardId string

	BuylistEntry
	InventoryEntry

	Difference float64
	Spread     float64

	AbsoluteDifference float64

	SellerName string
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

	for cardId, blEntries := range buylist {
		invEntries, found := inventory[cardId]
		if !found {
			continue
		}

		// Look up the first NM printing to use as base
		nmIndex := 0
		if vendor.Info().MultiCondBuylist {
			for nmIndex = range blEntries {
				if blEntries[nmIndex].Conditions == "NM" {
					break
				}
			}
		}
		blEntry := blEntries[nmIndex]

		for _, invEntry := range invEntries {
			price := invEntry.Price * rate

			// When invEntry is not NM, we need to account for conditions, which
			// means either take a percentage off, or use a differen blEntry entirely
			if invEntry.Conditions != "NM" {
				if vendor.Info().MultiCondBuylist {
					i := 0
					for i = range blEntries {
						if blEntries[i].Conditions == invEntry.Conditions {
							break
						}
					}
					blEntry = blEntries[i]
					// If, after looping, a matching condition was not found,
					// just skip the current invEntry
					if blEntry.Conditions != invEntry.Conditions {
						continue
					}
				} else {
					grade := vendor.Info().Grading(cardId, blEntries[nmIndex])
					blEntry.Conditions = invEntry.Conditions
					blEntry.BuyPrice = blEntries[nmIndex].BuyPrice * grade[invEntry.Conditions]
					blEntry.TradePrice = blEntries[nmIndex].TradePrice * grade[invEntry.Conditions]

				}
			}

			blPrice := blEntry.BuyPrice
			if useTrades {
				blPrice = blEntry.TradePrice
			}

			if price == 0 || blPrice == 0 {
				continue
			}

			spread := 100 * (blPrice - price) / price
			difference := blPrice - price

			// Find the minimum amount tradable
			qty := invEntry.Quantity
			if blEntry.Quantity != 0 {
				qty = blEntry.Quantity
				if invEntry.Quantity < blEntry.Quantity {
					qty = invEntry.Quantity
				}
			}

			absDifference := difference * float64(qty)

			if difference > minDiff && spread > minSpread {
				res := ArbitEntry{
					CardId:             cardId,
					BuylistEntry:       blEntry,
					InventoryEntry:     invEntry,
					Difference:         difference,
					AbsoluteDifference: absDifference,
					Spread:             spread,
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

	minDiff := DefaultMismatchMinDiff
	minSpread := DefaultMismatchMinSpread

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
			quantity += entry.InventoryEntry.Quantity
			totalPrice += entry.InventoryEntry.Price * float64(entry.InventoryEntry.Quantity)
			totalBuylistPrice += entry.BuylistEntry.BuyPrice * float64(entry.InventoryEntry.Quantity)
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
	CardId string

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

	for cardId, refEntries := range referenceInv {
		invEntries, found := probeInv[cardId]
		if !found {
			continue
		}

		for _, refEntry := range refEntries {
			if refEntry.Price == 0 {
				continue
			}
			for _, invEntry := range invEntries {
				if refEntry.Conditions != invEntry.Conditions {
					continue
				}
				refPrice := refEntry.Price
				price := invEntry.Price

				if invEntry.Price == 0 {
					continue
				}

				spread := 100 * (refPrice - price) / price
				difference := refPrice - price

				if difference > minDiff && spread > minSpread {
					res := MismatchEntry{
						CardId:         cardId,
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

type PennystockEntry struct {
	CardId string
	InventoryEntry
}

func Pennystock(seller Seller) (result []PennystockEntry, err error) {
	inventory, err := seller.Inventory()
	if err != nil {
		return nil, err
	}

	for cardId, entries := range inventory {
		co, err := mtgmatcher.GetUUID(cardId)
		if err != nil {
			return nil, err
		}
		isRare := co.Card.Rarity == "rare"
		isMythic := co.Card.Rarity == "mythic"
		if !(isRare || isMythic) {
			continue
		}
		if co.Card.BorderColor == "silver" {
			continue
		}
		for _, entry := range entries {
			pennyMythic := !co.Foil && isMythic && entry.Price <= 0.25
			pennyRare := !co.Foil && isRare && entry.Price <= 0.07
			pennyFoil := co.Foil && entry.Price <= 0.05
			pennyInteresting := false
			switch {
			case strings.Contains(co.Card.Name, "Signet") && entry.Price <= 0.20:
				pennyInteresting = true
			case strings.Contains(co.Card.Name, "Talisman") && entry.Price <= 0.20:
				pennyInteresting = true
			case strings.Contains(co.Card.Name, "Diamond") && entry.Price <= 0.20:
				pennyInteresting = true
			case strings.Contains(co.Card.Name, "Curse of") && entry.Price <= 0.25:
				pennyInteresting = true
			case co.Card.Name == "Sakura-Tribe Elder" && entry.Price <= 0.20:
				pennyInteresting = true
			case co.Card.Name == "Sol Ring" && entry.Price <= 1.50:
				pennyInteresting = true
			}

			if pennyMythic || pennyRare || pennyFoil || pennyInteresting {
				result = append(result, PennystockEntry{
					CardId:         cardId,
					InventoryEntry: entry,
				})
			}
		}
	}
	return
}
