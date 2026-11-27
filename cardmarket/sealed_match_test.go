package cardmarket

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// sealedMatchBackend is sealedFallbackBackend's One Piece shelf with a bundle
// sold in two finishes under one Cardmarket id.
func sealedMatchBackend() *matcher.Backend {
	b := &matcher.Backend{
		Game:           matcher.GameOnePiece,
		UUIDs:          map[string]*matcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*matcher.Set{
			"OP01": {Name: "Romance Dawn", Code: "OP01"},
			"OP02": {Name: "Paramount War", Code: "OP02"},
			"OP03": {Name: "Pillars of Strength", Code: "OP03"},
		},
	}
	b.AddSealed("op01-box", "Romance Dawn - Booster Box", "OP01", "",
		map[string]string{"tcgplayerProductId": "10", "mcmId": "1"})
	b.AddSealed("op01-pack", "Romance Dawn - Booster Pack", "OP01", "",
		map[string]string{"tcgplayerProductId": "5"})
	b.AddSealed("op02-box", "Paramount War - Booster Box", "OP02", "", nil)
	b.AddSealed("op02-bundle", "Paramount War - Bundle", "OP02", "",
		map[string]string{"mcmId": "13"})
	b.AddSealed("op02-bundle-foil", "Paramount War - Bundle Foil", "OP02", "",
		map[string]string{"mcmId": "13"})
	b.AddSealed("op03-wave1", "Pillars of Strength - Booster Box (Wave 1)", "OP03", "",
		map[string]string{"tcgplayerProductId": "30"})
	b.AddSealed("op03-wave2", "Pillars of Strength - Booster Box (Wave 2)", "OP03", "",
		map[string]string{"tcgplayerProductId": "31"})
	b.SortSealed()
	b.IndexSets()
	return b
}

// sealedMatchCatalog has a row for every way a product settles, out of id
// order so the sort is tested too.
var sealedMatchCatalog = []cm.ProductList{
	{IDProduct: 12, Name: "Paramount War Booster Box (24 Packs)", CategoryName: "One Piece Booster Boxes"},
	{IDProduct: 1, Name: "Romance Dawn Booster Box", CategoryName: "One Piece Booster Boxes"},
	{IDProduct: 2, Name: "Nothing The Resolver Knows", CategoryName: "One Piece Boosters"},
	{IDProduct: 3, Name: "Paramount War Booster Box", CategoryName: "One Piece Booster Boxes"},
	{IDProduct: 7, Name: "Pillars of Strength Booster Box (Pre-Errata)", CategoryName: "One Piece Booster Boxes"},
	{IDProduct: 8, Name: "Pillars of Strength Booster Box", CategoryName: "One Piece Booster Boxes"},
	{IDProduct: 5, Name: "Paramount War Booster Box", CategoryName: "One Piece Coins"},
	{IDProduct: 9, Name: "Paramount War Booster Box (Asia Region Legal)", CategoryName: "One Piece Booster Boxes"},
	{IDProduct: 10, Name: "Nothing The Resolver Knows", CategoryName: "One Piece Boosters"},
	{IDProduct: 13, Name: "Paramount War Bundle", CategoryName: "One Piece Bundles"},
}

// sealedMatchBridge links Cardmarket id 2 to the pack's TCGplayer id.
var sealedMatchBridge = map[int]int{2: 5}

// TestMatchSealedCatalogAgreesWithLoad pins that the exported matching places
// every product exactly where Load's placing would price it.
func TestMatchSealedCatalogAgreesWithLoad(t *testing.T) {
	b := sealedMatchBackend()
	mkm, err := newSealedMatcher(b, sealedMatchBridge)
	if err != nil {
		t.Fatal(err)
	}

	lookup := mkm.newLookup()
	mkm.addPrintRuns(&lookup, sealedMatchCatalog)
	placed := mkm.placeProducts(sealedMatchCatalog, lookup)

	matches := mkm.matchCatalog(sealedMatchCatalog)
	exported := map[int]string{}
	for _, match := range matches {
		id, err := strconv.Atoi(match.StoreID)
		if err != nil {
			t.Fatalf("store id %q is not Cardmarket's: %v", match.StoreID, err)
		}
		if match.UUID != "" {
			exported[id] = match.UUID
		}
	}
	for _, id := range placed.productIDs {
		if exported[id] != placed.productMap[id][0] {
			t.Errorf("%d: Load prices it on %s, the export says %q", id, placed.productMap[id][0], exported[id])
		}
	}
	for id, uuid := range exported {
		if !slices.Contains(placed.productIDs, id) {
			t.Errorf("%d: the export places it on %s, Load does not price it", id, uuid)
		}
	}
}

// TestMatchSealedCatalogRows pins what each row of the fixture settles as.
func TestMatchSealedCatalogRows(t *testing.T) {
	b := sealedMatchBackend()
	mkm, err := newSealedMatcher(b, sealedMatchBridge)
	if err != nil {
		t.Fatal(err)
	}
	refusal := matcher.ErrCardDoesNotExist.Error()
	want := []mtgban.SealedMatch{
		{StoreID: "1", Name: "Romance Dawn Booster Box", UUID: "op01-box", Via: mtgban.SealedByID},
		{StoreID: "2", Name: "Nothing The Resolver Knows", UUID: "op01-pack", Via: mtgban.SealedByBridge},
		{StoreID: "3", Name: "Paramount War Booster Box", UUID: "op02-box", Via: mtgban.SealedByName},
		{StoreID: "5", Name: "Paramount War Booster Box", Via: mtgban.SealedSkipped, Reason: sealedShelves["Coins"]},
		{StoreID: "7", Name: "Pillars of Strength Booster Box (Pre-Errata)", UUID: "op03-wave1", Via: mtgban.SealedByName},
		{StoreID: "8", Name: "Pillars of Strength Booster Box", UUID: "op03-wave2", Via: mtgban.SealedByName},
		{StoreID: "9", Name: "Paramount War Booster Box (Asia Region Legal)", Via: mtgban.SealedSkipped, Reason: "foreign printing"},
		{StoreID: "10", Name: "Nothing The Resolver Knows", Via: mtgban.SealedRefused, Reason: refusal},
		{StoreID: "12", Name: "Paramount War Booster Box (24 Packs)", Via: mtgban.SealedRefused, Reason: "subsumed by another listing"},
		{StoreID: "13", Name: "Paramount War Bundle", UUID: "op02-bundle", Via: mtgban.SealedByID},
	}

	got := mkm.matchCatalog(sealedMatchCatalog)
	if len(got) != len(want) {
		t.Fatalf("got %d matches, want one per product id, %d:\n%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestMatchSealedCatalogOnePerID pins one match per Cardmarket id, sorted by
// id, however often the catalog lists one.
func TestMatchSealedCatalogOnePerID(t *testing.T) {
	b := sealedMatchBackend()
	mkm, err := newSealedMatcher(b, nil)
	if err != nil {
		t.Fatal(err)
	}
	catalog := []cm.ProductList{
		{IDProduct: 30, Name: "Paramount War Booster Box", CategoryName: "One Piece Booster Boxes"},
		{IDProduct: 4, Name: "Nothing The Resolver Knows", CategoryName: "One Piece Boosters"},
		{IDProduct: 30, Name: "Nothing The Resolver Knows", CategoryName: "One Piece Boosters"},
		{IDProduct: 100, Name: "Paramount War Booster Box", CategoryName: "One Piece Coins"},
	}
	got := mkm.matchCatalog(catalog)
	var ids []string
	for _, match := range got {
		ids = append(ids, match.StoreID)
	}
	if !slices.Equal(ids, []string{"4", "30", "100"}) {
		t.Fatalf("got ids %v, want [4 30 100]", ids)
	}
	// Load prices id 30 on what its placed row found, so the export says so.
	if got[1].UUID != "op02-box" || got[1].Via != mtgban.SealedByName {
		t.Errorf("id 30: got %+v, want its placed row", got[1])
	}
}

// TestMatchSealedCatalogRefusesMagic pins the games the export refuses before
// it downloads anything: Magic, whose sealed map is ids alone, and a game the
// sealed scraper is not registered for.
func TestMatchSealedCatalogRefusesMagic(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, game := range []matcher.Game{matcher.GameMagic, matcher.GamePalworld} {
		b := &matcher.Backend{Game: game}
		matches, err := MatchSealedCatalog(ctx, b, nil)
		if err == nil {
			t.Errorf("%s: got %d matches, want an error", game, len(matches))
		}
		if errors.Is(err, context.Canceled) {
			t.Errorf("%s: reached the network", game)
		}
	}
}

// TestMatchSealedCatalogDownload pins the export over its download: the
// list it hands back matched, a failed download its error, and an empty
// list an error rather than an empty answer.
func TestMatchSealedCatalogDownload(t *testing.T) {
	mkm, err := newSealedMatcher(sealedMatchBackend(), map[int]int{2: 5, 3: 5})
	if err != nil {
		t.Fatal(err)
	}
	var asked cm.Game
	listed := func(_ context.Context, game cm.Game) ([]cm.ProductList, error) {
		asked = game
		return sealedMatchCatalog, nil
	}
	got, err := mkm.matchSealedCatalog(context.Background(), listed)
	if err != nil {
		t.Fatalf("matchSealedCatalog(): %v", err)
	}
	if asked != cm.GameOnePiece {
		t.Errorf("downloaded game %v, want One Piece", asked)
	}
	want := mkm.matchCatalog(sealedMatchCatalog)
	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}

	boom := errors.New("boom")
	failed := func(context.Context, cm.Game) ([]cm.ProductList, error) {
		return nil, boom
	}
	got, err = mkm.matchSealedCatalog(context.Background(), failed)
	if !errors.Is(err, boom) {
		t.Errorf("got %d matches and %v, want the download's error", len(got), err)
	}

	empty := func(context.Context, cm.Game) ([]cm.ProductList, error) {
		return nil, nil
	}
	got, err = mkm.matchSealedCatalog(context.Background(), empty)
	if err == nil {
		t.Errorf("got %d matches from an empty list, want an error", len(got))
	}
}
