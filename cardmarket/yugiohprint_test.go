package cardmarket

import (
	"errors"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// yugiohEuropeanPrint answers the uuid of the row numbered exactly number
// in setName, or "" where the datastore has no row at that number - so a
// case can hold on either shape of the datastore.
func yugiohEuropeanPrint(t *testing.T, b *mtgmatcher.Backend, setName, number string) string {
	t.Helper()
	set, err := b.GetSetByName(setName)
	if err != nil {
		t.Fatalf("GetSetByName(%q) = %v", setName, err)
	}
	for _, uuid := range b.GetUUIDsInSet(set.Code) {
		co, err := b.GetUUID(uuid)
		if err == nil && co.Number == number {
			return uuid
		}
	}
	return ""
}

// TestYugiohIndexPrints pins what the version index names on the shelves
// Cardmarket split into one product per regional print: a print, never a
// run. Each product, number, expansion and bridge id is the live catalog's.
func TestYugiohIndexPrints(t *testing.T) {
	b := loadYugiohBackend(t)
	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}

	// Once datastore-gen mints the card's own European first print, the
	// resolver gives up the CardTrader bridge to the North American row for
	// its own numbered printing instead; hold either outcome.
	europeanWant, europeanErr := "", errForeign
	uuid := yugiohEuropeanPrint(t, b, "Magic Ruler", "MRL-E047")
	if uuid != "" {
		europeanWant, europeanErr = uuid, nil
	}

	for _, tt := range []struct {
		desc                    string
		mkmID, bridgeTCG        int
		name, number, expansion string
		want, wantFoil          string
		wantErr                 error
	}{
		{
			desc:  "the North American print takes the default run, the first edition beside it",
			mkmID: 579784, name: "Mystical Space Typhoon (V.2 - Ultra Rare)", number: "047", expansion: "Magic Ruler",
			want: "mrl-047_22255_unlimited", wantFoil: "mrl-047_22255_1stedition",
		},
		{
			desc:  "the European print gives the bridged North American row up",
			mkmID: 104729, bridgeTCG: 22255, name: "Mystical Space Typhoon (V.1 - Ultra Rare)", number: "047", expansion: "Magic Ruler",
			want: europeanWant, wantErr: europeanErr,
		},
		{
			desc:  "the European print keeps a row of its own",
			mkmID: 105448, bridgeTCG: 22981, name: "Serpent Night Dragon (V.1 - Secret Rare)", number: "130", expansion: "Magic Ruler",
			want: "mrl-e130_229418_unlimited", wantFoil: "mrl-e130_229418_1stedition",
		},
		{
			desc:  "a worldwide reprint numbered bare reaches its own set",
			mkmID: 579296, name: "Cannon Soldier (V.3 - Rare)", number: "106", expansion: "Metal Raiders",
			want: "mrd-en106_476730_unlimited",
		},
		{
			desc:  "an Asian English print numbered bare is refused",
			mkmID: 578096, name: "Dark Magician (V.3 - Ultra Rare)", number: "005", expansion: "Legend of Blue Eyes White Dragon",
			wantErr: errForeign,
		},
		{
			desc:  "a product the split left behind names no print",
			mkmID: 106089, name: "Toon Summoned Skull", expansion: "Magic Ruler",
			wantErr: errTwin,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			mkm.tcgBridge = map[int]int{}
			if tt.bridgeTCG != 0 {
				mkm.tcgBridge[tt.mkmID] = tt.bridgeTCG
			}
			product := &cm.Product{IDProduct: tt.mkmID, Name: tt.name, Number: tt.number, ExpansionName: tt.expansion}
			got, gotFoil, _, err := mkm.resolveProduct(product)
			if !errors.Is(err, tt.wantErr) || got != tt.want || gotFoil != tt.wantFoil {
				t.Errorf("resolveProduct(%q #%s) = %q, %q, %v; want %q, %q, %v",
					tt.name, tt.number, got, gotFoil, err, tt.want, tt.wantFoil, tt.wantErr)
			}
		})
	}
}
