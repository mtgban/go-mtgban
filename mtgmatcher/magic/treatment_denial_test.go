package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestTreatmentDenial pins the three treatments a storefront that stocks them
// always names, so silence about one is evidence the listing is not it. Each
// pair below is the same card sold twice by the same shop: once with the
// treatment written down and once without, where reading the second listing
// forgivingly costs the first its printing.
func TestTreatmentDenial(t *testing.T) {
	for _, tt := range []struct {
		desc        string
		in          mtgmatcher.InputCard
		wantSet     string
		wantNum     string
		wantPromo   string
		wantNoPromo string
	}{
		{
			desc:    "a Commander listing that says only Extended Art is not the Chocobo Track printing",
			in:      mtgmatcher.InputCard{Name: "Tataru Taru", Edition: "Commander: FINAL FANTASY", Variation: "Extended Art", Foil: true},
			wantSet: "FIC", wantNum: "138", wantPromo: "boosterfun",
		},
		{
			desc:    "and the listing that does say it still lands there",
			in:      mtgmatcher.InputCard{Name: "Tataru Taru", Edition: "Commander: FINAL FANTASY", Variation: "Borderless Chocobo Track Foil", Foil: true},
			wantSet: "FIC", wantNum: "466", wantPromo: "chocobotrackfoil",
		},
		{
			desc:    "a listing that names the Silver Scroll treatment is the printing wearing it",
			in:      mtgmatcher.InputCard{Name: "Ad Nauseam", Edition: "Secrets of Strixhaven Mystical Archive", Variation: "JP Alternate Art Silver Scroll Foil", Foil: true},
			wantSet: "SOA", wantNum: "155", wantPromo: "silverscroll",
		},
		{
			desc:    "and the one that names no treatment is the plain printing beside it",
			in:      mtgmatcher.InputCard{Name: "Ad Nauseam", Edition: "Secrets of Strixhaven Mystical Archive", Foil: true},
			wantSet: "SOA", wantNum: "25", wantNoPromo: "silverscroll",
		},
		{
			desc:    "a foil that names no treatment at all is the untreated printing",
			in:      mtgmatcher.InputCard{Name: "Valor's Flagship", Edition: "Aetherdrift", Foil: true},
			wantSet: "DFT", wantNum: "35", wantNoPromo: "boosterfun",
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
			// The border and the two frames are promo types of their own, so
			// an untreated printing is one without the type that tells it
			// from its sibling, not one with an empty list.
			if tt.wantNoPromo != "" && co.HasPromoType(tt.wantNoPromo) {
				t.Errorf("Match(%v) = %s|%s with promo types %v, want none of them to be %q", tt.in, co.SetCode, co.Number, co.PromoTypes, tt.wantNoPromo)
			}
		})
	}
}
