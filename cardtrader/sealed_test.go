package cardtrader

import (
	"os"
	"reflect"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestBuildProductMap pins the sealed product-map fallbacks blueprint by
// blueprint: the datastore's cardtraderId index answers first, the
// TCGplayer id bridges a blueprint that index does not answer even when
// it answers others, and the name pass stays off for Magic while staying
// on for every other game. The fixtures are drawn from the datastore so
// the test holds across its releases.
func TestBuildProductMap(t *testing.T) {
	datastorePath := os.Getenv("ALLPRINTINGS5_PATH")
	if datastorePath == "" {
		t.Skip("Need ALLPRINTINGS5_PATH variable set to run this test")
	}
	reader, err := os.Open(datastorePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	backend, err := magic.Load(reader)
	if err != nil {
		t.Fatal(err)
	}
	mtgmatcher.SetGlobalDatastore(backend)

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
