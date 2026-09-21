package cardkingdom

import (
	"errors"
	"testing"

	"github.com/mtgban/go-cardkingdom"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestPreprocessPunchCard pins that a punch card reaches the Punchcard row of
// its own sheet, under the name the datastore files it with rather than the
// title CK sells it under or the id the row publishes.
//
// Neither of the other two anchors answers for all of them: the sku's number
// is CK's index rather than the card's, and the scryfall id is missing on
// three, unknown to the datastore on two, and on Lorwyn Eclipsed names the
// Treefolk token instead - which is what the punch card was priced as.
func TestPreprocessPunchCard(t *testing.T) {
	b := realDatastore(t)
	for _, tt := range []struct {
		product         cardkingdom.Product
		setCode, number string
	}{
		{
			product: cardkingdom.Product{
				SKU: "THOU-014", Name: "Hour of Devastation Punch Card",
				Edition: "Hour of Devastation",
			},
			setCode: "THOU", number: "13",
		},
		{
			// CK numbers this one 026, which the sheet happens to
			// agree with - the name still has to carry it
			product: cardkingdom.Product{
				SKU: "TAKH-026", Name: "Amonkhet Punch Card",
				Edition: "Amonkhet",
			},
			setCode: "TAKH", number: "26",
		},
		{
			// The row publishes the Treefolk token's scryfall id, so
			// the id fallback would price it as a Treefolk
			product: cardkingdom.Product{
				SKU: "TECL-0003X", Name: "Lorwyn Eclipsed Punch Card",
				Edition: "Lorwyn Eclipsed",
			},
			setCode: "TECL", number: "13",
		},
		{
			// No scryfall id at all on this one
			product: cardkingdom.Product{
				SKU: "TFIN-0001X", Name: "Final Fantasy Punch Card",
				Edition: "Final Fantasy",
			},
			setCode: "TFIN", number: "37",
		},
	} {
		t.Run(tt.product.SKU, func(t *testing.T) {
			theCard, err := Preprocess(b, tt.product)
			if err != nil {
				t.Fatalf("Preprocess(%v) = %v", tt.product, err)
			}
			cardID, err := b.Match(theCard)
			if err != nil {
				t.Fatalf("Match(%v) = %v, want %s %s", theCard, err, tt.setCode, tt.number)
			}
			co, err := b.GetUUID(cardID)
			if err != nil {
				t.Fatalf("GetUUID(%s) = %v", cardID, err)
			}
			if co.Name != punchcardName || co.SetCode != tt.setCode || co.Number != tt.number {
				t.Errorf("Match(%v) = %s %s %q, want %s %s %q",
					theCard, co.SetCode, co.Number, co.Name, tt.setCode, tt.number, punchcardName)
			}
		})
	}

	// A sheet holding no punch card has no printing for the row to reach,
	// and nothing the log can add: New Capenna Commander's and The
	// Brothers' War's do not.
	for _, product := range []cardkingdom.Product{
		{
			SKU: "TNCC-006X", Name: "Streets of New Capenna Commander Punch Card",
			Variation: "006 // 005", Edition: "Streets of New Capenna Commander Decks",
		},
		{
			SKU: "TBRO-001X", Name: "The Brothers' War Punch Card",
			Edition: "The Brothers' War",
		},
	} {
		t.Run(product.SKU, func(t *testing.T) {
			theCard, err := Preprocess(b, product)
			if !errors.Is(err, mtgmatcher.ErrUnsupported) {
				t.Errorf("Preprocess(%v) = %v, %v, want ErrUnsupported", product, theCard, err)
			}
		})
	}
}
