package gundam

import "testing"

// TestSetUUIDsIndexesRealCards guards the regression this loader once had:
// SetUUIDs stayed nil because nothing populated it, so GetUUIDsInSet always
// answered empty and mtgban-website's edition-only searches (s:CODE) - which
// seed candidates from this index alone when there is no text to search -
// came back empty for every set of this game.
func TestSetUUIDsIndexesRealCards(t *testing.T) {
	b := loadBackend(t)

	if len(b.SetUUIDs) == 0 {
		t.Fatal("SetUUIDs is empty; IndexSetUUIDs was not called, or the datastore has no cards")
	}
	checked := 0
	for _, code := range b.AllSets {
		for _, uuid := range b.SetUUIDs[code] {
			co, err := b.GetUUID(uuid)
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
