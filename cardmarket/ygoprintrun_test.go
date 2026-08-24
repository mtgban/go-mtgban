package cardmarket

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/yugioh"
)

// ygoDatastore is the published Yu-Gi-Oh datastore cut down to the printings
// these tests turn on, every row copied verbatim from it: Guardian Elma, the
// one row Dark Crisis is numbered by where Cardmarket sells the North
// American and the European print as two products; Good & Evil in the
// Burning Abyss, which Secrets of Eternity carries at both its own number
// and the special edition's; and Damage Condenser, whose number opens on a
// deck letter the datastore writes after the region infix.
const ygoDatastore = `{
 "game": "yugioh",
 "sets": {
  "DCR": {"name": "Dark Crisis", "releaseDate": "2007-10-12"},
  "SECE": {"name": "Secrets of Eternity", "releaseDate": "2015-01-16"},
  "G2970": {"name": "Speed Duel GX: Duel Academy Box", "releaseDate": "2022-04-01"}
 },
 "cards": [
  {"attribute": "WIND", "externalLinks": {"tcgPlayerId": 22823}, "finish": "1st Edition", "id": "dcr-005_22823_1e", "name": "Guardian Elma", "number": "DCR-005", "rarity": "Common", "setCode": "DCR", "type": "Effect Monster"},
  {"attribute": "WIND", "externalLinks": {"tcgPlayerId": 22823}, "finish": "Unlimited", "id": "dcr-005_22823_unl", "name": "Guardian Elma", "number": "DCR-005", "rarity": "Common", "setCode": "DCR", "type": "Effect Monster"},
  {"attribute": "SPELL", "externalLinks": {"tcgPlayerId": 95478}, "finish": "1st Edition", "id": "sece-en086_95478_1e", "name": "Good & Evil in the Burning Abyss", "number": "SECE-EN086", "rarity": "Super Rare", "setCode": "SECE", "type": "Normal Spell"},
  {"attribute": "SPELL", "externalLinks": {"tcgPlayerId": 95478}, "finish": "Unlimited", "id": "sece-en086_95478_unl", "name": "Good & Evil in the Burning Abyss", "number": "SECE-EN086", "rarity": "Super Rare", "setCode": "SECE", "type": "Normal Spell"},
  {"attribute": "SPELL", "externalLinks": {"tcgPlayerId": 96145}, "finish": "Limited", "id": "sece-ens14_96145_lim", "name": "Good & Evil in the Burning Abyss", "number": "SECE-ENS14", "promoTypes": ["se"], "rarity": "Super Rare", "setCode": "SECE", "type": "Normal Spell", "variant": "SE"},
  {"attribute": "TRAP", "externalLinks": {"tcgPlayerId": 266282}, "finish": "1st Edition", "id": "sgx1-end19_266282_1e", "name": "Damage Condenser", "number": "SGX1-END19", "rarity": "Common", "setCode": "G2970", "type": "Normal Trap"}
 ]
}`

// TestMatchProductPrintRunPrefix pins which of the print runs Cardmarket
// sells an old Yu-Gi-Oh set in the name fallback answers for.
//
// The matcher reads a collector number that opens on digits or on a set
// code, so a Cardmarket number carrying a print run's own prefix is dropped
// and the name alone decides - which lands the European print, the Asian one
// and the special edition all on the single row the set is numbered by. The
// prefix has to be the answer's own, the region infix the datastore writes
// and Cardmarket omits aside.
func TestMatchProductPrintRunPrefix(t *testing.T) {
	err := mtgmatcher.LoadDatastore(strings.NewReader(ygoDatastore))
	if err != nil {
		t.Fatal(err)
	}

	mkm := &Index{gameID: GameYuGiOh}
	for _, tt := range []struct {
		name, expansion, product, number, want string
	}{
		{
			name:      "the North American print keeps the row the set is numbered by",
			expansion: "Dark Crisis",
			product:   "Guardian Elma (V.1 - Common)",
			number:    "005",
			want:      "dcr-005_22823_unl",
		},
		{
			// The row is DCR-005 and this product is DCR-EN005, a run we
			// carry no printing of: it has to refuse rather than fight the
			// product above for the one row.
			name:      "the European print refuses the row rather than folding onto it",
			expansion: "Dark Crisis",
			product:   "Guardian Elma (V.2 - Common)",
			number:    "EN005",
			want:      "",
		},
		{
			name:      "the set's own number still answers with the set's printing",
			expansion: "Secrets of Eternity",
			product:   "Good & Evil in the Burning Abyss (V.1 - Super Rare)",
			number:    "086",
			want:      "sece-en086_95478_unl",
		},
		{
			// The special edition is a printing we do carry, so refusing
			// SECE-EN086 has to narrow onto SECE-ENS14 rather than lose the
			// product: the set code prepended to the digits reaches it.
			name:      "the special edition narrows onto its own printing",
			expansion: "Secrets of Eternity",
			product:   "Good & Evil in the Burning Abyss (V.2 - Super Rare)",
			number:    "S14",
			want:      "sece-ens14_96145_lim",
		},
		{
			// "D19" is the tail of "SGX1-END19" with the region infix the
			// datastore writes and Cardmarket omits taken off, so the prefix
			// is the answer's own and the product is the row.
			name:      "a prefix the answer's own number carries is no other run",
			expansion: "Speed Duel GX: Duel Academy Box",
			product:   "Damage Condenser",
			number:    "D19",
			want:      "sgx1-end19_266282_1e",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got := mkm.matchProduct(&MKMProduct{
				Name:          tt.product,
				Number:        tt.number,
				ExpansionName: tt.expansion,
			})
			if got != tt.want {
				t.Errorf("matchProduct(%q, %q) = %q, want %q", tt.product, tt.number, got, tt.want)
			}
		})
	}
}
