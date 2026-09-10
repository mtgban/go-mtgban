package lorcana

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
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

	// The uuids are the datastore's to spell, and a card sold in two
	// finishes is still one card to Match: folding its finishes by the
	// shape of their uuids aliased every such card the moment the builder
	// spelled them another way.
	var twoFinish *mtgmatcher.CardObject
	for _, uuid := range b.AllUUIDs {
		co := b.UUIDs[uuid]
		if co.Sealed || len(co.Finishes) < 2 {
			continue
		}
		twoFinish = co
		break
	}
	if twoFinish == nil {
		t.Fatal("no card is sold in two finishes, so the fold is not being tested")
	}
	in := mtgmatcher.InputCard{Name: twoFinish.Name, Edition: twoFinish.Edition, Variation: twoFinish.Number}
	got, err := b.Match(&in)
	if err != nil {
		t.Fatalf("Match(%v) = %v, want the card's plain printing", in, err)
	}
	if got != twoFinish.FoilUUIDs[mtgmatcher.FinishNonfoil] {
		t.Errorf("Match(%v) = %s, want %s", in, got, twoFinish.FoilUUIDs[mtgmatcher.FinishNonfoil])
	}
}

// stampPrintingIDs renames the uuid of every printing a card carries, to one
// the spelling below could not arrive at, and answers the uuids it wrote.
func stampPrintingIDs(t *testing.T, data []byte) ([]byte, map[string]bool) {
	t.Helper()
	want := map[string]bool{}
	out := restamp(t, data, func(id int, finish string) string {
		uuid := fmt.Sprintf("published-%d-%s", id, finish)
		want[uuid] = true
		return uuid
	})
	return out, want
}

// restamp rewrites every printing's uuid, in whichever shape the datastore
// publishes them: printings[] carries a finish and its uuid together, and
// printingIds is the map a datastore published before it. Both are stamped
// so this pins the invariant against either, and nothing here has to know
// which one it was handed.
func restamp(t *testing.T, data []byte, name func(id int, finish string) string) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	rows, ok := doc["cards"].([]any)
	if !ok {
		t.Fatal("the payload holds no cards")
	}
	for _, item := range rows {
		row, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("a card is %T, not an object", item)
		}
		id, ok := row["id"].(float64)
		if !ok {
			t.Fatalf("a card id is %T, not a number", row["id"])
		}
		if printings, listed := row["printings"].([]any); listed {
			for _, raw := range printings {
				printing, ok := raw.(map[string]any)
				if !ok {
					t.Fatalf("a printing is %T, not an object", raw)
				}
				finish, ok := printing["finish"].(string)
				if !ok || finish == "" {
					t.Fatalf("a printing of card %d names no finish", int(id))
				}
				printing["id"] = name(int(id), finish)
			}
			continue
		}
		ids, listed := row["printingIds"].(map[string]any)
		if !listed || len(ids) == 0 {
			continue
		}
		for finish := range ids {
			ids[finish] = name(int(id), finish)
		}
	}
	doc["cards"] = rows
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
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

// stampVendorPrintingIDs renames every printing's uuid to one naming the
// finish it should be stored under, keyed the way the builder keys it - by
// the TCGplayer printing name - and answers each against that finish.
func stampVendorPrintingIDs(t *testing.T, data []byte) ([]byte, map[string]string) {
	t.Helper()
	want := map[string]string{}
	out := restamp(t, data, func(id int, printing string) string {
		finish := canonicalFinish(printing)
		// One finish named twice is one printing, stored once.
		uuid := fmt.Sprintf("vendor-%d-%s", id, finish)
		want[uuid] = finish
		return uuid
	})
	return out, want
}
