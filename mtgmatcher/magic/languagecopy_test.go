package magic

import (
	"testing"
)

// A Secret Lair number listed for a Japanese copy brings its starred sibling.
func TestSLDJapaneseCopies(t *testing.T) {
	realDatastore(t)

	for _, uuid := range testBackend.SetUUIDs["SLD"] {
		co, err := testBackend.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		if co.Number == "1858★jpn" {
			return
		}
	}
	t.Error("SLD 1858★ has no Japanese copy")
}
