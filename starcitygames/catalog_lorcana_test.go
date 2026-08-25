package starcitygames

import "testing"

// TestLorcanaNumber pins the collector number a Lorcana sku is read as. The
// sku is the more specific of the two numbers a product carries - it keeps the
// printing marker the number field drops and spells a promo's number under the
// series that issued it - but the product's own number vetoes it when the two
// name different cards, which is what a stray digit and a marker the datastore
// numbers separately both do.
func TestLorcanaNumber(t *testing.T) {
	for _, tt := range []struct {
		name, sku, number, want string
	}{
		{"a plain number is what both agree on", "SGL-LOR-001-036-ENC", "036", "036"},
		{"the printing marker the number field drops rides along",
			"SGL-LOR-002-117M-ENN", "117", "117M"},
		{"a marker both sides write stays", "SGL-LOR-002-073b-ENC", "073b", "073b"},
		{"the datastore's own variant letter stays", "SGL-LOR-003-004a-ENN", "004a", "004a"},
		{"a promo series is a heading, not part of the number",
			"SGL-LOR-PRM-P3_031-ENC", "P3_031", "031"},
		{"a promo series the number field already dropped",
			"SGL-LOR-PRM-P01_005-ENK", "005", "005"},
		{"a stray digit in the sku loses to the number field",
			"SGL-LOR-001-0142-ENN", "042", "042"},
		{"a marker the datastore numbers as a card of its own loses too",
			"SGL-LOR-PRM-P02_032B-ENK", "033", "033"},
		{"a segment naming no number at all leaves the number field to answer",
			"SGL-LOR-008-T03-ENN", "000", "000"},
		{"a sku short of a number segment has nothing to read",
			"SGL-LOR-008", "012", "012"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			p := CatalogProduct{SKU: tt.sku, CollectorNumber: tt.number}
			if got := lorcanaNumber(p); got != tt.want {
				t.Errorf("lorcanaNumber(%q, %q) = %q, want %q", tt.sku, tt.number, got, tt.want)
			}
		})
	}
}
