package starcitygames

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The catalog spells two-part languages with a dash mtgjson does not use, and
// English - which an absent language also means - is the empty tag, the one
// that claims nothing about which printing is being sold.
func TestCatalogLanguageTag(t *testing.T) {
	tests := []struct {
		catalog, want string
	}{
		{"English", ""},
		{"", ""},
		{"Japanese", "Japanese"},
		{"Italian", "Italian"},
		{"German", "German"},
		{"Korean", "Korean"},
		{"Chinese - Traditional", "Chinese Traditional"},
		{"Chinese - Simplified", "Chinese Simplified"},
		{"Portuguese", "Portuguese"},
	}
	for _, test := range tests {
		if got := catalogLanguageTag(test.catalog); got != test.want {
			t.Errorf("catalogLanguageTag(%q) = %q, want %q", test.catalog, got, test.want)
		}
	}
}

// mtgjson keeps one printing for the inherently foreign sets while SCG
// sells them in several languages, so only the matching one resolves.
func TestResolveForeignLanguages(t *testing.T) {
	withMagic(t)

	withMagic(t)
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
			if claimed := catalogLanguageTag(test.product.Language); claimed != "" &&
				!strings.Contains(co.Language, claimed) {
				t.Errorf("resolved to a %s printing for a %s product", co.Language, test.product.Language)
			}
		})
	}
}

// A printing that was only ever made in one language other than English is
// still what the shop is selling: the sku carries "-EN" because that is what
// the grammar emits, and the collector number beside it has already said which
// printing is meant. Dropping these lost the Dwarvish cards in The Hobbit, the
// Phyrexian Secret Lairs and the promos printed as a gimmick in a single tongue.
func TestResolveEnglishTagOnForeignPrinting(t *testing.T) {
	withMagic(t)

	for _, test := range []struct {
		product          CatalogProduct
		wantSet, wantNum string
		wantCardLanguage string
	}{
		{
			CatalogProduct{
				Name: "Mox Amber", Set: "The Hobbit Eternal", Language: "English",
				CollectorNumber: "096", SKU: "SGL-MTG-HOC-096-ENN",
				Finish: "Non-foil", FinishGroup: "Non-foil",
			}, "HOC", "96", "Dwarvish",
		},
		{
			CatalogProduct{
				Name: "Jin-Gitaxias, Progress Tyrant", Set: "Kamigawa: Neon Dynasty",
				Language: "English", CollectorNumber: "307", SKU: "SGL-MTG-NEO2-307-ENN",
				Finish: "Non-foil", FinishGroup: "Non-foil",
			}, "NEO", "307", "Phyrexian",
		},
		{
			CatalogProduct{
				Name: "Sheoldred, Whispering One", Set: "Secret Lair Drop",
				Language: "English", CollectorNumber: "211",
				SKU:    "SGL-MTG-PRM-SECRET_SLD_211-ENN",
				Finish: "Non-foil", FinishGroup: "Non-foil",
			}, "SLD", "211", "Phyrexian",
		},
	} {
		t.Run(test.product.SKU, func(t *testing.T) {
			id, err := resolveProduct(GameMagic, test.product)
			if err != nil {
				t.Fatalf("resolveProduct: %v", err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != test.wantSet || co.Number != test.wantNum ||
				co.Language != test.wantCardLanguage {
				t.Errorf("got %s #%s (%s), want %s #%s (%s)", co.SetCode, co.Number,
					co.Language, test.wantSet, test.wantNum, test.wantCardLanguage)
			}
		})
	}
}
