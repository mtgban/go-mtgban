package cardmarket

import (
	"os"
	"sync"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"
	_ "github.com/mtgban/go-mtgban/mtgmatcher/riftbound"
)

var (
	riftboundBackendOnce sync.Once
	riftboundBackendErr  error
	riftboundBackend     *mtgmatcher.Backend
)

// loadRiftboundBackend installs the real Riftbound datastore RIFTBOUND_PATH
// names, once per test binary run: the gallery envelope Load parses is too
// shaped to fake convincingly, so the bridge guard below is graded against
// real rows instead (see the *_finish_test.go equivalents for other games).
func loadRiftboundBackend(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	riftboundBackendOnce.Do(func() {
		path := os.Getenv("RIFTBOUND_PATH")
		if path == "" {
			return
		}
		riftboundBackend, riftboundBackendErr = datastore.Read("riftbound", path)
	})
	if riftboundBackendErr != nil {
		t.Fatal(riftboundBackendErr)
	}
	if os.Getenv("RIFTBOUND_PATH") == "" {
		t.Skip("Need RIFTBOUND_PATH set to run this test")
	}
	return riftboundBackend
}

// TestRiftboundBridgeLandsTheRune pins the case the bridge exists for: the
// SFD/UNL R0Nb runes wear the AliasEdition heading "<Set>: Promos", which
// folds onto the one shared "Promos" pool of candidates, and Fury Rune R01b
// sits in it three times over (OPP, SFD, UNL) - wording alone always lands
// the OPP vendetta rune. The bridge's TCGplayer id says which one outright,
// and CloseName passes since both sides say "Fury Rune".
func TestRiftboundBridgeLandsTheRune(t *testing.T) {
	b := loadRiftboundBackend(t)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	mkm.tcgBridge = map[int]int{874107: 680274}
	// This product's only real printing is foil (opp-709748_foil is foil
	// too), so emitPrices prices it from the guide's foil pair.
	mkm.priceGuide = map[int]cm.PriceGuide{874107: {IDProduct: 874107, FoilLowPrice: 1, FoilTrendPrice: 2}}

	product := cm.Product{
		IDProduct:     874107,
		Name:          "Fury Rune",
		Number:        "R01b",
		ExpansionName: "Spiritforged: Promos",
	}
	channel := make(chan responseChan, 8)
	if err := mkm.processProduct(channel, &product); err != nil {
		t.Fatalf("processProduct(%+v) = %v", product, err)
	}
	close(channel)
	var got string
	for res := range channel {
		if res.cardID != "" {
			got = res.cardID
			break
		}
	}
	const want = "sfd-680274_foil"
	if got != want {
		t.Errorf("processProduct landed %q, want %q (the SFD row the bridge names, not opp-709748_foil the wording alone finds)", got, want)
	}
}

// TestRiftboundBridgeRejectsFusedToken pins the guard's other half: a bridge
// link to a fused two-face token ("Shadow Clone // Tentacle") for a product
// named after one face alone ("Tentacle") is not the same card, and
// CloseName says so. The product falls to the wording path instead of
// landing the wrong printing - which still refuses here, since the gallery
// carries no card named "Tentacle" on its own; that refusal is the correct,
// unchanged outcome, not a regression the guard introduced.
func TestRiftboundBridgeRejectsFusedToken(t *testing.T) {
	b := loadRiftboundBackend(t)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	// 709916 is ven-709916, "Shadow Clone // Tentacle" - a real bad link
	// cardtrader's blueprints carry for this product.
	mkm.tcgBridge = map[int]int{898153: 709916}
	mkm.priceGuide = map[int]cm.PriceGuide{898153: {IDProduct: 898153, LowPrice: 1, TrendPrice: 2}}

	product := cm.Product{
		IDProduct:     898153,
		Name:          "Tentacle",
		Number:        "T5",
		ExpansionName: "Vendetta",
	}
	cardID, cardIDFoil, _, err := mkm.resolveProduct(&product)
	if cardID == "ven-709916" || cardIDFoil == "ven-709916" {
		t.Fatalf("resolveProduct(%+v) landed the bridge's fused-token link %q, want it rejected", product, "ven-709916")
	}
	if err == nil && cardID != "" {
		t.Errorf("resolveProduct(%+v) = (%q, %q, nil), want a wording refusal now that the bridge link is rejected", product, cardID, cardIDFoil)
	}
}
