package mtgban

import (
	"math"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
	"golang.org/x/exp/slices"
)

type ArbitOpts struct {
	// Extra factor to modify Inventory prices
	Rate float64

	// Minimum Inventory price
	MinPrice float64

	// Minimum Buylist price
	MinBuyPrice float64

	// Minimum difference between prices
	MinDiff float64

	// Minimum Inventory quantities
	MinQuantity int

	// Minimum spread % between prices
	MinSpread float64

	// Maximum Spread % between prices
	MaxSpread float64

	// Maximum price ratio of Inventory
	MaxPriceRatio float64

	// Use credit for Buylist prices
	UseTrades bool

	// Whether to consider foils
	NoFoil   bool
	OnlyFoil bool

	// Whether to skip non-rl cards
	OnlyReserveList bool

	// List of conditions to ignore
	Conditions []string

	// List of rarities to ignore
	Rarities []string

	// List of editions (or set codes) to ignore
	Editions []string

	// List of editions (or set codes) to select
	OnlyEditions []string

	// List of per-edition collector numbers to select
	OnlyCollectorNumberRanges map[string][2]int

	// Only run for products with static decklists
	SealedDecklist bool

	// Only select entries which are part of a bundle
	OnlyBundles bool

	// List of seller name that wil be considered
	Sellers []string

	// Custom function to be run on the card
	// It returns a custom factor to be applied on the buylist price,
	// and whether the entry shoul be skipped
	CustomCardFilter func(co *mtgmatcher.CardObject) (float64, bool)

	// Constant used to offset prices (the higher the value, the less impactful
	// lower prices will be, and viceversa)
	ProfitabilityConstant float64

	// Minimum profitability value
	MinProfitability float64
}

type ArbitEntry struct {
	// ID of the card
	CardId string

	// The buylist used to determine Arbit
	BuylistEntry BuylistEntry

	// The actual entry that matches either of the above
	InventoryEntry InventoryEntry

	// The inventory used to determine Mismatch
	ReferenceEntry InventoryEntry

	// Difference of the prices
	Difference float64

	// Spread between the the prices
	Spread float64

	// Difference of the prices accounting for quantities available
	AbsoluteDifference float64

	// Amount of cards that can be applied
	Quantity int

	// The higher the number the better the arbit is. Using this formula
	// Profitability Index (PI) = (Difference / (Sell Price + 10)) * log(1 + Spread) * sqrt(Units)
	Profitability float64
}

func Arbit(opts *ArbitOpts, vendor Vendor, seller Seller) (result []ArbitEntry, err error) {
	minDiff := 0.0
	minSpread := 0.0
	useTrades := false
	rate := 1.0
	profitabilityConstant := 0.0

	minPrice := 0.0
	minBuyPrice := 0.0
	minQty := 0
	maxSpread := 0.0
	maxPriceRatio := 0.0
	minProfitability := 0.0
	filterFoil := false
	filterOnlyFoil := false
	filterRLOnly := false
	filterDecksOnly := false
	filterBundle := false
	var filterConditions []string
	var filterRarities []string
	var filterEditions []string
	var filterSelectedEditions []string
	var filterSelectedCNRange map[string][2]int
	var filterSellers []string
	var filterFunc func(co *mtgmatcher.CardObject) (float64, bool)

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
		if opts.ProfitabilityConstant > 0 {
			profitabilityConstant = opts.ProfitabilityConstant
		}
		useTrades = opts.UseTrades

		minPrice = opts.MinPrice
		minBuyPrice = opts.MinBuyPrice
		minQty = opts.MinQuantity
		maxPriceRatio = opts.MaxPriceRatio
		maxSpread = opts.MaxSpread
		minProfitability = opts.MinProfitability
		filterFoil = opts.NoFoil
		filterOnlyFoil = opts.OnlyFoil
		filterRLOnly = opts.OnlyReserveList
		filterDecksOnly = opts.SealedDecklist
		filterBundle = opts.OnlyBundles
		filterFunc = opts.CustomCardFilter

		if len(opts.Conditions) != 0 {
			filterConditions = opts.Conditions
		}
		if len(opts.Rarities) != 0 {
			filterRarities = opts.Rarities
		}
		if len(opts.Editions) != 0 {
			filterEditions = opts.Editions
		}
		if len(opts.OnlyEditions) != 0 {
			filterSelectedEditions = opts.OnlyEditions
		}
		if len(opts.OnlyCollectorNumberRanges) != 0 {
			filterSelectedCNRange = opts.OnlyCollectorNumberRanges
		}
		if len(opts.Sellers) != 0 {
			filterSellers = opts.Sellers
		}
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

		// The first entry is always NM
		blEntry := blEntries[0]

		if maxPriceRatio != 0 && blEntry.PriceRatio > maxPriceRatio {
			continue
		}

		if blEntry.BuyPrice < minBuyPrice {
			continue
		}

		co, err := mtgmatcher.GetUUID(cardId)
		if err != nil {
			continue
		}
		if slices.Contains(filterRarities, co.Rarity) {
			continue
		}
		if filterFoil && (co.Foil || co.Etched) {
			continue
		}
		if filterOnlyFoil && !co.Foil && !co.Etched {
			continue
		}
		if filterRLOnly && !co.IsReserved {
			continue
		}
		if filterDecksOnly && co.Sealed && !mtgmatcher.SealedHasDecklist(co.SetCode, cardId) {
			continue
		}
		if slices.Contains(filterEditions, co.Edition) || slices.Contains(filterEditions, co.SetCode) {
			continue
		}
		if filterSelectedEditions != nil && !slices.Contains(filterSelectedEditions, co.Edition) && !slices.Contains(filterSelectedEditions, co.SetCode) {
			continue
		}
		cnRange, found := filterSelectedCNRange[co.Edition]
		if found {
			cn, err := strconv.Atoi(co.Number)
			if err == nil && (cn < cnRange[0] || cn > cnRange[1]) {
				continue
			}
		}

		if filterFunc != nil {
			factor, skip := filterFunc(co)
			if skip {
				continue
			}
			blEntry.BuyPrice *= factor
		}

		for _, invEntry := range invEntries {
			if slices.Contains(filterConditions, invEntry.Conditions) {
				continue
			}
			if filterSellers != nil && !slices.Contains(filterSellers, invEntry.SellerName) {
				continue
			}
			if filterBundle && !invEntry.Bundle {
				continue
			}
			if !seller.Info().NoQuantityInventory && invEntry.Quantity < minQty {
				continue
			}
			if invEntry.Price < minPrice {
				continue
			}

			price := invEntry.Price * rate

			// When invEntry is not NM, we need to account for conditions
			if invEntry.Conditions != "NM" {
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
			}

			blPrice := blEntry.BuyPrice
			if useTrades {
				blPrice = blEntry.TradePrice
			}

			if price == 0 || blPrice == 0 {
				continue
			}

			// Check again to account for conditions
			if blPrice < minBuyPrice {
				continue
			}

			spread := 100 * (blPrice - price) / price
			difference := blPrice - price

			if maxSpread != 0 && spread > maxSpread {
				continue
			}
			if difference < minDiff {
				continue
			}
			if spread < minSpread {
				continue
			}

			// Find the minimum amount tradable
			qty := invEntry.Quantity
			if blEntry.Quantity != 0 {
				qty = blEntry.Quantity
				if invEntry.Quantity < blEntry.Quantity {
					qty = invEntry.Quantity
				}
			}

			profitability := (difference / (price + profitabilityConstant)) * math.Log(1+spread)
			if qty > 1 {
				profitability *= math.Pow(float64(qty), 0.25)
			}

			if profitability < minProfitability {
				continue
			}

			res := ArbitEntry{
				CardId:             cardId,
				BuylistEntry:       blEntry,
				InventoryEntry:     invEntry,
				Difference:         difference,
				AbsoluteDifference: difference * float64(qty),
				Spread:             spread,
				Quantity:           qty,
				Profitability:      profitability,
			}
			result = append(result, res)
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

	minDiff := 0.0
	minSpread := 0.0

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

// A generic grading map that estimates common deductions
var defaultGradeMap = map[string]float64{
	"NM": 1, "SP": 0.8, "MP": 0.6, "HP": 0.4,
}

func Mismatch(opts *ArbitOpts, reference Seller, probe Seller) (result []ArbitEntry, err error) {
	minDiff := 0.0
	minSpread := 0.0
	maxSpread := 0.0
	minPrice := 0.0
	minQty := 0
	minProfitability := 0.0
	profitabilityConstant := 0.0
	filterFoil := false
	filterOnlyFoil := false
	filterRLOnly := false
	filterDecksOnly := false
	var filterConditions []string
	var filterRarities []string
	var filterEditions []string
	var filterSelectedEditions []string
	var filterSelectedCNRange map[string][2]int

	if opts != nil {
		if opts.MinDiff != 0 {
			minDiff = opts.MinDiff
		}
		if opts.MinSpread != 0 {
			minSpread = opts.MinSpread
		}
		if opts.ProfitabilityConstant > 0 {
			profitabilityConstant = opts.ProfitabilityConstant
		}

		minPrice = opts.MinPrice
		maxSpread = opts.MaxSpread
		minQty = opts.MinQuantity
		minProfitability = opts.MinProfitability
		filterFoil = opts.NoFoil
		filterOnlyFoil = opts.OnlyFoil
		filterRLOnly = opts.OnlyReserveList
		filterDecksOnly = opts.SealedDecklist

		if len(opts.Conditions) != 0 {
			filterConditions = opts.Conditions
		}
		if len(opts.Rarities) != 0 {
			filterRarities = opts.Rarities
		}
		if len(opts.Editions) != 0 {
			filterEditions = opts.Editions
		}
		if len(opts.OnlyEditions) != 0 {
			filterSelectedEditions = opts.OnlyEditions
		}
		if len(opts.OnlyCollectorNumberRanges) != 0 {
			filterSelectedCNRange = opts.OnlyCollectorNumberRanges
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

		co, err := mtgmatcher.GetUUID(cardId)
		if err != nil {
			continue
		}
		if slices.Contains(filterRarities, co.Rarity) {
			continue
		}
		if filterFoil && (co.Foil || co.Etched) {
			continue
		}
		if filterOnlyFoil && !co.Foil && !co.Etched {
			continue
		}
		if filterDecksOnly && co.Sealed && !mtgmatcher.SealedHasDecklist(co.SetCode, cardId) {
			continue
		}
		if filterRLOnly && !co.IsReserved {
			continue
		}
		if slices.Contains(filterEditions, co.Edition) || slices.Contains(filterEditions, co.SetCode) {
			continue
		}
		if filterSelectedEditions != nil && !slices.Contains(filterSelectedEditions, co.Edition) && !slices.Contains(filterSelectedEditions, co.SetCode) {
			continue
		}
		cnRange, found := filterSelectedCNRange[co.Edition]
		if found {
			cn, _ := strconv.Atoi(co.Number)
			if err != nil && (cn < cnRange[0] || cn > cnRange[1]) {
				continue
			}
		}

		for _, refEntry := range refEntries {
			if refEntry.Price == 0 {
				continue
			}
			if slices.Contains(filterConditions, refEntry.Conditions) {
				continue
			}

			for _, invEntry := range invEntries {
				if slices.Contains(filterConditions, invEntry.Conditions) {
					continue
				}
				if !probe.Info().NoQuantityInventory && invEntry.Quantity < minQty {
					continue
				}
				if invEntry.Price < minPrice {
					continue
				}

				refPrice := refEntry.Price
				price := invEntry.Price

				// We need to account for conditions, using a default ladder
				refPrice *= defaultGradeMap[invEntry.Conditions]

				if price == 0 {
					continue
				}

				spread := 100 * (refPrice - price) / price
				difference := refPrice - price

				if maxSpread != 0 && spread > maxSpread {
					continue
				}
				if difference < minDiff {
					continue
				}
				if spread < minSpread {
					continue
				}

				// Find the minimum amount tradable
				qty := invEntry.Quantity
				if refEntry.Quantity != 0 {
					qty = refEntry.Quantity
					if invEntry.Quantity < refEntry.Quantity {
						qty = invEntry.Quantity
					}
				}

				profitability := (difference / (price + profitabilityConstant)) * math.Log(1+spread)
				if qty > 1 {
					profitability *= math.Pow(float64(qty), 0.25)
				}

				if profitability < minProfitability {
					continue
				}

				res := ArbitEntry{
					CardId:         cardId,
					InventoryEntry: invEntry,
					ReferenceEntry: refEntry,
					Difference:     difference,
					Spread:         spread,
					Quantity:       qty,
					Profitability:  profitability,
				}
				result = append(result, res)
			}
		}
	}

	return
}

type PennystockEntry struct {
	CardId string
	InventoryEntry
}

func Pennystock(seller Seller, full bool, thresholds ...float64) (result []PennystockEntry, err error) {
	inventory, err := seller.Inventory()
	if err != nil {
		return nil, err
	}

	for cardId, entries := range inventory {
		co, err := mtgmatcher.GetUUID(cardId)
		if err != nil {
			continue
		}

		isRare := co.Card.Rarity == "rare"
		isMythic := co.Card.Rarity == "mythic"
		isLand := mtgmatcher.IsBasicLand(co.Name)
		isPromo := co.Card.IsPromo || strings.HasSuffix(co.Edition, "Promos")
		if !isRare && !isMythic && !isLand && !isPromo {
			continue
		}

		// Silver is to catch ULST, IsFunny to catch anything after Unfinity
		switch co.BorderColor {
		case "gold", "silver", "white":
			continue
		}
		if co.IsFunny || co.HasPromoType(mtgjson.PromoTypeThickDisplay) {
			continue
		}

		priceThreshold := []float64{0.12, 0.02, 0.05, 0.02, 0.01, 0.02}
		for i := range thresholds {
			if i > len(priceThreshold) {
				break
			}
			if thresholds[i] == 0 {
				continue
			}

			priceThreshold[i] = thresholds[i]
		}

		for _, entry := range entries {
			if entry.Conditions == "PO" || entry.Conditions == "HP" {
				continue
			}

			isFoil := co.Foil || co.Etched
			var pennyMythic, pennyRare, pennyLand, pennyFoil, pennyPromo bool
			pennyMythic = isMythic && !isFoil && entry.Price <= priceThreshold[0]
			if full {
				pennyRare = isRare && ((!isFoil && entry.Price <= priceThreshold[1]) || (co.Foil && entry.Price <= priceThreshold[2]))
				pennyLand = isLand && ((!isFoil && co.Card.IsFullArt) || isFoil) && entry.Price <= priceThreshold[3]
				pennyFoil = isFoil && !isPromo && !isLand && entry.Price <= priceThreshold[4]
				pennyPromo = isPromo && entry.Price <= priceThreshold[5]
			}

			if pennyMythic || pennyRare || pennyLand || pennyFoil || pennyPromo {
				result = append(result, PennystockEntry{
					CardId:         cardId,
					InventoryEntry: entry,
				})
			}
		}
	}
	return
}
