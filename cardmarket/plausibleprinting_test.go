package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestPlausiblePrintingDefersOnImplausibleWCD pins plausiblePrinting's own
// rule directly: a World Championship Deck expansion only ever sells a
// WC97-WC04 memorabilia printing, a Pro Tour 1996 one a PTC printing, and
// an Oversized shelf only ever sells an oversized one. mtgjson's own id links have drifted for real cards onto an
// unrelated printing of neither shape - 249617 to The Brothers' War Retro
// Artifacts' foil Phyrexian Processor instead of a WCD one, 21364 to the
// ordinary-sized 30th Anniversary Edition Nether Shadow instead of an
// oversized one - so this is not a synthetic shape.
func TestPlausiblePrintingDefersOnImplausibleWCD(t *testing.T) {
	b := realDatastore(t)

	for _, tt := range []struct {
		name          string
		expansionName string
		cardID        string
		want          bool
	}{
		{"a WCD product whose only candidate is a Brothers' War printing", "WCD 2000: Janosch Kühn", "a1a10730-b01e-5b6f-a9a4-1ae999729ef4", false},
		{"a WCD product whose candidate is its own WC00 printing", "WCD 2000: Janosch Kühn", "e2b509c6-cb8a-5170-b0cb-7093e0823747", true},
		{"a Pro Tour 1996 product whose only candidate is a Brothers' War printing", "Pro Tour 1996: Mark Justice", "a1a10730-b01e-5b6f-a9a4-1ae999729ef4", false},
		{"a Pro Tour 1996 product whose candidate is its own PTC printing", "Pro Tour 1996: Mark Justice", "693c127a-d748-5fc8-9b6e-94c3448a7ac3", true},
		{"an Oversized product whose only candidate is an ordinary-sized printing", "Oversized 6x9 Promos", "bf14fe40-3e5c-5790-bb75-4c8221c04883", false},
		{"an expansion neither guard constrains", "Judge Rewards Promos", "bf14fe40-3e5c-5790-bb75-4c8221c04883", true},
		{"an empty id decides nothing either way", "WCD 2000: Janosch Kühn", "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := plausiblePrinting(b, tt.expansionName, tt.cardID); got != tt.want {
				t.Errorf("plausiblePrinting(%q, %q) = %v, want %v", tt.expansionName, tt.cardID, got, tt.want)
			}
		})
	}
}

// TestResolveUUIDsDefersOnImplausibleOversized pins the map route this guard
// closes: resolveMapped answers from resolveUUIDs before Fallback's own mcmId
// check ever runs, so a map entry naming the same implausible uuid used to
// price straight through it. 21364 and 21387 are the id map's real entries
// for two Oversized 6x9 Promos products, each carrying exactly the wrong,
// ordinary-sized 30th Anniversary Edition printing mtgjson links them to.
func TestResolveUUIDsDefersOnImplausibleOversized(t *testing.T) {
	b := realDatastore(t)
	r := &resolver{backend: b, gameID: cm.GameMagic}

	for _, tt := range []struct {
		name    string
		product cm.Product
		uuids   []string
	}{
		{
			"21364 Nether Shadow's only map uuid is the ordinary 30A printing",
			cm.Product{IDProduct: 21364, Name: "Nether Shadow (V.2)", ExpansionName: "Oversized 6x9 Promos"},
			[]string{"bf14fe40-3e5c-5790-bb75-4c8221c04883"},
		},
		{
			"21387 Swords to Plowshares' only map uuid is the ordinary 30A printing",
			cm.Product{IDProduct: 21387, Name: "Swords to Plowshares (V.2)", ExpansionName: "Oversized 6x9 Promos"},
			[]string{"e411a1e9-bc7a-58bb-b436-b0b45b37f27a"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cardID, cardIDFoil := r.resolveUUIDs(&tt.product, tt.uuids)
			if cardID != "" || cardIDFoil != "" {
				co, _ := b.GetUUID(cardID)
				t.Fatalf("resolveUUIDs = (%q, %q), want (\"\", \"\") - kept %s instead of falling through to resolveProduct", cardID, cardIDFoil, co)
			}
		})
	}
}

// TestResolveUUIDsFollowsTheProductNumber pins numberedPrinting: mtgjson
// links Double Masters 2022's Consecrated Sphinx #345 and #427 each to the
// other's Cardmarket product, and the product's own number, which CardTrader
// agrees with, names the printing. The ids and entries are the map's own.
func TestResolveUUIDsFollowsTheProductNumber(t *testing.T) {
	b := realDatastore(t)
	r := &resolver{backend: b, gameID: cm.GameMagic}

	for _, tt := range []struct {
		product          cm.Product
		uuids            []string
		wantID, wantFoil string
	}{
		{
			cm.Product{IDProduct: 664915, Name: "Consecrated Sphinx (V.1)", Number: "345", ExpansionName: "Double Masters 2022: Extras"},
			[]string{"bf3e52bb-4017-5760-8db6-838ee858421d"},
			"106eb1f8-1452-55cd-8bfb-7db43bc5a3ce", "106eb1f8-1452-55cd-8bfb-7db43bc5a3ce_f",
		},
		{
			cm.Product{IDProduct: 664250, Name: "Consecrated Sphinx (V.2)", Number: "427", ExpansionName: "Double Masters 2022: Extras"},
			[]string{"106eb1f8-1452-55cd-8bfb-7db43bc5a3ce"},
			"bf3e52bb-4017-5760-8db6-838ee858421d", "bf3e52bb-4017-5760-8db6-838ee858421d",
		},
	} {
		cardID, cardIDFoil := r.resolveUUIDs(&tt.product, tt.uuids)
		if cardID != tt.wantID || cardIDFoil != tt.wantFoil {
			t.Errorf("%d #%s: resolveUUIDs = (%q, %q), want (%q, %q)", tt.product.IDProduct, tt.product.Number, cardID, cardIDFoil, tt.wantID, tt.wantFoil)
		}
	}
}
