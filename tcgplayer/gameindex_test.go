package tcgplayer

import (
	"testing"

	goTCGPlayer "github.com/mtgban/go-tcgplayer"
)

func TestProductPrintingUsesUnambiguousCatalogFinish(t *testing.T) {
	tcg := TCGGameIndex{
		printings: map[int]string{
			168: "Normal",
			169: "Holofoil",
		},
	}

	product := goTCGPlayer.Product{
		ProductID: 659325,
		Skus: []goTCGPlayer.SKU{
			{ProductID: 659325, PrintingID: 169, ConditionID: 1},
			{ProductID: 659325, PrintingID: 169, ConditionID: 2},
			{ProductID: 659325, PrintingID: 169, ConditionID: 3},
			{ProductID: 659325, PrintingID: 169, ConditionID: 4},
			{ProductID: 659325, PrintingID: 169, ConditionID: 5},
		},
	}
	if got := tcg.productPrinting(&product, "Normal"); got != "Holofoil" {
		t.Fatalf("productPrinting(%d) = %q, want Holofoil", product.ProductID, got)
	}
}

func TestProductPrintingKeepsReportedFinishForMultipleCatalogFinishes(t *testing.T) {
	tcg := TCGGameIndex{
		printings: map[int]string{
			168: "Normal",
			169: "Holofoil",
		},
	}

	product := goTCGPlayer.Product{
		Skus: []goTCGPlayer.SKU{{PrintingID: 168}, {PrintingID: 169}},
	}
	if got := tcg.productPrinting(&product, "Normal"); got != "Normal" {
		t.Fatalf("productPrinting() = %q, want reported Normal", got)
	}
}
