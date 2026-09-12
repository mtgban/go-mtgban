package mtgmatcher

import (
	"sync"
	"testing"
)

func TestGlobalDatastoreBeforeFirstPublication(t *testing.T) {
	previous := defaultBackend.Load()
	t.Cleanup(func() { defaultBackend.Store(previous) })
	defaultBackend.Store(nil)
	if _, err := GetUUID("anything"); err != ErrDatastoreEmpty {
		t.Fatalf("GetUUID before publication = %v", err)
	}
	if _, err := Match(&InputCard{Name: "Anything"}); err != ErrDatastoreEmpty {
		t.Fatalf("Match before publication = %v", err)
	}
	if len(AllNames("canonical", false)) != 0 || len(GetUUIDsInSet("A")) != 0 {
		t.Fatal("uninitialized indexes were not empty")
	}
}

func TestGlobalDatastorePinsSnapshot(t *testing.T) {
	previous := GlobalDatastore()
	t.Cleanup(func() { SetGlobalDatastore(previous) })

	first := &Backend{UUIDs: map[string]*CardObject{"first": {Card: Card{UUID: "first"}}}}
	SetGlobalDatastore(first)
	pinned := GlobalDatastore()
	if first.sealedIdx != nil {
		t.Fatal("publication mutated the caller's backend")
	}
	// Reassigning a field on the caller's value or a captured copy must not
	// replace that field on the published snapshot. Nested data stays shared.
	first.UUIDs = nil
	captured := GlobalDatastore()
	captured.UUIDs = nil
	if _, err := GetUUID("first"); err != nil {
		t.Fatalf("caller changed the published snapshot: %v", err)
	}
	SetGlobalDatastore(&Backend{})
	if _, err := GetUUID("first"); err != ErrDatastoreEmpty {
		t.Fatalf("global lookup after replacement = %v", err)
	}
	if _, err := pinned.GetUUID("first"); err != nil {
		t.Fatalf("captured snapshot changed after replacement: %v", err)
	}
}

func TestGlobalDatastoreConcurrentPublication(t *testing.T) {
	previous := GlobalDatastore()
	t.Cleanup(func() { SetGlobalDatastore(previous) })

	backends := []*Backend{}
	for _, id := range []string{"first", "second"} {
		backends = append(backends, &Backend{
			AllUUIDs:            []string{id},
			UUIDs:               map[string]*CardObject{id: {Card: Card{UUID: id}}},
			ExternalIdentifiers: map[string]map[string]string{IDSpaceTCGplayer: {"123": id}},
		})
	}
	SetGlobalDatastore(backends[0])
	var wg sync.WaitGroup
	for worker := 0; worker < 4; worker++ {
		wg.Go(func() {
			for i := 0; i < 500; i++ {
				// Concurrent publishers may share their input. Publication
				// must neither mutate it nor expose mixed index generations.
				SetGlobalDatastore(backends[i%2])
				b := GlobalDatastore()
				id := b.GetUUIDs()[0]
				if _, err := b.GetUUID(id); err != nil {
					t.Errorf("mixed snapshot: %q: %v", id, err)
				}
				got, err := MatchID("123")
				if err != nil || (got != "first" && got != "second") {
					t.Errorf("global MatchID during publication = %q, %v", got, err)
				}
			}
		})
	}
	wg.Wait()
}
