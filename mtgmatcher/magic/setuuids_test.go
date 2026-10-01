package magic

import "testing"

// TestSetUUIDsIndexesRealCards pins the index GetUUIDsInSet answers from,
// which the site's edition-only searches (s:CODE) seed their candidates
// from alone: an empty one answers nothing for every set.
func TestSetUUIDsIndexesRealCards(t *testing.T) {
	realDatastore(t)

	if len(testBackend.SetUUIDs) == 0 {
		t.Fatal("SetUUIDs is empty, or the datastore has no cards")
	}
	checked := 0
	for _, code := range testBackend.AllSets {
		for _, uuid := range testBackend.SetUUIDs[code] {
			co, err := testBackend.GetUUID(uuid)
			if err != nil {
				t.Errorf("SetUUIDs[%s] holds %s, which GetUUID does not know: %v", code, uuid, err)
				continue
			}
			if co.SetCode != code {
				t.Errorf("SetUUIDs[%s] holds %s, whose SetCode is %q", code, uuid, co.SetCode)
			}
			if co.Sealed {
				t.Errorf("SetUUIDs[%s] holds the sealed uuid %s", code, uuid)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no uuid in any SetUUIDs bucket; the index is present but empty")
	}
}
