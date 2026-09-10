package cardmarket

import (
	cm "github.com/mtgban/go-cardmarket"
)

// processProduct resolves one product and lands its prices, the way the walk
// does for each product of an expansion, for the tests that drive a product
// through both halves at once.
func (mkm *Index) processProduct(channel chan<- responseChan, product *cm.Product) error {
	cardID, cardIDFoil, byName, err := mkm.resolveProduct(product)
	if err != nil || cardID == "" {
		return err
	}
	return mkm.emitPrices(channel, product, cardID, cardIDFoil, byName)
}
