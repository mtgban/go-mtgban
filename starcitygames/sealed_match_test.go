package starcitygames

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// sealedMatchBackend is a Lorcana set of three products: the box carries an
// scgId, the gift set and the case carry none and are found by name.
func sealedMatchBackend(t *testing.T) *matcher.Backend {
	t.Helper()
	b := &matcher.Backend{
		Game:           matcher.GameLorcana,
		UUIDs:          map[string]*matcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*matcher.Set{
			"WSP": {Name: "Winterspell", Code: "WSP"},
		},
	}
	b.AddSealed("wsp-box", "Winterspell Booster Box", "WSP", "", map[string]string{"scgId": "SLD-LOR-BBX-011-EN"})
	b.AddSealed("wsp-gift", "Winterspell Scrooge McDuck Gift Set", "WSP", "", nil)
	b.AddSealed("wsp-case", "Winterspell Booster Box Case", "WSP", "", nil)
	b.SortSealed()
	b.IndexSets()

	for _, name := range []string{"Lorcana: Mystery Assortment", "Lorcana: Mystery Bundle", "Lorcana: Winterspell Booster Case"} {
		_, err := b.ResolveSealed(name)
		if err == nil {
			t.Fatalf("the resolver placed %q, so the fixture proves nothing", name)
		}
	}
	return b
}

// sealedMatchCatalog is the fixture catalog: one product per way a sealed
// product is matched, plus a single and another game's box the scraper
// does not read, and the box listed a second time.
func sealedMatchCatalog() []CatalogProduct {
	product := func(sku, name, language string) CatalogProduct {
		return CatalogProduct{
			SKU: sku, Name: name, Game: "Disney Lorcana", ProductType: ProductTypeSealed, Language: language,
			Variants: []CatalogVariant{{SKU: sku + "-NM", Qty: 1, Price: "100.00", SellListPrice: "50.00"}},
		}
	}
	single := product("LOR-WSP-001-EN", "Scrooge McDuck", "English")
	single.ProductType = "Singles"
	other := product("SLD-MTG-BBX-FDN-EN", "Magic: The Gathering - Foundations Booster Box", "English")
	other.Game = "Magic: The Gathering"
	return []CatalogProduct{
		product("SLD-LOR-REF-011-EN", "Lorcana: Mystery Bundle", "English"),
		product("SLD-LOR-BBX-011-EN", "Lorcana: Mystery Assortment", "English"),
		product("SLD-LOR-GFT-011-EN", "Lorcana: Winterspell - Scrooge McDuck Gift Set", "English"),
		product("SLD-LOR-CAS-011-EN", "Lorcana: Winterspell Booster Case", "English"),
		product("SLD-LOR-BBX-011-JP", "Lorcana: Winterspell Booster Box", "Japanese"),
		product("SLD-LOR-BBX-011-DE", "Lorcana: Winterspell Booster Box (German)", ""),
		single,
		other,
		product("SLD-LOR-BBX-011-EN", "Lorcana: Mystery Assortment", "English"),
	}
}

// TestMatchSealedCatalogAgreesWithLoad pins that what Load prices and what
// the export reports land on the same product for every sku.
func TestMatchSealedCatalogAgreesWithLoad(t *testing.T) {
	b := sealedMatchBackend(t)
	catalog := sealedMatchCatalog()

	scg, err := NewScraperSealed(b, "")
	if err != nil {
		t.Fatal(err)
	}
	scg.productMap = buildProductMap(b)
	for _, p := range catalog {
		scg.processProduct(p)
	}
	priced := map[string]string{}
	for uuid, entries := range scg.inventory {
		for _, entry := range entries {
			priced[entry.OriginalID] = uuid
		}
	}
	for uuid, entries := range scg.buylist {
		for _, entry := range entries {
			sku := entry.OriginalID[:len(entry.OriginalID)-len("-NM")]
			if priced[sku] != "" && priced[sku] != uuid {
				t.Errorf("%s: Load sells it as %q and buys it as %q", sku, priced[sku], uuid)
			}
			priced[sku] = uuid
		}
	}

	matcherScg, err := newSealedMatcher(b)
	if err != nil {
		t.Fatal(err)
	}
	exported := map[string]string{}
	for _, match := range matcherScg.matchCatalog(catalog) {
		exported[match.StoreID] = match.UUID
	}

	for sku, uuid := range priced {
		if exported[sku] != uuid {
			t.Errorf("%s: Load prices it on %q, the export says %q", sku, uuid, exported[sku])
		}
	}
	for sku, uuid := range exported {
		if priced[sku] != uuid {
			t.Errorf("%s: the export says %q, Load prices it on %q", sku, uuid, priced[sku])
		}
	}
}

// TestMatchSealedCatalogRows pins every row of the fixture: one match per
// sku, sorted, the drop reasons Load counts as the skip reasons.
func TestMatchSealedCatalogRows(t *testing.T) {
	scg, err := newSealedMatcher(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	got := scg.matchCatalog(sealedMatchCatalog())
	want := []mtgban.SealedMatch{
		{StoreID: "SLD-LOR-BBX-011-DE", Name: "Lorcana: Winterspell Booster Box (German)", Via: mtgban.SealedSkipped, Reason: "language variant"},
		{StoreID: "SLD-LOR-BBX-011-EN", Name: "Lorcana: Mystery Assortment", UUID: "wsp-box", Via: mtgban.SealedByID},
		{StoreID: "SLD-LOR-BBX-011-JP", Name: "Lorcana: Winterspell Booster Box", Via: mtgban.SealedSkipped, Reason: "not sold in English"},
		{StoreID: "SLD-LOR-CAS-011-EN", Name: "Lorcana: Winterspell Booster Case", UUID: "wsp-case", Via: mtgban.SealedByName},
		{StoreID: "SLD-LOR-GFT-011-EN", Name: "Lorcana: Winterspell - Scrooge McDuck Gift Set", UUID: "wsp-gift", Via: mtgban.SealedByName},
		{StoreID: "SLD-LOR-REF-011-EN", Name: "Lorcana: Mystery Bundle", Via: mtgban.SealedRefused, Reason: matcher.ErrCardDoesNotExist.Error()},
	}
	if len(got) != len(want) {
		t.Errorf("got %d matches, want %d: %+v", len(got), len(want), got)
	}
	for i := range min(len(got), len(want)) {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestMatchSealedCatalogOnePerSKU pins one match per sku however often the
// catalog lists it, a placed answer kept over a later one placing nothing.
func TestMatchSealedCatalogOnePerSKU(t *testing.T) {
	scg, err := newSealedMatcher(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	box := func(name string) CatalogProduct {
		return CatalogProduct{SKU: "SLD-LOR-X", Name: name, Game: "Lorcana", ProductType: ProductTypeSealed}
	}
	got := scg.matchCatalog([]CatalogProduct{box("Lorcana: Winterspell Booster Case"), box("Lorcana: Mystery Bundle")})
	if len(got) != 1 || got[0].UUID != "wsp-case" {
		t.Errorf("got %+v, want one match on wsp-case", got)
	}
}

// TestMatchSealedCatalogRefuses pins the games and the key the export will
// not match with, refused before any request.
func TestMatchSealedCatalogRefuses(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, game := range []matcher.Game{matcher.GameMagic, matcher.GameOnePiece} {
		matches, err := MatchSealedCatalog(ctx, &matcher.Backend{Game: game}, "key")
		if err == nil {
			t.Errorf("%s: got %d matches, want an error", game, len(matches))
			continue
		}
		if errors.Is(err, context.Canceled) {
			t.Errorf("%s: reached the network", game)
		}
	}
	matches, err := MatchSealedCatalog(ctx, &matcher.Backend{Game: matcher.GameLorcana}, "")
	if err == nil || errors.Is(err, context.Canceled) {
		t.Errorf("an empty key gave %d matches and %v, want a refusal", len(matches), err)
	}
}

// TestMatchSealedCatalogSorts pins the order: by sku.
func TestMatchSealedCatalogSorts(t *testing.T) {
	scg, err := newSealedMatcher(sealedMatchBackend(t))
	if err != nil {
		t.Fatal(err)
	}
	var catalog []CatalogProduct
	for _, sku := range []string{"C", "A", "B"} {
		catalog = append(catalog, CatalogProduct{SKU: sku, Name: "Lorcana: Mystery Bundle", Game: "Lorcana", ProductType: ProductTypeSealed})
	}
	var skus []string
	for _, match := range scg.matchCatalog(catalog) {
		skus = append(skus, match.StoreID)
	}
	if !slices.Equal(skus, []string{"A", "B", "C"}) {
		t.Errorf("got %v, want A, B, C", skus)
	}
}
