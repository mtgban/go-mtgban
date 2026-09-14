package mtgmatcher

import (
	"slices"
	"testing"
)

// TestIndexSetUUIDs guards the bucketing itself, independent of any one
// game's loader: every non-Magic loader once built AllUUIDs and UUIDs but
// never called this, leaving SetUUIDs nil and GetUUIDsInSet silently
// answering empty for every set of every game but Magic (whose loader
// built the same buckets by hand). Sealed uuids are excluded the same way
// the sealed counterpart, SetSealedUUIDs, keeps to AddSealed's own uuids: a
// uuid with no matching CardObject (a stale AllUUIDs entry) is skipped
// rather than panicking.
func TestIndexSetUUIDs(t *testing.T) {
	var b Backend
	b.UUIDs = map[string]*CardObject{
		"a1": {Card: Card{UUID: "a1", SetCode: "AAA"}},
		"a2": {Card: Card{UUID: "a2", SetCode: "AAA"}},
		"b1": {Card: Card{UUID: "b1", SetCode: "BBB"}},
		"s1": {Card: Card{UUID: "s1", SetCode: "AAA"}, Sealed: true},
	}
	// AllUUIDs deliberately unsorted and out of set order, and "stale"
	// names a uuid IndexSetUUIDs must skip rather than crash on.
	b.AllUUIDs = []string{"a2", "b1", "a1", "stale"}

	b.IndexSetUUIDs()

	if got := b.SetUUIDs["AAA"]; !slices.Equal(got, []string{"a1", "a2"}) {
		t.Errorf("SetUUIDs[AAA] = %v, want [a1 a2] (sorted)", got)
	}
	if got := b.SetUUIDs["BBB"]; !slices.Equal(got, []string{"b1"}) {
		t.Errorf("SetUUIDs[BBB] = %v, want [b1]", got)
	}
	if got, found := b.SetUUIDs["CCC"]; found {
		t.Errorf("SetUUIDs[CCC] = %v, want no entry", got)
	}
	// s1 is sealed and never listed in AllUUIDs, so it must not appear in
	// any bucket - GetUUIDsInSet's callers assume this index holds cards
	// alone, mirroring the doc comment on IndexSetUUIDs.
	for code, uuids := range b.SetUUIDs {
		if slices.Contains(uuids, "s1") {
			t.Errorf("SetUUIDs[%s] contains the sealed uuid s1", code)
		}
	}
}
