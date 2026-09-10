package gamenerdz

import (
	"strconv"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestResolveProductEtchedByID pins the etched flag on the id path. The
// retail feed carries the catalog's own TCGplayer id for nearly every
// product, and an etched product carries the etched printing's; asked for
// with the foil flag alone, the matcher hands back the foil sibling, which
// is the class the wording path stopped doing and the id path, answering
// first, kept on.
func TestResolveProductEtchedByID(t *testing.T) {
	realDatastore(t)
	// An etched printing filed beside its foil on one card is the shape
	// that folds: one sold etched alone has nothing to fold to.
	var etchedUUID, tcgID string
	for _, uuid := range mtgmatcher.GetUUIDs() {
		co, err := mtgmatcher.GetUUID(uuid)
		if err != nil || !co.Etched || !co.HasFinish(mtgmatcher.FinishFoil) {
			continue
		}
		if id := co.Identifiers["tcgplayerEtchedProductId"]; id != "" {
			etchedUUID, tcgID = uuid, id
			break
		}
	}
	if etchedUUID == "" {
		t.Skip("no etched printing carries a TCGplayer id")
	}
	co, _ := mtgmatcher.GetUUID(etchedUUID)

	gn := NewScraper(GameMagic)
	id, err := strconv.ParseInt(tcgID, 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	product := GNProduct{
		ID:             "etched",
		DisplayName:    co.Name + " (Foil Etched)",
		SelectedFinish: "Foil",
		ProductData:    GNProductData{TCGProductID: id},
	}
	got, err := gn.resolveProduct(modeRetail, product)
	if err != nil {
		t.Fatal(err)
	}
	if got != etchedUUID {
		t.Errorf("resolveProduct(%q, tcg %s) = %s, want the etched printing %s", product.DisplayName, tcgID, got, etchedUUID)
	}
}
