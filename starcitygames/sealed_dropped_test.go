package starcitygames

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// TestSealedPricedCount covers the count a run signs off with. What it priced
// is what it came away with a price for on either side of the counter, and the
// two sides are far from the same list: Star City Games quotes a buy price on
// plenty of sealed product it holds no stock of, so counting the shelf alone
// would move the number with stock rather than with coverage - the reading the
// drop tally beside it is there to prevent.
func TestSealedPricedCount(t *testing.T) {
	scg, err := NewScraperSealed(&matcher.Backend{Game: "lorcana"}, "")
	if err != nil {
		t.Fatal(err)
	}
	// One product on both sides, one it has stock of and will not buy, one it
	// will buy and has no stock of. Three products carry a price; neither side
	// on its own says three.
	for _, uuid := range []string{"both", "sell-only"} {
		if err := scg.inventory.Add(uuid, &mtgban.InventoryEntry{Price: 1, Quantity: 1}); err != nil {
			t.Fatal(err)
		}
	}
	for _, uuid := range []string{"both", "buy-only"} {
		if err := scg.buylist.Add(uuid, &mtgban.BuylistEntry{BuyPrice: 1}); err != nil {
			t.Fatal(err)
		}
	}

	if got := scg.pricedProducts(); got != 3 {
		t.Errorf("pricedProducts() = %d, want 3 (inventory holds %d, buylist %d)",
			got, len(scg.inventory), len(scg.buylist))
	}
}

// TestSealedDropAccounting covers the products a sealed run turns down. For
// the games whose datastore carries no sku, every product goes through the
// name resolver, and each refusal names itself and is counted under its
// reason, so a game losing coverage does not look like a game with nothing
// more to sell.
func TestSealedDropAccounting(t *testing.T) {
	b := withLorcana(t)

	scg, err := NewScraperSealed(b, "")
	if err != nil {
		t.Fatal(err)
	}
	var logs []string
	scg.logCallback = func(format string, a ...any) { logs = append(logs, fmt.Sprintf(format, a...)) }
	scg.productMap = map[string]string{}

	// A gift set the datastore does not carry, a Japanese box, and a product
	// of another game entirely, which is not this run's to account for.
	scg.processProduct(CatalogProduct{
		SKU: "SLD-LOR-BXS-011-EN", Name: "Lorcana: Winterspell - Scrooge McDuck Gift Set",
		Game: "Disney Lorcana", ProductType: ProductTypeSealed, Language: "English",
	})
	scg.processProduct(CatalogProduct{
		SKU: "SLD-LOR-BBX-011-JP", Name: "Lorcana: Winterspell Booster Box",
		Game: "Disney Lorcana", ProductType: ProductTypeSealed, Language: "Japanese",
	})
	scg.processProduct(CatalogProduct{
		SKU: "SLD-MTG-BBX-FDN-EN", Name: "Magic: The Gathering - Foundations Booster Box",
		Game: "Magic: The Gathering", ProductType: ProductTypeSealed, Language: "English",
	})

	if got := scg.dropped["not sold in English"]; got != 1 {
		t.Errorf("counted %d non-English drops, want 1", got)
	}
	total := 0
	for _, count := range scg.dropped {
		total += count
	}
	if total != 2 {
		t.Errorf("counted %d drops in %v, want 2", total, scg.dropped)
	}
	var named int
	for _, entry := range logs {
		if strings.Contains(entry, "Scrooge McDuck Gift Set") {
			named++
		}
	}
	if named != 1 {
		t.Errorf("the refused name was logged %d times, want 1", named)
	}
}

// TestSealedByScgID pins that a non-Magic product carrying an scgId is
// priced under that sku, its name never asked.
func TestSealedByScgID(t *testing.T) {
	b := &matcher.Backend{
		Game:           matcher.GameLorcana,
		UUIDs:          map[string]*matcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*matcher.Set{
			"WSP": {Name: "Winterspell", Code: "WSP"},
		},
	}
	b.AddSealed("wsp-box", "Winterspell Booster Box", "WSP", "",
		map[string]string{"scgId": "SLD-LOR-BBX-011-EN"})
	b.SortSealed()
	b.IndexSets()

	scg, err := NewScraperSealed(b, "")
	if err != nil {
		t.Fatal(err)
	}
	scg.productMap = buildProductMap(b)

	scg.processProduct(CatalogProduct{
		SKU: "SLD-LOR-BBX-011-EN", Name: "Lorcana: Mystery Assortment",
		Game: "Disney Lorcana", ProductType: ProductTypeSealed, Language: "English",
		Variants: []CatalogVariant{{SKU: "SLD-LOR-BBX-011-EN-NM", Qty: 1, Price: "100.00"}},
	})
	if len(scg.inventory["wsp-box"]) != 1 {
		t.Errorf("the box holds %v, want one price (dropped %v)", scg.inventory["wsp-box"], scg.dropped)
	}
}

// TestSealedNameKeepsOffAnIDsProduct pins that a product carrying a sku is
// priced under its sku and no other: a sku the datastore lacks whose name
// reaches it is dropped under a reason of its own and named in the log.
func TestSealedNameKeepsOffAnIDsProduct(t *testing.T) {
	b := &matcher.Backend{
		Game:           matcher.GameLorcana,
		UUIDs:          map[string]*matcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*matcher.Set{
			"WSP": {Name: "Winterspell", Code: "WSP"},
		},
	}
	b.AddSealed("wsp-box", "Winterspell Booster Box", "WSP", "",
		map[string]string{"scgId": "SLD-LOR-BBX-011-EN"})
	b.AddSealed("wsp-pack", "Winterspell Booster Pack", "WSP", "", nil)
	b.SortSealed()
	b.IndexSets()

	scg, err := NewScraperSealed(b, "")
	if err != nil {
		t.Fatal(err)
	}
	var logs []string
	scg.logCallback = func(format string, a ...any) { logs = append(logs, fmt.Sprintf(format, a...)) }
	scg.productMap = buildProductMap(b)

	for sku, name := range map[string]string{
		"SLD-LOR-BBX-999-EN": "Lorcana: Winterspell Booster Box",
		"SLD-LOR-BPK-011-EN": "Lorcana: Winterspell Booster Pack",
	} {
		scg.processProduct(CatalogProduct{
			SKU: sku, Name: name,
			Game: "Disney Lorcana", ProductType: ProductTypeSealed, Language: "English",
			Variants: []CatalogVariant{{SKU: sku + "-NM", Qty: 1, Price: "100.00"}},
		})
	}

	if len(scg.inventory["wsp-box"]) != 0 {
		t.Errorf("the box holds %v, want no price", scg.inventory["wsp-box"])
	}
	if len(scg.inventory["wsp-pack"]) != 1 {
		t.Errorf("the pack holds %v, want one price by name", scg.inventory["wsp-pack"])
	}
	if len(scg.dropped) != 1 || scg.dropped[sealedHoldsOwnID] != 1 {
		t.Errorf("dropped %v, want one under %q", scg.dropped, sealedHoldsOwnID)
	}
	if len(logs) != 1 || !strings.HasSuffix(logs[0], "(sku=SLD-LOR-BBX-999-EN): "+sealedHoldsOwnID) {
		t.Errorf("logged %q, want the refused sku named", logs)
	}
}
