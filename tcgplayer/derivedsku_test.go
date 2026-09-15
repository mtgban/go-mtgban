package tcgplayer

import "testing"

// TestDerivedSkuMatches pins the filter a two-sided token sheet's combined
// entity uses to find its own price rows in the other face's sku list. The
// sku catalog spells a nonfoil printing "NON FOIL" - confirmed against the
// real file - not "NORMAL", which this function used to compare against and
// which silently zeroed every nonfoil two-sided token sheet's price rather
// than erroring (an empty sku list looks identical to "priced elsewhere").
func TestDerivedSkuMatches(t *testing.T) {
	ownIDs := map[string]bool{"111": true}

	tests := []struct {
		desc         string
		sku          TCGSku
		wantFoil     bool
		wantLanguage string
		want         bool
	}{
		{
			desc:         "a real nonfoil sku for the pairing's own id matches",
			sku:          TCGSku{ProductID: 111, Printing: "NON FOIL", Condition: "NEAR MINT", Language: "ENGLISH"},
			wantFoil:     false,
			wantLanguage: "",
			want:         true,
		},
		{
			desc:         "the old, wrong nonfoil string never matches a real sku",
			sku:          TCGSku{ProductID: 111, Printing: "NORMAL", Condition: "NEAR MINT", Language: "ENGLISH"},
			wantFoil:     false,
			wantLanguage: "",
			want:         false,
		},
		{
			desc:         "a foil sku is refused for a nonfoil entity",
			sku:          TCGSku{ProductID: 111, Printing: "FOIL", Condition: "NEAR MINT", Language: "ENGLISH"},
			wantFoil:     false,
			wantLanguage: "",
			want:         false,
		},
		{
			desc:         "a foil sku matches a foil entity",
			sku:          TCGSku{ProductID: 111, Printing: "FOIL", Condition: "NEAR MINT", Language: "ENGLISH"},
			wantFoil:     true,
			wantLanguage: "",
			want:         true,
		},
		{
			desc:         "an id belonging to a different product is refused",
			sku:          TCGSku{ProductID: 222, Printing: "NON FOIL", Condition: "NEAR MINT", Language: "ENGLISH"},
			wantFoil:     false,
			wantLanguage: "",
			want:         false,
		},
		{
			desc:         "an unopened condition is refused",
			sku:          TCGSku{ProductID: 111, Printing: "NON FOIL", Condition: "UNOPENED", Language: "ENGLISH"},
			wantFoil:     false,
			wantLanguage: "",
			want:         false,
		},
		{
			desc:         "an etched finish is refused",
			sku:          TCGSku{ProductID: 111, Printing: "NON FOIL", Finish: "ETCHED", Condition: "NEAR MINT", Language: "ENGLISH"},
			wantFoil:     false,
			wantLanguage: "",
			want:         false,
		},
		{
			desc:         "a language the entity does not carry, but English, still matches",
			sku:          TCGSku{ProductID: 111, Printing: "NON FOIL", Condition: "NEAR MINT", Language: "ENGLISH"},
			wantFoil:     false,
			wantLanguage: "German",
			want:         true,
		},
		{
			desc:         "a non-English sku matches only the entity's own language",
			sku:          TCGSku{ProductID: 111, Printing: "NON FOIL", Condition: "NEAR MINT", Language: "GERMAN"},
			wantFoil:     false,
			wantLanguage: "German",
			want:         true,
		},
		{
			desc:         "a non-English sku in a language the entity does not carry is refused",
			sku:          TCGSku{ProductID: 111, Printing: "NON FOIL", Condition: "NEAR MINT", Language: "GERMAN"},
			wantFoil:     false,
			wantLanguage: "",
			want:         false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			got := derivedSkuMatches(tt.sku, ownIDs, tt.wantFoil, tt.wantLanguage)
			if got != tt.want {
				t.Errorf("derivedSkuMatches(%+v) = %v, want %v", tt.sku, got, tt.want)
			}
		})
	}
}
