package magic

import (
	"encoding/json"
	"testing"
)

// A card entry of a sealed product counts toward the one finish its flags
// name, which is how MTGJSON buckets it into sourceProducts. The entries
// are spelled the way MTGJSON publishes them: an etched one carries no foil
// flag, as with the Etched bonus card of a Secret Lair drop.
func TestContentsContainCardFinish(t *testing.T) {
	const uuid = "fb7fcf6c-e9b4-5ac2-be9f-4e55b126f4f5"
	for _, tc := range []struct {
		entry string
		want  string
	}{
		{`{"finishes":["nonfoil"],"uuid":"` + uuid + `"}`, "nonfoil"},
		{`{"finishes":["foil"],"foil":true,"uuid":"` + uuid + `"}`, "foil"},
		{`{"etched":true,"finishes":["etched"],"uuid":"` + uuid + `"}`, "etched"},
	} {
		var contents map[string][]SealedContent
		err := json.Unmarshal([]byte(`{"card":[`+tc.entry+`]}`), &contents)
		if err != nil {
			t.Fatal(err)
		}
		for _, finish := range []string{"nonfoil", "foil", "etched"} {
			got := contentsContainCard(nil, contents, uuid, finish)
			if got != (finish == tc.want) {
				t.Errorf("%s as %s: got %v", tc.entry, finish, got)
			}
		}
	}
}
