package lorcana

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
// datastore may publish a row per card, carrying the foil type list for
// this loader to explode, or a row per printing, naming the one foil type
// it is and the uuid that printing prices - and the second is the shape
// every other game here publishes. Whichever arrives, the backend has to
// come out the same: the same uuids, the same finish on each, the same
// FoilUUIDs and FinishAliases, the same cards filed in the same sets.
//
// The aliases are the reason this is pinned on the whole set rather than a
// sample: which foil answers the plain foil flag, and which answers
// TCGplayer's Holofoil, is decided by the order the foil types are
// published in, and after the split that order lives in the order of the
// rows. The twelve cards sold in three finishes are where that is load
// bearing.
func TestPerFinishRowsLoadAlike(t *testing.T) {
	path := os.Getenv("LORCANA_PATH")
	if path == "" {
		t.Skip("LORCANA_PATH not set; skipping Lorcana matcher suite")
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
		out = append(out, fmt.Sprintf("%s|%s|%s|%s|%s|%v|%v|%v|%v|%v",
			uuid, co.Name, co.SetCode, co.Number, co.Finish, co.Foil,
			co.Finishes, co.FoilUUIDs, co.FinishAliases, co.Identifiers))
	}
	sort.Strings(out)
	return out
}

// splitRowsPerFinish rewrites a payload's card rows into one row per
// printing, the way a datastore publishing printings would: the foil type
// named on the row, and the uuid that printing prices beside it. The id
// stays the card's, which is what the rows group under and what
// promoIds and the other upstream cross-references point at.
func splitRowsPerFinish(t *testing.T, data []byte) []byte {
	t.Helper()
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	rows, ok := doc["cards"].([]any)
	if !ok {
		t.Fatal("the payload holds no cards")
	}
	var split []any
	for _, item := range rows {
		row, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("a card is %T, not an object", item)
		}
		id, ok := row["id"].(float64)
		if !ok {
			t.Fatalf("a card's id is %T, not a number", row["id"])
		}
		base := cardUUID(int(id))
		foilTypes, _ := row["foilTypes"].([]any)
		if len(foilTypes) == 0 {
			split = append(split, row)
			continue
		}
		for _, raw := range foilTypes {
			foilType, ok := raw.(string)
			if !ok {
				t.Fatalf("a foil type is %T, not a string", raw)
			}
			printing := map[string]any{}
			for k, v := range row {
				printing[k] = v
			}
			delete(printing, "foilTypes")
			printing["finish"] = foilType
			printing["uuid"] = base
			if finish := canonicalFinish(foilType); finish != mtgmatcher.FinishNonfoil {
				printing["uuid"] = base + "_" + finish
			}
			split = append(split, printing)
		}
	}
	doc["cards"] = split
	out, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
