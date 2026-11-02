package magic

import (
	"encoding/json"
	"maps"
	"testing"
)

// A card entry of a sealed product counts toward the one finish its flags
// name, which is how MTGJSON buckets it into sourceProducts. The entries
// are spelled the way MTGJSON publishes them: an etched one carries no foil
// flag, as with the Etched bonus card of a Secret Lair drop.
func TestSealedCardFinish(t *testing.T) {
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
		held := map[sealedCard]bool{}
		addContents(held, nil, "product", contents)
		for _, finish := range []string{"nonfoil", "foil", "etched"} {
			got := held[sealedCard{"product", uuid, finish}]
			if got != (finish == tc.want) {
				t.Errorf("%s as %s: got %v", tc.entry, finish, got)
			}
		}
	}
}

// A pack holds its foil sheets' cards as foil and etched, its other sheets'
// as nonfoil; a deck holds each card in the finish its flags name.
func TestSealedPackAndDeckFinish(t *testing.T) {
	var set Set
	err := json.Unmarshal([]byte(`{
		"booster": {"draft": {"sheets": {
			"foil": {"foil": true, "cards": {"foil-uuid": 1}},
			"common": {"cards": {"plain-uuid": 1}}
		}}},
		"decks": [{"name": "Deck", "mainBoard": [{"isEtched": true, "uuid": "etched-uuid"}]}]
	}`), &set)
	if err != nil {
		t.Fatal(err)
	}
	var contents map[string][]SealedContent
	err = json.Unmarshal([]byte(`{
		"pack": [{"set": "abc", "code": "draft"}],
		"deck": [{"set": "abc", "name": "Deck"}]
	}`), &contents)
	if err != nil {
		t.Fatal(err)
	}
	held := map[sealedCard]bool{}
	addContents(held, map[string]*Set{"ABC": &set}, "product", contents)
	want := map[sealedCard]bool{
		{"product", "foil-uuid", "foil"}:     true,
		{"product", "foil-uuid", "etched"}:   true,
		{"product", "plain-uuid", "nonfoil"}: true,
		{"product", "etched-uuid", "etched"}: true,
	}
	if !maps.Equal(held, want) {
		t.Errorf("got %v, want %v", held, want)
	}
}
