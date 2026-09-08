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

// TestPrintingIDsNamedTheVendorsWay pins the crossing between the two
// vocabularies. The builder names a finish the way TCGplayer prices it
// ("Normal", "Cold Foil", "Holofoil"); this package spells finishes its own
// way. Read the keys without crossing and every one of them files a finish
// nothing ever asks for, stranding the uuid it names - silently, since a
// uuid nobody stored resolves to nothing rather than erroring.
func TestPrintingIDsNamedTheVendorsWay(t *testing.T) {
	path := os.Getenv("LORCANA_PATH")
	if path == "" {
		t.Skip("LORCANA_PATH not set; skipping Lorcana matcher suite")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	stamped, want := stampVendorPrintingIDs(t, data)
	b, err := Load(bytes.NewReader(stamped))
	if err != nil {
		t.Fatal(err)
	}
	if len(want) == 0 {
		t.Fatal("no printing was stamped, so nothing is being tested")
	}
	for uuid, finish := range want {
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatalf("uuid %s published under a TCGplayer name was not stored: %v", uuid, err)
		}
		if co.Finish != finish {
			t.Errorf("uuid %s carries finish %q, want %q", uuid, co.Finish, finish)
		}
	}
	t.Logf("%d uuids keyed by TCGplayer's own names reached their finish", len(want))
}

// stampVendorPrintingIDs writes a printingIds map keyed the way the builder
// keys it - by the TCGplayer printing the card's own externalLinks name -
// and answers each uuid it wrote against the finish it should be stored
// under.
func stampVendorPrintingIDs(t *testing.T, data []byte) ([]byte, map[string]string) {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	rows, ok := doc["cards"].([]any)
	if !ok {
		t.Fatal("the payload holds no cards")
	}
	want := map[string]string{}
	for _, item := range rows {
		row, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("a card is %T, not an object", item)
		}
		id, ok := row["id"].(float64)
		if !ok {
			t.Fatalf("a card id is %T, not a number", row["id"])
		}
		links, ok := row["externalLinks"].(map[string]any)
		if !ok {
			continue
		}
		printings, listed := links["tcgPrintings"].([]any)
		if !listed || len(printings) == 0 {
			continue
		}
		ids := map[string]any{}
		for _, raw := range printings {
			name, ok := raw.(string)
			if !ok {
				t.Fatalf("a TCGplayer printing is %T, not a string", raw)
			}
			finish := canonicalFinish(name)
			// One finish named twice is one printing, stored once.
			uuid := fmt.Sprintf("vendor-%d-%s", int(id), finish)
			ids[name] = uuid
			want[uuid] = finish
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
