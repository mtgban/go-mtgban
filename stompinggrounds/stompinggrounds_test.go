package stompinggrounds

import (
	"encoding/json"
	"os"
	"slices"
	"sync"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/internal/gametest"
	"github.com/mtgban/go-mtgban/matcher"
	_ "github.com/mtgban/go-mtgban/matcher/magic"
	_ "github.com/mtgban/go-mtgban/matcher/pokemon"
)

func TestProductID(t *testing.T) {
	tests := map[string]string{
		"https://cdn.shopify.com/s/files/1/0794/9162/9342/files/722944.jpg?v=1795687103":                                                       "722944",
		"https://cdn.shopify.com/s/files/1/0794/9162/9342/files/696492_317fe558-7efd-4244-897b-48410d7eae58.jpg":                               "696492",
		"https://cdn.shopify.com/s/files/1/0794/9162/9342/files/126ea71a-9a64-4d1e-a8ca-1efce3b8346f_168a9828-2301-424f-8f96-c4a19fcade58.png": "126ea71a-9a64-4d1e-a8ca-1efce3b8346f",
		"https://cdn.shopify.com/s/files/1/0794/9162/9342/files/New.jpg":                                                                       "",
	}
	for link, want := range tests {
		product := sfProduct{}
		product.FeaturedImage = &struct {
			URL string `json:"url"`
		}{link}
		got := productID(product)
		if got != want {
			t.Errorf("%s: got %q, want %q", link, got, want)
		}
	}
	if productID(sfProduct{}) != "" {
		t.Error("a product with no photo names an id")
	}
}

func TestFinishOf(t *testing.T) {
	tests := []struct {
		title, variant string
		want           string
	}{
		{"Sheoldred, the Apocalypse [Dominaria United]", "Lightly Played", "Normal"},
		{"Sheoldred, the Apocalypse [Dominaria United]", "Lightly Played Foil", "Foil"},
		{"Eevee (173) [Scarlet & Violet: Black Star Promos]", "Near Mint Holofoil", "Holofoil"},
		{"Winter's Bite (Blue) [U-ELE171] (Tales of Aria Unlimited)  Unlimited Rainbow Foil", "Near Mint Unlimited Rainbow Foil", "Unlimited Edition Rainbow Foil"},
		{"Falling Star (029/298) - Origins Foil", "Near Mint", "Foil"},
		{"Crobat 076  - Holofoil ME Mega Evolution Promo - Promo", "Near Mint", "Holofoil"},
		{"Lumiose City 077/088  ME03 Perfect Order - Uncommon", "Near Mint", "Normal"},
		{"Lotus Petal (Foil Etched) [30th Anniversary Promos]", "Near Mint Foil", "Etched"},
		{"Seal of Removal (MH2-269) - Modern Horizons 2 Etched Foil", "Near Mint", "Etched"},
		{"Duress [Iconic Masters]", "Near Mint Foil Japanese", ""},
		{"Lugia V (138/195) [Sword & Shield: Silver Tempest]", "Graded - PSA 10", ""},
		{"Etched Champion [Scars of Mirrodin]", "Near Mint", "Normal"},
	}
	for _, test := range tests {
		var got string
		_, printing, ok := splitVariant(test.variant)
		if ok {
			finish, ok := finishOf(test.title, printing)
			if ok {
				got = finish
			}
		}
		if got != test.want {
			t.Errorf("%q %q: got %q, want %q", test.title, test.variant, got, test.want)
		}
	}
}

var (
	magicOnce sync.Once
	magicErr  error
	magicB    *matcher.Backend
)

// withMagic loads AllPrintings the first time a test asks for it, and skips
// where the run carries none.
func withMagic(t *testing.T) *matcher.Backend {
	t.Helper()
	magicOnce.Do(func() {
		path := os.Getenv("ALLPRINTINGS5_PATH")
		if path == "" {
			return
		}
		magicB, magicErr = datastore.Read(matcher.GameMagic, path)
	})
	if magicErr != nil {
		t.Fatal(magicErr)
	}
	if magicB == nil {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	return magicB
}

// testSingles prices a fixture's products and checks each landed on the
// printings named, "<set> <number> <finish>", or on none.
func testSingles(t *testing.T, b *matcher.Backend, fixture string, want map[string][]string) {
	t.Helper()
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatal(err)
	}
	var products []sfProduct
	err = json.Unmarshal(data, &products)
	if err != nil {
		t.Fatal(err)
	}
	sg, err := NewScraper(b)
	if err != nil {
		t.Fatal(err)
	}
	titles := map[string]string{}
	for _, product := range products {
		titles[gidNumber(product.ID)] = product.Title
		sg.processProduct(product)
	}
	got := map[string][]string{}
	for uuid, entries := range sg.inventory {
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			title := titles[entry.OriginalID]
			landing := co.SetCode + " " + co.Number + " " + co.Finish
			if !slices.Contains(got[title], landing) {
				got[title] = append(got[title], landing)
			}
		}
	}
	for _, product := range products {
		slices.Sort(got[product.Title])
		if !slices.Equal(got[product.Title], want[product.Title]) {
			t.Errorf("%s: got %q, want %q", product.Title, got[product.Title], want[product.Title])
		}
	}
}

func TestMagicSingles(t *testing.T) {
	testSingles(t, withMagic(t), "testdata/magic.json", map[string][]string{
		// The photo is the foil numbered apart; the listing is the plain card
		"Shifting Sky [Eighth Edition]": {"8ED 100 nonfoil"},
		// The photo is another card's
		"The Destined Black Mage 0447 (Borderless) [FINAL FANTASY] (FIC)": nil,
		// The photo names no id the datastore carries; the SKU numbers it
		"Grief [Modern Horizons 2]":                                 {"MH2 87 nonfoil"},
		"Seal of Removal (MH2-269) - Modern Horizons 2 Etched Foil": {"MH2 269 etched"},
		// A flavor name is left to the id
		"Lucille [Secret Lair Drop Promos]": {"SLD 581 foil"},
		// A finish the printing is not sold in is not clamped onto another
		"Curse-Marred Demon (Borderless) (FRA-328) - Reality Fracture Foil": nil,
		"Teferi's Puzzle Box (Foil Etched) [Secret Lair Drop Series]":       nil,
	})
}

func TestPokemonSingles(t *testing.T) {
	testSingles(t, gametest.Backend(t, matcher.GamePokemon), "testdata/pokemon.json", map[string][]string{
		"Fennekin 080  - Holofoil ME Mega Evolution Promo - Promo": {"MEP 080 holofoil"},
		"Raichu (29/62) [Fossil 1st Edition]":                      {"FO 29 1stedition"},
	})
}
