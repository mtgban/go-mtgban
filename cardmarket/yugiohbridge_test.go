package cardmarket

import (
	"os"
	"sync"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/yugioh"
)

var (
	yugiohBackendOnce sync.Once
	yugiohBackendErr  error
	yugiohBackend     *mtgmatcher.Backend
)

// loadYugiohBackend reads the real Yu-Gi-Oh datastore YUGIOH_PATH
// names, once per test binary run, for the tests below: they need real
// sibling rarities and konamiId, not a synthetic subset.
func loadYugiohBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	yugiohBackendOnce.Do(func() {
		path := os.Getenv("YUGIOH_PATH")
		if path == "" {
			return
		}
		yugiohBackend, yugiohBackendErr = datastore.Read("yugioh", path)
	})
	if yugiohBackendErr != nil {
		t.Fatal(yugiohBackendErr)
	}
	if os.Getenv("YUGIOH_PATH") == "" {
		t.Skip("Need YUGIOH_PATH set to run this test")
	}
	return yugiohBackend
}

// TestYugiohBridgeDistrustsSiblingRarity pins that CardTrader's blueprint
// can agree with a product's name while its own tcg_player_id belongs to
// the sibling rarity, or - Griggle - to another card outright. Each
// product, number and expansion below is Cardmarket's own catalog naming,
// and each bridge id is a CardTrader blueprint id.
func TestYugiohBridgeDistrustsSiblingRarity(t *testing.T) {
	b := loadYugiohBackend(t)

	for _, tt := range []struct {
		desc                    string
		mkmID, bridgeTCG        int
		name, number, expansion string
		want                    string
	}{
		{
			desc:      "Gem-Knight Tourmaline Common bridged to its own Shatterfoil twin",
			mkmID:     283203,
			bridgeTCG: 99879,
			name:      "Gem-Knight Tourmaline (V.1 - Common)", number: "001", expansion: "Star Pack Arc-V",
			want: "sp15-en001_99880_1stedition",
		},
		{
			desc:      "the Shatterfoil twin the bridge id belongs to, unbridged, still lands by name",
			mkmID:     283204,
			bridgeTCG: 0,
			name:      "Gem-Knight Tourmaline (V.2 - Shatterfoil)", number: "001", expansion: "Star Pack Arc-V",
			want: "sp15-en001_99879_1stedition",
		},
		{
			desc:      "Purrely V.4 Platinum Secret Rare bridged to the Prismatic Ultimate Rare",
			mkmID:     769941,
			bridgeTCG: 550973,
			name:      "Purrely (V.4 - Platinum Secret Rare)", number: "018", expansion: "25th Anniversary Rarity Collection II",
			want: "ra02-en018_550971_1stedition",
		},
		{
			desc:      "Purrely V.6 Collectors Rare bridged to the Platinum Secret Rare",
			mkmID:     770107,
			bridgeTCG: 550971,
			name:      "Purrely (V.6 - Collectors Rare)", number: "018", expansion: "25th Anniversary Rarity Collection II",
			want: "ra02-en018_550974_1stedition",
		},
		{
			desc:      "Purrely V.7 Ultimate Rare bridged to the Prismatic Collector's Rare",
			mkmID:     770188,
			bridgeTCG: 550974,
			name:      "Purrely (V.7 - Ultimate Rare)", number: "018", expansion: "25th Anniversary Rarity Collection II",
			want: "ra02-en018_550973_1stedition",
		},
		{
			desc:      "Skullcrobat Joker Secret Rare bridged to its Common sibling",
			mkmID:     900149,
			bridgeTCG: 708270,
			name:      "Performapal Skullcrobat Joker (V.2 - Secret Rare)", number: "O03", expansion: "Legendary Arc-V Decks",
			want: "lavd-eno03_708269_1stedition",
		},
		{
			desc:      "the swapped Common half of the same pair",
			mkmID:     900150,
			bridgeTCG: 708269,
			name:      "Performapal Skullcrobat Joker (V.1 - Common)", number: "O03", expansion: "Legendary Arc-V Decks",
			want: "lavd-eno03_708270_1stedition",
		},
		{
			desc:      "Artifact Scythe Super Rare bridged to the ENSP1 Ultra Rare",
			mkmID:     266821,
			bridgeTCG: 82546,
			name:      "Artifact Scythe (V.1 - Super Rare)", number: "000", expansion: "Primal Origin",
			want: "prio-en000_82626_unlimited",
		},
		{
			desc:      "the ENSP1 Ultra Rare the bridge id belongs to, unbridged, still lands by name",
			mkmID:     294681,
			bridgeTCG: 0,
			name:      "Artifact Scythe (V.2 - Ultra Rare)", number: "SP1", expansion: "Primal Origin",
			want: "prio-ensp1_82546_limited",
		},
		{
			desc:      "Griggle bridged to Rescue-ACE Monitor, another card entirely",
			mkmID:     579753,
			bridgeTCG: 579753,
			name:      "Griggle (V.2 - Common)", number: "016", expansion: "Magic Ruler",
			want: "mrl-016_22052_1stedition",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			mkm, err := NewScraperIndex(b)
			if err != nil {
				t.Fatalf("NewScraperIndex(b) = %v", err)
			}
			bridge := map[int]int{}
			if tt.bridgeTCG != 0 {
				bridge[tt.mkmID] = tt.bridgeTCG
			}
			mkm.tcgBridge = bridge
			// Both pairs are set because the landed printing may be the
			// base run or the 1st Edition one, and emitPrices reads one
			// pair or the other depending on which the id turns out to be.
			mkm.priceGuide = map[int]cm.PriceGuide{tt.mkmID: {
				IDProduct: tt.mkmID, LowPrice: 1, TrendPrice: 2, FoilLowPrice: 1, FoilTrendPrice: 2,
			}}
			product := cm.Product{
				IDProduct:     tt.mkmID,
				Name:          tt.name,
				Number:        tt.number,
				ExpansionName: tt.expansion,
			}
			channel := make(chan responseChan, 8)
			if err := mkm.processProduct(channel, &product); err != nil {
				t.Fatalf("processProduct(%q) = %v", tt.name, err)
			}
			close(channel)
			var got string
			for res := range channel {
				if res.cardID != "" {
					got = res.cardID
					break
				}
			}
			if got != tt.want {
				t.Errorf("processProduct(%q) named %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
