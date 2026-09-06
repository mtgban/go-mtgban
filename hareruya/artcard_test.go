package hareruya

import (
	"errors"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestArtCardUnsupported pins that the art cards a set booster carries are
// refused quietly: they are filed in art series sets the datastore does not
// carry, and 2,898 of them named no card in the night of 2026-09-06.
func TestArtCardUnsupported(t *testing.T) {
	for _, product := range []Product{
		{
			ProductName:   "【アート・カード】恐怖昇りの小道(024)[ZNR]",
			ProductNameEN: "【Art Card】Grimclimb Pathway(024)[ZNR]",
			CardName:      "【Art Card】Grimclimb Pathway",
		},
		{
			ProductName:   "【アート・カード】平地(001)[LCI]",
			ProductNameEN: "【Art Card】Plains(001)[LCI]",
			CardName:      "【Art Card】Plains",
		},
	} {
		_, err := Preprocess(product)
		if !errors.Is(err, mtgmatcher.ErrUnsupported) {
			t.Errorf("Preprocess(%q) = %v, want ErrUnsupported", product.CardName, err)
		}
	}
}
