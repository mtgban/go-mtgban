package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/cardtrader"
	"github.com/mtgban/go-mtgban/matcher"
)

// TestCardmarketNeedsItsBridge pins that a Cardmarket target Cardmarket cannot
// identify on its own refuses to be built without the CardTrader bridge, all
// three scrapers alike - sealed, singles, and the live-listing market.
//
// The scraper can price by name where the bridge is missing, and that is the
// weaker answer, not the same one: the bridge settles a product by an id both
// catalogs carry, while a name reaches about half of what each datastore holds
// and reaches it on a spelling. A run that quietly delivered the weaker answer
// would be a run nobody was told about, so the failure is loud instead -
// scraperResources returning an error ends the run.
func TestCardmarketNeedsItsBridge(t *testing.T) {
	t.Setenv("CARDTRADER_TOKEN_BEARER", "")
	// The singles and market scrapers read their catalog before they ask for
	// the bridge, so the catalog has to be there for the bridge to be what
	// is missing.
	catalog := filepath.Join(t.TempDir(), "catalog.json")
	err := os.WriteFile(catalog, []byte(`{"data":{"products":{"1":{"expansionId":1,"name":"Blue-Eyes White Dragon"}}}}`), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CARDMARKET_CATALOG_PATH", catalog)

	_, err = scraperResources(&matcher.Backend{Game: matcher.GamePokemon}, "cardmarket_sealed")
	if err == nil || !strings.Contains(err.Error(), "CARDTRADER_TOKEN_BEARER") {
		t.Errorf("sealed was built without a bridge: %v", err)
	}
	_, err = scraperResources(&matcher.Backend{Game: matcher.GameYuGiOh}, "cardmarket")
	if err == nil || !strings.Contains(err.Error(), "CARDTRADER_TOKEN_BEARER") {
		t.Errorf("singles was built without a bridge: %v", err)
	}
	_, err = scraperResources(&matcher.Backend{Game: matcher.GameYuGiOh}, "cardmarket_market")
	if err == nil || !strings.Contains(err.Error(), "CARDTRADER_TOKEN_BEARER") {
		t.Errorf("market was built without a bridge: %v", err)
	}
	for _, game := range []matcher.Game{matcher.GameOnePiece, matcher.GameGundam} {
		for _, store := range []string{"cardmarket", "cardmarket_market"} {
			_, err = scraperResources(&matcher.Backend{Game: game}, store)
			if err == nil || !strings.Contains(err.Error(), "CARDTRADER_TOKEN_BEARER") {
				t.Errorf("%s %s was built without a bridge: %v", game, store, err)
			}
		}
	}
	// Riftbound and Lorcana are named by the catalog, so they go ahead.
	for _, game := range []matcher.Game{matcher.GameRiftbound, matcher.GameLorcana} {
		for _, store := range []string{"cardmarket", "cardmarket_market"} {
			_, err = scraperResources(&matcher.Backend{Game: game}, store)
			if err != nil {
				t.Errorf("%s %s refused without a bridge: %v", game, store, err)
			}
		}
	}
}

// TestCardmarketNeedsACompleteBridge pins that a bridge which lost an
// expansion is refused wherever the bridge is required, and used for what it
// holds everywhere else.
func TestCardmarketNeedsACompleteBridge(t *testing.T) {
	t.Setenv("CARDTRADER_TOKEN_BEARER", "token")
	catalog := filepath.Join(t.TempDir(), "catalog.json")
	err := os.WriteFile(catalog, []byte(`{"data":{"products":{"1":{"expansionId":1,"name":"Blue-Eyes White Dragon"}}}}`), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CARDMARKET_CATALOG_PATH", catalog)

	fetch := blueprintsForGame
	t.Cleanup(func() { blueprintsForGame = fetch })
	blueprintsForGame = func(context.Context, *cardtrader.CTAuthClient, matcher.Game, string, func(string, ...any)) ([]cardtrader.Blueprint, []cardtrader.Expansion, error) {
		blueprints := []cardtrader.Blueprint{{TCGplayerID: 1, CardMarketIDs: []int{2}}}
		return blueprints, nil, fmt.Errorf("%w: lost 1 of 2 expansions", cardtrader.ErrIncomplete)
	}

	for _, test := range []struct {
		store  string
		games  []matcher.Game
		refuse bool
		// opts counts the bridge, and the catalog where the store takes one.
		opts int
	}{
		{"cardmarket", []matcher.Game{matcher.GamePokemon, matcher.GameYuGiOh, matcher.GameFleshAndBlood, matcher.GameOnePiece, matcher.GameGundam}, true, 0},
		{"cardmarket_market", []matcher.Game{matcher.GamePokemon, matcher.GameYuGiOh, matcher.GameFleshAndBlood, matcher.GameOnePiece, matcher.GameGundam}, true, 0},
		{"cardmarket_sealed", []matcher.Game{matcher.GamePokemon, matcher.GameRiftbound, matcher.GameLorcana}, true, 0},
		{"cardmarket", []matcher.Game{matcher.GameRiftbound, matcher.GameLorcana}, false, 2},
		{"cardmarket_market", []matcher.Game{matcher.GameRiftbound, matcher.GameLorcana}, false, 2},
		{"cardgamecorner", []matcher.Game{matcher.GameOnePiece, matcher.GamePokemon, matcher.GameYuGiOh}, false, 1},
	} {
		for _, game := range test.games {
			opts, err := scraperResources(&matcher.Backend{Game: game}, test.store)
			if test.refuse {
				if !errors.Is(err, cardtrader.ErrIncomplete) {
					t.Errorf("%s %s was built on a short bridge: %v", game, test.store, err)
				}
				continue
			}
			if err != nil || len(opts) != test.opts {
				t.Errorf("%s %s dropped a short bridge: %d options, error %v", game, test.store, len(opts), err)
			}
		}
	}
}
