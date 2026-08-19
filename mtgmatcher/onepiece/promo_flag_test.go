package onepiece

import "testing"

// TestPromoFlag pins what "is:promo" answers with. The datastore types no
// set, so the flag is read off the set name; the Premium Booster sets are
// the trap, their codes beginning "PRB" while they are an ordinary product.
func TestPromoFlag(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		name string
		want bool
	}{
		{"One Piece Promotion Cards", true},
		{"Paramount War Pre-Release Cards", true},
		{"Super Pre-Release Starter Deck 1: Straw Hat Crew", true},
		{"Premium Booster -The Best-", false},
		{"Extra Booster: Memorial Collection", false},
		{"Starter Deck 31: RED Monkey.D.Luffy", false},
	} {
		if got := setIsPromotional(tt.name); got != tt.want {
			t.Errorf("setIsPromotional(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}

	// The printing that started this: a promo Nami no bare name could reach.
	uuids, err := b.SearchEquals("Nami (Premium Card Collection -Best Selection Vol. 6-)")
	if err != nil {
		t.Fatal(err)
	}
	for _, uuid := range uuids {
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		if !co.IsPromo {
			t.Errorf("%s %s is in %q and is not flagged promotional", co.SetCode, co.Number, co.Edition)
		}
	}

	var promo int
	for _, uuid := range b.AllUUIDs {
		co, err := b.GetUUID(uuid)
		if err == nil && co.IsPromo {
			promo++
		}
	}
	if promo == 0 {
		t.Error("no printing is flagged promotional, so is:promo answers with nothing")
	}
}
