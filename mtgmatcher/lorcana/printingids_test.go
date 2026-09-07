package lorcana

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// TestPublishedPrintingIDsWin pins that the datastore's own uuids are used
// where it publishes them, rather than being spelled from the foil type
// here. 3,200 of this game's uuids are reached by spelling a foil type
// through canonicalFinish, and a uuid is what a price is keyed on: if this
// package spelled one and the builder published another, the two would
// disagree silently, since a uuid nobody stored resolves to nothing rather
// than erroring.
func TestPublishedPrintingIDsWin(t *testing.T) {
	path := os.Getenv("LORCANA_PATH")
	if path == "" {
		t.Skip("LORCANA_PATH not set; skipping Lorcana matcher suite")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	stamped, want := stampPrintingIDs(t, data)
	b, err := Load(bytes.NewReader(stamped))
	if err != nil {
		t.Fatal(err)
	}
	if len(want) == 0 {
		t.Fatal("no printing was stamped, so nothing is being tested")
	}
	for uuid := range want {
		if _, err := b.GetUUID(uuid); err != nil {
			t.Fatalf("published uuid %s was not stored: %v", uuid, err)
		}
	}
	t.Logf("%d published uuids honoured", len(want))
}

// stampPrintingIDs writes a printingIds map onto every card that lists a
// foil type, naming a uuid the spelling below could not arrive at, and
// answers the uuids it wrote.
func stampPrintingIDs(t *testing.T, data []byte) ([]byte, map[string]bool) {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	rows, ok := doc["cards"].([]any)
	if !ok {
		t.Fatal("the payload holds no cards")
	}
	want := map[string]bool{}
	for _, item := range rows {
		row, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("a card is %T, not an object", item)
		}
		id, ok := row["id"].(float64)
		if !ok {
			t.Fatalf("a card id is %T, not a number", row["id"])
		}
		foilTypes, listed := row["foilTypes"].([]any)
		if !listed || len(foilTypes) == 0 {
			continue
		}
		ids := map[string]any{}
		for _, raw := range foilTypes {
			foilType, ok := raw.(string)
			if !ok {
				t.Fatalf("a foil type is %T, not a string", raw)
			}
			// The same sub-type listed twice is one printing, and the
			// loader stores it once, so it is expected once.
			uuid := fmt.Sprintf("published-%d-%s", int(id), foilType)
			ids[foilType] = uuid
			want[uuid] = true
		}
		row["printingIds"] = ids
	}
	doc["cards"] = rows
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out, want
}
