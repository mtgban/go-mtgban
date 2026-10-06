package cardmarket

import (
	"context"
	"fmt"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgban"
)

// filed builds a product as the walk files it, on an expansion.
func filed(expansion, id int) *cm.Product {
	product := &cm.Product{IDProduct: id}
	product.Expansion.IDExpansion = expansion
	return product
}

// entry builds the one shape the index produces, a single index price for
// one printing.
func entry(price float64, ogID int) mtgban.InventoryEntry {
	return mtgban.InventoryEntry{
		Conditions: mtgban.NM,
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
	const uuid = "lod-005_22583_unlimited"

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
		{
			// The pool walks the expansions in whatever order they
			// finish, and the catalog's order is the one that holds
			// from one run to the next.
			name: "two prices of one printing keep the catalog's order, not the pool's",
			results: []responseChan{
				{ogID: 106410, cardID: uuid, entry: entry(1.5, 106410), product: filed(20, 106410)},
				{ogID: 106409, cardID: uuid, entry: entry(2.5, 106409), product: filed(10, 106409)},
			},
			want: 2.5,
		},
		{
			// 7ED's Scathe Zombies prices its foil column on the
			// Simplified Chinese alternate art, which has its own product.
			name: "the product the datastore files the printing under holds it first",
			results: []responseChan{
				{ogID: 2923, cardID: uuid, entry: entry(5.67, 2923), product: filed(37, 2923)},
				{ogID: 257634, cardID: uuid, entry: entry(0.02, 257634), product: filed(1401, 257634), owned: true},
			},
			want: 0.02,
		},
		{
			name: "and within one expansion the lower product id holds it",
			results: []responseChan{
				{ogID: 106410, cardID: uuid, entry: entry(1.5, 106410), product: filed(10, 106410)},
				{ogID: 106409, cardID: uuid, entry: entry(2.5, 106409), product: filed(10, 106409)},
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
	b := loadFabDatastore(t)

	const uuid = "mon092_237847_1stedition"
	// The bridge knows the second of the two, so the first resolves by name.
	products := []cm.Product{
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

	mkm, err := NewScraperIndex(b)
	if err != nil {
		t.Fatalf("NewScraperIndex(b) = %v", err)
	}
	mkm.exchangeRate = 1
	mkm.maxConcurrency = 1
	mkm.tcgBridge = map[int]int{999001: 237847}
	mkm.priceGuide = map[int]cm.PriceGuide{
		602755: {IDProduct: 602755, LowPrice: 9, TrendPrice: 10},
		999001: {IDProduct: 999001, LowPrice: 1, TrendPrice: 2},
	}

	mkm.collectPrices(context.Background(), []cm.Expansion{{Name: "Monarch - First"}},
		func(_ context.Context, _ cm.Expansion, channel chan<- responseChan) error {
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
// never mistaken for a price. Every price waits for flush, and the count
// flush reports is the named ones.
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
	if added != 0 {
		t.Errorf("prices added before flush = %d, want 0", added)
	}
	if got, _ := collector.flush(); got != 1 {
		t.Errorf("named prices flushed = %d, want 1", got)
	}
	if added != 2 {
		t.Errorf("prices added by flush = %d, want 2", added)
	}
}

// TestNamedLastOnePerPrinting pins that a printing is priced by one product
// in every column: a second product gives way even in a column the first
// has no price in, so the Low and Trend shelves cannot name two products
// for one card.
func TestNamedLastOnePerPrinting(t *testing.T) {
	const uuid = "m3c-223_f"
	low, trend := entry(1, 772984), entry(3, 772984)
	low.SellerName, trend.SellerName = availableIndexNames[0], availableIndexNames[1]
	otherLow, otherTrend := entry(0.5, 774875), entry(4, 774875)
	otherLow.SellerName, otherTrend.SellerName = availableIndexNames[0], availableIndexNames[1]

	for _, tt := range []struct {
		name    string
		results []responseChan
		want    []mtgban.InventoryEntry
		clashes int
	}{
		{
			name: "one product keeps both of its columns",
			results: []responseChan{
				{ogID: 772984, cardID: uuid, entry: low},
				{ogID: 772984, cardID: uuid, entry: trend},
			},
			want: []mtgban.InventoryEntry{low, trend},
		},
		{
			name: "a second product gives way where the first has no price",
			results: []responseChan{
				{ogID: 774875, cardID: uuid, entry: otherLow, byName: true},
				{ogID: 774875, cardID: uuid, entry: otherTrend, byName: true},
				{ogID: 772984, cardID: uuid, entry: trend},
			},
			want:    []mtgban.InventoryEntry{trend},
			clashes: 2,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			inventory := mtgban.InventoryRecord{}
			var heard int
			collector := namedLast{
				add: func(result responseChan) {
					_ = inventory.AddUnique(result.cardID, &result.entry)
				},
				clash: func(result responseChan, held int) {
					heard++
					if held != 772984 {
						t.Errorf("%d gave way to %d, want 772984", result.ogID, held)
					}
				},
			}
			for _, result := range tt.results {
				collector.collect(result)
			}
			collector.flush()

			got := inventory[uuid]
			if len(got) != len(tt.want) {
				t.Fatalf("got %d entries, want %d: %v", len(got), len(tt.want), got)
			}
			for _, want := range tt.want {
				found := false
				for _, entry := range got {
					found = found || (entry.SellerName == want.SellerName && entry.OriginalID == want.OriginalID && entry.Price == want.Price)
				}
				if !found {
					t.Errorf("missing %s from %s at %v", want.SellerName, want.OriginalID, want.Price)
				}
			}
			if collector.clashes != tt.clashes || heard != tt.clashes {
				t.Errorf("clashes = %d, heard %d, want %d", collector.clashes, heard, tt.clashes)
			}
		})
	}
}

// TestCollectPricesOneProductPerMagicPrinting pins the Magic shape behind
// it: the Extras shelf sells Commander: Modern Horizons 3's ripple foils as
// products of their own, named onto the foil the base product's id already
// prices. The base product holds both shelves even where it has no Low,
// whichever product the walk reaches first. Extras is filed ahead of it in
// the catalog's order here, so only the datastore's id can decide it.
func TestCollectPricesOneProductPerMagicPrinting(t *testing.T) {
	b := realDatastore(t)

	const plain, foil = "00a85170-a441-5911-bc5b-626e7e8cb5ec", "00a85170-a441-5911-bc5b-626e7e8cb5ec_f"
	extras := cm.Product{IDProduct: 774875, Name: "Beast Within", Number: "223", ExpansionName: "Commander: Modern Horizons 3: Extras"}
	extras.Expansion.IDExpansion = 1
	base := cm.Product{IDProduct: 772984, Name: "Beast Within", Number: "223", ExpansionName: "Commander: Modern Horizons 3"}
	base.Expansion.IDExpansion = 2

	for _, tt := range []struct {
		name     string
		products []cm.Product
	}{
		{"Extras walked first", []cm.Product{extras, base}},
		{"base walked first", []cm.Product{base, extras}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			// Built by hand: the test datastore is loaded without the game
			// name NewScraperIndex reads.
			mkm := &Index{inventory: mtgban.InventoryRecord{}, exchangeRate: 1, maxConcurrency: 1}
			mkm.gameID = cm.GameMagic
			mkm.resolver.backend = b
			mkm.resolver.printf = mkm.printf
			mkm.priceGuide = map[int]cm.PriceGuide{
				772984: {IDProduct: 772984, LowPrice: 1, TrendPrice: 2, FoilTrendPrice: 3},
				774875: {IDProduct: 774875, FoilLowPrice: 0.5, FoilTrendPrice: 4},
			}

			mkm.collectPrices(context.Background(), []cm.Expansion{{Name: "Commander: Modern Horizons 3"}},
				func(_ context.Context, _ cm.Expansion, channel chan<- responseChan) error {
					for i := range tt.products {
						err := mkm.processProduct(channel, &tt.products[i])
						if err != nil {
							return err
						}
					}
					return nil
				})

			if len(mkm.inventory[plain]) != 2 {
				t.Errorf("got %d entries for the nonfoil, want Low and Trend", len(mkm.inventory[plain]))
			}
			entries := mkm.inventory[foil]
			if len(entries) != 1 {
				t.Fatalf("got %d entries for the foil, want its Trend alone: %v", len(entries), entries)
			}
			if entries[0].OriginalID != "772984" || entries[0].SellerName != availableIndexNames[1] {
				t.Errorf("the foil kept %s from %s, want Trend from 772984", entries[0].SellerName, entries[0].OriginalID)
			}
		})
	}
}
