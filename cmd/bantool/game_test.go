package main

import "testing"

// TestScraperGame pins the naming the run's datastore is chosen by: the
// game is the last word of a scraper's name, and a name ending on no game
// is Magic's, whatever else it is suffixed with.
func TestScraperGame(t *testing.T) {
	for _, tt := range []struct {
		name, want string
	}{
		{"cardmarket", "magic"},
		{"cardmarket_sealed", "magic"},
		{"tcg_index", "magic"},
		{"sealed_ev", "magic"},
		{"cardkingdom_graded", "magic"},
		{"cardmarket_pokemon", "pokemon"},
		{"cardmarket_sealed_pokemon", "pokemon"},
		{"starcitygames_sealed_lorcana", "lorcana"},
		{"tcg_syp_fleshandblood", "fleshandblood"},
		{"cardtrader_riftbound", "riftbound"},
	} {
		if got := scraperGame(tt.name); got != tt.want {
			t.Errorf("scraperGame(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}

// TestRunGame pins that a run names one game or refuses: one datastore is
// loaded, so scrapers of two games cannot share a run.
func TestRunGame(t *testing.T) {
	enabled := func(names ...string) map[string]*scraperOption {
		out := map[string]*scraperOption{"cardmarket_yugioh": {}}
		for _, name := range names {
			out[name] = &scraperOption{Enabled: true}
		}
		return out
	}
	for _, tt := range []struct {
		desc    string
		options map[string]*scraperOption
		want    string
		wantErr bool
	}{
		{"one game", enabled("cardmarket_pokemon", "tcg_syp_pokemon"), "pokemon", false},
		{"magic by default", enabled("cardmarket", "cardmarket_sealed"), "magic", false},
		{"nothing enabled", enabled(), "", true},
		{"two games", enabled("cardmarket_pokemon", "cardmarket"), "", true},
	} {
		got, err := runGame(tt.options)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("%s: runGame() = %q, %v; want %q, error %v", tt.desc, got, err, tt.want, tt.wantErr)
		}
	}
}
