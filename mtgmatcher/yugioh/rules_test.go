package yugioh

import "testing"

// TestSuffixRarity pins the tails cardtrader appends to its bare collector
// numbers. A tail the map does not carry reads as no rarity at all, which
// leaves every printing of the number standing and surfaces as an aliasing
// error - the wording cannot rescue it either, since a rarity named by its
// treatment alone ("Shatterfoil") is missing the label's "Rare".
func TestSuffixRarity(t *testing.T) {
	tests := []struct {
		name   string
		number string
		want   string
	}{
		{"ultra rare", "019u", "Ultra Rare"},
		{"secret rare", "014sec", "Secret Rare"},
		{"quarter century secret rare", "019qsec", "Quarter Century Secret Rare"},
		{"collector's rare", "022cr", "Collector's Rare"},
		{"ultimate rare", "030ul", "Ultimate Rare"},
		{"platinum secret rare", "007psec", "Platinum Secret Rare"},
		{"shatterfoil rare", "010sh", "Shatterfoil Rare"},
		{"the alternate-art tail rides behind a rarity", "019ua", "Ultra Rare"},
		{"and behind a longer one", "019qseca", "Quarter Century Secret Rare"},
		{"a bare alternate-art tail names no rarity", "019a", ""},
		{"a plain number names no rarity", "019", ""},
		{"the misprint reissue tail names no rarity", "EOJ-EN004K", ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := suffixRarity(test.number); got != test.want {
				t.Errorf("suffixRarity(%q) = %q, want %q", test.number, got, test.want)
			}
		})
	}
}
