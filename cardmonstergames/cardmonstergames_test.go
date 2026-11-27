package cardmonstergames

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgban"

	api "github.com/mtgban/go-tcgplayer"

	_ "github.com/mtgban/go-mtgban/matcher/magic"
)

// testCatalog holds the TCGplayer products and skus the fixture's singles
// carry.
var testCatalog = &api.CatalogDump{
	Category:  api.Category{CategoryID: api.CategoryMagic, Name: "Magic"},
	Languages: []api.Language{{LanguageID: 1, Name: "English"}},
	Printings: []api.Printing{{PrintingID: 1, Name: "Normal"}, {PrintingID: 2, Name: "Foil"}},
	Products: []api.Product{
		{ProductID: 169083, Name: "Omniscience", ProductType: "Cards", Skus: []api.SKU{
			{SKUID: 3765175, ProductID: 169083, LanguageID: 1, PrintingID: 1, ConditionID: 1},
		}},
		{ProductID: 490536, Name: "Tribute to the World Tree", ProductType: "Cards", Skus: []api.SKU{
			{SKUID: 7053662, ProductID: 490536, LanguageID: 1, PrintingID: 2, ConditionID: 1},
		}},
		{ProductID: 262050, Name: "Boseiju, Who Endures", ProductType: "Cards", Skus: []api.SKU{
			{SKUID: 5375534, ProductID: 262050, LanguageID: 1, PrintingID: 2, ConditionID: 1},
		}},
		{ProductID: 631051, Name: "Yuna, Hope of Spira (Borderless) (Surge Foil)", ProductType: "Cards", Skus: []api.SKU{
			{SKUID: 8700810, ProductID: 631051, LanguageID: 1, PrintingID: 2, ConditionID: 1},
		}},
		{ProductID: 221774, Name: "Valakut Awakening", ProductType: "Cards", Skus: []api.SKU{
			{SKUID: 4539307, ProductID: 221774, LanguageID: 1, PrintingID: 1, ConditionID: 1},
		}},
	},
}

// TestProcessProducts prices each available single by its sku, in the
// finish the sku names even where the title leaves the foil out, and skips
// sealed product, a listing with no sku and one sold out.
func TestProcessProducts(t *testing.T) {
	path := os.Getenv("ALLPRINTINGS5_PATH")
	if path == "" {
		t.Skip("Need ALLPRINTINGS5_PATH set to run this test")
	}
	b, err := datastore.Read("magic", path)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/products.json")
	if err != nil {
		t.Fatal(err)
	}
	var feed struct {
		Products []shopifyProduct `json:"products"`
	}
	err = json.Unmarshal(data, &feed)
	if err != nil {
		t.Fatal(err)
	}

	cmg := NewScraper(b)
	cmg.catalog = testCatalog
	err = cmg.processProducts(feed.Products)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string]float64{
		"M19 65 false": 7,
		"MOM 211 true": 19,
		"NEO 266 true": 60,
		"FIN 549 true": 170,
	}
	got := map[string]float64{}
	for cardID, entries := range cmg.inventory {
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if entry.Conditions != mtgban.NM {
				t.Errorf("%s %s: graded %s", co.SetCode, co.Number, entry.Conditions)
			}
			got[fmt.Sprintf("%s %s %v", co.SetCode, co.Number, co.Foil)] = entry.Price
		}
	}
	if len(got) != len(want) {
		t.Errorf("got %v, want %v", got, want)
	}
	for card, price := range want {
		if got[card] != price {
			t.Errorf("%s: got %v, want %v", card, got[card], price)
		}
	}
}
