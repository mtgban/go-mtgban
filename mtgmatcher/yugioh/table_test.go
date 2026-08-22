package yugioh

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// aliasBackend holds the set names the cases below need, indexed the way
// AdjustEdition asks for them. Only the set index is consulted, so the
// entries need nothing but their name.
func aliasBackend(t *testing.T, names ...string) *mtgmatcher.Backend {
	t.Helper()
	b := &mtgmatcher.Backend{NormalizedSets: map[string]*mtgmatcher.Set{}}
	for _, name := range names {
		b.NormalizedSets[mtgmatcher.Normalize(name)] = &mtgmatcher.Set{Name: name}
	}
	return b
}

// TestAdjustEditionAliases pins that the table only ever answers for an
// edition no set is named by. A rewrite of a name the datastore carries
// would move every printing under it onto another set.
func TestAdjustEditionAliases(t *testing.T) {
	b := aliasBackend(t,
		"Duelist Pack 11: Crow",
		"Duelist Pack: Crow",
		"Legendary Collection Kaiba",
		"Speed Duel Decks: Ultimate Predators",
		"Yu-Gi-Oh! Championship Series 2025 Prize Cards",
	)

	tests := []struct {
		name    string
		edition string
		want    string
	}{
		{"a storefront name the catalog numbers", "Legendary Collection Kaiba Mega Pack", "Legendary Collection Kaiba"},
		{"a deck the catalog files as a speed duel one", "Starter Deck: Ultimate Predators", "Speed Duel Decks: Ultimate Predators"},
		{"the game-name prefix comes off first", "Yu-Gi-Oh! Legendary Collection Kaiba Mega Pack", "Legendary Collection Kaiba"},
		{"and so does the singles suffix", "Legendary Collection Kaiba Mega Pack Singles", "Legendary Collection Kaiba"},
		{"a set the datastore carries is never rewritten", "Duelist Pack: Crow", "Duelist Pack: Crow"},
		{"a set name carrying the game name is left whole", "Yu-Gi-Oh! Championship Series 2025 Prize Cards", "Yu-Gi-Oh! Championship Series 2025 Prize Cards"},
		{"an edition naming nothing is left as it is", "Some Storefront Heading", "Some Storefront Heading"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inCard := &mtgmatcher.InputCard{Edition: test.edition}
			Rules{}.AdjustEdition(b, inCard)
			if inCard.Edition != test.want {
				t.Errorf("AdjustEdition(%q) = %q, want %q", test.edition, inCard.Edition, test.want)
			}
		})
	}
}

// TestEditionAliasesAreDistinct pins that no two spellings in the table
// normalize to one key, which would leave whichever the map iterated last
// silently deciding for both.
func TestEditionAliasesAreDistinct(t *testing.T) {
	seen := map[string]string{}
	for name := range editionAliases {
		normalized := mtgmatcher.Normalize(name)
		if other, found := seen[normalized]; found {
			t.Errorf("%q and %q both normalize to %q", name, other, normalized)
		}
		seen[normalized] = name
	}
	if len(normalizedEditionAliases()) != len(editionAliases) {
		t.Errorf("indexed %d aliases of %d", len(normalizedEditionAliases()), len(editionAliases))
	}
}
