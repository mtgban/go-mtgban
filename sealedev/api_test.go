package sealedev

import (
	"math"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// installCards puts a datastore behind GetUUID for one test. Every price
// lookup reads the card to know which finish it is quoting.
func installCards(t *testing.T, cards map[string]*mtgmatcher.CardObject) {
	t.Helper()
	mtgmatcher.SetGlobalDatastore(&mtgmatcher.Backend{UUIDs: cards})
	t.Cleanup(func() {
		mtgmatcher.SetGlobalDatastore(&mtgmatcher.Backend{})
	})
}

func priced(conditions map[string]float64) *BanPrice {
	return &BanPrice{Conditions: conditions}
}

// TestGetPriceReadsTheFinishBeingQuoted pins which condition key answers for
// a card. A foil is quoted under its own key, so reading the plain one would
// price a foil at its nonfoil copy's price.
func TestGetPriceReadsTheFinishBeingQuoted(t *testing.T) {
	installCards(t, map[string]*mtgmatcher.CardObject{
		"plain":  {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}},
		"foil":   {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}, Foil: true},
		"etched": {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}, Etched: true},
	})

	conditions := map[string]float64{
		"NM": 1, "NM_foil": 10, "NM_etched": 100,
	}
	for _, tt := range []struct {
		uuid string
		want float64
	}{
		{"plain", 1},
		{"foil", 10},
		{"etched", 100},
	} {
		t.Run(tt.uuid, func(t *testing.T) {
			if got := getPrice(tt.uuid, priced(conditions)); got != tt.want {
				t.Errorf("getPrice(%q) = %v, want %v", tt.uuid, got, tt.want)
			}
		})
	}
}

// A card with no near mint copy is quoted at its played one rather than at
// nothing, which would drop it out of the value entirely.
func TestGetPriceFallsBackToPlayed(t *testing.T) {
	installCards(t, map[string]*mtgmatcher.CardObject{
		"plain": {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}},
		"foil":  {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}, Foil: true},
	})

	if got := getPrice("plain", priced(map[string]float64{"SP": 3})); got != 3 {
		t.Errorf("getPrice = %v, want the played price 3", got)
	}
	// The fallback keeps the finish: a foil falls back to the foil played
	// price, not to the plain one.
	if got := getPrice("foil", priced(map[string]float64{"SP": 3, "SP_foil": 30})); got != 30 {
		t.Errorf("getPrice = %v, want the foil played price 30", got)
	}
}

// TestGetPriceCapsAMisprice pins the guard on a single card's contribution.
// One bad price would otherwise carry the whole product's value with it,
// except in the editions where a four-figure card is ordinary.
func TestGetPriceCapsAMisprice(t *testing.T) {
	installCards(t, map[string]*mtgmatcher.CardObject{
		"modern":  {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}},
		"vintage": {Card: mtgmatcher.Card{Name: "Card", SetCode: "LEA"}},
	})

	over := priced(map[string]float64{"NM": MaxSinglePrice + 1})
	if got := getPrice("modern", over); got != 0 {
		t.Errorf("getPrice = %v, want 0 for a price past the cap", got)
	}
	if got := getPrice("vintage", over); got != MaxSinglePrice+1 {
		t.Errorf("getPrice = %v, want the price kept for a set that reaches it", got)
	}
	// The cap itself is not past the cap.
	if got := getPrice("modern", priced(map[string]float64{"NM": MaxSinglePrice})); got != MaxSinglePrice {
		t.Errorf("getPrice = %v, want the cap itself kept", got)
	}
}

// Nothing to read is worth nothing, rather than a panic.
func TestGetPriceOfNothing(t *testing.T) {
	installCards(t, map[string]*mtgmatcher.CardObject{
		"plain": {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}},
	})
	if got := getPrice("plain", nil); got != 0 {
		t.Errorf("getPrice(nil) = %v, want 0", got)
	}
	if got := getPrice("unknown", priced(map[string]float64{"NM": 5})); got != 0 {
		t.Errorf("getPrice of a card the datastore lacks = %v, want 0", got)
	}
}

// TestMaxStorePriceTakesTheBest pins that the value uses the best price a
// card can be had at across the named stores, not the first one listed.
func TestMaxStorePriceTakesTheBest(t *testing.T) {
	installCards(t, map[string]*mtgmatcher.CardObject{
		"card": {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}},
	})

	prices := map[string]map[string]*BanPrice{
		"card": {
			"CK":  priced(map[string]float64{"NM": 4}),
			"SCG": priced(map[string]float64{"NM": 7}),
			"MP":  priced(map[string]float64{"NM": 2}),
		},
	}
	if got := maxStorePrice("card", prices, []string{"CK", "SCG"}); got != 7 {
		t.Errorf("maxStorePrice = %v, want 7", got)
	}
	if got := maxStorePrice("card", prices, []string{"MP"}); got != 2 {
		t.Errorf("maxStorePrice = %v, want 2", got)
	}
	// A store none of them stock is worth nothing rather than an error.
	if got := maxStorePrice("card", prices, []string{"NOPE"}); got != 0 {
		t.Errorf("maxStorePrice = %v, want 0", got)
	}
}

// TestCT0FeesLadder pins the fee taken off a CardTrader Zero price at each
// step and, more to the point, at the boundaries: the ladder is written with
// <=, so a price sitting exactly on a rung pays that rung's fee.
func TestCT0FeesLadder(t *testing.T) {
	for _, tt := range []struct {
		price, want float64
	}{
		{0.10, 0.09},
		{0.25, 0.09},
		{0.26, 0.10},
		{3, 0.10},
		{5, 0.11},
		{7, 0.14},
		{10, 0.15},
		{15, 0.21},
		{20, 0.27},
		{30, 0.40},
		{40, 0.52},
		{40.01, 0.64},
		{1000, 0.64},
	} {
		if got := getCT0fees(tt.price); got != tt.want {
			t.Errorf("getCT0fees(%v) = %v, want %v", tt.price, got, tt.want)
		}
	}
}

// TestSetPriceKeepsTheFinish pins that a price written back is filed under
// the key the reader will look for, which is the finish's own.
func TestSetPriceKeepsTheFinish(t *testing.T) {
	installCards(t, map[string]*mtgmatcher.CardObject{
		"plain":  {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}},
		"foil":   {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}, Foil: true},
		"etched": {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}, Etched: true},
	})

	r := &BANPriceResponse{
		Retail:  map[string]map[string]*BanPrice{},
		Buylist: map[string]map[string]*BanPrice{},
	}
	for _, uuid := range []string{"plain", "foil", "etched"} {
		r.setRetail(uuid, "CK", 5)
		r.setBuylist(uuid, "CK", 3)
		if got := r.getRetail(uuid, "CK"); got != 5 {
			t.Errorf("getRetail(%q) = %v, want 5 back", uuid, got)
		}
		if got := r.getBuylist(uuid, "CK"); got != 3 {
			t.Errorf("getBuylist(%q) = %v, want 3 back", uuid, got)
		}
	}

	// A card the datastore cannot name has no finish to file under, so
	// nothing is written rather than something filed wrongly.
	r.setRetail("unknown", "CK", 5)
	if _, found := r.Retail["unknown"]; found {
		t.Error("a price was filed for a card the datastore does not hold")
	}
}

// valueFromCache is the sum the whole value rests on: weighted by probability
// for the deterministic pass, unweighted for one simulated draw.
func TestValueFromCache(t *testing.T) {
	unit := map[string]float64{"a": 10, "b": 4, "missing": 0}

	if got := valueFromCache([]string{"a", "b"}, unit, nil); got != 14 {
		t.Errorf("an unweighted draw = %v, want 14", got)
	}
	got := valueFromCache([]string{"a", "b"}, unit, []float64{0.5, 0.25})
	if math.Abs(got-6) > 1e-9 {
		t.Errorf("a weighted pass = %v, want 6", got)
	}
	// A pick with no resolved price contributes nothing rather than
	// dropping the rest of the draw.
	if got := valueFromCache([]string{"a", "unpriced"}, unit, nil); got != 10 {
		t.Errorf("a draw carrying an unpriced pick = %v, want 10", got)
	}
	if got := valueFromCache(nil, unit, nil); got != 0 {
		t.Errorf("an empty draw = %v, want 0", got)
	}
}

// passthroughFirst is what the deterministic parameters use in place of a
// statistic: the single value they produced.
func TestPassthroughFirst(t *testing.T) {
	got, err := passthroughFirst([]float64{7, 9})
	if err != nil || got != 7 {
		t.Errorf("passthroughFirst = %v, %v; want 7, nil", got, err)
	}
}
