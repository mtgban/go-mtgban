package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestRegisteredHalvesAreHonoured walks the targets that ask for one half and
// checks each can actually be told to drop the other. A scraper registered
// for an option it cannot answer fails mtgban.NewScraper with "does not
// implement mtgban.ScraperConfig" rather than silently publishing both
// halves, which is how Vegas Singles published an empty Magic shelf twice a
// day. A target that needs a secret cannot be built on a checkout without
// one, and is counted as unverified rather than as honoured; any other
// failure is a fault of its own.
func TestRegisteredHalvesAreHonoured(t *testing.T) {
	var checked, unverified int
	for game, scrapers := range options {
		for key, opt := range scrapers {
			if !opt.OnlyVendor && !opt.OnlySeller {
				continue
			}

			half := mtgban.WithRetailOnly()
			if opt.OnlyVendor {
				half = mtgban.WithBuylistOnly()
			}

			backend := &mtgmatcher.Backend{Game: strings.ToLower(string(game))}
			_, err := mtgban.NewScraper(backend, key, mtgban.MapAuthenticator{}, half)
			switch {
			case err == nil:
				checked++
			case errors.Is(err, mtgban.ErrMissingSecret):
				unverified++
				t.Logf("%s/%s not verified: %v", game, key, err)
			default:
				t.Errorf("%s/%s cannot honour its own override: %v", game, key, err)
			}
		}
	}
	if checked == 0 {
		t.Fatalf("no target asking for a single half could be built (%d unverified)", unverified)
	}
	t.Logf("checked %d targets that ask for a single half, %d unverified", checked, unverified)
}
