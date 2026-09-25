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
	if mtgmatcher.CloseName(co.Name, name) {
		return true
	}
	// A promo's name carries what sets it apart, "(JP Exclusive)" and the like.
	bare, decorated := strings.CutSuffix(co.Name, ")")
	if i := strings.LastIndex(bare, " ("); decorated && i > 0 {
		return mtgmatcher.CloseName(bare[:i], name)
	}
	return false
}

// riftboundShelves spells a Cardmarket shelf of promotional prints as the set
// the datastore files them in and the bundle it labels them with; a starred
// number is the bundle's serial-numbered print.
var riftboundShelves = map[string]struct{ edition, label, starLabel string }{
	"T1 2025 Worlds Champion Collection": {
		edition:   "Riftbound Promotional Cards",
		label:     "T1 Worlds Champion Signature Edition Bundle",
		starLabel: "T1 Worlds Champion Signature Edition Bundle Serial Numbered",
	},
}
