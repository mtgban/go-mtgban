package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// TestOnlyGame pins the wrapper every single-game store's Init goes through:
// it answers for the one game it was given and refuses everything else by
// name, without touching the constructor it wraps.
func TestOnlyGame(t *testing.T) {
	var built bool
	build := onlyGame("magic", func() (mtgban.Scraper, error) {
		built = true
		return nil, nil
	})

	if _, err := build("lorcana"); err == nil {
		t.Error("a game other than the one supported was not refused")
	}
	if built {
		t.Error("the wrapped constructor ran for a game it does not support")
	}

	if _, err := build("magic"); err != nil {
		t.Errorf("the supported game was refused: %v", err)
	}
	if !built {
		t.Error("the wrapped constructor did not run for the game it supports")
	}
}

// allGames is every game mtgmatcher registers plus Magic, which is not in
// that list because bantool reads it off a scraper's name rather than a
// registered loader.
var allGames = []string{
	"magic", "fleshandblood", "gundam", "lorcana", "onepiece",
	"palworld", "pokemon", "riftbound", "yugioh",
}

// TestOptionsSupportsExactlyItsRegisteredGames pins which games each store's
// Init answers for, with no credentials configured at all: every constructor
// checks its own game table before it reads an env var, so "does not
// support" and a missing-credential error are never confused for one
// another, and this needs nothing but the table itself to run everywhere.
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
		var got []string
		for _, game := range allGames {
			_, err := opt.Init(game)
			if err == nil || !strings.Contains(err.Error(), "does not support") && !strings.Contains(err.Error(), "unsupported") {
				got = append(got, game)
			}
		}
		slices.Sort(got)
		if !slices.Equal(got, wantGames) {
			t.Errorf("%s supports %v, want %v", name, got, wantGames)
		}
	}
}
