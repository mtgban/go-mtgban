package mtgban

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// ArbitOpts narrows what Arbit, Mismatch and Pennystock will report. Every
// field is a filter whose zero value means "do not filter", so the empty
// struct returns everything the two sides have in common.
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

	// List of seller name that will be considered
	Sellers []string

	// Custom function to be run on the card object
	// It returns a custom factor to be applied on the compared price,
	// and whether the entry should be skipped or not
	CustomCardFilter func(co *mtgmatcher.CardObject) (float64, bool)

	// Custom function to be run on the probed inventory entry
	// It returns a custom factor to be applied on the compared price,
	// and whether the entry should be skipped or not
	CustomPriceFilter func(string, InventoryEntry) (float64, bool)

	// Constant used to offset prices (the higher the value, the less impactful
	// lower prices will be, and vice versa)
	ProfitabilityConstant float64

	// Minimum profitability value
	MinProfitability float64

	// List of languages to ignore
	Languages []string

	// List of languages to select
	OnlyLanguages []string
}

// ArbitEntry is one card worth acting on, carrying both sides of the
// comparison that produced it so a caller can show its working. Which side is
// filled depends on the function that returned it: Arbit sets BuylistEntry
// against InventoryEntry, Mismatch sets ReferenceEntry against InventoryEntry.
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
	// Profitability Index (PI) =
	//   (Difference / (Sell Price + k)) * log10(1 + Spread) * sqrt(Units)
	// where k is ProfitabilityConstant, a denominator-stabilizing constant
	// that keeps cheap cards from dominating (reference value 10; configurable
	// via ArbitOpts, defaults to 0). The sqrt(Units) term is applied only when
	// Units > 1.
	Profitability float64
}

// ArbitEntry implements the Stringer interface
func (ae ArbitEntry) String() string {
	co, err := mtgmatcher.GetUUID(ae.CardId)
	if err != nil {
		return ""
	}
	if ae.BuylistEntry.BuyPrice != 0 {
		return fmt.Sprintf("%s (%d): %0.2f -> %0.2f", co, ae.Quantity, ae.InventoryEntry.Price, ae.BuylistEntry.BuyPrice)
	}
	return fmt.Sprintf("%s (%d): %0.2f ~ %0.2f", co, ae.Quantity, ae.InventoryEntry.Price, ae.ReferenceEntry.Price)
}

// resolvedOpts holds the resolved filter and threshold values from ArbitOpts,
// with defaults applied for nil opts.
type resolvedOpts struct {
	minDiff                float64
	minSpread              float64
	maxSpread              float64
	minPrice               float64
	minBuyPrice            float64
	minQty                 int
	minProfitability       float64
	maxPriceRatio          float64
	rate                   float64
	profitabilityConstant  float64
	filterFoil             bool
	filterOnlyFoil         bool
	filterRLOnly           bool
	filterDecksOnly        bool
	filterBundle           bool
	filterConditions       []string
	filterRarities         []string
	filterEditions         []string
	filterSelectedEditions []string
	filterSelectedCNRange  map[string][2]int
	filterSellers          []string
	filterFunc             func(co *mtgmatcher.CardObject) (float64, bool)
	filterPriceFunc        func(string, InventoryEntry) (float64, bool)

	filterLanguages         []string
	filterSelectedLanguages []string
}

func resolveOpts(opts *ArbitOpts) resolvedOpts {
	r := resolvedOpts{
		rate: 1.0,
	}
	if opts == nil {
		return r
	}

	if opts.MinDiff != 0 {
		r.minDiff = opts.MinDiff
	}
	if opts.MinSpread != 0 {
		r.minSpread = opts.MinSpread
	}
	if opts.Rate != 0 {
		r.rate = opts.Rate
	}
	if opts.ProfitabilityConstant > 0 {
		r.profitabilityConstant = opts.ProfitabilityConstant
	}
	r.minPrice = opts.MinPrice
	r.minBuyPrice = opts.MinBuyPrice
	r.minQty = opts.MinQuantity
	r.maxPriceRatio = opts.MaxPriceRatio
	r.maxSpread = opts.MaxSpread
	r.minProfitability = opts.MinProfitability
	r.filterFoil = opts.NoFoil
	r.filterOnlyFoil = opts.OnlyFoil
	r.filterRLOnly = opts.OnlyReserveList
	r.filterDecksOnly = opts.SealedDecklist
	r.filterBundle = opts.OnlyBundles
	r.filterFunc = opts.CustomCardFilter
	r.filterPriceFunc = opts.CustomPriceFilter
	r.filterConditions = opts.Conditions
	r.filterRarities = opts.Rarities
	r.filterEditions = opts.Editions
	r.filterSelectedEditions = opts.OnlyEditions
	r.filterLanguages = opts.Languages
	r.filterSelectedLanguages = opts.OnlyLanguages
	r.filterSelectedCNRange = opts.OnlyCollectorNumberRanges
	r.filterSellers = opts.Sellers

	return r
}

// filterCard checks whether a card should be skipped based on the resolved
// options. Returns the custom factor and true if the card should be kept.
func (r *resolvedOpts) filterCard(cardID string) (*mtgmatcher.CardObject, float64, bool) {
	co, err := mtgmatcher.GetUUID(cardID)
	if err != nil {
		return nil, 0, false
	}
	if slices.Contains(r.filterRarities, co.Rarity) {
		return nil, 0, false
	}
	if r.filterFoil && (co.Foil || co.Etched) {
		return nil, 0, false
	}
	if r.filterOnlyFoil && !co.Foil && !co.Etched {
		return nil, 0, false
	}
	if r.filterDecksOnly && co.Sealed && !mtgmatcher.SealedHasDecklist(co.SetCode, cardID) {
		return nil, 0, false
	}
	if r.filterRLOnly && !co.IsReserved {
		return nil, 0, false
	}
	if slices.Contains(r.filterEditions, co.Edition) || slices.Contains(r.filterEditions, co.SetCode) {
		return nil, 0, false
	}
	if r.filterSelectedEditions != nil && !slices.Contains(r.filterSelectedEditions, co.Edition) && !slices.Contains(r.filterSelectedEditions, co.SetCode) {
		return nil, 0, false
	}
	if slices.Contains(r.filterLanguages, co.Language) {
		return nil, 0, false
	}
	if r.filterSelectedLanguages != nil && !slices.Contains(r.filterSelectedLanguages, co.Language) {
		return nil, 0, false
	}
	cnRange, found := r.filterSelectedCNRange[co.Edition]
	if found {
		cn, err := strconv.Atoi(co.Number)
		if err == nil && (cn < cnRange[0] || cn > cnRange[1]) {
			return nil, 0, false
		}
	}

	customFactor := 1.0
	if r.filterFunc != nil {
		factor, skip := r.filterFunc(co)
		if skip {
			return nil, 0, false
		}
		customFactor = factor
	}

	return co, customFactor, true
}

// Arbit reports the cards a vendor buys for more than a seller asks, the
// trade that pays for itself. Only cards both sides carry are considered, and
// opts filters the rest.
func Arbit(opts *ArbitOpts, vendor Vendor, seller Seller) []ArbitEntry {
	var result []ArbitEntry

	r := resolveOpts(opts)

	for cardID, blEntries := range vendor.Buylist() {
		invEntries, found := seller.Inventory()[cardID]
		if !found {
			continue
		}

		// The first entry is always NM, and gates the whole card
		nmEntry := blEntries[0]

		if r.maxPriceRatio != 0 && nmEntry.PriceRatio > r.maxPriceRatio {
			continue
		}

		if nmEntry.BuyPrice < r.minBuyPrice {
			continue
		}

		_, customFactor, ok := r.filterCard(cardID)
		if !ok {
			continue
		}

		initialFactor := customFactor
		for _, invEntry := range invEntries {
			// Re-anchor on NM for every inventory entry: the condition
			// matching below only rebinds this when the entry is not NM,
			// so a value carried over from a previous iteration would
			// price this entry against another entry's grade
			blEntry := nmEntry

			if slices.Contains(r.filterConditions, invEntry.Conditions) {
				continue
			}
			if r.filterSellers != nil && !slices.Contains(r.filterSellers, invEntry.SellerName) && !slices.Contains(r.filterSellers, invEntry.CustomFields["SubSellerName"]) {
				continue
			}
			if r.filterBundle && !invEntry.Bundle {
				continue
			}
			if !seller.Info().NoQuantityInventory && invEntry.Quantity < r.minQty {
				continue
			}
			if invEntry.Price < r.minPrice {
				continue
			}

			if r.filterPriceFunc != nil {
				factor, skip := r.filterPriceFunc(cardID, invEntry)
				if skip {
					continue
				}

				customFactor = initialFactor * factor
			}

			// Apply the optional previously established factor
			price := invEntry.Price * customFactor * r.rate

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

			if price == 0 || blPrice == 0 {
				continue
			}

			// Check again to account for conditions
			if blPrice < r.minBuyPrice {
				continue
			}

			spread := 100 * (blPrice - price) / price
			difference := blPrice - price

			if r.maxSpread != 0 && spread > r.maxSpread {
				continue
			}
			if difference < r.minDiff {
				continue
			}
			if spread < r.minSpread {
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

			profitability := (difference / (price + r.profitabilityConstant)) * math.Log10(1+spread)
			if qty > 1 {
				profitability *= math.Sqrt(float64(qty))
			}

			if profitability < r.minProfitability {
				continue
			}

			res := ArbitEntry{
				CardId:             cardID,
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

	return result
}

// A generic grading map that estimates common deductions
var defaultGradeMap = map[string]float64{
	"NM": 1, "SP": 0.8, "MP": 0.6, "HP": 0.4, "PO": 0,
}

// Mismatch compares two sellers rather than the two sides of one book,
// reporting where probe asks less than reference for the same card. Same
// arithmetic as Arbit, against a price the market has settled on instead of
// against an offer to buy.
func Mismatch(opts *ArbitOpts, reference Seller, probe Seller) []ArbitEntry {
	var result []ArbitEntry

	r := resolveOpts(opts)

	for cardID, refEntries := range reference.Inventory() {
		invEntries, found := probe.Inventory()[cardID]
		if !found {
			continue
		}

		_, customFactor, ok := r.filterCard(cardID)
		if !ok {
			continue
		}

		initialFactor := customFactor
		for _, refEntry := range refEntries {
			if slices.Contains(r.filterConditions, refEntry.Conditions) {
				continue
			}
			if refEntry.Price < r.minPrice {
				continue
			}

			if r.filterPriceFunc != nil {
				factor, skip := r.filterPriceFunc(cardID, refEntry)
				if skip {
					continue
				}
				customFactor = initialFactor * factor
			}

			for _, invEntry := range invEntries {
				if slices.Contains(r.filterConditions, invEntry.Conditions) {
					continue
				}
				if !probe.Info().NoQuantityInventory && invEntry.Quantity < r.minQty {
					continue
				}
				if invEntry.Price < r.minPrice {
					continue
				}

				// Apply the optional previously established factor
				refPrice := refEntry.Price * customFactor
				price := invEntry.Price

				// Undo the reference's own grade before applying the probe's,
				// otherwise a non-NM reference is compared against a rescaled
				// copy of itself. A zero factor cannot be divided by, so skip
				// the pair rather than report an infinite spread.
				refGrade, found := defaultGradeMap[refEntry.Conditions]
				if !found || refGrade <= 0 {
					continue
				}
				invGrade, found := defaultGradeMap[invEntry.Conditions]
				if !found || invGrade <= 0 {
					continue
				}
				refPrice *= invGrade / refGrade

				if price == 0 {
					continue
				}

				spread := 100 * (refPrice - price) / price
				difference := refPrice - price

				if r.maxSpread != 0 && spread > r.maxSpread {
					continue
				}
				if difference < r.minDiff {
					continue
				}
				if spread < r.minSpread {
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

				profitability := (difference / (price + r.profitabilityConstant)) * math.Log10(1+spread)
				if qty > 1 {
					profitability *= math.Sqrt(float64(qty))
				}

				if profitability < r.minProfitability {
					continue
				}

				res := ArbitEntry{
					CardId:         cardID,
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

	return result
}

// Pennystock reports cards priced near the floor that have somewhere to fall
// from: rares, mythics, basic lands and promos, skipping the borders and promo
// types that are cheap for reasons which will not change. thresholds overrides
// the per-rarity ceilings in order, and a zero leaves that position at its
// default.
func Pennystock(seller Seller, full bool, thresholds ...float64) []ArbitEntry {
	var result []ArbitEntry

	for cardID, entries := range seller.Inventory() {
		co, err := mtgmatcher.GetUUID(cardID)
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
		if co.IsFunny || co.HasPromoType(magic.PromoTypeThickDisplay) {
			continue
		}

		priceThreshold := []float64{0.12, 0.02, 0.05, 0.02, 0.01, 0.02}
		for i := range thresholds {
			// The last index this slice holds is len-1, so the equal case
			// is one past the end rather than the final entry
			if i >= len(priceThreshold) {
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
				result = append(result, ArbitEntry{
					CardId:         cardID,
					InventoryEntry: entry,
				})
			}
		}
	}

	return result
}
