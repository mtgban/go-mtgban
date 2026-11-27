package cardmarket

import (
	"fmt"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// TestSealedPricedLine pins the line a run ends on: the products the
// inventory holds a price for, by how each was placed. The listings are fed
// to Load's consumer as processProduct sends them: the bundle in both of its
// finishes (one product, two entries), the bridged pack and a print run. The
// box placed by id and the box placed by name are placed and never priced.
func TestSealedPricedLine(t *testing.T) {
	mkm, err := newSealedMatcher(sealedMatchBackend(), sealedMatchBridge)
	if err != nil {
		t.Fatal(err)
	}
	lookup := mkm.newLookup()
	mkm.addPrintRuns(&lookup, sealedMatchCatalog)
	placed := mkm.placeProducts(sealedMatchCatalog, lookup)

	mkm.inventory = mtgban.InventoryRecord{}
	var logs []string
	mkm.logCallback = func(format string, a ...any) {
		logs = append(logs, fmt.Sprintf(format, a...))
	}
	listing := func(uuid, id, article string) responseChan {
		return responseChan{cardID: uuid, entry: mtgban.InventoryEntry{
			Conditions: mtgban.NM, Price: 10, Quantity: 1, SellerName: "seller",
			OriginalID: id, InstanceID: article,
		}}
	}
	for _, result := range []responseChan{
		listing("op02-bundle", "13", "1"),
		listing("op02-bundle-foil", "13", "2"),
		listing("op01-pack", "2", "3"),
		listing("op03-wave1", "7", "4"),
	} {
		mkm.addResult(result)
	}
	if len(mkm.inventory["op02-bundle"]) != 1 || len(mkm.inventory["op02-bundle-foil"]) != 1 {
		t.Fatalf("the bundle holds %v, want an entry per finish", mkm.inventory)
	}

	mkm.printPriced(placed)
	want := []string{"[MKMSealed] Priced 1 products by id, 1 through a bridge, 1 by name"}
	if !slices.Equal(logs, want) {
		t.Errorf("logged %q, want %q", logs, want)
	}
}
