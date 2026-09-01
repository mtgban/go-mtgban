package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The sku carries detail the product fields have lost: which of two
// Portal versions a card is, and which duel deck an Anthology card came
// from.
func TestSkuSegments(t *testing.T) {
	tests := []struct{ sku, set, number string }{
		{"SGL-MTG-EVG-28-ENN", "EVG", "28"},
		{"SGL-MTG-GVL-28-ENN", "GVL", "28"},
		{"SGL-MTG-POR-6b-ENN", "POR", "6b"},
		{"SGL-MTG-POR-6a-ENN", "POR", "6a"},
		{"SGL-MTG-4BB-249-KON", "4BB", "249"},
		{"short", "", ""},
	}
	for _, test := range tests {
		if got := skuSetCode(test.sku); got != test.set {
			t.Errorf("skuSetCode(%q) = %q, want %q", test.sku, got, test.set)
		}
		if got := skuNumber(test.sku); got != test.number {
			t.Errorf("skuNumber(%q) = %q, want %q", test.sku, got, test.number)
		}
	}
}

// Portal printed two versions of six cards; mtgjson numbers the second
// with a d suffix while SCG marks it b in the sku, and the scryfall id
// SCG sends names the first for both.
func TestResolvePortalVariants(t *testing.T) {
	withMagic(t)

	if len(mtgmatcher.GetUUIDs()) == 0 {
		t.Skip("datastore not loaded")
	}
	base := CatalogProduct{
		Name: "Armored Pegasus", Set: "Portal", Language: "English",
		CollectorNumber: "006", FinishGroup: "Non-foil",
		ScryfallID: "a81b61af-cdb7-468f-9ff0-db82aa084023",
	}

	first := base
	first.SKU = "SGL-MTG-POR-6a-ENN"
	second := base
	second.SKU = "SGL-MTG-POR-6b-ENN"

	idA, err := resolveProduct(GameMagic, first)
	if err != nil {
		t.Fatalf("a: %v", err)
	}
	idB, err := resolveProduct(GameMagic, second)
	if err != nil {
		t.Fatalf("b: %v", err)
	}
	if idA == idB {
		t.Fatalf("both Portal versions resolved to %s", idA)
	}
	coA, _ := mtgmatcher.GetUUID(idA)
	coB, _ := mtgmatcher.GetUUID(idB)
	if coA.Number != "6" {
		t.Errorf("a resolved to #%s, want #6", coA.Number)
	}
	if coB.Number != "6d" {
		t.Errorf("b resolved to #%s, want #6d", coB.Number)
	}
}
