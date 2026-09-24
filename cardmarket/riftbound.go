package cardmarket

import (
	"strings"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// bridgeNamesCard reports whether a bridged printing is the card the
// product names, allowing for the datastore's own shortened or misspelt
// wording. It is what keeps a bridge link to a fused-token product (whose
// TCGplayer id names an unrelated dual-faced card) from landing.
func bridgeNamesCard(b *mtgmatcher.Backend, product *cm.Product, cardID string) bool {
	co, err := b.GetUUID(cardID)
	if err != nil {
		return false
	}
	name, _, _ := strings.Cut(product.Name, " (V.")
	return mtgmatcher.CloseName(co.Name, name)
}
