package main

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
)

// TestRunGame pins that a run names one game or refuses: one datastore is
// loaded, so scrapers of two games cannot share a run.
func TestRunGame(t *testing.T) {
	enabled := func(targets map[mtgban.Game][]string) map[mtgban.Game]map[string]*scraperOption {
		out := map[mtgban.Game]map[string]*scraperOption{
			mtgban.GameYuGiOh: {"cardmarket": {}},
		}
		for game, names := range targets {
			if out[game] == nil {
				out[game] = map[string]*scraperOption{}
			}
			for _, name := range names {
				out[game][name] = &scraperOption{Enabled: true}
			}
		}
		return out
	}
	for _, tt := range []struct {
		desc    string
		options map[mtgban.Game]map[string]*scraperOption
		want    mtgban.Game
		wantErr bool
	}{
		{"one game", enabled(map[mtgban.Game][]string{
			mtgban.GamePokemon: {"cardmarket", "tcg_syplist"},
		}), mtgban.GamePokemon, false},
		{"magic by default", enabled(map[mtgban.Game][]string{
			mtgban.GameMagic: {"cardmarket", "cardmarket_sealed"},
		}), mtgban.GameMagic, false},
		{"nothing enabled", enabled(nil), "", true},
		{"two games", enabled(map[mtgban.Game][]string{
			mtgban.GamePokemon: {"cardmarket"},
			mtgban.GameMagic:   {"cardmarket"},
		}), "", true},
	} {
		got, err := runGame(tt.options)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("%s: runGame() = %q, %v; want %q, error %v", tt.desc, got, err, tt.want, tt.wantErr)
		}
	}
}

// TestScraperFlagName pins the naming a target is enabled by: the store's own
// name for Magic, and store+"_"+game for every other game, whatever else the
// store's own name is suffixed with.
func TestScraperFlagName(t *testing.T) {
	for _, tt := range []struct {
		game mtgban.Game
		name string
		want string
	}{
		{mtgban.GameMagic, "cardmarket", "cardmarket"},
		{mtgban.GameMagic, "cardmarket_sealed", "cardmarket_sealed"},
		{mtgban.GameMagic, "tcg_index", "tcg_index"},
		{mtgban.GameMagic, "sealed_ev", "sealed_ev"},
		{mtgban.GameMagic, "cardkingdom_graded", "cardkingdom_graded"},
		{mtgban.GamePokemon, "cardmarket", "cardmarket_pokemon"},
		{mtgban.GamePokemon, "cardmarket_sealed", "cardmarket_sealed_pokemon"},
		{mtgban.GameLorcana, "starcitygames_sealed", "starcitygames_sealed_lorcana"},
		{mtgban.GameFleshAndBlood, "tcg_market", "tcg_market_fleshandblood"},
		{mtgban.GameYuGiOh, "cardtrader", "cardtrader_yugioh"},
	} {
		if got := scraperFlagName(tt.game, tt.name); got != tt.want {
			t.Errorf("scraperFlagName(%q, %q) = %q, want %q", tt.game, tt.name, got, tt.want)
		}
	}
}

// TestFlattenOptionsSharesPointers pins that the flat view is the same
// registry seen from outside: enabling a target by its flag name is what
// runGame later reads off the nested map.
func TestFlattenOptionsSharesPointers(t *testing.T) {
	nested := map[mtgban.Game]map[string]*scraperOption{
		mtgban.GameLorcana: {"cardtrader": {}},
	}
	flat := flattenOptions(nested)
	if flat["cardtrader_lorcana"] == nil {
		t.Fatalf("flattenOptions() = %v, want a cardtrader_lorcana entry", flat)
	}
	flat["cardtrader_lorcana"].Enabled = true
	if !nested[mtgban.GameLorcana]["cardtrader"].Enabled {
		t.Error("enabling a target through the flat view left the nested one disabled")
	}
}

// TestFlattenOptionsRefusesCollision pins that two games claiming one flag
// name stop the run rather than silently dropping whichever entry a random
// map iteration wrote first.
func TestFlattenOptionsRefusesCollision(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("flattenOptions() accepted one name registered under two games")
		}
	}()
	flattenOptions(map[mtgban.Game]map[string]*scraperOption{
		mtgban.GameMagic:   {"cardmarket_lorcana": {}},
		mtgban.GameLorcana: {"cardmarket": {}},
	})
}
