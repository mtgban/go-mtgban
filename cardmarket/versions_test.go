package cardmarket

import (
	"slices"
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

// TestResolveMappedReadsVersionPrintings pins versionPrintings over the
// datastore's own links: Sim Han How's Forest V.1, which mtgjson links to
// shh328 where Cardmarket's image is shh347, and V.2, linked to Fourth
// Edition's black bordered Forest; Mark Justice's Plains V.3 and Chronicles'
// Urza's Mine V.2, each linked to several printings. A version the links get
// right still lands.
func TestResolveMappedReadsVersionPrintings(t *testing.T) {
	b := realDatastore(t)
	r := &resolver{backend: b, gameID: cm.GameMagic}

	for _, tt := range []struct {
		id        int
		name      string
		expansion string
		want      string
	}{
		{249453, "Forest (V.1)", "WCD 2002: Sim Han How", "shh347"},
		{249454, "Forest (V.2)", "WCD 2002: Sim Han How", "shh348"},
		{249456, "Forest (V.4)", "WCD 2002: Sim Han How", "shh350"},
		{22804, "Plains (V.3)", "Pro Tour 1996: Mark Justice", "mj365"},
		{272488, "Urza's Mine (V.2)", "Chronicles: Japanese", "114b"},
	} {
		got := r.resolveMapped(tt.id, cm.CatalogProduct{Name: tt.name}, cm.Expansion{Name: tt.expansion})
		co, err := b.GetUUID(got.cardID)
		if err != nil || co.Number != tt.want {
			t.Errorf("%d %s: landed on %v, want %s", tt.id, tt.name, co, tt.want)
		}
	}
}

// TestMagicPrintings holds the printings the datastore links to a product
// to the list MTGJSON's CardmarketIdentifiers published for it.
func TestMagicPrintings(t *testing.T) {
	b := realDatastore(t)
	r := &resolver{backend: b, gameID: cm.GameMagic}

	for id, want := range map[int][]string{
		249456: {"b62bd4c3-8dc6-580f-8ba6-6ad5dcdae6f4"},
		22804:  {"283db89c-0433-5b67-b376-01c6aa2646c1", "32fa4b3e-318b-5ade-bb55-2d13214ebbbf"},
		272488: {"6d974ddd-afbf-53f1-a5a6-190162fbad06", "9c4a6da7-4427-5379-85ff-7121dcc4dc03", "a0a9e2ae-b768-51ef-a002-71ccb6c3ceee", "aeb3807a-6bae-5382-8823-f1c884f1a46d"},
	} {
		if got := r.magicPrintings(id); !slices.Equal(got, want) {
			t.Errorf("%d: %v, want %v", id, got, want)
		}
	}
}
