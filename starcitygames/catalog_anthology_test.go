package starcitygames

import (
	"testing"
)

// Duel Decks: Anthology reprints four earlier decks and mtgjson keeps
// them under the original codes, which survive only in the sku.
func TestResolveAnthologySubdeck(t *testing.T) {
	b := withMagic(t)
	base := CatalogProduct{
		Name: "Forest", Set: "Duel Decks: Anthology", Language: "English",
		CollectorNumber: "28", FinishGroup: "Non-foil",
	}
	evg := base
	evg.SKU = "SGL-MTG-EVG-28-ENN"
	gvl := base
	gvl.SKU = "SGL-MTG-GVL-28-ENN"

	idE, err := resolveProduct(b, GameMagic, evg)
	if err != nil {
		t.Fatalf("EVG: %v", err)
	}
	idG, err := resolveProduct(b, GameMagic, gvl)
	if err != nil {
		t.Fatalf("GVL: %v", err)
	}
	if idE == idG {
		t.Fatalf("both sub-decks resolved to %s", idE)
	}
	coE, _ := b.GetUUID(idE)
	coG, _ := b.GetUUID(idG)
	if coE.SetCode != "EVG" {
		t.Errorf("EVG sku resolved into %s", coE.SetCode)
	}
	if coG.SetCode != "GVL" {
		t.Errorf("GVL sku resolved into %s", coG.SetCode)
	}
}
