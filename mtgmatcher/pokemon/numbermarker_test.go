package pokemon

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestNumberMarker pins the letter a storefront hangs off a collector number
// the catalog carries without one. Strikezone numbers the Master Ball
// patterns "074M" beside the plain "074" and the run-marked reprints "001A",
// and the catalog numbers neither that way, so the number as written reaches
// nothing and the listing used to go unpriced. Each case below is a listing
// the storefront publishes, beside the same card numbered as the catalog
// writes it.
func TestNumberMarker(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc    string
		in      mtgmatcher.InputCard
		wantSet string
		wantNum string
		wantVar string
	}{
		{
			desc:    "the marked number reaches the pattern the wording names",
			in:      mtgmatcher.InputCard{Name: "Umbreon", Edition: "SV: Prismatic Evolutions", Variation: "059M 1st Edition Master Ball Pattern"},
			wantSet: "PRE", wantNum: "59", wantVar: "Master Ball Pattern",
		},
		{
			desc:    "the same number written plain still reaches the plain printing",
			in:      mtgmatcher.InputCard{Name: "Umbreon", Edition: "SV: Prismatic Evolutions", Variation: "059 1st Edition"},
			wantSet: "PRE", wantNum: "59", wantVar: "",
		},
		{
			desc:    "and the pattern is reachable without the marker too",
			in:      mtgmatcher.InputCard{Name: "Umbreon", Edition: "SV: Prismatic Evolutions", Variation: "059 Master Ball Pattern"},
			wantSet: "PRE", wantNum: "59", wantVar: "Master Ball Pattern",
		},
		{
			desc:    "a marked number beside the run the printing was reprinted in",
			in:      mtgmatcher.InputCard{Name: "Alakazam", Edition: "Base Set Shadowless", Variation: "001A Unlimited"},
			wantSet: "BSS", wantNum: "1",
		},
		{
			// The letter is the catalog's own here, not the storefront's:
			// the alternate art promos are numbered for the set they
			// reprint and lettered apart from it. Cool Stuff Inc sells
			// them under that set's name, which is the one set the
			// printing is not in.
			desc:    "a lettered promo the edition admits none of",
			in:      mtgmatcher.InputCard{Name: "Garbodor", Edition: "SM Guardians Rising", Variation: "51a/145 Alt Art"},
			wantSet: "PR-1938", wantNum: "51a", wantVar: "Cosmos Holo",
		},
		{
			desc:    "the same, for a number the set code is written into",
			in:      mtgmatcher.InputCard{Name: "Tapu Koko", Edition: "SM Promos", Variation: "SM30a Alt Art"},
			wantSet: "PR-1938", wantNum: "SM30a",
		},
		{
			desc:    "and the unlettered number still reaches the set it names",
			in:      mtgmatcher.InputCard{Name: "Garbodor", Edition: "SM Guardians Rising", Variation: "51/145"},
			wantSet: "SM02", wantNum: "51",
		},
		{
			desc:    "a marked number in a gallery, whose prefix the storefront rebuilds",
			in:      mtgmatcher.InputCard{Name: "Abomasnow", Edition: "SWSH10: Astral Radiance Trainer Gallery", Variation: "TG001C 1st Edition"},
			wantSet: "SWSH10-TG", wantNum: "TG01",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			if err != nil {
				t.Fatalf("Match(%v) = %v", tt.in, err)
			}
			co, err := b.GetUUID(id)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", id, err)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("Match(%v) = %s|%s, want %s|%s", tt.in, co.SetCode, co.Number, tt.wantSet, tt.wantNum)
			}
			gotVar := ""
			if len(co.PromoTypes) > 0 {
				gotVar = co.PromoTypes[0]
			}
			wantVar := mtgmatcher.PromoTypeSlug(tt.wantVar)
			if tt.wantVar == "" {
				wantVar = ""
			}
			if gotVar != wantVar {
				t.Errorf("Match(%v) = %s|%s with promo types %v, want %q", tt.in, co.SetCode, co.Number, co.PromoTypes, wantVar)
			}
		})
	}
}

// TestLetteredPromoAmbiguous pins what happens where the catalog carries a
// lettered number twice. TCGplayer sells several of these promos both at
// their own size and as an oversized jumbo, filing them under the same name,
// number and finish with nothing on the card to tell them apart - the set is
// what does, and a wording naming it picks. One that names neither gets no
// answer: dropping the letter would reach the plain card the promo reprints,
// which is a third card again.
func TestLetteredPromoAmbiguous(t *testing.T) {
	b := loadBackend(t)

	t.Run("the wording names the set and picks it", func(t *testing.T) {
		in := mtgmatcher.InputCard{Name: "Darkrai-GX", Edition: "SM Burning Shadows", Variation: "88a/147 Alt Art"}
		id, err := b.Match(&in)
		if err != nil {
			t.Fatalf("Match(%v) = %v", in, err)
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.SetCode != "PR-1938" || co.Number != "88a" {
			t.Errorf("Match = %s|%s, want PR-1938|88a", co.SetCode, co.Number)
		}
	})

	t.Run("a wording naming neither set gets no answer", func(t *testing.T) {
		in := mtgmatcher.InputCard{Name: "Rayquaza-GX", Edition: "SM Celestial Storm", Variation: "177a/168 Shiny"}
		id, err := b.Match(&in)
		if err == nil {
			co, _ := b.GetUUID(id)
			t.Fatalf("Match = %s|%s, want a refusal", co.SetCode, co.Number)
		}
	})
}
