package cardmarket

import "testing"

// TestForeignShelf pins which expansion names are a catalog of their own.
// Cardmarket shelves a set's non-English printings beside the English ones
// under the same card names, and the datastore carries only the English, so a
// price from one of those shelves lands on a printing it is not.
func TestForeignShelf(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		{"Metal Raiders (Japanese)", true},
		{"Metal Raiders (Korean)", true},
		{"Metal Raiders (PMT)", true},
		{"Metal Raiders", false},
		{"Metal Raiders (25th Anniversary Edition)", false},
		{"Legend of Blue Eyes White Dragon (25th Anniversary Edition)", false},
		// The tail has to end the name: a set spelling one of those words
		// somewhere else is still the English catalog.
		{"Japanese Collection Tin", false},
		{"", false},
	} {
		if got := foreignShelf(tt.name); got != tt.want {
			t.Errorf("foreignShelf(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
