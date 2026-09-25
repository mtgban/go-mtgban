package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestVersionPrintingsAreReal holds versionPrintings to the datastore: every
// row names exactly one printing of its set, and no two rows the same one.
func TestVersionPrintingsAreReal(t *testing.T) {
	b := realDatastore(t)

	seen := map[string]int{}
	for id, printing := range versionPrintings {
		set, found := b.Sets[printing.set]
		if !found {
			t.Errorf("%d: no set %s", id, printing.set)
			continue
		}
		var n int
		for _, card := range set.Cards {
			if card.Number == printing.number {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%d: %s %s names %d printings, want 1", id, printing.set, printing.number, n)
		}
		key := printing.set + " " + printing.number
		if other, taken := seen[key]; taken {
			t.Errorf("%d and %d both name %s", id, other, key)
		}
		seen[key] = id
	}
}

// TestResolveMappedReadsVersionPrintings pins versionPrintings over the map's
// own entries: Sim Han How's Forest V.1, which the map files on shh328 where
// Cardmarket's image is shh347, and V.2, filed on Fourth Edition's black
// bordered Forest; Mark Justice's Plains V.3 and Chronicles' Urza's Mine V.2,
// each filed on several printings. A version the map gets right still lands.
func TestResolveMappedReadsVersionPrintings(t *testing.T) {
	b := realDatastore(t)
	r := &resolver{backend: b, gameID: cm.GameMagic}

	for _, tt := range []struct {
		id        int
		name      string
		expansion string
		uuids     []string
		want      string
	}{
		{249453, "Forest (V.1)", "WCD 2002: Sim Han How", []string{"8a4c6500-8bb7-51d7-a7e1-8a0d3eabfc9a"}, "shh347"},
		{249454, "Forest (V.2)", "WCD 2002: Sim Han How", []string{"54b0d1ae-9db1-50aa-b5ce-909a7b8f977a"}, "shh348"},
		{249456, "Forest (V.4)", "WCD 2002: Sim Han How", []string{"b62bd4c3-8dc6-580f-8ba6-6ad5dcdae6f4"}, "shh350"},
		{22804, "Plains (V.3)", "Pro Tour 1996: Mark Justice", []string{"283db89c-0433-5b67-b376-01c6aa2646c1", "32fa4b3e-318b-5ade-bb55-2d13214ebbbf"}, "mj365"},
		{272488, "Urza's Mine (V.2)", "Chronicles: Japanese", []string{"6d974ddd-afbf-53f1-a5a6-190162fbad06", "9c4a6da7-4427-5379-85ff-7121dcc4dc03", "a0a9e2ae-b768-51ef-a002-71ccb6c3ceee", "aeb3807a-6bae-5382-8823-f1c884f1a46d"}, "114b"},
	} {
		got := r.resolveMapped(tt.id, cm.CatalogProduct{Name: tt.name, UUIDs: tt.uuids}, cm.Expansion{Name: tt.expansion})
		co, err := b.GetUUID(got.cardID)
		if err != nil || co.Number != tt.want {
			t.Errorf("%d %s: landed on %v, want %s", tt.id, tt.name, co, tt.want)
		}
	}
}
