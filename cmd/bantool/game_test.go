package main

import "testing"

// TestScraperFlagName pins the naming every workflow and -scrapers/-sellers/
// -vendors caller already depends on: a Magic target keeps its bare name,
// and every other game's is suffixed with it.
func TestScraperFlagName(t *testing.T) {
	for _, tt := range []struct {
		game, name, want string
	}{
		{"magic", "cardmarket", "cardmarket"},
		{"magic", "cardmarket_sealed", "cardmarket_sealed"},
		{"magic", "tcg_index", "tcg_index"},
		{"pokemon", "cardmarket", "cardmarket_pokemon"},
		{"pokemon", "cardmarket_sealed", "cardmarket_sealed_pokemon"},
		{"lorcana", "starcitygames_sealed", "starcitygames_sealed_lorcana"},
		{"fleshandblood", "tcg_syplist", "tcg_syplist_fleshandblood"},
		{"riftbound", "cardtrader", "cardtrader_riftbound"},
	} {
		if got := scraperFlagName(tt.game, tt.name); got != tt.want {
			t.Errorf("scraperFlagName(%q, %q) = %q, want %q", tt.game, tt.name, got, tt.want)
		}
	}
}

// TestFlattenOptionsKeepsEveryEntryUnderItsFlagName pins the other side of
// the same naming: flattening a nested game:store table has to reach every
// entry, under the name scraperFlagName gives it, and reach it as the same
// *scraperOption the nested table holds - a copy would let the two drift
// apart the moment a flag flipped one and not the other.
func TestFlattenOptionsKeepsEveryEntryUnderItsFlagName(t *testing.T) {
	cardmarket := &scraperOption{}
	cardmarketPokemon := &scraperOption{}
	nested := map[string]map[string]*scraperOption{
		"magic":   {"cardmarket": cardmarket},
		"pokemon": {"cardmarket": cardmarketPokemon},
	}

	flat := flattenOptions(nested)
	if len(flat) != 2 {
		t.Fatalf("flattenOptions() has %d entries, want 2: %v", len(flat), flat)
	}
	if flat["cardmarket"] != cardmarket {
		t.Errorf(`flat["cardmarket"] = %p, want the Magic entry %p`, flat["cardmarket"], cardmarket)
	}
	if flat["cardmarket_pokemon"] != cardmarketPokemon {
		t.Errorf(`flat["cardmarket_pokemon"] = %p, want the Pokemon entry %p`, flat["cardmarket_pokemon"], cardmarketPokemon)
	}

	flat["cardmarket"].Enabled = true
	if !nested["magic"]["cardmarket"].Enabled {
		t.Error("enabling the flattened entry left the nested one untouched")
	}
}

// TestRunGame pins that a run names one game or refuses: one datastore is
// loaded, so scrapers of two games cannot share a run. Every case carries a
// disabled entry in a third game, which a naive count of games present
// (rather than games with something enabled) would misread.
func TestRunGame(t *testing.T) {
	disabledThirdGame := map[string]*scraperOption{"cardmarket": {}}
	for _, tt := range []struct {
		desc    string
		options map[string]map[string]*scraperOption
		want    string
		wantErr bool
	}{
		{
			"one game",
			map[string]map[string]*scraperOption{
				"yugioh":  disabledThirdGame,
				"pokemon": {"cardmarket": {Enabled: true}, "tcg_syplist": {Enabled: true}},
			},
			"pokemon", false,
		},
		{
			"magic by default",
			map[string]map[string]*scraperOption{
				"yugioh": disabledThirdGame,
				"magic":  {"cardmarket": {Enabled: true}, "cardmarket_sealed": {Enabled: true}},
			},
			"magic", false,
		},
		{
			"nothing enabled",
			map[string]map[string]*scraperOption{"yugioh": disabledThirdGame},
			"", true,
		},
		{
			"two games",
			map[string]map[string]*scraperOption{
				"yugioh":  disabledThirdGame,
				"pokemon": {"cardmarket": {Enabled: true}},
				"magic":   {"cardmarket": {Enabled: true}},
			},
			"", true,
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := runGame(tt.options)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Errorf("runGame() = %q, %v; want %q, error %v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

// TestOptionsHasNoCrossGameNameCollision guards the production table against
// the failure mode flattenOptions exists to refuse: two entries under
// different games computing the same scraperFlagName, which would silently
// keep only one of them (flattenOptions panics on that; this pins that the
// real table never reaches it, and that the count comes out whole).
func TestOptionsHasNoCrossGameNameCollision(t *testing.T) {
	var want int
	for _, scrapers := range options {
		want += len(scrapers)
	}
	if got := len(flattenOptions(options)); got != want {
		t.Errorf("flattenOptions(options) has %d entries, want %d", got, want)
	}
}
