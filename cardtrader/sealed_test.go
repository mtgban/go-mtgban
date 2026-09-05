package cardtrader

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestMain loads the datastore once for the whole package, where a run
// carries one: most of this package reads no cards, and the games this
// scraper is scheduled for run under jobs holding their own datastore.
func TestMain(m *testing.M) {
	if path := os.Getenv("ALLPRINTINGS5_PATH"); path != "" {
		reader, err := datastore.Open(path)
		if err != nil {
			log.Fatalln(err)
		}
		backend, err := magic.Load(reader)
		reader.Close()
		if err != nil {
			log.Fatalln(err)
		}
		mtgmatcher.SetGlobalDatastore(backend)
	}
	os.Exit(m.Run())
}

// TestBuildProductMap pins the sealed product-map fallbacks blueprint by
// blueprint: the datastore's cardtraderId index answers first, the
// TCGplayer id bridges a blueprint that index does not answer even when
// it answers others, and the name pass stays off for Magic while staying
// on for every other game. The fixtures are drawn from the datastore so
// the test holds across its releases.
func TestBuildProductMap(t *testing.T) {
	if len(mtgmatcher.GetAllSets()) == 0 {
		t.Skip("ALLPRINTINGS5_PATH not set; skipping the datastore-backed cases")
	}

	ctMap := mtgmatcher.BuildSealedProductMap("cardtraderId")
	tcgMap := mtgmatcher.BuildSealedProductMap("tcgplayerProductId")
	if len(ctMap) == 0 || len(tcgMap) == 0 {
		t.Fatal("datastore carries no sealed identifiers")
	}

	// A cardtraderId-indexed product, and a TCGplayer-indexed product
	// that names a different one, so the two answers are tellable apart.
	var ctID, tcgID int
	var ctUUIDs, tcgUUIDs []string
	for id, uuids := range ctMap {
		ctID, ctUUIDs = id, uuids
		break
	}
	for id, uuids := range tcgMap {
		if !reflect.DeepEqual(uuids, ctUUIDs) {
			tcgID, tcgUUIDs = id, uuids
			break
		}
	}
	if tcgID == 0 {
		t.Fatal("no distinct tcg-indexed product found")
	}

	// A name the resolver answers, to probe the name pass gate with.
	var resolvableName string
	var resolvedUUID string
	for _, uuids := range ctMap {
		co, err := mtgmatcher.GetUUID(uuids[0])
		if err != nil {
			continue
		}
		// A language variant would be dropped before the resolver runs,
		// failing the assertion for the wrong reason.
		if mtgmatcher.SealedIsLanguageVariant(co.Name) {
			continue
		}
		uuid, err := mtgmatcher.ResolveSealed(co.Name)
		if err == nil {
			resolvableName = co.Name
			resolvedUUID = uuid
			break
		}
	}
	if resolvableName == "" {
		t.Fatal("no name-resolvable sealed product found")
	}

	// An id the cardtraderId index does not answer.
	orphanID := 1 << 30
	for _, found := ctMap[orphanID]; found; _, found = ctMap[orphanID] {
		orphanID++
	}

	blueprints := map[int]*Blueprint{
		// Already answered: the conflicting TCGplayer id must not win.
		ctID: {ID: ctID, TCGplayerID: tcgID},
		// Unanswered with a usable TCGplayer id: the bridge must fire
		// even though the cardtraderId index answers other blueprints.
		orphanID: {ID: orphanID, TCGplayerID: tcgID},
		// Unanswered, no id, a resolvable name: the name pass decides.
		orphanID + 1: {ID: orphanID + 1, Name: resolvableName},
	}

	ct := &Sealed{gameID: GameMagic}
	productMap := ct.buildProductMap(blueprints)
	if !reflect.DeepEqual(productMap[ctID], ctUUIDs) {
		t.Errorf("answered blueprint overridden: got %v, want %v", productMap[ctID], ctUUIDs)
	}
	if !reflect.DeepEqual(productMap[orphanID], tcgUUIDs) {
		t.Errorf("bridge did not fire: got %v, want %v", productMap[orphanID], tcgUUIDs)
	}
	if uuids, found := productMap[orphanID+1]; found {
		t.Errorf("magic name pass fired: got %v", uuids)
	}

	ct = &Sealed{gameID: GamePokemon}
	productMap = ct.buildProductMap(blueprints)
	if !reflect.DeepEqual(productMap[orphanID+1], []string{resolvedUUID}) {
		t.Errorf("name pass did not fire: got %v, want %v", productMap[orphanID+1], []string{resolvedUUID})
	}
}

// sealedRunsBackend is a set carrying one product in both of its print runs,
// which is how a catalog names a Flesh and Blood or a YuGiOh set. CardTrader
// names both blueprints the same and shelves them apart.
func sealedRunsBackend() *mtgmatcher.Backend {
	backend := &mtgmatcher.Backend{
		UUIDs:          map[string]*mtgmatcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*mtgmatcher.Set{
			"CRU": {Name: "Crucible of War", Code: "CRU"},
		},
	}
	backend.AddSealed("cru-1e", "Crucible of War Booster Box [1st Edition]", "CRU", "", 0)
	backend.AddSealed("cru-unl", "Crucible of War Booster Box [Unlimited Edition]", "CRU", "", 0)
	backend.SortSealed()
	backend.IndexSets()
	return backend
}

// TestBuildProductMapReadsExpansion pins that the expansion a blueprint is
// filed under reaches the resolver. It is the only thing CardTrader publishes
// that says which print run a blueprint is - both runs carry the same product
// name and no TCGplayer id - so throwing it away leaves the two of them
// answering to one name and neither of them priced.
func TestBuildProductMapReadsExpansion(t *testing.T) {
	mtgmatcher.SetGlobalDatastore(sealedRunsBackend())

	blueprint := &Blueprint{ID: 1, Name: "Crucible of War Booster Box"}
	blueprints := map[int]*Blueprint{1: blueprint}
	ct := &Sealed{gameID: GameFleshAndBlood}

	if uuids, found := ct.buildProductMap(blueprints)[1]; found {
		t.Errorf("a blueprint naming no run resolved to %v", uuids)
	}

	blueprint.Expansion.Name = "Crucible of War - Unlimited"
	if got := ct.buildProductMap(blueprints)[1]; !reflect.DeepEqual(got, []string{"cru-unl"}) {
		t.Errorf("shelved under the unlimited run: got %v, want [cru-unl]", got)
	}

	blueprint.Expansion.Name = "Crucible of War - First"
	if got := ct.buildProductMap(blueprints)[1]; !reflect.DeepEqual(got, []string{"cru-1e"}) {
		t.Errorf("shelved under the first run: got %v, want [cru-1e]", got)
	}
}

// TestBuildProductMapNamesLanguageDrops pins that a blueprint dropped for
// being a non-English printing says which one it was, the way every other
// drop in this pass does. A count with no names behind it cannot be checked
// against the catalog.
func TestBuildProductMapNamesLanguageDrops(t *testing.T) {
	mtgmatcher.SetGlobalDatastore(sealedRunsBackend())

	var logged []string
	ct := &Sealed{gameID: GameFleshAndBlood}
	ct.LogCallback = func(format string, a ...any) {
		logged = append(logged, fmt.Sprintf(format, a...))
	}
	ct.buildProductMap(map[int]*Blueprint{
		1: {ID: 1, Name: "Crucible of War Japanese Booster Box"},
	})

	var named bool
	for _, line := range logged {
		if strings.Contains(line, "Crucible of War Japanese Booster Box") &&
			strings.Contains(line, "language variant") {
			named = true
		}
	}
	if !named {
		t.Errorf("the language-variant drop named no product: %v", logged)
	}
}

// TestBuildProductMapDropsAccessories pins the accessories out of the sealed
// map. Every product a set holds shares its wording, so a deck box named for
// the collection it was packed in resolves to that collection and prices a
// ten-euro box at the collection's fifty. A blueprint CardTrader files as an
// accessory and names as one is the case there is no doubt about - and the
// category alone is not enough, because CardTrader files real sealed product
// under its accessory categories often enough to matter.
func TestBuildProductMapDropsAccessories(t *testing.T) {
	mtgmatcher.SetGlobalDatastore(sealedAccessoryBackend())

	ct := &Sealed{gameID: GameFleshAndBlood}
	for _, tt := range []struct {
		desc     string
		category int
		name     string
		want     bool
	}{
		{
			"a sleeve named as one is the accessory, not the box it names",
			CategoryFleshAndBloodSleeves, "Crucible of War Booster Box Sleeves", false,
		},
		{
			"a blueprint miscategorised as an accessory still names its product",
			CategoryFleshAndBloodSleeves, "Crucible of War Booster Box", true,
		},
		{
			"a category that sells sealed product is left alone",
			CategoryFleshAndBloodBoosterBoxes, "Crucible of War Booster Box Sleeves", true,
		},
	} {
		blueprints := map[int]*Blueprint{
			1: {ID: 1, Name: tt.name, CategoryID: tt.category},
		}
		_, found := ct.buildProductMap(blueprints)[1]
		if found != tt.want {
			t.Errorf("%s: %q resolved=%v, want %v", tt.desc, tt.name, found, tt.want)
		}
	}
}

// sealedAccessoryBackend is one product on a shelf, beside a set whose name
// donates "sleeves" to the pooled vocabulary - which is what makes the word
// free against every product of the game, and so what lets a sleeve's name
// reach the box it was printed for.
func sealedAccessoryBackend() *mtgmatcher.Backend {
	backend := &mtgmatcher.Backend{
		UUIDs:          map[string]*mtgmatcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*mtgmatcher.Set{
			"CRU": {Name: "Crucible of War", Code: "CRU"},
			"SLV": {Name: "Sleeves Promos", Code: "SLV"},
		},
	}
	backend.AddSealed("cru-box", "Crucible of War Booster Box", "CRU", "", 0)
	backend.SortSealed()
	backend.IndexSets()
	return backend
}

// TestBuildProductMapDropsSubsumed pins that two blueprints do not price one
// product. The resolver is asked about one name at a time, so a catalog that
// lists both the box and the bundle of boxes gets the box's uuid twice, and
// the bundle's price then lands on a single box. The longer name is the one
// that loses: its extra word is the thing it sells.
func TestBuildProductMapDropsSubsumed(t *testing.T) {
	mtgmatcher.SetGlobalDatastore(sealedSubsumedBackend())

	ct := &Sealed{gameID: GameFleshAndBlood}
	// The bundle alone is nothing but a spelling of the product, and
	// dropping it there would cost the product its only price.
	alone := ct.buildProductMap(map[int]*Blueprint{
		2: {ID: 2, Name: "Crucible of War Booster Box Bundle"},
	})
	if got := alone[2]; !reflect.DeepEqual(got, []string{"cru-box"}) {
		t.Fatalf("the only name reaching the product was dropped: got %v", got)
	}

	productMap := ct.buildProductMap(map[int]*Blueprint{
		1: {ID: 1, Name: "Crucible of War Booster Box"},
		2: {ID: 2, Name: "Crucible of War Booster Box Bundle"},
	})
	if got := productMap[1]; !reflect.DeepEqual(got, []string{"cru-box"}) {
		t.Errorf("the name that says the product lost it: got %v, want [cru-box]", got)
	}
	if uuids, found := productMap[2]; found {
		t.Errorf("the bundle kept the box's uuid: %v", uuids)
	}

	// Two spellings of the same words are one product said twice, and
	// neither of them is the one built on the other.
	spellings := ct.buildProductMap(map[int]*Blueprint{
		1: {ID: 1, Name: "Crucible of War Booster Box"},
		3: {ID: 3, Name: "Crucible of War Booster Booster Box"},
	})
	if len(spellings) != 2 {
		t.Errorf("one of two spellings of a product was dropped: %v", spellings)
	}

	// The set a storefront files a product under is a word it prepends
	// freely, so a name that adds nothing else is the same name again.
	shelved := ct.buildProductMap(map[int]*Blueprint{
		1: {ID: 1, Name: "Crucible of War Booster Box"},
		4: {ID: 4, Name: "Booster Box"},
	})
	if len(shelved) != 2 {
		t.Errorf("a product filed under its own set was dropped: %v", shelved)
	}
}

// sealedSubsumedBackend is one product on a shelf, beside a set whose name
// donates "bundle" to the pooled vocabulary - which is what lets a vendor say
// it about any product of the game, and so what lets the bundle's name reach
// the box at all.
func sealedSubsumedBackend() *mtgmatcher.Backend {
	backend := &mtgmatcher.Backend{
		UUIDs:          map[string]*mtgmatcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*mtgmatcher.Set{
			"CRU": {Name: "Crucible of War", Code: "CRU"},
			"BND": {Name: "Bundle Promos", Code: "BND"},
		},
	}
	backend.AddSealed("cru-box", "Booster Box", "CRU", "", 0)
	backend.SortSealed()
	backend.IndexSets()
	return backend
}

// sealedShelfCodeBackend is two sets a storefront files under a Bandai set
// code, one of them named for the number that code carries.
func sealedShelfCodeBackend() *mtgmatcher.Backend {
	backend := &mtgmatcher.Backend{
		UUIDs:          map[string]*mtgmatcher.CardObject{},
		Hashes:         map[string][]string{},
		SetSealedUUIDs: map[string][]string{},
		Sets: map[string]*mtgmatcher.Set{
			"GD05": {Name: "Freedom Ascension", Code: "GD05"},
			"ST10": {Name: "Starter Deck 10: Generation Pulse", Code: "ST10"},
		},
	}
	backend.AddSealed("gd05-case", "Freedom Ascension Booster Box Case", "GD05", "", 0)
	backend.AddSealed("st10-deck", "Starter Deck 10: Generation Pulse", "ST10", "", 0)
	backend.SortSealed()
	backend.IndexSets()
	return backend
}

// TestBuildProductMapTrimsShelfCode pins the set code a storefront opens a
// sealed name with. The resolver's vocabulary is built from set names and
// never sees a code, so the letters read as the vendor naming a different
// product and the only candidate is dropped. The trim is guarded on the
// shelf's own code, and keeps the number the catalog spells into the name.
func TestBuildProductMapTrimsShelfCode(t *testing.T) {
	mtgmatcher.SetGlobalDatastore(sealedShelfCodeBackend())
	ct := &Sealed{gameID: GameGundam}

	for _, tt := range []struct {
		name  string
		shelf string
		want  []string
	}{
		{"GD-05: Freedom Ascension Booster Box Case", "GD-05: Freedom Ascension", []string{"gd05-case"}},
		// CardTrader lowercases the code on some of its own shelves.
		{"GD-05: Freedom Ascension Booster Box Case", "Gd-05: Freedom Ascension", []string{"gd05-case"}},
		// The number is the one word telling the starter decks apart.
		{"ST-10: Generation Pulse Starter Deck", "ST-10: Generation Pulse", []string{"st10-deck"}},
		// A shelf naming no code cannot vouch for a head that merely
		// looks like one, and a code that is not its own vouches for
		// nothing.
		{"EX-01: Freedom Ascension Booster Box Case", "Gundam Promos", nil},
		{"XX-05: Freedom Ascension Booster Box Case", "GD-05: Freedom Ascension", nil},
		{"GD-06: Freedom Ascension Booster Box Case", "GD-05: Freedom Ascension", nil},
	} {
		blueprint := &Blueprint{ID: 1, Name: tt.name}
		blueprint.Expansion.Name = tt.shelf
		got := ct.buildProductMap(map[int]*Blueprint{1: blueprint})[1]
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q on %q: got %v, want %v", tt.name, tt.shelf, got, tt.want)
		}
	}
}
