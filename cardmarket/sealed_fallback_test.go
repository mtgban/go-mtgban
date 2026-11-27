package cardmarket

import (
	"maps"
	"slices"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/matcher"
)

// sealedFallbackBackend holds one product with a Cardmarket id, one with only
// a TCGplayer id, one with neither, and a box held in two print runs.
func sealedFallbackBackend() *matcher.Backend {
	backend := &matcher.Backend{
		UUIDs:          map[string]*matcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*matcher.Set{
			"OP01": {Name: "Romance Dawn", Code: "OP01"},
			"OP02": {Name: "Paramount War", Code: "OP02"},
			"OP03": {Name: "Pillars of Strength", Code: "OP03"},
		},
	}
	backend.AddSealed("op01-box", "Romance Dawn - Booster Box", "OP01", "",
		matcher.SealedIDs{"tcgplayerProductId": {"10"}, "mcmId": {"1"}})
	backend.AddSealed("op01-pack", "Romance Dawn - Booster Pack", "OP01", "",
		matcher.SealedIDs{"tcgplayerProductId": {"5"}})
	backend.AddSealed("op02-box", "Paramount War - Booster Box", "OP02", "", nil)
	backend.AddSealed("op03-wave1", "Pillars of Strength - Booster Box (Wave 1)", "OP03", "",
		matcher.SealedIDs{"tcgplayerProductId": {"30"}})
	backend.AddSealed("op03-wave2", "Pillars of Strength - Booster Box (Wave 2)", "OP03", "",
		matcher.SealedIDs{"tcgplayerProductId": {"31"}})
	backend.SortSealed()
	backend.IndexSets()
	return backend
}

// fallbackLookup prepares what Load would for a catalog: the datastore's
// Cardmarket ids, a bridge from Cardmarket to TCGplayer ids, and the print
// runs the catalog marks.
func fallbackLookup(b *matcher.Backend, bridge map[int]int, catalog []cm.ProductList) sealedLookup {
	byID := b.BuildSealedProductMap("mcmId")
	lookup := sealedLookup{
		byID:      byID,
		bridged:   bridgedProducts(bridge, b.BuildSealedProductMap("tcgplayerProductId"), byID),
		printRuns: newPrintRunIndex(b),
		earlyRuns: map[string]bool{},
	}
	for _, product := range catalog {
		if namesEarlyPrintRun(product.Name) {
			lookup.earlyRuns[sealedBaseKey(product.Name)] = true
		}
	}
	return lookup
}

// sealedWant is what one product should settle as.
type sealedWant struct {
	product cm.ProductList
	how     sealedSettled
	uuid    string
	reason  string
}

func checkMatches(t *testing.T, mkm *Sealed, lookup sealedLookup, wants []sealedWant) {
	t.Helper()
	for _, want := range wants {
		got := mkm.matchProduct(want.product, lookup)
		if got.how != want.how || got.reason != want.reason {
			t.Errorf("%q (%d) settled as %d %q, want %d %q", want.product.Name, want.product.IDProduct,
				got.how, got.reason, want.how, want.reason)
			continue
		}
		if want.uuid == "" {
			if len(got.uuids) != 0 {
				t.Errorf("%q (%d) placed on %v, want nothing", want.product.Name, want.product.IDProduct, got.uuids)
			}
			continue
		}
		if !slices.Equal(got.uuids, []string{want.uuid}) {
			t.Errorf("%q (%d) placed on %v, want %s", want.product.Name, want.product.IDProduct, got.uuids, want.uuid)
		}
	}
}

// TestSealedFallbackWithSomeIDs pins that one Cardmarket id in a datastore
// does not switch the bridge and the name pass off for the rest of the game.
func TestSealedFallbackWithSomeIDs(t *testing.T) {
	b := sealedFallbackBackend()
	catalog := []cm.ProductList{
		{IDProduct: 1, Name: "Romance Dawn Booster Box", CategoryName: "One Piece Booster Boxes"},
		{IDProduct: 2, Name: "Nothing The Resolver Knows", CategoryName: "One Piece Boosters"},
		{IDProduct: 3, Name: "Paramount War Booster Box", CategoryName: "One Piece Booster Boxes"},
	}
	lookup := fallbackLookup(b, map[int]int{2: 5}, catalog)
	mkm := &Sealed{backend: b, gameID: cm.GameOnePiece}

	checkMatches(t, mkm, lookup, []sealedWant{
		{product: catalog[0], how: settledByID, uuid: "op01-box"},
		{product: catalog[1], how: settledByBridge, uuid: "op01-pack"},
		{product: catalog[2], how: settledByName, uuid: "op02-box"},
	})
}

// TestSealedFallbackGuardsStillRun pins that a product with no id still meets
// the shelf and language guards, ahead of the bridge, beside a product that
// has one.
func TestSealedFallbackGuardsStillRun(t *testing.T) {
	b := sealedFallbackBackend()
	catalog := []cm.ProductList{
		{IDProduct: 4, Name: "Paramount War Booster Box (Asia Region Legal)", CategoryName: "One Piece Booster Boxes"},
		{IDProduct: 5, Name: "Paramount War Booster Box", CategoryName: "One Piece Coins"},
		{IDProduct: 6, Name: "Romance Dawn Booster Pack (Japanese)", CategoryName: "One Piece Boosters"},
	}
	// The bridge would place the foreign pack on the English one.
	lookup := fallbackLookup(b, map[int]int{6: 5}, catalog)
	mkm := &Sealed{backend: b, gameID: cm.GameOnePiece}

	checkMatches(t, mkm, lookup, []sealedWant{
		{product: catalog[0], how: settledSkipped, reason: "foreign printing"},
		{product: catalog[1], how: settledSkipped, reason: sealedShelves["Coins"]},
		{product: catalog[2], how: settledSkipped, reason: "foreign printing"},
	})
}

// TestSealedIDBeatsGuards pins that an id the datastore holds is trusted over
// what the name or the shelf says.
func TestSealedIDBeatsGuards(t *testing.T) {
	b := sealedFallbackBackend()
	catalog := []cm.ProductList{
		{IDProduct: 1, Name: "Romance Dawn Booster Box (Japanese)", CategoryName: "One Piece Coins"},
	}
	lookup := fallbackLookup(b, nil, catalog)
	mkm := &Sealed{backend: b, gameID: cm.GameOnePiece}

	checkMatches(t, mkm, lookup, []sealedWant{
		{product: catalog[0], how: settledByID, uuid: "op01-box"},
	})
}

// TestSealedMagicIDOnly pins that Magic settles by id alone, a name that
// would resolve included.
func TestSealedMagicIDOnly(t *testing.T) {
	b := sealedFallbackBackend()
	catalog := []cm.ProductList{
		{IDProduct: 1, Name: "Romance Dawn Booster Box", CategoryName: "Magic Booster Boxes"},
		{IDProduct: 2, Name: "Nothing The Resolver Knows", CategoryName: "Magic Boosters"},
		{IDProduct: 3, Name: "Paramount War Booster Box", CategoryName: "Magic Booster Boxes"},
	}
	lookup := sealedLookup{byID: b.BuildSealedProductMap("mcmId")}
	mkm := &Sealed{backend: b, gameID: cm.GameMagic}

	checkMatches(t, mkm, lookup, []sealedWant{
		{product: catalog[0], how: settledByID, uuid: "op01-box"},
		{product: catalog[1], how: settledSkipped, reason: sealedUnlinked},
		{product: catalog[2], how: settledSkipped, reason: sealedUnlinked},
	})
}

// TestSealedFallbackWithoutIDs pins the path every non-Magic datastore takes
// today, with no Cardmarket id anywhere: the print runs beat the bridge, the
// bridge beats the name, and a name nobody can place is the resolver's refusal.
func TestSealedFallbackWithoutIDs(t *testing.T) {
	b := sealedFallbackBackend()
	catalog := []cm.ProductList{
		{IDProduct: 1, Name: "Romance Dawn Booster Box", CategoryName: "One Piece Booster Boxes"},
		{IDProduct: 2, Name: "Nothing The Resolver Knows", CategoryName: "One Piece Boosters"},
		{IDProduct: 3, Name: "Paramount War Booster Box", CategoryName: "One Piece Booster Boxes"},
		{IDProduct: 7, Name: "Pillars of Strength Booster Box (Pre-Errata)", CategoryName: "One Piece Booster Boxes"},
		{IDProduct: 8, Name: "Pillars of Strength Booster Box", CategoryName: "One Piece Booster Boxes"},
		{IDProduct: 9, Name: "Paramount War Booster Box (Asia Region Legal)", CategoryName: "One Piece Booster Boxes"},
	}
	// The bridge lumps both runs onto the first.
	lookup := fallbackLookup(b, map[int]int{2: 5, 7: 30, 8: 30, 9: 10}, catalog)
	lookup.byID = map[int][]string{}
	mkm := &Sealed{backend: b, gameID: cm.GameOnePiece}

	refusal := matcher.ErrCardDoesNotExist.Error()
	checkMatches(t, mkm, lookup, []sealedWant{
		{product: catalog[0], how: settledByName, uuid: "op01-box"},
		{product: catalog[1], how: settledByBridge, uuid: "op01-pack"},
		{product: catalog[2], how: settledByName, uuid: "op02-box"},
		{product: catalog[3], how: settledByName, uuid: "op03-wave1"},
		{product: catalog[4], how: settledByName, uuid: "op03-wave2"},
		{product: catalog[5], how: settledSkipped, reason: "foreign printing"},
		{product: cm.ProductList{IDProduct: 10, Name: catalog[1].Name}, how: settledRefused, reason: refusal},
	})
	// The print-run index answers like an id: never pruned, never counted
	// as resolved by name.
	printRun := mkm.matchProduct(catalog[3], lookup)
	byName := mkm.matchProduct(catalog[2], lookup)
	if !printRun.printRun || byName.printRun {
		t.Error("only the print-run index's answer is marked as one")
	}
}

// sealedPlaceBackend is one box, holding Cardmarket id 1 when byID is true.
func sealedPlaceBackend(t *testing.T, byID bool) *matcher.Backend {
	t.Helper()
	b := &matcher.Backend{
		UUIDs:          map[string]*matcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*matcher.Set{
			"OGN": {Name: "Origins", Code: "OGN"},
		},
	}
	var ids matcher.SealedIDs
	if byID {
		ids = matcher.SealedIDs{"mcmId": {"1"}}
	}
	b.AddSealed("ogn-box", "Booster Box", "OGN", "", ids)
	b.SortSealed()
	b.IndexSets()
	for _, name := range []string{"Booster Box", "Booster Box (18 Boosters)"} {
		uuid, err := b.ResolveSealed(name)
		if err != nil || uuid != "ogn-box" {
			t.Fatalf("%q resolved to (%q, %v), so the fixture proves nothing", name, uuid, err)
		}
	}
	return b
}

// TestPlaceProductsPrunesBesideAnID pins that a product placed by its
// Cardmarket id is never pruned, and that a name reaching the same product
// is pruned beside it exactly as it is beside a product placed by name.
func TestPlaceProductsPrunesBesideAnID(t *testing.T) {
	for _, tt := range []struct {
		desc         string
		byID         bool
		catalog      []cm.ProductList
		kept         []int
		resolved     int
		subsumedDrop int
	}{
		{
			"both by name: the longer name loses the box",
			false,
			[]cm.ProductList{{IDProduct: 1, Name: "Booster Box"}, {IDProduct: 2, Name: "Booster Box (18 Boosters)"}},
			[]int{1}, 1, 1,
		},
		{
			"the box by id: the longer name still loses it",
			true,
			[]cm.ProductList{{IDProduct: 1, Name: "Booster Box"}, {IDProduct: 2, Name: "Booster Box (18 Boosters)"}},
			[]int{1}, 0, 1,
		},
		{
			"the longer name by id is never pruned",
			true,
			[]cm.ProductList{{IDProduct: 1, Name: "Booster Box (18 Boosters)"}, {IDProduct: 2, Name: "Booster Box"}},
			[]int{1, 2}, 1, 0,
		},
	} {
		b := sealedPlaceBackend(t, tt.byID)
		mkm := &Sealed{backend: b, gameID: cm.GameRiftbound}
		placed := mkm.placeProducts(tt.catalog, fallbackLookup(b, nil, tt.catalog))

		got := slices.Sorted(maps.Keys(placed.productMap))
		if !slices.Equal(got, tt.kept) {
			t.Errorf("%s: kept %v, want %v", tt.desc, got, tt.kept)
		}
		if !slices.Equal(placed.productIDs, tt.kept) {
			t.Errorf("%s: left %v to be priced, want %v", tt.desc, placed.productIDs, tt.kept)
		}
		if placed.resolved != tt.resolved {
			t.Errorf("%s: resolved %d by name, want %d", tt.desc, placed.resolved, tt.resolved)
		}
		subsumed := placed.dropped["names a product built on another"]
		if subsumed != tt.subsumedDrop {
			t.Errorf("%s: dropped %d as subsumed, want %d", tt.desc, subsumed, tt.subsumedDrop)
		}
	}
}

// TestPlaceProductsTallies pins what the loop counts: a foreign printing is
// a drop of its own, and a print run's answer is placed without counting as
// resolved by name or being asked about by the pruning.
func TestPlaceProductsTallies(t *testing.T) {
	b := sealedFallbackBackend()
	catalog := []cm.ProductList{
		{IDProduct: 3, Name: "Paramount War Booster Box", CategoryName: "One Piece Booster Boxes"},
		{IDProduct: 7, Name: "Pillars of Strength Booster Box (Pre-Errata)", CategoryName: "One Piece Booster Boxes"},
		{IDProduct: 8, Name: "Pillars of Strength Booster Box", CategoryName: "One Piece Booster Boxes"},
		{IDProduct: 9, Name: "Paramount War Booster Box (Asia Region Legal)", CategoryName: "One Piece Booster Boxes"},
		{IDProduct: 11, Name: "Romance Dawn Booster Pack (Japanese)", CategoryName: "One Piece Boosters"},
	}
	lookup := fallbackLookup(b, nil, catalog)
	lookup.byID = map[int][]string{}
	mkm := &Sealed{backend: b, gameID: cm.GameOnePiece}
	placed := mkm.placeProducts(catalog, lookup)

	want := map[string]int{"foreign printing": 2}
	if !maps.Equal(placed.dropped, want) {
		t.Errorf("dropped %v, want %v", placed.dropped, want)
	}
	if placed.resolved != 1 {
		t.Errorf("resolved %d by name, want 1", placed.resolved)
	}
	wantNamed := map[string][]int{"op02-box": {3}}
	if !maps.EqualFunc(placed.named, wantNamed, slices.Equal) {
		t.Errorf("named %v, want %v", placed.named, wantNamed)
	}
	for id, uuid := range map[int]string{3: "op02-box", 7: "op03-wave1", 8: "op03-wave2"} {
		if !slices.Equal(placed.productMap[id], []string{uuid}) {
			t.Errorf("%d placed on %v, want %s", id, placed.productMap[id], uuid)
		}
	}
	if !slices.Equal(placed.productIDs, []int{3, 7, 8}) {
		t.Errorf("left %v to be priced, want [3 7 8]", placed.productIDs)
	}
}

// TestBridgedProducts pins that the bridge holds only what it will settle:
// a Cardmarket id the datastore already carries is the id's to settle.
func TestBridgedProducts(t *testing.T) {
	tcgMap := map[int][]string{10: {"op01-box"}, 5: {"op01-pack"}}
	bridge := map[int]int{1: 10, 2: 5, 3: 99}

	got := bridgedProducts(bridge, tcgMap, map[int][]string{})
	want := map[int][]string{1: {"op01-box"}, 2: {"op01-pack"}}
	if !maps.EqualFunc(got, want, slices.Equal) {
		t.Errorf("without ids: bridged %v, want %v", got, want)
	}

	got = bridgedProducts(bridge, tcgMap, map[int][]string{1: {"op01-box"}})
	want = map[int][]string{2: {"op01-pack"}}
	if !maps.EqualFunc(got, want, slices.Equal) {
		t.Errorf("beside an id: bridged %v, want %v", got, want)
	}
}
