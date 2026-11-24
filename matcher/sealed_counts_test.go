package matcher_test

import (
	"math"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/matcher/sealed"
)

// A product holding copies of another lists the held product's entries as
// they are, each standing for as many more copies as it holds: the chance
// in one copy is unchanged, and the total multiplies.
func TestProductCountsCountTheCopiesHeld(t *testing.T) {
	realDatastore(t)
	b := testBackend

	var checked, nested int
	for _, code := range b.GetAllSets() {
		set, err := b.GetSet(code)
		if err != nil {
			continue
		}
		for _, product := range set.SealedProduct {
			held := product.Contents["sealed"]
			if len(product.Contents) != 1 || len(held) != 1 || held[0].Count < 2 {
				continue
			}
			co, err := b.GetUUID(held[0].UUID)
			if err != nil {
				continue
			}
			inner, err := sealed.ProductCounts(b, co.SetCode, co.UUID)
			if err != nil || len(inner) == 0 {
				continue
			}
			outer, err := sealed.ProductCounts(b, set.Code, product.UUID)
			if err != nil {
				t.Fatal(err)
			}

			perCopy, total := map[string]float64{}, map[string]float64{}
			for _, count := range inner {
				perCopy[count.UUID] += count.ExpectedCount
				total[count.UUID] += count.Total() * float64(held[0].Count)
				if count.Copies > 1 {
					nested++
				}
			}
			for _, count := range outer {
				perCopy[count.UUID] -= count.ExpectedCount
				total[count.UUID] -= count.Total()
			}
			for uuid := range perCopy {
				if math.Abs(perCopy[uuid]) > 1e-9 || math.Abs(total[uuid]) > 1e-9 {
					t.Errorf("%s: %s is off by %v per copy and %v in total", product.Name, uuid, perCopy[uuid], total[uuid])
				}
			}
			checked++
		}
	}
	if checked == 0 || nested == 0 {
		t.Fatalf("checked %d products, %d entries already held more than once; want both", checked, nested)
	}
}

// A card copied into a set of its own names only products that draw it, and
// not those its original comes from.
func TestCopiedCardsNameTheirOwnProducts(t *testing.T) {
	realDatastore(t)
	b := testBackend

	draws := map[string]map[string]bool{}
	for _, code := range []string{
		"FBBFR", "FBBDE", "FWB", "FWBFR", "FWBDE", "4BBES", "4BBPT", "4BBKO", "4BBZHT",
		"RENDE", "LEGITA", "DRKITA", "4EDALT",
	} {
		for _, card := range b.Sets[code].Cards {
			for _, source := range card.SourceProducts["nonfoil"] {
				if draws[source] == nil {
					draws[source] = map[string]bool{}
					product, err := b.GetUUID(source)
					if err != nil {
						t.Fatal(err)
					}
					counts, err := sealed.ProductCounts(b, product.SetCode, source)
					if err != nil {
						t.Fatal(err)
					}
					for _, count := range counts {
						draws[source][count.UUID] = true
					}
				}
				if !draws[source][card.UUID] {
					t.Errorf("%s %s names %s, which does not draw it", code, card.Name, source)
				}
			}
		}
	}
}

// A set packed like another opens like it, from its own cards alone, and each
// card names the products that draw it, and no others, as where it comes from.
// The other sets copied from one of the two name no product.
func TestCopiedSetsPackLikeTheOriginal(t *testing.T) {
	realDatastore(t)
	b := testBackend

	total := func(product string) (float64, map[string]bool) {
		t.Helper()
		uuids, err := b.SearchSealedEquals(product)
		if err != nil {
			t.Fatal(product, err)
		}
		co, err := b.GetUUID(uuids[0])
		if err != nil {
			t.Fatal(err)
		}
		counts, err := sealed.ProductCounts(b, co.SetCode, co.UUID)
		if err != nil {
			t.Fatal(product, err)
		}
		var sum float64
		drawn := map[string]bool{}
		for _, count := range counts {
			sum += count.Total()
			drawn[count.UUID] = true
		}
		return sum, drawn
	}

	boosters := []string{"Booster Pack", "Booster Box", "Booster Box Case"}
	withStarters := append(slices.Clone(boosters), "Starter Deck", "Starter Deck Display")
	for _, tt := range []struct {
		code, name, like string
		products         []string
	}{
		{"FBB", "Foreign Black Border", "Revised Edition", withStarters},
		{"FWB", "Foreign White Border", "Revised Edition", withStarters},
		{"4BB", "Fourth Edition Black Bordered", "Fourth Edition", withStarters},
		{"LEGITA", "Legends Italian", "Legends", boosters},
		{"DRKITA", "The Dark Italian", "The Dark", boosters},
		{"RENDE", "Renaissance German", "Renaissance French", boosters},
	} {
		for _, product := range tt.products {
			want, like := total(tt.like + " " + product)
			got, drawn := total(tt.name + " " + product)
			if math.Abs(got-want) > 1e-6 || len(drawn) != len(like) {
				t.Errorf("%s %s opens to %v of %d cards, want %v of %d", tt.name, product, got, len(drawn), want, len(like))
			}
			for uuid := range drawn {
				co, err := b.GetUUID(uuid)
				if err != nil || co.SetCode != tt.code {
					t.Errorf("%s %s draws %v", tt.name, product, co)
				}
			}
		}

		for _, code := range []string{tt.code, b.Sets[tt.code].ParentCode} {
			for _, card := range b.Sets[code].Cards {
				for _, source := range card.SourceProducts["nonfoil"] {
					product, err := b.GetUUID(source)
					if err != nil {
						t.Fatal(err)
					}
					_, drawn := total(product.Name)
					if !drawn[card.UUID] {
						t.Errorf("%s %s does not come from %s", code, card.Name, product.Name)
					}
				}
			}
		}
		_, drawn := total(tt.name + " Booster Pack")
		for uuid := range drawn {
			co, err := b.GetUUID(uuid)
			if err != nil || len(co.SourceProducts["nonfoil"]) == 0 {
				t.Errorf("%v names no product", co)
			}
		}
	}

	for _, code := range []string{"FBBFR", "FBBDE", "FWBFR", "FWBDE", "4BBES", "4BBPT", "4BBKO", "4BBZHT", "4EDALT"} {
		for _, card := range b.Sets[code].Cards {
			if len(card.SourceProducts) > 0 {
				t.Errorf("%s %s comes from %v", code, card.Name, card.SourceProducts)
			}
		}
	}
}
