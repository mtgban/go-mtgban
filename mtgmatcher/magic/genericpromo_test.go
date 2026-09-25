package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestGenericPromoUsesBackendTokenNames pins isGenericPromo reading the
// backend's own token list. The same input is a generic promo against a
// backend that has never heard of it and not one against a backend that
// carries it as a token, so the answer follows the backend the caller
// asked rather than any datastore installed elsewhere.
func TestGenericPromoUsesBackendTokenNames(t *testing.T) {
	in := &mtgmatcher.InputCard{Name: "Test Reward", Variation: "Promo"}
	if !isGenericPromo(&mtgmatcher.Backend{}, in) {
		t.Fatal("a backend with no tokens refused a generic promo")
	}
	if isGenericPromo(&mtgmatcher.Backend{Tokens: []string{in.Name}}, in) {
		t.Fatal("a backend ignored its own token list")
	}
}
