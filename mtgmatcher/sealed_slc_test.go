package mtgmatcher_test

import (
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/sealed"
)

// slcProduct finds a Secret Lair Countdown Kit, the one product whose deck
// carries a chance of a foil rather than a fixed finish.
func slcProduct(t *testing.T) (*mtgmatcher.Backend, string) {
	t.Helper()
	realDatastore(t)
	b := testBackend
	set, err := b.GetSet("SLC")
	if err != nil {
		t.Fatal("no SLC in this datastore:", err)
	}
	for _, product := range set.SealedProduct {
		if sealed.HasDecklist(b, "SLC", product.UUID) {
			return b, product.UUID
		}
	}
	t.Fatal("no SLC product with a decklist")
	return nil, ""
}

func finishes(t *testing.T, b *mtgmatcher.Backend, uuids []string) (foil, nonfoil int) {
	t.Helper()
	for _, uuid := range uuids {
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		if co.Foil {
			foil++
		} else {
			nonfoil++
		}
	}
	return foil, nonfoil
}

// A decklist is what a product always holds, so asking twice has to answer
// twice the same. The Countdown Kit upgrades some of its cards to foil at
// random, which is a fact about one copy rather than about the product.
func TestProductDecklistIsTheSameEveryTime(t *testing.T) {
	b, uuid := slcProduct(t)

	first, err := sealed.ProductDecklist(b, "SLC", uuid)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) == 0 {
		t.Fatal("the product opens into nothing")
	}

	for i := 0; i < 5; i++ {
		again, err := sealed.ProductDecklist(b, "SLC", uuid)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(again, first) {
			foil, nonfoil := finishes(t, b, again)
			t.Fatalf("call %d answered differently: %d cards (%d foil, %d nonfoil)",
				i+2, len(again), foil, nonfoil)
		}
	}

	// A kit does promise one foil - the Lotus Field, the Alhammarret's Archive
	// - and the data says so. What it does not promise is the third of the
	// deck a roll upgrades, so a quarter separates the two cleanly
	// without pinning either number.
	foil, _ := finishes(t, b, first)
	if foil*4 >= len(first) {
		t.Errorf("%d of %d cards are foil, which reads as a roll rather than as the data",
			foil, len(first))
	}
}

// The chance itself is not lost: it belongs to opening a copy, which is what
// the simulation does.
func TestProductPicksStillRollsTheFoils(t *testing.T) {
	b, uuid := slcProduct(t)

	var sawFoil bool
	for i := 0; i < 10 && !sawFoil; i++ {
		picks, err := sealed.ProductPicks(b, "SLC", uuid)
		if err != nil {
			t.Fatal(err)
		}
		if foil, _ := finishes(t, b, picks); foil > 0 {
			sawFoil = true
		}
	}
	if !sawFoil {
		t.Error("ten simulated copies produced no foil at all")
	}
}

// And the expected value reads the chance from the counts, where each card is
// listed in both finishes with the odds of each.
func TestCountsCarryBothFinishes(t *testing.T) {
	b, uuid := slcProduct(t)

	probs, err := sealed.ProductCounts(b, "SLC", uuid)
	if err != nil {
		t.Fatal(err)
	}

	var foilOdds, nonfoilOdds int
	for _, prob := range probs {
		co, err := b.GetUUID(prob.UUID)
		if err != nil {
			continue
		}
		if co.Foil && prob.ExpectedCount == 0.3 {
			foilOdds++
		}
		if !co.Foil && prob.ExpectedCount == 0.7 {
			nonfoilOdds++
		}
	}
	if foilOdds == 0 || nonfoilOdds == 0 {
		t.Errorf("the odds are gone: %d cards at 0.3 foil, %d at 0.7 nonfoil", foilOdds, nonfoilOdds)
	}
}

// The roll spares only the bonus card, which the product lists apart from the
// deck, so enough copies show every printing the counts give a chance.
func TestProductPicksDrawsWhatTheCountsList(t *testing.T) {
	b, uuid := slcProduct(t)

	probs, err := sealed.ProductCounts(b, "SLC", uuid)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		picks, err := sealed.ProductPicks(b, "SLC", uuid)
		if err != nil {
			t.Fatal(err)
		}
		for _, pick := range picks {
			seen[pick] = true
		}
	}

	for _, prob := range probs {
		if prob.ExpectedCount > 0 && !seen[prob.UUID] {
			co, _ := b.GetUUID(prob.UUID)
			t.Errorf("%s #%s (foil %v) has a %.1f chance but never came up",
				co.Name, co.Number, co.Foil, prob.ExpectedCount)
		}
	}
}
