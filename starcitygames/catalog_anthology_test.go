package starcitygames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Duel Decks: Anthology reprints four earlier decks and mtgjson keeps
// them under the original codes, which survive only in the sku.
func TestResolveAnthologySubdeck(t *testing.T) {
	if len(mtgmatcher.GetUUIDs()) == 0 {
		t.Skip("datastore not loaded")
	}
	base := CatalogProduct{
		Name: "Forest", Set: "Duel Decks: Anthology", Language: "English",
		CollectorNumber: "28", FinishGroup: "Non-foil",
	}
	evg := base
	evg.SKU = "SGL-MTG-EVG-28-ENN"
	gvl := base
	gvl.SKU = "SGL-MTG-GVL-28-ENN"

	idE, err := resolveProduct(GameMagic, evg)
	if err != nil {
		t.Fatalf("EVG: %v", err)
	}
	idG, err := resolveProduct(GameMagic, gvl)
	if err != nil {
		t.Fatalf("GVL: %v", err)
	}
	if idE == idG {
		t.Fatalf("both sub-decks resolved to %s", idE)
	}
	coE, _ := mtgmatcher.GetUUID(idE)
	coG, _ := mtgmatcher.GetUUID(idG)
	if coE.SetCode != "EVG" {
		t.Errorf("EVG sku resolved into %s", coE.SetCode)
	}
	if coG.SetCode != "GVL" {
		t.Errorf("GVL sku resolved into %s", coG.SetCode)
	}
}
