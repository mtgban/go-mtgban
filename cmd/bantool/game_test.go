package main

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func testOptions() map[mtgmatcher.Game]map[string]*scraperOption {
	return map[mtgmatcher.Game]map[string]*scraperOption{
		mtgmatcher.GameMagic:   {"cardmarket": {}, "starcitygames": {}},
		mtgmatcher.GameLorcana: {"cardmarket": {}, "cardtrader": {}},
	}
}

// TestResolveGame pins -game's validation: empty, unknown, and a match
// returning exactly the game the caller asked for.
func TestResolveGame(t *testing.T) {
	options := testOptions()
	for _, tt := range []struct {
		desc    string
		name    string
		want    mtgmatcher.Game
		wantErr string
	}{
		{"registered", "lorcana", mtgmatcher.GameLorcana, ""},
		{"empty", "", "", "no -game given"},
		{"unknown", "nosuch", "", `unknown game "nosuch"`},
	} {
		got, err := resolveGame(options, tt.name)
		if got != tt.want {
			t.Errorf("%s: resolveGame() = %q, want %q", tt.desc, got, tt.want)
		}
		switch {
		case tt.wantErr == "" && err != nil:
			t.Errorf("%s: resolveGame() error = %v, want nil", tt.desc, err)
		case tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)):
			t.Errorf("%s: resolveGame() error = %v, want it to contain %q", tt.desc, err, tt.wantErr)
		}
	}
}

// TestEnableStores pins how -store, -sellers and -vendors turn on a game's
// scrapers: -store alone enables both halves, -sellers/-vendors pin one,
// and naming the same store through two of the three leaves both standing
// rather than one clearing what the other set.
func TestEnableStores(t *testing.T) {
	t.Run("store enables both halves", func(t *testing.T) {
		scrapers := map[string]*scraperOption{"cardmarket": {}}
		err := enableStores(scrapers, mtgmatcher.GameLorcana, "cardmarket", "", "")
		if err != nil {
			t.Fatal(err)
		}
		opt := scrapers["cardmarket"]
		if !opt.Enabled || opt.OnlySeller || opt.OnlyVendor {
			t.Errorf("got %+v, want enabled with neither half pinned", opt)
		}
	})

	t.Run("sellers pins retail", func(t *testing.T) {
		scrapers := map[string]*scraperOption{"cardmarket": {}}
		err := enableStores(scrapers, mtgmatcher.GameLorcana, "", "cardmarket", "")
		if err != nil {
			t.Fatal(err)
		}
		opt := scrapers["cardmarket"]
		if !opt.Enabled || !opt.OnlySeller || opt.OnlyVendor {
			t.Errorf("got %+v, want OnlySeller", opt)
		}
	})

	t.Run("vendors pins buylist", func(t *testing.T) {
		scrapers := map[string]*scraperOption{"cardmarket": {}}
		err := enableStores(scrapers, mtgmatcher.GameLorcana, "", "", "cardmarket")
		if err != nil {
			t.Fatal(err)
		}
		opt := scrapers["cardmarket"]
		if !opt.Enabled || !opt.OnlyVendor || opt.OnlySeller {
			t.Errorf("got %+v, want OnlyVendor", opt)
		}
	})

	t.Run("store and vendors on the same name leave both standing", func(t *testing.T) {
		scrapers := map[string]*scraperOption{"cardmarket": {}}
		err := enableStores(scrapers, mtgmatcher.GameLorcana, "cardmarket", "", "cardmarket")
		if err != nil {
			t.Fatal(err)
		}
		opt := scrapers["cardmarket"]
		if !opt.Enabled || !opt.OnlyVendor {
			t.Errorf("got %+v, want enabled and OnlyVendor", opt)
		}
	})

	t.Run("unknown store lists what is registered", func(t *testing.T) {
		scrapers := map[string]*scraperOption{"cardmarket": {}, "cardtrader": {}}
		err := enableStores(scrapers, mtgmatcher.GameLorcana, "nosuch", "", "")
		if err == nil || !strings.Contains(err.Error(), "cardmarket") || !strings.Contains(err.Error(), "cardtrader") {
			t.Errorf("enableStores() error = %v, want it to name the registered stores", err)
		}
	})

	t.Run("unknown seller is refused before it is enabled", func(t *testing.T) {
		scrapers := map[string]*scraperOption{"cardmarket": {}}
		err := enableStores(scrapers, mtgmatcher.GameLorcana, "", "nosuch", "")
		if err == nil {
			t.Error("enableStores() = nil, want an error naming the unregistered seller")
		}
	})

	t.Run("nothing given is refused", func(t *testing.T) {
		scrapers := map[string]*scraperOption{"cardmarket": {}}
		err := enableStores(scrapers, mtgmatcher.GameLorcana, "", "", "")
		if err == nil {
			t.Error("enableStores() = nil, want an error naming no store given")
		}
	})
}
