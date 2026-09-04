package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestAliasEditionProjectsAdjustEdition pins AliasEdition to its definition:
// AdjustEdition run over a card carrying nothing but an edition. The corpus
// is every set name and code, every alias the edition table knows, and the
// decorated spellings storefronts hang on them, so a rule added to one path
// and not the other fails here instead of drifting.
func TestAliasEditionProjectsAdjustEdition(t *testing.T) {
	seen := map[string]bool{}
	var corpus []string
	add := func(edition string) {
		if !seen[edition] {
			seen[edition] = true
			corpus = append(corpus, edition)
		}
	}

	var bases []string
	for code, set := range testBackend.Sets {
		bases = append(bases, code, set.Name)
	}
	for from, to := range EditionTable {
		bases = append(bases, from, to)
	}
	bases = append(bases,
		"Ravnica Weekend - Boros",
		"Guild Kit: Azorius",
		"Guild Kit",
		"Chronicles Japanese",
		"Chronicles FBB",
		"Oversized Commander",
		"Phyrexia: All Will Be One Concept Praetors",
		"Time Spiral Timeshifted",
		"Mystery Booster Retro Frame",
		"Secret Lair Drop Series",
		"",
		"   ",
	)
	for _, base := range bases {
		add(base)
		for _, decorated := range []string{
			"Magic: The Gathering - " + base,
			base + ": Extras",
			base + " Collector Booster",
			base + " - Etched",
			"UB: " + base,
			base + " Promos",
			base + " Promo Pack",
		} {
			add(decorated)
		}
	}

	rules := Rules{}
	for _, edition := range corpus {
		card := mtgmatcher.InputCard{Edition: edition}
		rules.AdjustEdition(testBackend, &card)
		alias := rules.AliasEdition(testBackend, edition)
		if alias != card.Edition {
			t.Errorf("AliasEdition(%q) = %q, but AdjustEdition projects %q",
				edition, alias, card.Edition)
		}
	}
	t.Logf("projected %d editions", len(corpus))
}
