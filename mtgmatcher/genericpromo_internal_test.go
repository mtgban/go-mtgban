package mtgmatcher

import "testing"

// TestGenericPromoUsesBackendTokenNames pins IsGenericPromo reading the
// receiver's own token list. The same input is a generic promo against a
// backend that has never heard of it and not one against a backend that
// carries it as a token, so the answer follows the backend the caller
// asked rather than any datastore installed elsewhere.
func TestGenericPromoUsesBackendTokenNames(t *testing.T) {
	in := &InputCard{Name: "Test Reward", Variation: "Promo"}
	if !(&Backend{}).IsGenericPromo(in) {
		t.Fatal("a backend with no tokens refused a generic promo")
	}
	if (&Backend{Tokens: []string{in.Name}}).IsGenericPromo(in) {
		t.Fatal("a backend ignored its own token list")
	}
}
