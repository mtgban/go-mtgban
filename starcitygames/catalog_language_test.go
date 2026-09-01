package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The catalog spells two-part languages with a dash mtgjson does not
// use, and an absent language means English on both sides.
func TestLanguageMatches(t *testing.T) {
	tests := []struct {
		catalog, card string
		want          bool
	}{
		{"English", "English", true},
		{"", "English", true},
		{"English", "", true},
		{"Japanese", "Japanese", true},
		{"Chinese - Traditional", "Chinese Traditional", true},
		{"Chinese - Simplified", "Chinese Simplified", true},
		{"German", "Italian", false},
		{"French", "Italian", false},
		{"Korean", "Japanese", false},
		{"Chinese - Traditional", "Japanese", false},
	}
	for _, test := range tests {
		if got := languageMatches(test.catalog, test.card); got != test.want {
			t.Errorf("languageMatches(%q, %q) = %v", test.catalog, test.card, got)
		}
	}
}

// mtgjson keeps one printing for the inherently foreign sets while SCG
// sells them in several languages, so only the matching one resolves.
func TestResolveForeignLanguages(t *testing.T) {
	withMagic(t)

	if len(mtgmatcher.GetUUIDs()) == 0 {
		t.Skip("datastore not loaded")
	}
	tests := []struct {
		desc    string
		product CatalogProduct
		wantOK  bool
	}{
		{
			desc: "4BB is Japanese, so the Japanese product resolves",
			product: CatalogProduct{
				Name: "Yotian Soldier", Set: "4th Edition - Black Border",
				Language: "Japanese", CollectorNumber: "360",
				SKU: "SGL-MTG-4BB-360-JAN", FinishGroup: "Non-foil",
			},
			wantOK: true,
		},
		{
			desc: "the Chinese one has no printing to land on",
			product: CatalogProduct{
				Name: "Yotian Soldier", Set: "4th Edition - Black Border",
				Language: "Chinese - Traditional", CollectorNumber: "360",
				SKU: "SGL-MTG-4BB-360-ZTN", FinishGroup: "Non-foil",
			},
			wantOK: false,
		},
		{
			desc: "FBB is Italian, so German is dropped",
			product: CatalogProduct{
				Name: "Weakness", Set: "3rd Edition - Black Border",
				Language: "German", CollectorNumber: "136",
				SKU: "SGL-MTG-3BB-136-DEN", FinishGroup: "Non-foil",
			},
			wantOK: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			id, err := resolveProduct(GameMagic, test.product)
			if test.wantOK && err != nil {
				t.Fatalf("resolveProduct: %v", err)
			}
			if !test.wantOK {
				if err == nil {
					co, _ := mtgmatcher.GetUUID(id)
					t.Fatalf("resolved to %s (%s), expected it to be skipped", id, co)
				}
				return
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if !languageMatches(test.product.Language, co.Language) {
				t.Errorf("resolved to a %s printing for a %s product", co.Language, test.product.Language)
			}
		})
	}
}
