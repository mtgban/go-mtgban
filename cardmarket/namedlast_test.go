package cardmarket

import (
	"context"
	"fmt"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// entry builds the one shape the index produces, a single index price for
// one printing.
func entry(price float64, ogID int) mtgban.InventoryEntry {
	return mtgban.InventoryEntry{
		Conditions: "NM",
		Price:      price,
		Quantity:   1,
		SellerName: availableIndexNames[0],
		OriginalID: fmt.Sprint(ogID),
	}
}

// TestNamedLast pins who wins a printing two products both reach. Cardmarket
// sells one expansion per old Yu-Gi-Oh set where the datastore holds a
// printing per print run, so a product the bridge knows and one only a name
// reached land on the same uuid, and AddUnique keeps whichever arrives
// first. The named one has to arrive second whatever order the pool walked
// the catalog in.
func TestNamedLast(t *testing.T) {
	const uuid = "lod-005_22583_unl"

	for _, tt := range []struct {
		name    string
		results []responseChan
		want    float64
	}{
		{
			name: "a named price arriving first still loses the printing",
			results: []responseChan{
				{ogID: 581132, cardID: uuid, entry: entry(2.5, 581132), byName: true},
				{ogID: 106409, cardID: uuid, entry: entry(1.5, 106409)},
			},
			want: 1.5,
		},
		{
			name: "a named price arriving second loses it too",
			results: []responseChan{
				{ogID: 106409, cardID: uuid, entry: entry(1.5, 106409)},
				{ogID: 581132, cardID: uuid, entry: entry(2.5, 581132), byName: true},
			},
			want: 1.5,
		},
		{
			name: "a printing no id reached is still priced by the name",
			results: []responseChan{
				{ogID: 581132, cardID: uuid, entry: entry(2.5, 581132), byName: true},
			},
			want: 2.5,
		},
		{
			name: "two named prices keep the order they arrived in",
			results: []responseChan{
				{ogID: 581132, cardID: uuid, entry: entry(2.5, 581132), byName: true},
				{ogID: 581133, cardID: uuid, entry: entry(3.5, 581133), byName: true},
			},
			want: 2.5,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			inventory := mtgban.InventoryRecord{}
			collector := namedLast{add: func(result responseChan) {
				_ = inventory.AddUnique(result.cardID, &result.entry)
			}}
			for _, result := range tt.results {
				collector.collect(result)
			}
			collector.flush()

			entries := inventory[uuid]
			if len(entries) != 1 {
				t.Fatalf("got %d entries for %s, want 1", len(entries), uuid)
			}
			if entries[0].Price != tt.want {
				t.Errorf("kept price %v, want %v", entries[0].Price, tt.want)
			}
		})
	}
}

// TestCollectPricesDefersNamed pins the wait where Load takes it, over the
// pipeline that produces the collision: two Cardmarket products of one
// printing, the guessed one walked first. The catalog really does sell a
// card twice in one expansion, and the pool walks it in catalog order, so
// without the wait the printing keeps whichever price the catalog happened
// to list first rather than the one an id vouches for.
func TestCollectPricesDefersNamed(t *testing.T) {
	loadFabDatastore(t)

	const uuid = "mon092_237847_1e"
	// The bridge knows the second of the two, so the first resolves by name.
	products := []MKMProduct{
		{
			IDProduct:     602755,
			Name:          "Prismatic Shield (Red) (Regular)",
			Number:        "MON092",
			ExpansionName: "Monarch - First",
		},
		{
			IDProduct:     999001,
			Name:          "Prismatic Shield (Red) (Regular)",
			Number:        "MON092",
			ExpansionName: "Monarch - First",
		},
	}

	mkm := &Index{
		gameID:         GameFleshAndBlood,
		exchangeRate:   1,
		MaxConcurrency: 1,
		inventory:      mtgban.InventoryRecord{},
		TCGBridge:      map[int]int{999001: 237847},
		priceGuide: map[int]PriceGuide{
			602755: {IDProduct: 602755, LowPrice: 9, TrendPrice: 10},
			999001: {IDProduct: 999001, LowPrice: 1, TrendPrice: 2},
		},
	}

	mkm.collectPrices(context.Background(), []MKMExpansion{{Name: "Monarch - First"}},
		func(_ context.Context, _ MKMExpansion, channel chan<- responseChan) error {
			for i := range products {
				err := mkm.processProduct(channel, &products[i])
				if err != nil {
					return err
				}
			}
			return nil
		})

	entries := mkm.inventory[uuid]
	if len(entries) != len(availableIndexNames) {
		t.Fatalf("got %d entries for %s, want %d", len(entries), uuid, len(availableIndexNames))
	}
	for _, entry := range entries {
		if entry.OriginalID != "999001" {
			t.Errorf("%s kept product %s at %v, want 999001", entry.SellerName, entry.OriginalID, entry.Price)
		}
	}
}

// TestCollectTally pins the run's tally riding the results channel: one
// record per edition, summed by the collector on its single goroutine, and
// never mistaken for a price.
func TestCollectTally(t *testing.T) {
	var added int
	collector := namedLast{add: func(responseChan) { added++ }}

	collector.collect(responseChan{cardID: "a", entry: entry(1, 1)})
	collector.collect(responseChan{tally: true, walked: 40, refused: 3})
	collector.collect(responseChan{cardID: "b", entry: entry(2, 2), byName: true})
	collector.collect(responseChan{tally: true, walked: 25, refused: 0})

	if collector.walked != 65 || collector.refused != 3 {
		t.Errorf("tally = %d/%d, want 65/3", collector.walked, collector.refused)
	}
	if added != 1 {
		t.Errorf("prices added before flush = %d, want 1", added)
	}
	if got, _ := collector.flush(); got != 1 {
		t.Errorf("named prices flushed = %d, want 1", got)
	}
}
