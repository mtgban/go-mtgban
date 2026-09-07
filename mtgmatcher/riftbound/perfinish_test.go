package riftbound

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPerFinishRowsLoadAlike pins the two shapes to one another. The
// datastore may publish a row per card, carrying the finish list for this
// loader to explode, or a row per printing, naming the one finish it is -
// and the second is the shape every other game here publishes. Whichever
// arrives, the backend has to come out the same: the same uuids, the same
// finish on each, the same FoilUUIDs, the same cards filed in the same
// sets. A shape change that moved a single uuid would silently repoint
// every price keyed on it.
func TestPerFinishRowsLoadAlike(t *testing.T) {
	path := os.Getenv("RIFTBOUND_PATH")
	if path == "" {
		t.Skip("RIFTBOUND_PATH not set; skipping Riftbound matcher suite")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	byCard, err := Load(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	byPrinting, err := Load(bytes.NewReader(splitRowsPerFinish(t, data)))
	if err != nil {
		t.Fatal(err)
	}

	was, is := describe(t, byCard), describe(t, byPrinting)
	if len(was) != len(is) {
		t.Fatalf("a row per card loads %d printings, a row per finish loads %d", len(was), len(is))
	}
	for i := range was {
		if was[i] != is[i] {
			t.Errorf("printing %d differs\n  row per card:   %s\n  row per finish: %s", i, was[i], is[i])
		}
	}
}

// describe is every printing the backend holds, in one comparable line
// apiece and a stable order.
func describe(t *testing.T, b *mtgmatcher.Backend) []string {
	t.Helper()
	out := make([]string, 0, len(b.AllUUIDs))
	for _, uuid := range b.AllUUIDs {
		co, err := b.GetUUID(uuid)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, fmt.Sprintf("%s|%s|%s|%s|%s|%v|%v|%v|%v",
			uuid, co.Name, co.SetCode, co.Number, co.Finish, co.Foil,
			co.Finishes, co.FoilUUIDs, co.Identifiers))
	}
	sort.Strings(out)
	return out
}

// splitRowsPerFinish rewrites a payload's gallery rows into one row per
// finish, the way a datastore publishing printings would: the finish named
// on the row, and the uuid it prices as the row's own id. A row the
// catalog sells nothing for carries no finish list and is sold in both,
// which is the fallback this has to reproduce rather than drop.
func splitRowsPerFinish(t *testing.T, data []byte) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	blades := doc["pageProps"].(map[string]any)["page"].(map[string]any)["blades"].([]any)
	for _, raw := range blades {
		blade, _ := raw.(map[string]any)
		if blade == nil || blade["type"] != "riftboundCardGallery" {
			continue
		}
		cards := blade["cards"].(map[string]any)
		var split []any
		for _, item := range cards["items"].([]any) {
			row := item.(map[string]any)
			finishes := []string{mtgmatcher.FinishNonfoil, mtgmatcher.FinishFoil}
			if listed, ok := row["finishes"].([]any); ok && len(listed) > 0 {
				finishes = nil
				for _, f := range listed {
					switch name, _ := f.(string); name {
					case mtgmatcher.FinishNonfoil, mtgmatcher.FinishFoil:
						finishes = append(finishes, name)
					}
				}
			}
			for _, finish := range finishes {
				printing := map[string]any{}
				for k, v := range row {
					printing[k] = v
				}
				delete(printing, "finishes")
				printing["id"] = fmt.Sprintf("%s_%s", row["id"], finish)
				printing["finish"] = finish
				split = append(split, printing)
			}
		}
		cards["items"] = split
	}
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
