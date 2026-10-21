package sealed_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/sealed"
)

// sealedBackend holds one set, AAA, decoded from data the way a datastore
// publishes it, and the nonfoil cards a, b and c.
func sealedBackend(t *testing.T, data string) *mtgmatcher.Backend {
	t.Helper()
	var set mtgmatcher.Set
	err := json.Unmarshal([]byte(data), &set)
	if err != nil {
		t.Fatal(err)
	}
	set.Code = "AAA"
	b := &mtgmatcher.Backend{
		UUIDs: map[string]*mtgmatcher.CardObject{},
		Sets:  map[string]*mtgmatcher.Set{"AAA": &set},
	}
	for _, id := range []string{"a", "b", "c"} {
		b.UUIDs[id] = &mtgmatcher.CardObject{Card: mtgmatcher.Card{UUID: id, SetCode: "AAA"}}
	}
	return b
}

// totals sums each uuid's Total across counts.
func totals(counts []sealed.Count) map[string]float64 {
	out := map[string]float64{}
	for _, count := range counts {
		out[count.UUID] += count.Total()
	}
	return out
}

// A product listed in its own contents, directly or through another, is
// corrupt data; every walk answers for it instead of recursing until the
// stack overflows, which no recover can catch.
func TestSealedWalksStopAtACycle(t *testing.T) {
	for name, data := range map[string]string{
		"self": `{"sealedProduct":[
			{"uuid":"x","name":"X","contents":{"card":[{"uuid":"a"}],"sealed":[{"set":"aaa","uuid":"x","count":1}],"deck":[{"set":"aaa","name":"D"}]}}],
			"decks":[{"name":"D","mainBoard":[{"uuid":"b","count":1}]}]}`,
		"two": `{"sealedProduct":[
			{"uuid":"x","name":"X","contents":{"card":[{"uuid":"a"}],"sealed":[{"set":"aaa","uuid":"y","count":2}]}},
			{"uuid":"y","name":"Y","contents":{"sealed":[{"set":"aaa","uuid":"x","count":2}],"deck":[{"set":"aaa","name":"D"}]}}],
			"decks":[{"name":"D","mainBoard":[{"uuid":"b","count":1}]}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			b := sealedBackend(t, data)
			if sealed.IsRandom(b, "AAA", "x") {
				t.Error("IsRandom: a product holding only cards and a deck is fixed")
			}
			if !sealed.HasDecklist(b, "AAA", "x") {
				t.Error("HasDecklist: the deck past the cycle was missed")
			}
			list, err := sealed.ProductDecklist(b, "AAA", "x")
			if err != nil {
				t.Errorf("ProductDecklist: %v", err)
			}
			if !slices.Contains(list, "b") {
				t.Errorf("ProductDecklist = %v, missing the deck's b", list)
			}
			_, err = sealed.ProductPicks(b, "AAA", "x")
			if err == nil {
				t.Error("ProductPicks opened a product that holds itself")
			}
			_, err = sealed.ProductCounts(b, "AAA", "x")
			if err == nil {
				t.Error("ProductCounts counted a product that holds itself")
			}
		})
	}
}

// A product held twice is not a cycle: a bundle of two identical boosters,
// or one booster under two kits, counts every copy.
func TestSealedWalksCountRepeats(t *testing.T) {
	b := sealedBackend(t, `{"sealedProduct":[
		{"uuid":"box","name":"Box","contents":{"sealed":[
			{"set":"aaa","uuid":"kit1","count":1},
			{"set":"aaa","uuid":"kit2","count":1},
			{"set":"aaa","uuid":"p","count":2}]}},
		{"uuid":"kit1","name":"Kit One","contents":{"sealed":[{"set":"aaa","uuid":"p","count":1}],"deck":[{"set":"aaa","name":"D"}]}},
		{"uuid":"kit2","name":"Kit Two","contents":{"sealed":[{"set":"aaa","uuid":"p","count":3}],"deck":[{"set":"aaa","name":"D"}]}},
		{"uuid":"p","name":"P","contents":{"card":[{"uuid":"a"}],"deck":[{"set":"aaa","name":"E"}]}}],
		"decks":[
			{"name":"D","mainBoard":[{"uuid":"b","count":2},{"uuid":"c","count":1}]},
			{"name":"E","mainBoard":[{"uuid":"c","count":1}]}]}`)

	// p is held 1 + 3 + 2 times and holds an a and deck E's c; D is held
	// twice
	want := map[string]float64{"a": 6, "b": 4, "c": 8}

	counts, err := sealed.ProductCounts(b, "AAA", "box")
	if err != nil {
		t.Fatal(err)
	}
	got := totals(counts)
	if len(got) != len(want) || got["a"] != want["a"] || got["b"] != want["b"] || got["c"] != want["c"] {
		t.Errorf("ProductCounts totals = %v, want %v", got, want)
	}

	for name, open := range map[string]func(*mtgmatcher.Backend, string, string) ([]string, error){
		"ProductPicks":    sealed.ProductPicks,
		"ProductDecklist": sealed.ProductDecklist,
	} {
		picks, err := open(b, "AAA", "box")
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for uuid, n := range want {
			c := 0
			for _, pick := range picks {
				if pick == uuid {
					c++
				}
			}
			if float64(c) != n {
				t.Errorf("%s holds %d of %s, want %v", name, c, uuid, n)
			}
		}
	}

	if !sealed.HasDecklist(b, "AAA", "box") || sealed.IsRandom(b, "AAA", "box") {
		t.Error("box holds fixed decks only")
	}
}

// A variable config publishing an empty list of odds is one of its equals,
// as one publishing none is.
func TestSealedWalksReadEmptyOdds(t *testing.T) {
	b := sealedBackend(t, `{"sealedProduct":[{"uuid":"v","name":"V","contents":{"variable":[{"configs":[
		{"card":[{"uuid":"a"}],"variable_config":[]},
		{"card":[{"uuid":"b"}]}]}]}}]}`)

	counts, err := sealed.ProductCounts(b, "AAA", "v")
	if err != nil {
		t.Fatal(err)
	}
	got := totals(counts)
	if got["a"] != 0.5 || got["b"] != 0.5 {
		t.Errorf("ProductCounts totals = %v, want a and b at 0.5", got)
	}
	picks, err := sealed.ProductPicks(b, "AAA", "v")
	if err != nil {
		t.Fatal(err)
	}
	if len(picks) != 1 {
		t.Errorf("ProductPicks = %v, want one card", picks)
	}
}

// A count no real product comes near is refused at once rather than opened
// copy by copy, which would spin or allocate until the process dies.
func TestSealedWalksRefuseAHugeCount(t *testing.T) {
	for name, data := range map[string]string{
		"product": `{"sealedProduct":[
			{"uuid":"x","name":"X","contents":{"sealed":[{"set":"aaa","uuid":"p","count":1000000000000}],"deck":[{"set":"aaa","name":"D"}]}},
			{"uuid":"p","name":"P","contents":{"card":[{"uuid":"a"}]}}],
			"decks":[{"name":"D","mainBoard":[{"uuid":"b","count":1}]}]}`,
		"nested": `{"sealedProduct":[
			{"uuid":"x","name":"X","contents":{"sealed":[{"set":"aaa","uuid":"y","count":5000}],"deck":[{"set":"aaa","name":"D"}]}},
			{"uuid":"y","name":"Y","contents":{"sealed":[{"set":"aaa","uuid":"p","count":5000}],"deck":[{"set":"aaa","name":"D"}]}},
			{"uuid":"p","name":"P","contents":{"card":[{"uuid":"a"}]}}],
			"decks":[{"name":"D","mainBoard":[{"uuid":"b","count":1}]}]}`,
		"deck": `{"sealedProduct":[
			{"uuid":"x","name":"X","contents":{"deck":[{"set":"aaa","name":"D"}]}}],
			"decks":[{"name":"D","mainBoard":[{"uuid":"b","count":1000000000000}]}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			b := sealedBackend(t, data)
			_, err := sealed.ProductPicks(b, "AAA", "x")
			if err == nil {
				t.Error("ProductPicks opened it")
			}
			_, err = sealed.ProductDecklist(b, "AAA", "x")
			if err == nil {
				t.Error("ProductDecklist listed it")
			}
			_, err = sealed.ProductCounts(b, "AAA", "x")
			if err == nil {
				t.Error("ProductCounts counted it")
			}
		})
	}
}

// A booster slot or fixed-sheet count no real booster comes near is refused
// at once rather than drawn card by card.
func TestBoosterPicksRefusesAHugeCount(t *testing.T) {
	for name, b := range map[string]*mtgmatcher.Backend{
		"slot": boosterBackend(mtgmatcher.Sheet{
			AllowDuplicates: true,
			Cards:           map[string]int{"a": 1},
		}, 1000000000000, 1, map[string][]string{"a": nil}),
		"fixed": boosterBackend(mtgmatcher.Sheet{
			Fixed: true,
			Cards: map[string]int{"a": 1000000000000},
		}, 1, 1, map[string][]string{"a": nil}),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := sealed.BoosterPicks(b, "AAA", "draft")
			if err == nil {
				t.Error("BoosterPicks opened it")
			}
		})
	}
}

// Packs held by a product draw from the product's budget: 300,000 packs of
// two cards each are past it with their cards, though no single pack is.
func TestProductPicksChargesItsPacks(t *testing.T) {
	b := boosterBackend(mtgmatcher.Sheet{
		AllowDuplicates: true,
		Cards:           map[string]int{"a": 1},
	}, 2, 1, map[string][]string{"a": nil})
	set := b.Sets["AAA"]
	set.SealedProduct = []mtgmatcher.SealedProduct{
		{UUID: "box", Name: "Box", Contents: map[string][]mtgmatcher.SealedContent{
			"sealed": {{Set: "aaa", UUID: "p", Count: 300000}},
		}},
		{UUID: "p", Name: "P", Contents: map[string][]mtgmatcher.SealedContent{
			"pack": {{Set: "aaa", Code: "draft"}},
		}},
	}
	_, err := sealed.ProductPicks(b, "AAA", "box")
	if err == nil {
		t.Error("ProductPicks opened 300,000 packs")
	}
}

// A variable entry that publishes no total weight is read the way its draw
// is: each config's chance against the sum of them all, a chance below one
// never coming up. Nothing to draw by is an error, never a NaN or an Inf.
func TestProductCountsReadUnweightedConfigs(t *testing.T) {
	b := sealedBackend(t, `{"sealedProduct":[
		{"uuid":"v","name":"V","contents":{"variable":[{"configs":[
			{"card":[{"uuid":"a"}],"variable_config":[{"chance":1,"weight":0}]},
			{"card":[{"uuid":"b"}],"variable_config":[{"chance":3,"weight":0}]},
			{"card":[{"uuid":"c"}],"variable_config":[{"chance":0,"weight":0}]}]}]}},
		{"uuid":"n","name":"N","contents":{"variable":[{"configs":[
			{"card":[{"uuid":"a"}],"variable_config":[{"chance":1,"weight":2}]},
			{"card":[{"uuid":"b"}],"variable_config":[{"chance":-1,"weight":2}]}]}]}},
		{"uuid":"z","name":"Z","contents":{"variable":[{"configs":[
			{"card":[{"uuid":"a"}],"variable_config":[{"chance":0,"weight":0}]},
			{"card":[{"uuid":"b"}],"variable_config":[{"chance":0,"weight":0}]}]}]}}]}`)

	for uuid, want := range map[string]map[string]float64{
		"v": {"a": 0.25, "b": 0.75, "c": 0},
		"n": {"a": 0.5, "b": 0},
	} {
		counts, err := sealed.ProductCounts(b, "AAA", uuid)
		if err != nil {
			t.Fatalf("%s: %v", uuid, err)
		}
		got := totals(counts)
		for id, n := range want {
			if got[id] != n {
				t.Errorf("%s: ProductCounts totals = %v, want %v", uuid, got, want)
				break
			}
		}
	}

	for range 50 {
		picks, err := sealed.ProductPicks(b, "AAA", "v")
		if err != nil {
			t.Fatal(err)
		}
		if slices.Contains(picks, "c") {
			t.Fatal("drew the config whose chance is zero")
		}
	}

	_, err := sealed.ProductCounts(b, "AAA", "z")
	if err == nil {
		t.Error("ProductCounts counted a variable entry with nothing to draw by")
	}
}
