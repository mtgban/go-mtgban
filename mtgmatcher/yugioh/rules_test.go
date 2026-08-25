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

// TestTierByRarity pins the rarity the wording names against the one the
// collector number's suffix encodes. The wording speaks first, and it speaks
// even when the storefront drops the apostrophe the catalog spells the
// rarity with: reading "Collectors Rare" as no rarity at all leaves the
// plain "Rare" candidate standing, since "rare" is a word of that wording
// too.
func TestTierByRarity(t *testing.T) {
	candidates := []mtgmatcher.Card{
		{UUID: "common", Rarity: "Common"},
		{UUID: "rare", Rarity: "Rare"},
		{UUID: "collectors", Rarity: "Collector's Rare"},
		{UUID: "quarter", Rarity: "Quarter Century Secret Rare"},
		{UUID: "secret", Rarity: "Secret Rare"},
	}

	tests := []struct {
		name      string
		variation string
		number    string
		want      string
	}{
		{"the apostrophe the storefront drops", "042cr Collectors Rare", "042cr", "collectors"},
		{"the apostrophe the catalog writes", "042cr Collector's Rare", "042cr", "collectors"},
		{"a wording naming no rarity falls to the suffix", "042cr", "042cr", "collectors"},
		{"a wording naming the rarity needs no suffix", "042 Collectors Rare", "042", "collectors"},
		{"the most specific wording wins", "019qsec Quarter Century Secret Rare", "019qsec", "quarter"},
		{"a plain rarity stays plain", "042 Rare", "042", "rare"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inCard := &mtgmatcher.InputCard{Variation: test.variation}
			var kept []string
			for _, card := range tierByRarity(inCard, candidates, test.number) {
				kept = append(kept, card.UUID)
			}
			if len(kept) != 1 || kept[0] != test.want {
				t.Errorf("tierByRarity(%q) = %v, want [%s]", test.variation, kept, test.want)
			}
		})
	}
}

// TestTokenNameHeldToTheListing pins the two places a token name used to be
// answered with more confidence than the listing earned. A name spelled
// "Token" and nothing else is five printings in three sets, four of them
// wearing a variant label; the variant tiering hands every such listing the
// one printing that wears none, so a dozen different tokens priced the same
// uuid. And the flip onto the catalog's word order is a guess that the
// storefront wrote the two words the other way round, which lands on a
// different card whenever the flipped spelling happens to name one in a set
// the listing does not name.
func TestTokenNameHeldToTheListing(t *testing.T) {
	b := loadBackend(t)

	for _, tt := range []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
		err  string
	}{
		{
			desc: "the bare token name with nothing to identify it",
			in:   mtgmatcher.InputCard{Name: "Token"},
			err:  mtgmatcher.ErrUnsupported.Error(),
		},
		{
			desc: "the bare token name with a number is identified",
			in:   mtgmatcher.InputCard{Name: "Token", Variation: "STP3-EN032"},
			want: "stp3-en032_267706_unl",
		},
		{
			desc: "a flip landing outside the named set is not taken",
			in:   mtgmatcher.InputCard{Name: "Zane Truesdale Token", Edition: "Legendary Duelists: Season 1"},
			err:  mtgmatcher.ErrCardDoesNotExist.Error(),
		},
		{
			desc: "a flip the listing names no set against still stands",
			in:   mtgmatcher.InputCard{Name: "Sheep Token", Variation: "TKN1-EN001"},
			want: "tkn1-en001_81303_unl",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			in := tt.in
			id, err := b.Match(&in)
			gotErr := ""
			if err != nil {
				gotErr = err.Error()
			}
			if id != tt.want || gotErr != tt.err {
				t.Errorf("Match(%q, %q, %q) = (%q, %q), want (%q, %q)",
					tt.in.Name, tt.in.Edition, tt.in.Variation, id, gotErr, tt.want, tt.err)
			}
		})
	}
}
