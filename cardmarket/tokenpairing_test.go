package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"
)

// TestResolveMagicResolvesTokenPairing pins a two-sided token product
// resolving to the combined entity mtgmatcher/magic derives for it, anchored
// by both face names plus the product's own edition - Cardmarket's own
// Number field ("T 3/6") is a catalog ordinal, not a collector number, so
// unlike Cool Stuff Inc's own feed there is no set+number anchor available
// here. Before this, the product's name went straight into Preprocess/Match,
// which was never built to read "X Token (...) // Y Token (...)", and
// refused every one of these listings outright.
func TestResolveMagicResolvesTokenPairing(t *testing.T) {
	b := realDatastore(t)
	res := &resolver{backend: b, gameID: cm.GameMagic}

	product := &cm.Product{
		IDProduct: 362664, Name: "Manifest Token // Angel Token (W 4/4)",
		ExpansionName: "Commander 2018", Number: "T 12/24",
	}
	cardID, _, err := res.resolveMagic(product)
	if err != nil {
		t.Fatalf("resolveMagic(%d) = %v", product.IDProduct, err)
	}
	if cardID == "" {
		t.Fatal("resolveMagic returned no id, want the derived Manifest // Angel pairing")
	}
	co, err := b.GetUUID(cardID)
	if err != nil {
		t.Fatalf("GetUUID(%s) = %v", cardID, err)
	}
	if co.Identifiers["derivedTokenPair"] != "true" {
		t.Errorf("resolved to %q, want a derived token pairing", co.Name)
	}
	if co.Name != "Manifest // Angel" {
		t.Errorf("resolved to %q, want \"Manifest // Angel\"", co.Name)
	}
}

// TestResolveMagicRefusesAmbiguousTokenPairing pins the reverse: Commander
// 2016 prints more than one "Bird" token and more than one "Spirit" token,
// so a listing naming only "Bird" and "Spirit" cannot say which pairing it
// means - the same collision TestTokenPairIndexCollision already pins at
// the mtgmatcher/magic level. A refusal here is correct: guessing one of
// several candidate pairings would risk pricing the wrong physical product.
func TestResolveMagicRefusesAmbiguousTokenPairing(t *testing.T) {
	b := realDatastore(t)
	res := &resolver{backend: b, gameID: cm.GameMagic}

	product := &cm.Product{
		IDProduct: 294192, Name: "Bird Token (W 1/1) // Spirit Token (W 1/1)",
		ExpansionName: "Commander 2016", Number: "T 2/6",
	}
	cardID, cardIDFoil, err := res.resolveMagic(product)
	if err != nil {
		t.Fatalf("resolveMagic(%d) = %v", product.IDProduct, err)
	}
	if cardID != "" || cardIDFoil != "" {
		t.Fatalf("resolveMagic(%d) = (%q, %q), want both empty: Commander 2016 prints several Bird and Spirit tokens, name+edition alone cannot say which pairing this is", product.IDProduct, cardID, cardIDFoil)
	}
}
