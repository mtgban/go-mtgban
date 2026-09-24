package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"
)

// TestYugiohRun pins how the index Cardmarket appends to a Yu-Gi-Oh product
// name is read. A set printed twice sells both runs under one name, one
// collector number and one rarity, so the index is the only thing left; it is
// read the way the catalog's prices bear out most often, and a set that reads
// the other way round is named in the override beside it.
func TestYugiohRun(t *testing.T) {
	for _, tt := range []struct {
		desc, name, expansion, want string
	}{
		{
			"the first index is the run a set keeps in print",
			"Blue-Eyes White Dragon (V.1 - Super Rare)", "Duelist Pack: Kaiba", "Unlimited",
		},
		{
			"the one after it is the scarcer first edition",
			"Blue-Eyes White Dragon (V.2 - Super Rare)", "Duelist Pack: Kaiba", "1st Edition",
		},
		{
			"and so is every one beyond that",
			"Suijin (V.3 - Super Rare)", "Metal Raiders", "1st Edition",
		},
		{
			"a product carrying no index says nothing about its run",
			"Blue-Eyes White Dragon", "Duelist Pack: Kaiba", "",
		},
		{
			"an index the name spells without a rarity beside it is not one",
			"Blue-Eyes White Dragon (V.2)", "Duelist Pack: Kaiba", "",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			product := &cm.Product{Name: tt.name, ExpansionName: tt.expansion}
			if got := yugiohRun(product); got != tt.want {
				t.Errorf("yugiohRun(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestYugiohRunOverride pins that a set reading the other way round is
// corrected by naming it, which is the whole point of the table: the index
// means something different from one set to the next, and the prices are what
// say which sets those are.
func TestYugiohRunOverride(t *testing.T) {
	const set = "Swapped Set"
	yugiohFirstAtIndexOne[set] = true
	defer delete(yugiohFirstAtIndexOne, set)

	for _, tt := range []struct{ name, want string }{
		{"Card (V.1 - Rare)", "1st Edition"},
		{"Card (V.2 - Rare)", "Unlimited"},
	} {
		product := &cm.Product{Name: tt.name, ExpansionName: set}
		if got := yugiohRun(product); got != tt.want {
			t.Errorf("with the override, yugiohRun(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}

	// A set nobody named keeps reading the default way.
	product := &cm.Product{Name: "Card (V.1 - Rare)", ExpansionName: "Ordinary Set"}
	if got := yugiohRun(product); got != "Unlimited" {
		t.Errorf("an unnamed set read as %q, want Unlimited", got)
	}
}

// battleFaderDatastore is Battle Fader's four rows in Absolute Powerforce,
// copied verbatim from the published datastore: an Ultra Rare and an
// Ultimate Rare at one number, each printed in both runs.
const battleFaderDatastore = `{"data": {
 "game": "yugioh",
 "sets": {"ABPF": {"name": "Absolute Powerforce", "releaseDate": "2010-02-16"}},
 "cards": [
  {"attribute": "DARK", "externalLinks": {"konamiId": 19665973, "tcgPlayerId": 34509}, "finish": "Unlimited", "id": "abpf-en006_34509_unlimited", "name": "Battle Fader", "number": "ABPF-EN006", "rarity": "Ultra Rare", "setCode": "ABPF", "type": "Effect Monster"},
  {"attribute": "DARK", "externalLinks": {"konamiId": 19665973, "tcgPlayerId": 34509}, "finish": "1st Edition", "id": "abpf-en006_34509_1stedition", "name": "Battle Fader", "number": "ABPF-EN006", "rarity": "Ultra Rare", "setCode": "ABPF", "type": "Effect Monster"},
  {"attribute": "DARK", "externalLinks": {"konamiId": 19665973, "tcgPlayerId": 58454}, "finish": "Unlimited", "id": "abpf-en006_58454_unlimited", "name": "Battle Fader", "number": "ABPF-EN006", "rarity": "Ultimate Rare", "setCode": "ABPF", "type": "Effect Monster"},
  {"attribute": "DARK", "externalLinks": {"konamiId": 19665973, "tcgPlayerId": 58454}, "finish": "1st Edition", "id": "abpf-en006_58454_1stedition", "name": "Battle Fader", "number": "ABPF-EN006", "rarity": "Ultimate Rare", "setCode": "ABPF", "type": "Effect Monster"}
 ]
}}`

// TestYugiohRarityIndex pins that an index counting rarities names no run.
// Cardmarket sells Battle Fader as V.1 Ultra Rare and V.2 Ultimate Rare,
// each product listing both runs, so V.2 has to resolve to the default run
// and hand Market the pair rather than a lone 1st Edition.
func TestYugiohRarityIndex(t *testing.T) {
	b := datastoreBackend(t, "yugioh", battleFaderDatastore)
	exp := cm.Expansion{IDExpansion: 1187, Name: "Absolute Powerforce", SetCode: "ABPF"}

	for _, tt := range []struct {
		desc, v1Rarity, want, wantFoil string
	}{
		{
			"a rarity split resolves to the default run",
			"Ultra Rare", "abpf-en006_58454_unlimited", "abpf-en006_58454_1stedition",
		},
		{
			"a second product of the same rarity is still read as a run",
			"Ultimate Rare", "abpf-en006_58454_1stedition", "abpf-en006_58454_1stedition",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			mkm, err := NewScraperIndex(b)
			if err != nil {
				t.Fatal(err)
			}
			v2 := cm.CatalogProduct{ExpansionID: 1187, Name: "Battle Fader (V.2 - Ultimate Rare)", Number: "006", Rarity: "Ultimate Rare", Version: 2}
			mkm.catalog = &cm.Catalog{Data: cm.CatalogData{
				Expansions: map[int]cm.CatalogExpansion{1187: {Name: exp.Name, Code: exp.SetCode}},
				Products: map[int]cm.CatalogProduct{
					110551: {ExpansionID: 1187, Name: "Battle Fader (V.1 - " + tt.v1Rarity + ")", Number: "006", Rarity: tt.v1Rarity, Version: 1},
					110645: v2,
				},
			}}
			r := mkm.resolveMapped(110645, v2, exp)
			if r.err != nil || r.cardID != tt.want || r.cardIDFoil != tt.wantFoil {
				t.Errorf("resolveMapped(110645) = (%q, %q, %v), want (%q, %q)", r.cardID, r.cardIDFoil, r.err, tt.want, tt.wantFoil)
			}
		})
	}
}
