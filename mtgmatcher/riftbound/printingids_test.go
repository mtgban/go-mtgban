package riftbound

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPublishedPrintingIDsWin pins that the datastore's own uuids are used
// where it publishes them, rather than being spelled from the finish name
// here. A uuid is what a price is keyed on: if this package spelled one and
// the builder published another, the two would disagree silently, since a
// uuid nobody stored resolves to nothing rather than erroring.
func TestPublishedPrintingIDsWin(t *testing.T) {
	path := os.Getenv("RIFTBOUND_PATH")
	if path == "" {
		t.Skip("RIFTBOUND_PATH not set; skipping Riftbound matcher suite")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	// Publish a uuid this package would never spell, on every printing.
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

// stampPrintingIDs renames the uuid of every printing a gallery row carries,
// to one the spelling below could not arrive at, and answers the uuids it
// wrote. It stamps whichever shape the datastore publishes - printings[]
// carries a finish and its uuid together, and printingIds is the map a
// datastore published before it - so the invariant is pinned against both
// rather than against the one on its way out.
func stampPrintingIDs(t *testing.T, data []byte) ([]byte, map[string]bool) {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	props, ok := doc["pageProps"].(map[string]any)
	if !ok {
		t.Fatal("the payload holds no pageProps")
	}
	page, ok := props["page"].(map[string]any)
	if !ok {
		t.Fatal("the payload holds no page")
	}
	blades, ok := page["blades"].([]any)
	if !ok {
		t.Fatal("the payload holds no blades")
	}
	want := map[string]bool{}
	for _, raw := range blades {
		blade, ok := raw.(map[string]any)
		if !ok || blade["type"] != "riftboundCardGallery" {
			continue
		}
		cards, ok := blade["cards"].(map[string]any)
		if !ok {
			t.Fatal("the gallery holds no cards")
		}
		items, ok := cards["items"].([]any)
		if !ok {
			t.Fatal("the gallery holds no card items")
		}
		for _, item := range items {
			row, ok := item.(map[string]any)
			if !ok {
				t.Fatalf("a card item is %T, not an object", item)
			}
			id, ok := row["id"].(string)
			if !ok {
				t.Fatalf("a card id is %T, not a string", row["id"])
			}
			if printings, listed := row["printings"].([]any); listed {
				for _, raw := range printings {
					printing, ok := raw.(map[string]any)
					if !ok {
						t.Fatalf("a printing is %T, not an object", raw)
					}
					finish, ok := printing["finish"].(string)
					if !ok || finish == "" {
						t.Fatalf("a printing of card %s names no finish", id)
					}
					uuid := "published-" + id + "-" + finish
					printing["id"] = uuid
					want[uuid] = true
				}
				continue
			}
			// Only the finishes the row is sold in are stored, so only
			// those are expected: stamping a uuid for a finish the card
			// has no printing of would be asking for one that should not
			// exist.
			sold := []string{mtgmatcher.FinishNonfoil, mtgmatcher.FinishFoil}
			if listed, ok := row["finishes"].([]any); ok && len(listed) > 0 {
				sold = nil
				for _, raw := range listed {
					if name, ok := raw.(string); ok {
						sold = append(sold, name)
					}
				}
			}
			ids := map[string]any{}
			for _, finish := range sold {
				uuid := "published-" + id + "-" + finish
				ids[finish] = uuid
				want[uuid] = true
			}
			row["printingIds"] = ids
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out, want
}
