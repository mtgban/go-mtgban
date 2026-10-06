package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"
)

// A product's printing in another language at the same number is priced
// from its own listings; one an upstream id names at another number is not.
func TestLanguageCopies(t *testing.T) {
	b := realDatastore(t)

	product := &cm.Product{IDProduct: 680699, Name: "Carrion Feeder"}
	cardID, cardIDFoil := Fallback(b, product)
	copies := languageCopies(b, product.IDProduct, cardID, cardIDFoil)
	if len(copies) != 1 {
		t.Fatalf("Carrion Feeder has %d language copies, want 1", len(copies))
	}
	co, err := b.GetUUID(copies[0][0])
	if err != nil {
		t.Fatal(err)
	}
	if co.SetCode != "SLD" || co.Number != "1114jpn" {
		t.Errorf("Carrion Feeder's copy is %s %s, want SLD 1114jpn", co.SetCode, co.Number)
	}

	// MTGJSON files this 2024 promo's product on the 2019 Japanese one too
	product = &cm.Product{IDProduct: 382566, Name: "Shock"}
	cardID, cardIDFoil = Fallback(b, product)
	copies = languageCopies(b, product.IDProduct, cardID, cardIDFoil)
	if len(copies) != 0 {
		t.Errorf("Shock has language copies %v, want none", copies)
	}
}
