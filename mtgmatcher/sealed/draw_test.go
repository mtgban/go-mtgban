package sealed_test

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/sealed"
)

// boosterBackend holds one set, AAA, with one booster type drawing count
// cards from sheet in its one layout, published as weighing total, and a card
// for every id in colors, of those colors.
func boosterBackend(sheet mtgmatcher.Sheet, count, total int, colors map[string][]string) *mtgmatcher.Backend {
	b := &mtgmatcher.Backend{
		UUIDs: map[string]*mtgmatcher.CardObject{},
		Sets:  map[string]*mtgmatcher.Set{},
	}
	for id, identity := range colors {
		b.UUIDs[id] = &mtgmatcher.CardObject{Card: mtgmatcher.Card{UUID: id, ColorIdentity: identity}}
	}
	booster := mtgmatcher.Booster{
		Sheets:              map[string]mtgmatcher.Sheet{"common": sheet},
		BoostersTotalWeight: total,
	}
	booster.Boosters = append(booster.Boosters, struct {
		Contents map[string]int `json:"contents"`
		Weight   int            `json:"weight"`
	}{Contents: map[string]int{"common": count}, Weight: 1})
	b.Sets["AAA"] = &mtgmatcher.Set{Code: "AAA", Booster: map[string]mtgmatcher.Booster{"draft": booster}}
	return b
}

// A draw goes by the weights themselves: a card weighing nothing never comes
// up, and published totals that disagree with them change nothing.
func TestBoosterPicksDrawsByTheWeights(t *testing.T) {
	b := boosterBackend(mtgmatcher.Sheet{
		AllowDuplicates: true,
		Cards:           map[string]int{"a": 1, "b": 0, "c": -2},
		TotalWeight:     100,
	}, 3, 7, map[string][]string{"a": nil, "b": nil, "c": nil})

	for range 50 {
		picks, err := sealed.BoosterPicks(b, "AAA", "draft")
		if err != nil {
			t.Fatal(err)
		}
		for _, pick := range picks {
			if pick != "a" {
				t.Fatalf("drew %q from a sheet where only a has weight", pick)
			}
		}
	}
}

// The counts read the same weights the draws do, so a card that is never
// drawn counts for nothing and a wrong published total changes nothing.
func TestSheetCountsGoByTheWeights(t *testing.T) {
	b := boosterBackend(mtgmatcher.Sheet{
		Cards:       map[string]int{"a": 1, "b": 0, "c": -2},
		TotalWeight: 100,
	}, 1, 7, map[string][]string{"a": nil, "b": nil, "c": nil})

	counts, err := sealed.BoosterCounts(b, "AAA", "draft")
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 1 || counts[0].UUID != "a" || counts[0].ExpectedCount != 1 {
		t.Errorf("counted %+v, want a alone, once", counts)
	}
}

// A color whose cards all weigh nothing cannot fill its balanced slot, which
// is an error to report rather than a draw to attempt.
func TestBoosterPicksRefusesAColorWithNoWeight(t *testing.T) {
	b := boosterBackend(mtgmatcher.Sheet{
		AllowDuplicates: true,
		BalanceColors:   true,
		Cards:           map[string]int{"w": 0, "u": 1},
		TotalWeight:     1,
	}, 2, 1, map[string][]string{"w": {"white"}, "u": {"blue"}})

	_, err := sealed.BoosterPicks(b, "AAA", "draft")
	if err == nil {
		t.Fatal("opened a balanced booster whose white cards weigh nothing")
	}
}
