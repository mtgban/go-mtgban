package main

import (
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
// day; any other failure here is a missing secret or resource, which a
// checkout with no credentials is expected to hit.
func TestRegisteredHalvesAreHonoured(t *testing.T) {
	var checked int
	for game, scrapers := range options {
		for key, opt := range scrapers {
			if !opt.OnlyVendor && !opt.OnlySeller {
				continue
			}
			checked++

			half := mtgban.WithRetailOnly()
			if opt.OnlyVendor {
				half = mtgban.WithBuylistOnly()
			}

			backend := &mtgmatcher.Backend{Game: strings.ToLower(string(game))}
			_, err := mtgban.NewScraper(backend, key, mtgban.MapAuthenticator{}, half)
			if err != nil && strings.Contains(err.Error(), "does not implement") {
				t.Errorf("%s/%s cannot honour its own override: %v", game, key, err)
			}
		}
	}
	if checked == 0 {
		t.Skip("no target asks for a single half")
	}
	t.Logf("checked %d targets that ask for a single half", checked)
}
