package tcgplayer

import (
	"strings"

	"github.com/mtgban/go-tcgplayer"
)

// isUnsupportedProduct reports catalog products that are inserts rather than
// cards the matcher can price. TCGplayer files these code-card products under
// the singles category even though the datastore carries no card identity for
// them.
func isUnsupportedProduct(product *tcgplayer.Product) bool {
	return strings.HasPrefix(product.Name, "Code Card -")
}
