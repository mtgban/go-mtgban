package main

import (
	"slices"
	"strings"
	"testing"
)

// allGames is every game mtgmatcher registers plus Magic, which is not in
// that list because bantool reads it off a scraper's name rather than a
// registered loader.
var allGames = []string{
	"magic", "fleshandblood", "gundam", "lorcana", "onepiece",
	"palworld", "pokemon", "riftbound", "yugioh",
}

// TestOptionsSupportsExactlyItsRegisteredGames pins every store's declared
// Supports directly - the one thing run() actually reads to decide whether
// a game reaches Init at all, and nil on any entry here would silently
// refuse it for every game rather than the one or few it should.
func TestOptionsSupportsExactlyItsRegisteredGames(t *testing.T) {
	want := map[string][]string{
		"abugames":               {"magic"},
		"abugames_sealed":        {"magic"},
		"arcanafrisia":           {"magic"},
		"cardkingdom":            {"magic"},
		"cardkingdom_graded":     {"magic"},
		"cardkingdom_sealed":     {"magic"},
		"hareruya":               {"magic"},
		"hareruya_sealed":        {"magic"},
		"magiccorner":            {"magic"},
		"manaleak":               {"magic"},
		"manapool":               {"magic"},
		"manapool_index":         {"magic"},
		"manapool_sealed":        {"magic"},
		"mintcard":               {"magic"},
		"mtgseattle":             {"magic"},
		"sealed_ev":              {"magic"},
		"trollandtoad":           {"magic"},
		"merlion":                {"riftbound"},
		"cardmarket":             {"fleshandblood", "lorcana", "magic", "onepiece", "pokemon", "riftbound", "yugioh"},
		"cardmarket_sealed":      {"fleshandblood", "lorcana", "magic", "onepiece", "pokemon", "riftbound", "yugioh"},
		"cardtrader":             {"fleshandblood", "gundam", "lorcana", "magic", "onepiece", "pokemon", "riftbound", "yugioh"},
		"cardtrader_sealed":      {"fleshandblood", "gundam", "lorcana", "magic", "onepiece", "pokemon", "riftbound", "yugioh"},
		"coolstuffinc":           {"gundam", "lorcana", "magic", "onepiece", "palworld", "pokemon", "riftbound", "yugioh"},
		"coolstuffinc_sealed":    {"lorcana", "magic", "onepiece", "pokemon", "riftbound", "yugioh"},
		"gamenerdz":              {"fleshandblood", "lorcana", "magic", "onepiece", "pokemon"},
		"miniaturemarket_sealed": {"fleshandblood", "gundam", "lorcana", "magic", "onepiece", "riftbound"},
		"starcitygames":          {"fleshandblood", "lorcana", "magic", "riftbound"},
		"starcitygames_sealed":   {"fleshandblood", "lorcana", "magic", "riftbound"},
		"strikezone":             {"fleshandblood", "lorcana", "magic", "pokemon"},
		"tcg_index":              {"fleshandblood", "gundam", "lorcana", "magic", "onepiece", "palworld", "pokemon", "riftbound", "yugioh"},
		"tcg_market":             {"fleshandblood", "gundam", "lorcana", "magic", "onepiece", "palworld", "pokemon", "riftbound", "yugioh"},
		"tcg_sealed":             {"fleshandblood", "gundam", "lorcana", "magic", "onepiece", "palworld", "pokemon", "riftbound", "yugioh"},
		"tcg_syplist":            {"magic", "pokemon"},
		"vegassingles":           {"gundam", "magic", "onepiece", "pokemon", "riftbound"},
	}

	if len(want) != len(options) {
		t.Fatalf("this test names %d stores, options has %d - one was added or removed without updating the other", len(want), len(options))
	}

	for name, opt := range options {
		wantGames, ok := want[name]
		if !ok {
			t.Errorf("%s: not named in this test's expectations", name)
			continue
		}

		got := slices.Clone(opt.Supports)
		slices.Sort(got)
		if !slices.Equal(got, wantGames) {
			t.Errorf("%s.Supports = %v, want %v", name, got, wantGames)
		}
	}
}

// TestMultiGameInitAgreesWithSupports cross-checks a multi-game family's own
// Init against what its Supports declares. Supports is what run() actually
// reads to gate a game before Init ever runs, but every multi-game family
// also keeps checking its own translation table from inside Init - the
// table Supports was copied from - so the two could still drift apart
// silently if only one of them were ever updated. Calling Init with no
// credentials configured at all is what makes this meaningful: every one
// of these constructors checks its table before it reads an env var, so
// "does not support"/"unsupported" and a missing-credential error are
// never confused for one another. Single-game entries have no such table
// of their own to check against - Supports is the only thing that gates
// them - so they are skipped here on the same signal that named them
// single-game in the first place: exactly one supported game.
func TestMultiGameInitAgreesWithSupports(t *testing.T) {
	for name, opt := range options {
		if len(opt.Supports) <= 1 {
			continue
		}

		var got []string
		for _, game := range allGames {
			_, err := opt.Init(game)
			if err == nil || !strings.Contains(err.Error(), "does not support") && !strings.Contains(err.Error(), "unsupported") {
				got = append(got, game)
			}
		}
		slices.Sort(got)

		want := slices.Clone(opt.Supports)
		slices.Sort(want)
		if !slices.Equal(got, want) {
			t.Errorf("%s: Init accepts %v, Supports declares %v", name, got, want)
		}
	}
}
