package magic

import "testing"

// A 1997-frame printing is a retroframe promo when the card itself came out
// after PromosForEverybodyYay, even in a set dated before it, and not when it
// is the frame its own era printed.
func TestRetroFrameDatesTheCard(t *testing.T) {
	realDatastore(t)
	for _, tt := range []struct {
		name, set, number string
		want              bool
	}{
		{"Ruin Crab", "PMEI", "2023-4", true},
		{"Gingerbrute", "PMEI", "2023-3", true},
		{"Pyromancer's Gauntlet", "PMEI", "2023-6", true},
		{"Command Tower", "PMEI", "2026-1", true},
		{"Wasteland", "TMP", "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cards := testBackend.MatchInSet(tt.name, tt.set)
			var checked int
			for _, card := range cards {
				if tt.number != "" && card.Number != tt.number {
					continue
				}
				checked++
				if got := card.HasPromoType(PromoTypeRetroFrame); got != tt.want {
					t.Errorf("%s %s #%s: retroframe = %v, want %v", tt.name, tt.set, card.Number, got, tt.want)
				}
			}
			if checked == 0 {
				t.Fatalf("no %s printing in %s", tt.name, tt.set)
			}
		})
	}
}
