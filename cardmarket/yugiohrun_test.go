package cardmarket

import "testing"

// TestYugiohRun pins how the index Cardmarket appends to a Yu-Gi-Oh product
// name is read. A set printed twice sells both runs under one name, one
// collector number and one rarity, so the index is the only thing left; it is
// read the way the catalog's prices bear out most often, and a set that reads
// the other way round is named in the override beside it.
func TestYugiohRun(t *testing.T) {
	for _, tt := range []struct {
		desc, name, expansion, want string
	}{
		{
			"the first index is the run a set keeps in print",
			"Blue-Eyes White Dragon (V.1 - Super Rare)", "Duelist Pack: Kaiba", "Unlimited",
		},
		{
			"the one after it is the scarcer first edition",
			"Blue-Eyes White Dragon (V.2 - Super Rare)", "Duelist Pack: Kaiba", "1st Edition",
		},
		{
			"and so is every one beyond that",
			"Suijin (V.3 - Super Rare)", "Metal Raiders", "1st Edition",
		},
		{
			"a product carrying no index says nothing about its run",
			"Blue-Eyes White Dragon", "Duelist Pack: Kaiba", "",
		},
		{
			"an index the name spells without a rarity beside it is not one",
			"Blue-Eyes White Dragon (V.2)", "Duelist Pack: Kaiba", "",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			product := &MKMProduct{Name: tt.name, ExpansionName: tt.expansion}
			if got := yugiohRun(product); got != tt.want {
				t.Errorf("yugiohRun(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

// TestYugiohRunOverride pins that a set reading the other way round is
// corrected by naming it, which is the whole point of the table: the index
// means something different from one set to the next, and the prices are what
// say which sets those are.
func TestYugiohRunOverride(t *testing.T) {
	const set = "Swapped Set"
	yugiohFirstAtIndexOne[set] = true
	defer delete(yugiohFirstAtIndexOne, set)

	for _, tt := range []struct{ name, want string }{
		{"Card (V.1 - Rare)", "1st Edition"},
		{"Card (V.2 - Rare)", "Unlimited"},
	} {
		product := &MKMProduct{Name: tt.name, ExpansionName: set}
		if got := yugiohRun(product); got != tt.want {
			t.Errorf("with the override, yugiohRun(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}

	// A set nobody named keeps reading the default way.
	product := &MKMProduct{Name: "Card (V.1 - Rare)", ExpansionName: "Ordinary Set"}
	if got := yugiohRun(product); got != "Unlimited" {
		t.Errorf("an unnamed set read as %q, want Unlimited", got)
	}
}
