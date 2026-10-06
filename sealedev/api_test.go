package sealedev

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// installCards builds a backend behind GetUUID for one test. Every price
// lookup reads the card to know which finish it is quoting.
func installCards(t *testing.T, cards map[string]*mtgmatcher.CardObject) *mtgmatcher.Backend {
	t.Helper()
	return &mtgmatcher.Backend{UUIDs: cards}
}

func TestSkipFromEV(t *testing.T) {
	for _, test := range []struct {
		name        string
		promoType   string
		probability float64
		wantSkip    bool
	}{
		{name: "serialized", promoType: magic.PromoTypeSerialized, probability: 1, wantSkip: true},
		{name: "cosmic foil", promoType: magic.PromoTypeCosmicFoil, probability: 1, wantSkip: true},
		{name: "bonus without fixed distribution", promoType: magic.PromoTypeSLDBonus, probability: 0.5, wantSkip: true},
		{name: "bonus with fixed distribution", promoType: magic.PromoTypeSLDBonus, probability: 1, wantSkip: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			co := &mtgmatcher.CardObject{Card: mtgmatcher.Card{
				PromoTypes: []string{test.promoType},
			}}
			if got := skipFromEV(co, nil, test.probability); got != test.wantSkip {
				t.Errorf("skipFromEV() = %v, want %v", got, test.wantSkip)
			}
		})
	}

	if !skipFromEV(nil, mtgmatcher.ErrCardUnknownID, 1) {
		t.Error("skipFromEV did not exclude an unresolvable card")
	}
	if skipFromEV(&mtgmatcher.CardObject{}, nil, 1) {
		t.Error("skipFromEV excluded an ordinary card")
	}
}

// TestReadSideReadsTheFinishBeingQuoted pins which finish answers for a
// card: each is quoted under its own, so reading another would price a foil
// at its nonfoil copy's price. It decodes the API's own JSON, so the field
// names are pinned too.
func TestReadSideReadsTheFinishBeingQuoted(t *testing.T) {
	b := installCards(t, map[string]*mtgmatcher.CardObject{
		"plain":  {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA", Finish: "nonfoil"}},
		"foil":   {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA", Finish: "foil"}, Foil: true},
		"etched": {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA", Finish: "etched"}, Etched: true},
	})
	finishes := `{"nonfoil": {"CK": [{"condition": "NM", "price": 1}]},
		"foil": {"CK": [{"condition": "NM", "price": 10}]},
		"etched": {"CK": [{"condition": "NM", "price": 100}]}}`
	doc := `{"retail": {"plain": ` + finishes + `, "foil": ` + finishes + `, "etched": ` + finishes +
		`, "unknown": ` + finishes + `}}`
	var raw v2Response
	err := json.Unmarshal([]byte(doc), &raw)
	if err != nil {
		t.Fatal(err)
	}
	prices := readSide(b, raw.Retail)
	for uuid, want := range map[string]float64{"plain": 1, "foil": 10, "etched": 100} {
		if got := prices[uuid]["CK"]; got != want {
			t.Errorf("%s read %v, want %v", uuid, got, want)
		}
	}
	if _, found := prices["unknown"]; found {
		t.Error("a price was read for a card the datastore does not hold")
	}
}

// A card with no near mint copy is quoted at its played one rather than at
// nothing, which would drop it out of the value entirely. An index price has
// no condition and counts as near mint.
func TestReadPriceFallsBackToPlayed(t *testing.T) {
	for _, tt := range []struct {
		name    string
		entries []v2Entry
		want    float64
	}{
		{"near mint", []v2Entry{{"NM", 2}, {"SP", 3}}, 2},
		{"played only", []v2Entry{{"SP", 3}, {"MP", 1}}, 3},
		{"index", []v2Entry{{"", 4}}, 4},
		{"worse than played only", []v2Entry{{"MP", 1}, {"HP", 0.5}}, 0},
		{"nothing", nil, 0},
	} {
		if got := readPrice(tt.entries); got != tt.want {
			t.Errorf("%s: readPrice = %v, want %v", tt.name, got, tt.want)
		}
	}
}

// TestGetPriceCapsAMisprice pins the guard on a single card's contribution.
// One bad price would otherwise carry the whole product's value with it,
// except in the editions where a four-figure card is ordinary.
func TestGetPriceCapsAMisprice(t *testing.T) {
	b := installCards(t, map[string]*mtgmatcher.CardObject{
		"modern":  {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}},
		"vintage": {Card: mtgmatcher.Card{Name: "Card", SetCode: "LEA"}},
	})

	if got := getPrice(b, "modern", MaxSinglePrice+1); got != 0 {
		t.Errorf("getPrice = %v, want 0 for a price past the cap", got)
	}
	if got := getPrice(b, "vintage", MaxSinglePrice+1); got != MaxSinglePrice+1 {
		t.Errorf("getPrice = %v, want the price kept for a set that reaches it", got)
	}
	// The cap itself is not past the cap.
	if got := getPrice(b, "modern", MaxSinglePrice); got != MaxSinglePrice {
		t.Errorf("getPrice = %v, want the cap itself kept", got)
	}
	if got := getPrice(b, "unknown", MaxSinglePrice+1); got != 0 {
		t.Errorf("getPrice past the cap for a card the datastore lacks = %v, want 0", got)
	}
}

// TestMaxStorePriceTakesTheBest pins that the value uses the best price a
// card can be had at across the named stores, not the first one listed.
func TestMaxStorePriceTakesTheBest(t *testing.T) {
	b := installCards(t, map[string]*mtgmatcher.CardObject{
		"card": {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}},
	})

	prices := map[string]map[string]float64{
		"card": {"CK": 4, "SCG": 7, "MP": 2},
	}
	if got := maxStorePrice(b, "card", prices, []string{"CK", "SCG"}); got != 7 {
		t.Errorf("maxStorePrice = %v, want 7", got)
	}
	if got := maxStorePrice(b, "card", prices, []string{"MP"}); got != 2 {
		t.Errorf("maxStorePrice = %v, want 2", got)
	}
	// A store none of them stock is worth nothing rather than an error.
	if got := maxStorePrice(b, "card", prices, []string{"NOPE"}); got != 0 {
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

// TestSetPriceReadsBack pins that a price written back is the one read, on
// either side, and that a card the datastore cannot name gets none.
func TestSetPriceReadsBack(t *testing.T) {
	b := installCards(t, map[string]*mtgmatcher.CardObject{
		"plain": {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}},
		"foil":  {Card: mtgmatcher.Card{Name: "Card", SetCode: "AAA"}, Foil: true},
	})

	r := &priceSnapshot{
		Retail:  map[string]map[string]float64{},
		Buylist: map[string]map[string]float64{},
	}
	for _, uuid := range []string{"plain", "foil"} {
		r.setRetail(b, uuid, "CK", 5)
		r.setBuylist(b, uuid, "CK", 3)
		if got := r.getRetail(b, uuid, "CK"); got != 5 {
			t.Errorf("getRetail(%q) = %v, want 5 back", uuid, got)
		}
		if got := r.getBuylist(b, uuid, "CK"); got != 3 {
			t.Errorf("getBuylist(%q) = %v, want 3 back", uuid, got)
		}
	}

	r.setRetail(b, "unknown", "CK", 5)
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
