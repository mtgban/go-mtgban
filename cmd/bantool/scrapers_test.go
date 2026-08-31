package main

import (
	"testing"

	"github.com/mtgban/go-mtgban/cardmarket"
	"github.com/mtgban/go-mtgban/cardtrader"
)

// TestCardmarketNeedsItsBridge pins that a Cardmarket target Cardmarket cannot
// identify on its own refuses to be built without the CardTrader bridge, sealed
// as well as singles.
//
// The scraper can price by name where the bridge is missing, and that is the
// weaker answer, not the same one: the bridge settles a product by an id both
// catalogs carry, while a name reaches about half of what each datastore holds
// and reaches it on a spelling. A run that quietly delivered the weaker answer
// would be a run nobody was told about, so the failure is loud instead - Init
// returning an error ends the run.
func TestCardmarketNeedsItsBridge(t *testing.T) {
	t.Setenv("MKM_APP_TOKEN", "token")
	t.Setenv("MKM_APP_SECRET", "secret")
	t.Setenv("CARDTRADER_TOKEN_BEARER", "")

	if _, err := cardmarketSealedScraper(cardmarket.GamePokemon, cardtrader.GamePokemon)(); err == nil {
		t.Error("the sealed scraper was built without a bridge")
	}
	if _, err := cardmarketBridgedIndexScraper(cardmarket.GameYuGiOh, cardtrader.GameYuGiOh)(); err == nil {
		t.Error("the singles scraper was built without a bridge")
	}
}
