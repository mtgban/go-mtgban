package mtgmatcher_test

import (
	"math"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher/sealed"
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
				total[count.UUID] += count.ExpectedCount * float64(count.Copies*held[0].Count)
				if count.Copies > 1 {
					nested++
				}
			}
			for _, count := range outer {
				perCopy[count.UUID] -= count.ExpectedCount
				total[count.UUID] -= count.ExpectedCount * float64(count.Copies)
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
