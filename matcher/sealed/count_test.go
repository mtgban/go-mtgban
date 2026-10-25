package sealed_test

import (
	"testing"

	"github.com/mtgban/go-mtgban/matcher/sealed"
)

// A card at 0.25 per booster in an entry standing for 36 boosters is nine
// copies in the whole product, not a quarter of one.
func TestCountTotalCountsEveryCopy(t *testing.T) {
	count := sealed.Count{UUID: "a", ExpectedCount: 0.25, Copies: 36}
	got := count.Total()
	if got != 9 {
		t.Errorf("Total() = %v, want 9", got)
	}
}
