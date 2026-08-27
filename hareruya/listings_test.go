package hareruya

import "testing"

// TestListingsFor pins which of the storefront's blocks belong to a lot. The
// lazy endpoint answers one block per lot, so a product sold in several
// conditions has several blocks; pairing on the product alone gave every lot
// of it every other lot's rows, and the quantities summed accordingly.
func TestListingsFor(t *testing.T) {
	rows := func(qty ...int) []Row {
		var out []Row
		for _, q := range qty {
			out = append(out, Row{Quantity: q, Condition: "NM", Price: 100})
		}
		return out
	}
	total := func(rows []Row) int {
		var n int
		for _, row := range rows {
			n += row.Quantity
		}
		return n
	}

	lazy := []LazyResult{
		{ProductID: "111", ProductClass: "1", Rows: rows(2, 3)},
		{ProductID: "111", ProductClass: "2", Rows: rows(5)},
		{ProductID: "222", ProductClass: "", Rows: rows(7)},
		{ProductID: "333", ProductClass: "1", Rows: rows(11)},
	}

	for _, tt := range []struct {
		desc    string
		product Product
		want    int
	}{
		{
			desc:    "a lot takes its own block and not its sibling's",
			product: Product{Product: "111", ProductClass: "1"},
			want:    5,
		},
		{
			desc:    "and the sibling takes the other",
			product: Product{Product: "111", ProductClass: "2"},
			want:    5,
		},
		{
			desc:    "a product with one lot is answered by a block that names none",
			product: Product{Product: "222", ProductClass: "1"},
			want:    7,
		},
		{
			desc:    "a product with no block of its own gets nothing",
			product: Product{Product: "444", ProductClass: "1"},
			want:    0,
		},
		{
			desc:    "a lot the storefront did not answer for gets nothing",
			product: Product{Product: "333", ProductClass: "9"},
			want:    0,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := total(listingsFor(tt.product, lazy)); got != tt.want {
				t.Errorf("listingsFor(%s/%s) totals %d, want %d",
					tt.product.Product, tt.product.ProductClass, got, tt.want)
			}
		})
	}

	// The whole page has to add up to what the storefront said, which is the
	// property the id-only join broke: it made 111 worth 10 twice.
	var all int
	for _, p := range []Product{
		{Product: "111", ProductClass: "1"},
		{Product: "111", ProductClass: "2"},
		{Product: "222", ProductClass: "1"},
		{Product: "333", ProductClass: "1"},
	} {
		all += total(listingsFor(p, lazy))
	}
	if want := 2 + 3 + 5 + 7 + 11; all != want {
		t.Errorf("the page totals %d, want %d", all, want)
	}
}
