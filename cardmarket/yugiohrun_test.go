package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"
)

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

// TestYugiohIndexNamesNoRun pins that the version index names no run.
// Cardmarket sells Battle Fader as V.1 Ultra Rare and V.2 Ultimate Rare,
// each product listing both runs, so V.2 has to resolve to the default run
// and hand Market the pair rather than a lone 1st Edition.
func TestYugiohIndexNamesNoRun(t *testing.T) {
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
			"a second product of the same rarity lands there too",
			"Ultimate Rare", "abpf-en006_58454_unlimited", "abpf-en006_58454_1stedition",
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
