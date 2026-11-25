package magic

import (
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
)

// TestAliasEditionProjectsAdjustEdition pins AliasEdition to its definition:
// AdjustEdition run over a card carrying nothing but an edition. The corpus
// is every set name and code, every alias the edition table knows, and the
// decorated spellings storefronts hang on them, so a rule added to one path
// and not the other fails here instead of drifting.
func TestAliasEditionProjectsAdjustEdition(t *testing.T) {
	realDatastore(t)
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
		card := matcher.InputCard{Edition: edition}
		rules.AdjustEdition(testBackend, &card)
		alias := rules.AliasEdition(testBackend, edition)
		if alias != card.Edition {
			t.Errorf("AliasEdition(%q) = %q, but AdjustEdition projects %q",
				edition, alias, card.Edition)
		}
	}
	t.Logf("projected %d editions", len(corpus))
}

// TestCommanderEditionLongestKeyword pins the set an edition holding several
// commander keywords resolves to, over enough calls to see an unstable one.
func TestCommanderEditionLongestKeyword(t *testing.T) {
	b := &matcher.Backend{CommanderKeywordMap: map[string]string{
		"Darkhold":     "Darkhold Destiny Commander",
		"Eldraine":     "Wilds of Eldraine Commander",
		"Horizons 3":   "Modern Horizons 3 Commander",
		"Super Heroes": "Marvel Super Heroes Commander",
		"Star Trek":    "Star Trek Commander",
	}}
	for _, tt := range []struct {
		edition, want string
	}{
		{"Marvel Super Heroes: Darkhold Destiny Commander", "Marvel Super Heroes Commander"},
		{"Marvel: Darkhold Destiny Commander", "Darkhold Destiny Commander"},
		{"Marvel: Darkhold Destiny Commander Variants", "Darkhold Destiny Commander"},
		{"Marvel Super Heroes Commander", "Marvel Super Heroes Commander"},
		{"Modern Horizons 3 Commander: Variants", "Modern Horizons 3 Commander"},
		{"Star Trek Commander", "Star Trek Commander"},
		{"Eldraine Darkhold Commander", "Darkhold Destiny Commander"},
	} {
		for range 100 {
			got := Rules{}.AliasEdition(b, tt.edition)
			if got != tt.want {
				t.Errorf("AliasEdition(%q) = %q, want %q", tt.edition, got, tt.want)
				break
			}
		}
	}
}

// TestCommanderKeywords pins the sets the loader finds by a phrase, which a
// set sharing only their longest word does not answer to.
func TestCommanderKeywords(t *testing.T) {
	realDatastore(t)
	for edition, want := range map[string]string{
		"Commander: Modern Horizons 3": "Modern Horizons 3 Commander",
		"Commander: Star Trek":         "Star Trek Commander",
		"Super Heroes Commander":       "Marvel Super Heroes Commander",
		"Jumpstart Commander":          "Jumpstart Commander",
		"Modern Horizons Commander":    "Modern Horizons Commander",
	} {
		got := Rules{}.AliasEdition(testBackend, edition)
		if got != want {
			t.Errorf("AliasEdition(%q) = %q, want %q", edition, got, want)
		}
	}
}
