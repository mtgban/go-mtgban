package magic

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// The lights are filed by the loader now rather than carried in a field of
// their own, which makes an unstamped datastore the failure to watch for:
// nothing would error, attraction listings would just stop being told
// apart. The invariant is pinned against the datastore itself rather than a
// handful of names, so a refresh cannot quietly empty it.
func TestAttractionLightsAreFiled(t *testing.T) {
	lit := map[string]bool{}
	printings := 0
	for _, card := range testBackend.Sets["UNF"].Cards {
		c := card
		tag := AttractionLights(&c)
		if tag == "" {
			continue
		}
		printings++
		lit[card.Name] = true
		if !strings.Contains(tag, "/") {
			t.Errorf("%s %s: %q spells no lit set", card.Name, card.Number, tag)
		}
	}
	if printings == 0 {
		t.Fatal("no attraction printings filed; the loader stopped stamping them")
	}
	t.Logf("%d printings across %d attractions", printings, len(lit))

	// Every attraction of a name answers, so a listing naming one is told
	// from its siblings whichever printing the matcher is holding.
	for name := range lit {
		if !hasAttractionPrinting(testBackend, name) {
			t.Errorf("hasAttractionPrinting(%q) = false, want true", name)
		}
	}
	for _, name := range []string{"Island", "Sol Ring"} {
		if hasAttractionPrinting(testBackend, name) {
			t.Errorf("hasAttractionPrinting(%q) = true, want false", name)
		}
	}
}

// The lights tell four otherwise identical printings apart, which is the
// whole reason they are kept: same name, same set, differing only in which
// bulbs are lit.
func TestAttractionLightsTellSiblingsApart(t *testing.T) {
	seen := map[string]string{}
	for _, card := range testBackend.MatchInSet("Balloon Stand", "UNF") {
		c := card
		tag := AttractionLights(&c)
		if tag == "" {
			t.Fatalf("%s has no lights filed", card.Number)
		}
		if other, dupe := seen[tag]; dupe {
			t.Errorf("%s and %s both spell %q", other, card.Number, tag)
		}
		seen[tag] = card.Number
	}
	if len(seen) < 2 {
		t.Fatalf("found %d Balloon Stand printings, want the siblings", len(seen))
	}
}

// A printing the set publishes no lights for carries none, so a plain
// listing is never compared against a lit set it has no business matching.
func TestAttractionLightsAbsentForPlainCards(t *testing.T) {
	card := mtgmatcher.Card{Identifiers: map[string]string{"mtgjsonId": "x"}}
	if tag := AttractionLights(&card); tag != "" {
		t.Errorf("AttractionLights = %q, want empty", tag)
	}
}
