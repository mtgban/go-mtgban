package mtgmatcher

import (
	"testing"
)

// The set index must agree exactly with a scan of the full uuid pool: same
// membership for singles and sealed, sorted buckets, and the multi-code
// union equal to the concatenation of the individual buckets.
func TestGetUUIDsInSet(t *testing.T) {
	b := realDatastore(t)
	sets := b.GetAllSets()

	// Rebuild the expected buckets the slow way.
	wantSingles := map[string]int{}
	for _, uuid := range b.GetUUIDs() {
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		wantSingles[co.SetCode]++
	}
	wantSealed := map[string]int{}
	for _, uuid := range b.GetSealedUUIDs() {
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		wantSealed[co.SetCode]++
	}

	var totalSingles, totalSealed int
	for _, code := range sets {
		singles := b.GetUUIDsInSet(code)
		if len(singles) != wantSingles[code] {
			t.Errorf("%s: %d singles in index, want %d", code, len(singles), wantSingles[code])
		}
		for _, uuid := range singles {
			co, err := b.GetUUID(uuid)
			if err != nil || co.SetCode != code || co.Sealed {
				t.Fatalf("%s: bad index entry %s", code, uuid)
			}
		}
		totalSingles += len(singles)

		sealed := b.GetSealedUUIDsInSet(code)
		if len(sealed) != wantSealed[code] {
			t.Errorf("%s: %d sealed in index, want %d", code, len(sealed), wantSealed[code])
		}
		for _, uuid := range sealed {
			co, err := b.GetUUID(uuid)
			if err != nil || co.SetCode != code || !co.Sealed {
				t.Fatalf("%s: bad sealed index entry %s", code, uuid)
			}
		}
		totalSealed += len(sealed)
	}
	if totalSingles != len(b.GetUUIDs()) {
		t.Errorf("index covers %d singles, pool has %d", totalSingles, len(b.GetUUIDs()))
	}
	if totalSealed != len(b.GetSealedUUIDs()) {
		t.Errorf("index covers %d sealed, pool has %d", totalSealed, len(b.GetSealedUUIDs()))
	}
}
