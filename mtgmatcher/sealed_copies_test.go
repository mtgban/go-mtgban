package mtgmatcher_test

import (
	"math"
	"testing"
)

// A product holding copies of another lists the held product's chances
// multiplied by the count, and says how many copies it multiplied by, so
// dividing gives back the chance in one copy.
func TestProbabilitiesCountTheCopiesHeld(t *testing.T) {
	realDatastore(t)
	b := testBackend

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
			child, err := b.GetProbabilitiesForSealed(co.SetCode, co.UUID)
			if err != nil || len(child) == 0 {
				continue
			}
			parent, err := b.GetProbabilitiesForSealed(set.Code, product.UUID)
			if err != nil {
				t.Fatal(err)
			}

			want := map[string]float64{}
			for _, prob := range child {
				want[prob.UUID] += prob.Probability / float64(prob.Copies)
			}
			got := map[string]float64{}
			for _, prob := range parent {
				if prob.Copies != held[0].Count {
					t.Fatalf("%s lists %s over %d copies, want %d", product.Name, prob.UUID, prob.Copies, held[0].Count)
				}
				got[prob.UUID] += prob.Probability / float64(prob.Copies)
			}
			for uuid, chance := range want {
				if math.Abs(got[uuid]-chance) > 1e-9 {
					t.Errorf("%s: %s is %v in one copy, want %v", product.Name, uuid, got[uuid], chance)
				}
			}
			return
		}
	}
	t.Fatal("no product holding several copies of one other")
}
