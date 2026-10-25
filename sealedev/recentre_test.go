package sealedev

import (
	"context"
	"math"
	"testing"

	"github.com/montanaflynn/stats"
	"github.com/mtgban/go-mtgban/matcher"
)

// boxBackend holds one set, AAA: a booster type drawing one of four cards, a
// pack holding one such booster, and a box holding packs of them. It is built
// by hand, unlike realDatastore's products, because it pins arithmetic rather
// than the shape of a real product.
func boxBackend(packs int) *matcher.Backend {
	b := &matcher.Backend{
		AllSets: []string{"AAA"},
		UUIDs:   map[string]*matcher.CardObject{},
		Sets:    map[string]*matcher.Set{},
	}
	cards := map[string]int{}
	for _, id := range []string{"c0", "c1", "c2", "c3"} {
		b.UUIDs[id] = &matcher.CardObject{Card: matcher.Card{UUID: id, SetCode: "AAA"}}
		cards[id] = 1
	}
	booster := matcher.Booster{
		Sheets:              map[string]matcher.Sheet{"common": {Cards: cards, TotalWeight: len(cards)}},
		BoostersTotalWeight: 1,
	}
	booster.Boosters = append(booster.Boosters, struct {
		Contents map[string]int `json:"contents"`
		Weight   int            `json:"weight"`
	}{Contents: map[string]int{"common": 1}, Weight: 1})

	set := &matcher.Set{Code: "AAA", Booster: map[string]matcher.Booster{"draft": booster}}
	set.SealedProduct = []matcher.SealedProduct{
		{UUID: "pack", Name: "Pack", Contents: map[string][]matcher.SealedContent{
			"pack": {{Set: "aaa", Code: "draft"}},
		}},
		{UUID: "box", Name: "Box", Contents: map[string][]matcher.SealedContent{
			"sealed": {{Set: "aaa", UUID: "pack", Name: "Pack", Count: packs}},
		}},
	}
	for _, product := range set.SealedProduct {
		b.UUIDs[product.UUID] = &matcher.CardObject{Card: matcher.Card{UUID: product.UUID, SetCode: "AAA"}, Sealed: true}
	}
	b.Sets["AAA"] = set
	return b
}

// TestHeldOpeningsAverageToTheExactValue pins the recentring: a box drawing
// its packs from a finite sample of their openings carries that sample's
// error, times every pack it holds, into each of its own openings unless the
// sample's mean is taken out. A thousand packs make that error far larger
// than the box's own spread, so every trial would show it.
func TestHeldOpeningsAverageToTheExactValue(t *testing.T) {
	b := boxBackend(1000)
	r := &priceSnapshot{
		Retail:  map[string]map[string]float64{},
		Buylist: map[string]map[string]float64{},
	}
	for id, price := range map[string]float64{"c0": 0, "c1": 1, "c2": 2, "c3": 10} {
		for _, parameter := range evParameters {
			for _, store := range parameter.SourceStores {
				r.setRetail(b, id, store, price)
				r.setBuylist(b, id, store, price)
			}
		}
	}
	box, err := b.GetUUID("box")
	if err != nil {
		t.Fatal(err)
	}

	for trial := range 5 {
		ss := NewScraper(b, "")
		ss.repetitions = 2000
		ss.prices = r
		ss.held = heldWhole(b)
		ss.kept = map[string]opened{}

		op := ss.open(context.Background(), box)
		if openingsKept(op) == 0 {
			t.Fatalf("trial %d: the box did not open", trial)
		}
		// Only drawing from what it holds opens the pack through open and
		// keeps it; opened card by card, the box would pin nothing here.
		_, found := ss.kept["pack"]
		if !found {
			t.Fatalf("trial %d: the box opened card by card, not from its packs' openings", trial)
		}
		for i := range evParameters {
			if !evParameters[i].Simulation {
				continue
			}
			dataset := op.datasets[i]
			mean, _ := stats.Mean(dataset)
			spread, _ := stats.StandardDeviation(dataset)
			bound := 6 * spread / math.Sqrt(float64(len(dataset)))
			if math.Abs(mean-op.expected[i]) > bound {
				t.Errorf("trial %d: %q averaged %.2f, want its exact %.2f within %.2f",
					trial, evParameters[i].Name, mean, op.expected[i], bound)
			}
		}
	}
}
