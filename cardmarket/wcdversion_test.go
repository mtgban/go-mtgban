package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// TestPreprocessRefusesAWCDVersion pins the refusal of a WCD "(V.N)" product
// no id settles. A version is one of the deck's printings of the name, and
// by name every version landed on the same one: Sim Han How's Forest V.2 on
// shh328 and Island V.1 on shh334, where the id route already lands V.1 and
// V.4, and Janosch Kühn's sideboard Phyrexian Processor on the main deck's
// jk306. A version its id settles still lands, and so does a name the deck
// prints once, the sideboard retry included.
func TestPreprocessRefusesAWCDVersion(t *testing.T) {
	b := realDatastore(t)
	r := &resolver{backend: b, gameID: cm.GameMagic}

	for _, tt := range []struct {
		product    cm.Product
		wantNumber string
	}{
		{cm.Product{IDProduct: 249456, Name: "Forest (V.4)", ExpansionName: "WCD 2002: Sim Han How"}, "shh350"},
		{cm.Product{IDProduct: 249454, Name: "Forest (V.2)", ExpansionName: "WCD 2002: Sim Han How"}, ""},
		{cm.Product{IDProduct: 249457, Name: "Island (V.1)", ExpansionName: "WCD 2002: Sim Han How"}, ""},
		{cm.Product{IDProduct: 249617, Name: "Phyrexian Processor (V.2)", ExpansionName: "WCD 2000: Janosch Kühn"}, ""},
	} {
		cardID, _, _, err := r.resolveProduct(&tt.product)
		if err != nil {
			t.Fatalf("%d: %v", tt.product.IDProduct, err)
		}
		co, _ := b.GetUUID(cardID)
		switch {
		case tt.wantNumber == "" && cardID != "":
			t.Errorf("%d %s: landed on %s, want no landing", tt.product.IDProduct, tt.product.Name, co)
		case tt.wantNumber != "" && (co == nil || co.Number != tt.wantNumber):
			t.Errorf("%d %s: landed on %v, want %s", tt.product.IDProduct, tt.product.Name, co, tt.wantNumber)
		}
	}

	for _, tt := range []struct {
		name, edition, wantNumber string
	}{
		{"Forest", "WCD 2000: Janosch Kühn", "jk347"},
		{"Gainsay", "WCD 2002: Sim Han How", "shh26sb"},
	} {
		theCard, err := Preprocess(b, tt.name, "", tt.edition)
		if err != nil {
			t.Fatalf("%s in %s: Preprocess: %v", tt.name, tt.edition, err)
		}
		id, err := b.Match(theCard)
		if err != nil {
			t.Fatalf("%s in %s: Match: %v", tt.name, tt.edition, err)
		}
		co, err := b.GetUUID(id)
		if err != nil {
			t.Fatal(err)
		}
		if co.Number != tt.wantNumber {
			t.Errorf("%s in %s: landed on %s, want %s", tt.name, tt.edition, co, tt.wantNumber)
		}
	}
}
