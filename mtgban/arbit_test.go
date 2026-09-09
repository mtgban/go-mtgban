package mtgban

import (
	"math"
	"reflect"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// installCards puts a datastore behind GetUUID for the duration of one test.
// Every arbitrage function reads the card to filter on it, so a card the
// datastore does not hold is skipped outright and the arithmetic below would
// never run.
func installCards(t *testing.T, cards map[string]*mtgmatcher.CardObject) {
	t.Helper()
	previous := mtgmatcher.GlobalDatastore()
	mtgmatcher.SetGlobalDatastore(&mtgmatcher.Backend{UUIDs: cards})
	t.Cleanup(func() {
		mtgmatcher.SetGlobalDatastore(previous)
	})
}

func plainCard() map[string]*mtgmatcher.CardObject {
	return map[string]*mtgmatcher.CardObject{
		"card": {
			Card: mtgmatcher.Card{
				Name: "Plain Card", Rarity: "rare", SetCode: "AAA", Number: "10",
			},
			Edition: "Alpha Set",
		},
	}
}

func sellerOf(inv InventoryRecord, info ScraperInfo) Seller {
	return NewSellerFromInventory(inv, info)
}

func vendorOf(bl BuylistRecord) Vendor {
	return NewVendorFromBuylist(bl, ScraperInfo{Name: "vendor"})
}

// TestArbitReportsTheTrade pins the arithmetic every row is built from: the
// difference and the spread the caller acts on, the quantity the smaller side
// allows, and the profitability index that ranks one row against another.
func TestArbitReportsTheTrade(t *testing.T) {
	installCards(t, plainCard())

	seller := sellerOf(InventoryRecord{
		"card": {{Conditions: "NM", Price: 10, Quantity: 4}},
	}, ScraperInfo{Name: "seller"})
	vendor := vendorOf(BuylistRecord{
		"card": {{Conditions: "NM", BuyPrice: 15, Quantity: 3}},
	})

	entries := Arbit(nil, vendor, seller)
	if len(entries) != 1 {
		t.Fatalf("Arbit returned %d entries, want 1", len(entries))
	}
	got := entries[0]
	if got.Difference != 5 {
		t.Errorf("Difference = %v, want 5", got.Difference)
	}
	if got.Spread != 50 {
		t.Errorf("Spread = %v, want 50", got.Spread)
	}
	// The vendor only buys three of the four on the shelf.
	if got.Quantity != 3 {
		t.Errorf("Quantity = %v, want 3", got.Quantity)
	}
	if got.AbsoluteDifference != 15 {
		t.Errorf("AbsoluteDifference = %v, want 15", got.AbsoluteDifference)
	}
	want := (5.0 / 10.0) * math.Log10(1+50) * math.Sqrt(3)
	if math.Abs(got.Profitability-want) > 1e-9 {
		t.Errorf("Profitability = %v, want %v", got.Profitability, want)
	}
}

// A single copy is not multiplied by the square root of one unit, which would
// be the same number, but the branch is what says so.
func TestArbitDoesNotScaleASingleCopy(t *testing.T) {
	installCards(t, plainCard())

	entries := Arbit(nil,
		vendorOf(BuylistRecord{"card": {{Conditions: "NM", BuyPrice: 15}}}),
		sellerOf(InventoryRecord{"card": {{Conditions: "NM", Price: 10, Quantity: 1}}},
			ScraperInfo{Name: "seller"}))
	if len(entries) != 1 {
		t.Fatalf("Arbit returned %d entries, want 1", len(entries))
	}
	want := (5.0 / 10.0) * math.Log10(1+50)
	if math.Abs(entries[0].Profitability-want) > 1e-9 {
		t.Errorf("Profitability = %v, want %v", entries[0].Profitability, want)
	}
}

// TestArbitPricesEachConditionAgainstItsOwn pins the re-anchoring the loop
// does on every inventory entry. The condition match only rebinds the buylist
// entry when the inventory entry is not NM, so a grade carried over from the
// previous iteration would price this one against another entry's offer.
func TestArbitPricesEachConditionAgainstItsOwn(t *testing.T) {
	installCards(t, plainCard())

	// The played copy comes first, so a stale binding would still be held
	// when the NM copy is reached.
	seller := sellerOf(InventoryRecord{
		"card": {
			{Conditions: "MP", Price: 4, Quantity: 1},
			{Conditions: "NM", Price: 10, Quantity: 1},
		},
	}, ScraperInfo{Name: "seller"})
	vendor := vendorOf(BuylistRecord{
		"card": {
			{Conditions: "NM", BuyPrice: 15},
			{Conditions: "MP", BuyPrice: 6},
		},
	})

	entries := Arbit(nil, vendor, seller)
	if len(entries) != 2 {
		t.Fatalf("Arbit returned %d entries, want 2", len(entries))
	}
	for _, entry := range entries {
		if entry.InventoryEntry.Conditions != entry.BuylistEntry.Conditions {
			t.Errorf("a %s copy was priced against the %s offer",
				entry.InventoryEntry.Conditions, entry.BuylistEntry.Conditions)
		}
	}
}

// A condition the vendor does not buy is not sold to it at another grade's
// price.
func TestArbitSkipsAConditionTheVendorDoesNotBuy(t *testing.T) {
	installCards(t, plainCard())

	entries := Arbit(nil,
		vendorOf(BuylistRecord{"card": {{Conditions: "NM", BuyPrice: 15}}}),
		sellerOf(InventoryRecord{"card": {{Conditions: "HP", Price: 1, Quantity: 1}}},
			ScraperInfo{Name: "seller"}))
	if len(entries) != 0 {
		t.Errorf("Arbit returned %d entries, want none", len(entries))
	}
}

// TestArbitFilters walks the thresholds one at a time against a trade that
// clears every one of them by default, so a row going missing names the
// filter that dropped it.
func TestArbitFilters(t *testing.T) {
	base := func() (Vendor, Seller) {
		return vendorOf(BuylistRecord{
				"card": {{Conditions: "NM", BuyPrice: 15, PriceRatio: 60}},
			}), sellerOf(InventoryRecord{
				"card": {{Conditions: "NM", Price: 10, Quantity: 4, SellerName: "shop"}},
			}, ScraperInfo{Name: "seller"})
	}

	for _, tt := range []struct {
		desc string
		opts *ArbitOpts
		want int
	}{
		{"no options keeps the trade", &ArbitOpts{}, 1},
		{"a price floor above the ask drops it", &ArbitOpts{MinPrice: 11}, 0},
		{"a price floor below the ask keeps it", &ArbitOpts{MinPrice: 9}, 1},
		{"a buy floor above the offer drops it", &ArbitOpts{MinBuyPrice: 16}, 0},
		{"a difference floor above the gap drops it", &ArbitOpts{MinDiff: 6}, 0},
		{"a spread floor above the spread drops it", &ArbitOpts{MinSpread: 51}, 0},
		{"a spread ceiling below the spread drops it", &ArbitOpts{MaxSpread: 49}, 0},
		{"a quantity floor above the stock drops it", &ArbitOpts{MinQuantity: 5}, 0},
		{"a price ratio ceiling below the vendor's drops it", &ArbitOpts{MaxPriceRatio: 50}, 0},
		{"a profitability floor above the index drops it", &ArbitOpts{MinProfitability: 100}, 0},
		{"an ignored condition drops it", &ArbitOpts{Conditions: []string{"NM"}}, 0},
		{"an ignored edition drops it", &ArbitOpts{Editions: []string{"Alpha Set"}}, 0},
		{"an ignored set code drops it too", &ArbitOpts{Editions: []string{"AAA"}}, 0},
		{"selecting another edition drops it", &ArbitOpts{OnlyEditions: []string{"Beta Set"}}, 0},
		{"selecting its edition keeps it", &ArbitOpts{OnlyEditions: []string{"Alpha Set"}}, 1},
		{"an ignored rarity drops it", &ArbitOpts{Rarities: []string{"rare"}}, 0},
		{"foils only drops a nonfoil", &ArbitOpts{OnlyFoil: true}, 0},
		{"reserved only drops an unreserved card", &ArbitOpts{OnlyReserveList: true}, 0},
		{"bundles only drops a loose copy", &ArbitOpts{OnlyBundles: true}, 0},
		{"another seller's name drops it", &ArbitOpts{Sellers: []string{"other"}}, 0},
		{"its own seller name keeps it", &ArbitOpts{Sellers: []string{"shop"}}, 1},
		{"a number outside the range drops it", &ArbitOpts{
			OnlyCollectorNumberRanges: map[string][2]int{"Alpha Set": {1, 5}}}, 0},
		{"a number inside the range keeps it", &ArbitOpts{
			OnlyCollectorNumberRanges: map[string][2]int{"Alpha Set": {1, 20}}}, 1},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			installCards(t, plainCard())
			vendor, seller := base()
			if got := len(Arbit(tt.opts, vendor, seller)); got != tt.want {
				t.Errorf("Arbit returned %d entries, want %d", got, tt.want)
			}
		})
	}
}

// A shop that publishes no counts is not held to a quantity floor, or every
// one of its rows would be dropped for saying nothing.
func TestArbitKeepsAQuantitylessSeller(t *testing.T) {
	installCards(t, plainCard())

	entries := Arbit(&ArbitOpts{MinQuantity: 5},
		vendorOf(BuylistRecord{"card": {{Conditions: "NM", BuyPrice: 15}}}),
		sellerOf(InventoryRecord{"card": {{Conditions: "NM", Price: 10}}},
			ScraperInfo{Name: "seller", NoQuantityInventory: true}))
	if len(entries) != 1 {
		t.Fatalf("Arbit returned %d entries, want 1", len(entries))
	}
}

// TestArbitAppliesTheRateAndFactors pins what moves the compared price: the
// flat rate, the card filter's factor and the price filter's, which multiply
// rather than replace one another.
func TestArbitAppliesTheRateAndFactors(t *testing.T) {
	for _, tt := range []struct {
		desc string
		opts *ArbitOpts
		want float64
	}{
		{"the rate scales the ask", &ArbitOpts{Rate: 1.2}, 15 - 12},
		{"a card factor scales it", &ArbitOpts{
			CustomCardFilter: func(*mtgmatcher.CardObject) (float64, bool) { return 0.5, false },
		}, 15 - 5},
		{"a price factor scales it", &ArbitOpts{
			CustomPriceFilter: func(string, InventoryEntry) (float64, bool) { return 2, false },
		}, 15 - 20},
		{"both factors multiply", &ArbitOpts{
			CustomCardFilter:  func(*mtgmatcher.CardObject) (float64, bool) { return 0.5, false },
			CustomPriceFilter: func(string, InventoryEntry) (float64, bool) { return 3, false },
		}, 0}, // the ask lands on the offer: 10 * 0.5 * 3
	} {
		t.Run(tt.desc, func(t *testing.T) {
			installCards(t, plainCard())
			// MinDiff defaults to zero, which would drop a trade that comes
			// out even, so ask for everything down to a loss.
			tt.opts.MinDiff = -1000
			tt.opts.MinSpread = -1000
			entries := Arbit(tt.opts,
				vendorOf(BuylistRecord{"card": {{Conditions: "NM", BuyPrice: 15}}}),
				sellerOf(InventoryRecord{"card": {{Conditions: "NM", Price: 10, Quantity: 1}}},
					ScraperInfo{Name: "seller"}))
			if len(entries) != 1 {
				t.Fatalf("Arbit returned %d entries, want 1", len(entries))
			}
			if math.Abs(entries[0].Difference-tt.want) > 1e-9 {
				t.Errorf("Difference = %v, want %v", entries[0].Difference, tt.want)
			}
		})
	}
}

// Either filter can refuse the card outright rather than reprice it.
func TestArbitFiltersCanSkip(t *testing.T) {
	for _, tt := range []struct {
		desc string
		opts *ArbitOpts
	}{
		{"the card filter", &ArbitOpts{
			CustomCardFilter: func(*mtgmatcher.CardObject) (float64, bool) { return 1, true },
		}},
		{"the price filter", &ArbitOpts{
			CustomPriceFilter: func(string, InventoryEntry) (float64, bool) { return 1, true },
		}},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			installCards(t, plainCard())
			entries := Arbit(tt.opts,
				vendorOf(BuylistRecord{"card": {{Conditions: "NM", BuyPrice: 15}}}),
				sellerOf(InventoryRecord{"card": {{Conditions: "NM", Price: 10, Quantity: 1}}},
					ScraperInfo{Name: "seller"}))
			if len(entries) != 0 {
				t.Errorf("Arbit returned %d entries, want none", len(entries))
			}
		})
	}
}

// A card the datastore does not hold is skipped: nothing can be said about
// what it is, so nothing is said about the trade.
func TestArbitSkipsAnUnknownCard(t *testing.T) {
	installCards(t, map[string]*mtgmatcher.CardObject{})

	entries := Arbit(nil,
		vendorOf(BuylistRecord{"card": {{Conditions: "NM", BuyPrice: 15}}}),
		sellerOf(InventoryRecord{"card": {{Conditions: "NM", Price: 10, Quantity: 1}}},
			ScraperInfo{Name: "seller"}))
	if len(entries) != 0 {
		t.Errorf("Arbit returned %d entries, want none", len(entries))
	}
}

// A card only one side carries is not a trade.
func TestArbitNeedsBothSides(t *testing.T) {
	installCards(t, plainCard())

	entries := Arbit(nil,
		vendorOf(BuylistRecord{"card": {{Conditions: "NM", BuyPrice: 15}}}),
		sellerOf(InventoryRecord{"other": {{Conditions: "NM", Price: 10, Quantity: 1}}},
			ScraperInfo{Name: "seller"}))
	if len(entries) != 0 {
		t.Errorf("Arbit returned %d entries, want none", len(entries))
	}
}

// TestArbitCardFilters covers the filters read off the card rather than the
// offer, which need a card that carries the property to be filtered on.
func TestArbitCardFilters(t *testing.T) {
	for _, tt := range []struct {
		desc string
		card *mtgmatcher.CardObject
		opts *ArbitOpts
		want int
	}{
		{"a foil is dropped when foils are excluded",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Shiny"}, Foil: true},
			&ArbitOpts{NoFoil: true}, 0},
		{"an etched card counts as one",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Shiny"}, Etched: true},
			&ArbitOpts{NoFoil: true}, 0},
		{"a foil is kept when only foils are wanted",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Shiny"}, Foil: true},
			&ArbitOpts{OnlyFoil: true}, 1},
		{"an ignored language drops the card",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Card", Language: "Japanese"}},
			&ArbitOpts{Languages: []string{"Japanese"}}, 0},
		{"selecting another language drops it",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Card", Language: "Japanese"}},
			&ArbitOpts{OnlyLanguages: []string{"English"}}, 0},
		{"selecting its own language keeps it",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Card", Language: "Japanese"}},
			&ArbitOpts{OnlyLanguages: []string{"Japanese"}}, 1},
		{"a sealed product with no decklist is dropped when one is required",
			&mtgmatcher.CardObject{Card: mtgmatcher.Card{Name: "Box"}, Sealed: true},
			&ArbitOpts{SealedDecklist: true}, 0},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			installCards(t, map[string]*mtgmatcher.CardObject{"card": tt.card})
			got := Arbit(tt.opts,
				vendorOf(BuylistRecord{"card": {{Conditions: "NM", BuyPrice: 15}}}),
				sellerOf(InventoryRecord{"card": {{Conditions: "NM", Price: 10, Quantity: 1}}},
					ScraperInfo{Name: "seller"}))
			if len(got) != tt.want {
				t.Errorf("Arbit returned %d entries, want %d", len(got), tt.want)
			}
		})
	}
}

// A side priced at nothing is not a trade, whichever side it is.
func TestArbitSkipsAPriceOfNothing(t *testing.T) {
	for _, tt := range []struct {
		desc              string
		askPrice, buyPice float64
	}{
		{"the shelf asks nothing", 0, 15},
		{"the vendor offers nothing", 10, 0},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			installCards(t, plainCard())
			got := Arbit(nil,
				vendorOf(BuylistRecord{"card": {{Conditions: "NM", BuyPrice: tt.buyPice}}}),
				sellerOf(InventoryRecord{"card": {{Conditions: "NM", Price: tt.askPrice, Quantity: 1}}},
					ScraperInfo{Name: "seller"}))
			if len(got) != 0 {
				t.Errorf("Arbit returned %d entries, want none", len(got))
			}
		})
	}
}

// The buy floor is checked again once the condition is matched: the NM offer
// that opened the card may clear it where the played offer being traded
// against does not.
func TestArbitRechecksTheBuyFloorPerCondition(t *testing.T) {
	installCards(t, plainCard())

	entries := Arbit(&ArbitOpts{MinBuyPrice: 10},
		vendorOf(BuylistRecord{"card": {
			{Conditions: "NM", BuyPrice: 15},
			{Conditions: "MP", BuyPrice: 5},
		}}),
		sellerOf(InventoryRecord{"card": {
			{Conditions: "MP", Price: 1, Quantity: 1},
		}}, ScraperInfo{Name: "seller"}))
	if len(entries) != 0 {
		t.Errorf("Arbit returned %d entries, want none: the played offer is below the floor", len(entries))
	}
}

// TestArbitrageReturnsACompleteRow pins what the quote is for: the row comes
// back carrying the side it was made against, rather than a caller having to
// finish it. A row that reached the caller half-filled would say nothing
// about where its number came from.
func TestArbitrageReturnsACompleteRow(t *testing.T) {
	r := resolveOpts(nil)

	bought := InventoryEntry{Conditions: "NM", Price: 10, Quantity: 2}
	offer := BuylistEntry{Conditions: "NM", BuyPrice: 15, Quantity: 5}
	row, ok := r.arbitrage("card", bought, bought.Price, buys(offer))
	if !ok {
		t.Fatal("arbitrage refused a trade that clears every threshold")
	}
	if !reflect.DeepEqual(row.BuylistEntry, offer) {
		t.Errorf("BuylistEntry = %+v, want the offer it was made against", row.BuylistEntry)
	}
	if !reflect.DeepEqual(row.ReferenceEntry, InventoryEntry{}) {
		t.Errorf("a buylist row carries a reference entry: %+v", row.ReferenceEntry)
	}

	shelf := InventoryEntry{Conditions: "NM", Price: 15, Quantity: 5}
	row, ok = r.arbitrage("card", bought, bought.Price, asks(shelf, shelf.Price))
	if !ok {
		t.Fatal("arbitrage refused a comparison that clears every threshold")
	}
	if !reflect.DeepEqual(row.ReferenceEntry, shelf) {
		t.Errorf("ReferenceEntry = %+v, want the shelf it was compared against", row.ReferenceEntry)
	}
	if !reflect.DeepEqual(row.BuylistEntry, BuylistEntry{}) {
		t.Errorf("a shelf row carries a buylist entry: %+v", row.BuylistEntry)
	}

	// Either side quoting nothing is not a comparison.
	if _, ok := r.arbitrage("card", bought, bought.Price, buys(BuylistEntry{})); ok {
		t.Error("arbitrage took a quote of nothing")
	}
}
