package starcitygames

import (
	"math"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// bulkRateEpsilon absorbs the last bit of a decimal parse. The closest two
// rates any tier quotes are a tenth of a cent apart, so nothing this wide can
// reach a neighbouring rate.
const bulkRateEpsilon = 1e-6

// scgMinBuyPrice is the least a published offer can be worth. The catalog
// quotes fractions of a cent on worn commons, $0.005 being its commonest such
// figure, and no one transacts at half a cent.
const scgMinBuyPrice = 0.01

// bulkBuyRates returns the prices SCG quotes for a single's bulk tier rather
// than for the card.
//
// The sell-your-cards index files every single under a bulk category and a
// status: buying_at_cost is an offer on that card, buying_in_bulk the rate for
// a whole tier. A tier quotes one number - every Magic Rare in it $0.08, every
// Common $0.008, 96,975 of the 159,874 Magic products - so the figure is a
// shelf label and not a bid, and publishing it prices the card at whatever the
// tier pays.
//
// The catalog export carries neither field: it has 14 keys per product and 8
// per variant, and none of them says which tier a card sits in. The tier does
// follow from rarity and finish, and each quotes only a handful of prices, so
// matching a variant's sell_list_price against its own tier's rates recovers
// the distinction. Measured against the index across all four games it splits
// 124,892 bulk rows from 80,489 offers, at 99.1% precision and 99.9% recall.
//
// A rarity or finish named nowhere here yields no rates and keeps its price,
// so an unfamiliar product loses nothing. That also covers the tiers SCG
// prices per card everywhere - Flesh and Blood commons, Riftbound non-foils -
// which appear under no rate at all.
func bulkBuyRates(game int, rarity, finish string) []float64 {
	foil := finish != "Non-foil"

	switch game {
	case GameMagic:
		switch rarity {
		case "Token":
			return []float64{0.001}
		case "Promo":
			return []float64{0.03}
		case "Basic Land":
			if foil {
				return []float64{0.01}
			}
			return []float64{0.01, 0.015}
		case "Rare":
			if foil {
				return []float64{0.1}
			}
			// Unstable's cards are ordinary rares and mythics to the
			// catalog and their own tenth-of-a-cent tier to the buylist,
			// 80 of the first and 42 of the second.
			return []float64{0.001, 0.08}
		case "Mythic Rare":
			if foil {
				return []float64{0.1}
			}
			return []float64{0.001, 0.25}
		case "Common", "Uncommon", "Special":
			if foil {
				return []float64{0.006, 0.02}
			}
			return []float64{0.006, 0.007, 0.008}
		}

	case GameFleshAndBlood:
		// Cold foils are one tier whatever the card's rarity, except the
		// promos: those are filed under the promo tier, which SCG quotes
		// per card and never in bulk.
		if finish == "Cold Foil" && rarity != "Promo" {
			return []float64{0.25}
		}
		if rarity == "Majestic" {
			if foil {
				return []float64{0.25}
			}
			return []float64{0.05}
		}

	case GameLorcana:
		switch rarity {
		case "Common", "Uncommon":
			if foil {
				return []float64{0.02}
			}
			return []float64{0.005}
		case "Promo":
			if foil {
				return []float64{0.02}
			}
		case "Rare", "Super Rare":
			if foil {
				return []float64{0.1}
			}
			return []float64{0.05}
		case "Legendary":
			if foil {
				return []float64{0.5}
			}
			return []float64{0.1}
		case "Epic":
			if foil {
				return []float64{0.25}
			}
		}

	case GameRiftbound:
		switch rarity {
		case "Showcase":
			return []float64{0.25}
		case "Rare", "Epic":
			return []float64{0.1}
		case "Common", "Uncommon":
			if foil {
				return []float64{0.03}
			}
		}
	}

	return nil
}

// buylistPrice reads a variant's buylist price and reports whether the shop is
// bidding on the card, and whether the figure it published was its bulk tier's
// rate. The two are not opposites: a figure can be neither, being empty,
// unparseable, or too small to be an offer at all.
//
// The tier is read before the floor is applied because most of what a tier
// quotes is itself under a cent, and the count of rates is what says this
// table is still current.
func buylistPrice(game int, p CatalogProduct, sellListPrice string) (price float64, priced, bulk bool) {
	price, err := mtgmatcher.ParsePrice(sellListPrice)
	if err != nil || price <= 0 {
		return 0, false, false
	}
	for _, rate := range bulkBuyRates(game, p.Rarity, p.Finish) {
		if math.Abs(price-rate) < bulkRateEpsilon {
			return 0, false, true
		}
	}
	if price < scgMinBuyPrice {
		return 0, false, false
	}
	return price, true, false
}
