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
// knows keeps its own spelling whichever order it is in. An edition that
// resolves to a set gates the flip: respellName has already asked that set,
// so a flip it could not confirm is refused rather than adopted into the
// wrong set.
func TestAdjustNameTokenOrder(t *testing.T) {
	b := respellBackend()
	for _, name := range []string{"Token: Sheep", "Token: Synthetic Seraphim", "Sky Striker Ace Token"} {
		b.CanonicalNames[mtgmatcher.Normalize(name)] = name
	}

	tests := []struct {
		name    string
		in      string
		edition string
		want    string
	}{
		{"the word moves to the front", "Sheep Token", "", "Token: Sheep"},
		{"a longer name flips whole", "Synthetic Seraphim Token", "", "Token: Synthetic Seraphim"},
		{"a name the catalog knows is left alone", "Sky Striker Ace Token", "", "Sky Striker Ace Token"},
		{"a token the catalog has neither way stays put", "Laval Token", "", "Laval Token"},
		{"a card that is not a token stays put", "Blue-Eyes White Dragon", "", "Blue-Eyes White Dragon"},
		{"a resolved edition that never confirmed the flip refuses it",
			"Sheep Token", "OTS Tournament Pack 9", "Sheep Token"},
		{"an edition naming no set leaves the flip to the catalog",
			"Sheep Token", "Some Unheard-Of Storefront Bucket", "Token: Sheep"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inCard := &mtgmatcher.InputCard{Name: test.in, Edition: test.edition}
			Rules{}.AdjustName(b, inCard)
			if inCard.Name != test.want {
				t.Errorf("AdjustName(%q) = %q, want %q", test.in, inCard.Name, test.want)
			}
		})
	}
}

// respellBackend indexes a handful of real printings the way the loader
// does, just enough for the edition lookups respellName leans on.
func respellBackend() *mtgmatcher.Backend {
	sets := map[string]*mtgmatcher.Set{
		"MFC": {Name: "Magician's Force", Cards: []mtgmatcher.Card{
			{Name: "Vampire Orchis", Number: "MFC-014"},
		}},
		"DASA": {Name: "Dark Saviors", Cards: []mtgmatcher.Card{
			{Name: "Vampiric Orchis", Number: "DASA-EN047"},
			{Name: "Vampiric Koala", Number: "DASA-EN048"},
		}},
		"OP08": {Name: "OTS Tournament Pack 8", Cards: []mtgmatcher.Card{
			{Name: "Token: Sky Striker Ace", Number: "OP08-EN026"},
		}},
		"OP09": {Name: "OTS Tournament Pack 9", Cards: []mtgmatcher.Card{
			{Name: `Token: Mecha Phantom Beast - "Dracossack"`, Number: "OP09-EN026"},
			{Name: `Token: Mecha Phantom Beast - "Harrliard"`, Number: "OP09-EN026"},
			{Name: `Token: Mecha Phantom Beast - "Megaraptor"`, Number: "OP09-EN026"},
		}},
		"OP19": {Name: "OTS Tournament Pack 19", Cards: []mtgmatcher.Card{
			{Name: "Slime Token", Number: "OP19-EN028"},
			{Name: "Mask Token", Number: "OP19-EN029"},
			{Name: "Kuwagata Alpha", Number: "OP19-EN013"},
		}},
		"TP1": {Name: "Tournament Pack 1", Cards: []mtgmatcher.Card{
			{Name: "Kuwagata", Number: "TP1-030"},
		}},
		"SDSA": {Name: "Structure Deck: Sacred Beasts", Cards: []mtgmatcher.Card{
			{Name: "Token: Phantasmal Martyr", Number: "SDSA-EN047"},
			{Name: "Token: Phantasm", Number: "SDSA-EN048"},
		}},
		"LDK2": {Name: "Legendary Decks II", Cards: []mtgmatcher.Card{
			{Name: "Token: Yugi", Number: "LDK2-ENT01"},
			{Name: "Token: Kaiba", Number: "LDK2-ENT02"},
			{Name: "Token: Joey", Number: "LDK2-ENT03"},
		}},
		"UP01": {Name: "Ultimate Tournament Pack 1", Cards: []mtgmatcher.Card{
			{Name: "Token", Number: "UP01-EN050"},
		}},
		"L26D": {Name: "Legendary Modern Decks 2026", Cards: []mtgmatcher.Card{
			{Name: "Sky Striker Ace Token", Number: "L26D-ENS36"},
		}},
	}
	b := &mtgmatcher.Backend{Sets: sets, CanonicalNames: map[string]string{}}
	b.IndexSets()
	return b
}

// TestRespellName pins the edition-guarded respelling both ways round: a
// name takes the spelling its own set files it under, and every guard that
// keeps it from taking anybody else's.
func TestRespellName(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		edition   string
		variation string
		want      string
	}{
		{"cardtrader's Vampiric respells to the set's Vampire",
			"Vampiric Orchis", "Magician's Force", "014 Common", "Vampire Orchis"},
		{"the real DASA Vampiric stays itself",
			"Vampiric Orchis", "Dark Saviors", "047 Super Rare", "Vampiric Orchis"},
		{"the pair reads in the other direction too",
			"Vampire Orchis", "Dark Saviors", "047", "Vampiric Orchis"},
		{"the token flip follows the set's word order",
			"Sky Striker Ace Token", "OTS Tournament Pack 8", "026", "Token: Sky Striker Ace"},
		{"a set naming the token the storefront's way keeps it",
			"Sky Striker Ace Token", "Legendary Modern Decks 2026", "S36", "Sky Striker Ace Token"},
		{"the set's token sheet answers a name the flip cannot",
			"Yugi & Dark Magician Token", "Legendary Decks II", "T01", "Token: Yugi"},
		{"the sheet's bare Token answers by number alone",
			"Laval Token", "Ultimate Tournament Pack 1", "050 Super Rare", "Token"},
		{"several arts under one number decide nothing",
			"Mecha Phantom Beast Token", "OTS Tournament Pack 9", "026", "Mecha Phantom Beast Token"},
		{"a set printing the name outranks a lying number",
			"Mask Token", "OTS Tournament Pack 19", "028", "Mask Token"},
		{"the written-out alpha respells to the set that drops it",
			"Kuwagata Alpha", "Tournament Pack 1", "030", "Kuwagata"},
		{"and the letterless name takes the spelling OTS writes out",
			"Kuwagata", "OTS Tournament Pack 19", "013", "Kuwagata Alpha"},
		{"a set printing the flip elsewhere outranks it too",
			"Phantasm Token", "Structure Deck: Sacred Beasts", "047", "Phantasm Token"},
		{"an edition naming no set decides nothing",
			"Yugi & Dark Magician Token", "Somebody's Binder", "T01", "Yugi & Dark Magician Token"},
	}

	b := respellBackend()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inCard := &mtgmatcher.InputCard{Name: test.in, Edition: test.edition, Variation: test.variation}
			// Through Prefilter rather than respellName itself, so the
			// pipeline hook owning the respelling stays pinned too.
			Rules{}.Prefilter(b, inCard)
			if inCard.Name != test.want {
				t.Errorf("respellName(%q, %q, %q) = %q, want %q",
					test.in, test.edition, test.variation, inCard.Name, test.want)
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
