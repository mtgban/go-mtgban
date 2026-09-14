package pokemon

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// envelopeFixture is a two-card, one-set cut used only to pin that Load
// reads the {"meta":...,"data":...} envelope the same as the legacy shape;
// it is not meant to exercise matching rules the other tests already cover.
const envelopeFixture = `{
	"game": "pokemon",
	"sets": {
		"BASE": {"name": "Base Set", "releaseDate": "1999-01-09"}
	},
	"cards": [
		{"externalLinks": {"tcgPlayerId": 100001}, "finish": "Holofoil", "id": "base-4_100001_holofoil", "image": "x", "name": "Charizard", "number": "4", "rarity": "Rare Holo", "setCode": "BASE", "type": "Fire"},
		{"externalLinks": {"tcgPlayerId": 100002}, "finish": "Normal", "id": "base-58_100002", "image": "x", "name": "Pikachu", "number": "58", "rarity": "Common", "setCode": "BASE", "type": "Lightning"}
	]
}`

// TestLoadAcceptsMetaDataEnvelope pins that Load reads a datastore wrapped
// in the {"meta":...,"data":...} envelope into the same Backend it reads
// from the legacy flat shape. The wrapped form strips "game" out of data
// The payload keeps every key it had, "game" included: the envelope moves
// the document under "data" and adds a meta beside it, and nothing about
// the document itself changes.
func TestLoadAcceptsMetaDataEnvelope(t *testing.T) {
	legacy, err := Load(strings.NewReader(envelopeFixture))
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(envelopeFixture), &payload); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := json.Marshal(map[string]any{
		"meta": map[string]any{"date": "2026-09-14", "version": "1"},
		"data": json.RawMessage(data),
	})
	if err != nil {
		t.Fatal(err)
	}

	wrapped, err := Load(strings.NewReader(string(envelope)))
	if err != nil {
		t.Fatal(err)
	}

	assertEquivalentBackends(t, legacy, wrapped)
}

// assertEquivalentBackends fails t unless want and got carry the same sets,
// sealed uuids and card data - the comparison Load's two accepted shapes
// must never disagree on.
func assertEquivalentBackends(t *testing.T, want, got *mtgmatcher.Backend) {
	t.Helper()

	wantSets := append([]string(nil), want.AllSets...)
	gotSets := append([]string(nil), got.AllSets...)
	sort.Strings(wantSets)
	sort.Strings(gotSets)
	if !reflect.DeepEqual(wantSets, gotSets) {
		t.Fatalf("set codes = %v, want %v", gotSets, wantSets)
	}

	wantSealed := append([]string(nil), want.AllSealedUUIDs...)
	gotSealed := append([]string(nil), got.AllSealedUUIDs...)
	sort.Strings(wantSealed)
	sort.Strings(gotSealed)
	if !reflect.DeepEqual(wantSealed, gotSealed) {
		t.Fatalf("sealed uuids = %v, want %v", gotSealed, wantSealed)
	}

	var wantUUIDs, gotUUIDs []string
	for uuid := range want.UUIDs {
		wantUUIDs = append(wantUUIDs, uuid)
	}
	for uuid := range got.UUIDs {
		gotUUIDs = append(gotUUIDs, uuid)
	}
	sort.Strings(wantUUIDs)
	sort.Strings(gotUUIDs)
	if !reflect.DeepEqual(wantUUIDs, gotUUIDs) {
		t.Fatalf("card uuids = %v, want %v", gotUUIDs, wantUUIDs)
	}

	for _, uuid := range wantUUIDs {
		if !reflect.DeepEqual(want.UUIDs[uuid], got.UUIDs[uuid]) {
			t.Errorf("card %s = %+v, want %+v", uuid, got.UUIDs[uuid], want.UUIDs[uuid])
		}
	}
}
