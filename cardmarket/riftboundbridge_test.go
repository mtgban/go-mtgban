package cardmarket

import (
	"os"
	"strings"
	"sync"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/matcher"
	_ "github.com/mtgban/go-mtgban/matcher/riftbound"
)

var (
	riftboundBackendOnce sync.Once
	riftboundBackendErr  error
	riftboundBackend     *matcher.Backend
)

// loadRiftboundBackend installs the real Riftbound datastore RIFTBOUND_PATH
// names, once per test binary run: the gallery envelope Load parses is too
// shaped to fake convincingly, so the bridge guard below is graded against
// real rows instead (see the *_finish_test.go equivalents for other games).
func loadRiftboundBackend(t *testing.T) *matcher.Backend {
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
// carries no card named "Tentacle" on its own; that refusal is the correct
// outcome.
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

// TestRiftboundFoilRunes pins that the Vendetta: Promos runes land on the
// foil of Vendetta's own rune in both price columns, leaving the plain rune
// to the set's product.
func TestRiftboundFoilRunes(t *testing.T) {
	b := loadRiftboundBackend(t)

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}

	for _, tt := range []struct {
		id     int
		name   string
		number string
	}{
		{917119, "Fury Rune (V.2 - Common)", "R01"},
		{917120, "Calm Rune (V.2 - Common)", "R02"},
		{917123, "Mind Rune (V.2 - Common)", "R03"},
		{917140, "Body Rune (V.2 - Common)", "R04"},
		{917141, "Chaos Rune (V.2 - Common)", "R05"},
		{917151, "Order Rune (V.2 - Common)", "R06"},
	} {
		product := cm.Product{IDProduct: tt.id, Name: tt.name, Number: tt.number, ExpansionName: "Vendetta: Promos"}
		cardID, cardIDFoil, _, err := mkm.resolveProduct(&product)
		if err != nil || cardID == "" || cardID != cardIDFoil {
			t.Errorf("resolveProduct(%d) = (%q, %q, %v), want one printing in both slots", tt.id, cardID, cardIDFoil, err)
			continue
		}
		co, err := b.GetUUID(cardID)
		if err != nil {
			t.Fatal(err)
		}
		want := "R" + strings.TrimPrefix(tt.number, "R0")
		if !co.Foil || co.SetCode != "VEN" || co.Number != want {
			t.Errorf("resolveProduct(%d) landed on %s (%s %s foil=%v), want the foil of VEN %s", tt.id, cardID, co.SetCode, co.Number, co.Foil, want)
		}
	}
}
