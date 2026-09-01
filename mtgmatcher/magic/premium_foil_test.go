package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPremiumFoilClaim pins the treatments a set prints once, beside the
// plain card and nowhere else, whose price is orders of magnitude above it:
// a buylist that named one was quoted the plain printing's identity, so the
// premium price landed on the cheap card and read as arbitrage. The plain
// listings are here too, since narrowing to the treatment is only right if
// silence still lands on the printing that wears none.
func TestPremiumFoilClaim(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		in        mtgmatcher.InputCard
		wantSet   string
		wantNum   string
		wantPromo string
	}{
		{
			desc:    "a listing that names Cosmic Foil is the printing wearing it",
			in:      mtgmatcher.InputCard{Name: "The Soul Stone", Edition: "Marvel's Spider-Man", Variation: "Cosmic Foil", Foil: true},
			wantSet: "SPM", wantNum: "242", wantPromo: "cosmicfoil",
		},
		{
			desc:    "and the plain foil beside it keeps the base printing",
			in:      mtgmatcher.InputCard{Name: "The Soul Stone", Edition: "Marvel's Spider-Man", Foil: true},
			wantSet: "SPM", wantNum: "66",
		},
		{
			desc:    "the borderless listing is the borderless printing, not the cosmic one",
			in:      mtgmatcher.InputCard{Name: "The Soul Stone", Edition: "Marvel's Spider-Man", Variation: "Borderless", Foil: true},
			wantSet: "SPM", wantNum: "243",
		},
		{
			desc:    "Cosmic Foil reads the same way in the set that shares the treatment",
			in:      mtgmatcher.InputCard{Name: "The Mind Stone", Edition: "Marvel Super Heroes", Variation: "Cosmic Foil", Foil: true},
			wantSet: "MSH", wantNum: "385", wantPromo: "cosmicfoil",
		},
		{
			desc:    "a listing that names Singularity Foil outranks the treatments it stays silent about",
			in:      mtgmatcher.InputCard{Name: "Sothera, the Supervoid", Edition: "Edge of Eternities", Variation: "Singularity Foil", Foil: true},
			wantSet: "EOE", wantNum: "382", wantPromo: "singularityfoil",
		},
		{
			desc:    "and the plain foil beside it is still the base printing",
			in:      mtgmatcher.InputCard{Name: "Sothera, the Supervoid", Edition: "Edge of Eternities", Foil: true},
			wantSet: "EOE", wantNum: "115",
		},
		{
			desc:    "the showcase listings of the same card keep their own printings",
			in:      mtgmatcher.InputCard{Name: "Sothera, the Supervoid", Edition: "Edge of Eternities", Variation: "Showcase Fracture Foil", Foil: true},
			wantSet: "EOE", wantNum: "386", wantPromo: "fracturefoil",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := testBackend.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			co, err := testBackend.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("Match(%v) = %s|%s, want %s|%s", tt.in, co.SetCode, co.Number, tt.wantSet, tt.wantNum)
			}
			if tt.wantPromo != "" && !co.HasPromoType(tt.wantPromo) {
				t.Errorf("Match(%v) = %s|%s with promo types %v, want one of them to be %q", tt.in, co.SetCode, co.Number, co.PromoTypes, tt.wantPromo)
			}
		})
	}
}
