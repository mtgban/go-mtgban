package cardtrader

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/mtgban"
)

// sealedMatchBackend is a Gundam shelf with a product for every way a
// blueprint settles: by its cardtraderId (a premium pack in two finishes
// among them), through its TCGplayer id, and by name.
func sealedMatchBackend() *matcher.Backend {
	b := &matcher.Backend{
		Game:           matcher.GameGundam,
		UUIDs:          map[string]*matcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*matcher.Set{
			"GD05": {Name: "Freedom Ascension", Code: "GD05"},
			"ST10": {Name: "Starter Deck 10: Generation Pulse", Code: "ST10"},
			"BND":  {Name: "Bundle Promos", Code: "BND"},
		},
	}
	b.AddSealed("gd05-box", "Freedom Ascension Booster Box", "GD05", "",
		map[string]string{"cardtraderId": "100"})
	b.AddSealed("gd05-premium", "Freedom Ascension Premium Pack", "GD05", "",
		map[string]string{"cardtraderId": "101"})
	b.AddSealed("gd05-premium-foil", "Freedom Ascension Premium Pack Foil", "GD05", "",
		map[string]string{"cardtraderId": "101"})
	b.AddSealed("gd05-pack", "Freedom Ascension Booster Pack", "GD05", "",
		map[string]string{"tcgplayerProductId": "555"})
	b.AddSealed("gd05-case", "Freedom Ascension Booster Box Case", "GD05", "", nil)
	b.AddSealed("st10-deck", "Starter Deck 10: Generation Pulse", "ST10", "", nil)
	b.SortSealed()
	b.IndexSets()
	return b
}

// sealedMatchBlueprints has a blueprint for every way one settles.
func sealedMatchBlueprints() map[int]*Blueprint {
	shelved := func(bp *Blueprint, shelf string) *Blueprint {
		bp.Expansion.Name = shelf
		return bp
	}
	return map[int]*Blueprint{
		100: {ID: 100, Name: "Mystery Assortment"},
		101: {ID: 101, Name: "Mystery Premium"},
		200: {ID: 200, Name: "Something Else Entirely", TCGplayerID: 555},
		300: shelved(&Blueprint{ID: 300, Name: "ST-10: Generation Pulse Starter Deck"}, "ST-10: Generation Pulse"),
		400: {ID: 400, Name: "Freedom Ascension Booster Box Case"},
		500: {ID: 500, Name: "Freedom Ascension Booster Box Case Bundle"},
		600: {ID: 600, Name: "Freedom Ascension Japanese Booster Box"},
		700: {ID: 700, Name: "Freedom Ascension Booster Box Sleeves", CategoryID: CategoryGundamSleeves},
		800: {ID: 800, Name: "Nothing The Resolver Knows"},
	}
}

// TestMatchSealedCatalogAgreesWithLoad pins that the exported matching places
// every blueprint exactly where Load's product map would price it.
func TestMatchSealedCatalogAgreesWithLoad(t *testing.T) {
	ct, err := newSealedMatcher(sealedMatchBackend())
	if err != nil {
		t.Fatal(err)
	}
	productMap := ct.buildProductMap(sealedMatchBlueprints())
	matches := ct.matchCatalog(sealedMatchBlueprints())

	exported := map[int]string{}
	for _, match := range matches {
		id, err := strconv.Atoi(match.StoreID)
		if err != nil {
			t.Fatalf("store id %q is not a blueprint id: %v", match.StoreID, err)
		}
		exported[id] = match.UUID
	}
	for id := range sealedMatchBlueprints() {
		var priced string
		uuids, found := productMap[id]
		if found {
			priced = uuids[0]
		}
		if exported[id] != priced {
			t.Errorf("%d: Load prices it on %q, the export says %q", id, priced, exported[id])
		}
	}
}

// TestMatchSealedCatalogRows pins what each blueprint of the fixture settles
// as, one match per id, sorted by id.
func TestMatchSealedCatalogRows(t *testing.T) {
	ct, err := newSealedMatcher(sealedMatchBackend())
	if err != nil {
		t.Fatal(err)
	}
	refusal := matcher.ErrCardDoesNotExist.Error()
	want := []mtgban.SealedMatch{
		{StoreID: "100", Name: "Mystery Assortment", UUID: "gd05-box", Via: mtgban.SealedByID},
		{StoreID: "101", Name: "Mystery Premium", UUID: "gd05-premium", Via: mtgban.SealedByID},
		{StoreID: "200", Name: "Something Else Entirely", UUID: "gd05-pack", Via: mtgban.SealedByBridge},
		{StoreID: "300", Name: "ST-10: Generation Pulse Starter Deck", UUID: "st10-deck", Via: mtgban.SealedByName},
		{StoreID: "400", Name: "Freedom Ascension Booster Box Case", UUID: "gd05-case", Via: mtgban.SealedByName},
		{StoreID: "500", Name: "Freedom Ascension Booster Box Case Bundle", Via: mtgban.SealedRefused, Reason: "subsumed by another listing"},
		{StoreID: "600", Name: "Freedom Ascension Japanese Booster Box", Via: mtgban.SealedSkipped, Reason: "language variant"},
		{StoreID: "700", Name: "Freedom Ascension Booster Box Sleeves", Via: mtgban.SealedSkipped, Reason: "accessory"},
		{StoreID: "800", Name: "Nothing The Resolver Knows", Via: mtgban.SealedRefused, Reason: refusal},
	}

	got := ct.matchCatalog(sealedMatchBlueprints())
	if len(got) != len(want) {
		t.Fatalf("got %d matches, want one per blueprint id, %d:\n%v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("row %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestMatchSealedCatalogSortsNumerically pins the order by blueprint id as a
// number, not as the decimal string it is reported as.
func TestMatchSealedCatalogSortsNumerically(t *testing.T) {
	ct, err := newSealedMatcher(sealedMatchBackend())
	if err != nil {
		t.Fatal(err)
	}
	got := ct.matchCatalog(map[int]*Blueprint{
		30:  {ID: 30, Name: "Nothing The Resolver Knows"},
		4:   {ID: 4, Name: "Nothing The Resolver Knows"},
		100: {ID: 100, Name: "Mystery Assortment"},
	})
	var ids []string
	for _, match := range got {
		ids = append(ids, match.StoreID)
	}
	if !slices.Equal(ids, []string{"4", "30", "100"}) {
		t.Errorf("got ids %v, want [4 30 100]", ids)
	}
}

// TestMatchSealedCatalogRefuses pins what the export turns down before it
// fetches anything: Magic, whose sealed map is ids alone, a game the sealed
// scraper is not registered for, and a missing token.
func TestMatchSealedCatalogRefuses(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, game := range []matcher.Game{matcher.GameMagic, matcher.GamePalworld} {
		b := &matcher.Backend{Game: game}
		matches, err := MatchSealedCatalog(ctx, b, "token")
		if err == nil {
			t.Errorf("%s: got %d matches, want an error", game, len(matches))
		}
		if errors.Is(err, context.Canceled) {
			t.Errorf("%s: reached the network", game)
		}
	}
	matches, err := MatchSealedCatalog(ctx, sealedMatchBackend(), "")
	if err == nil {
		t.Errorf("no token: got %d matches, want an error", len(matches))
	}
	if errors.Is(err, context.Canceled) {
		t.Error("no token: reached the network")
	}
}
