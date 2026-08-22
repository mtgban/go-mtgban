package yugioh

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

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

// TestAdjustNameTokenOrder pins the token flip: the catalog writes the word
// first and the storefronts write it last, and a name the datastore already
// knows keeps its own spelling whichever order it is in.
func TestAdjustNameTokenOrder(t *testing.T) {
	b := &mtgmatcher.Backend{CanonicalNames: map[string]string{}}
	for _, name := range []string{"Token: Sheep", "Token: Synthetic Seraphim", "Sky Striker Ace Token"} {
		b.CanonicalNames[mtgmatcher.Normalize(name)] = name
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"the word moves to the front", "Sheep Token", "Token: Sheep"},
		{"a longer name flips whole", "Synthetic Seraphim Token", "Token: Synthetic Seraphim"},
		{"a name the catalog knows is left alone", "Sky Striker Ace Token", "Sky Striker Ace Token"},
		{"a token the catalog has neither way stays put", "Laval Token", "Laval Token"},
		{"a card that is not a token stays put", "Blue-Eyes White Dragon", "Blue-Eyes White Dragon"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inCard := &mtgmatcher.InputCard{Name: test.in}
			Rules{}.AdjustName(b, inCard)
			if inCard.Name != test.want {
				t.Errorf("AdjustName(%q) = %q, want %q", test.in, inCard.Name, test.want)
			}
		})
	}
}
