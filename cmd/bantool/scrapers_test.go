package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

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
	// The singles scraper reads its catalog before it asks for the bridge,
	// so the catalog has to be there for the bridge to be what is missing.
	catalog := filepath.Join(t.TempDir(), "catalog.json")
	err := os.WriteFile(catalog, []byte(`{"data":{"products":{"1":{"expansionId":1,"name":"Blue-Eyes White Dragon"}}}}`), 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("MTGJSON_MKMID_PATH", catalog)

	_, err = cardmarketSealedScraper(cm.GamePokemon, cardtrader.GamePokemon)()
	if err == nil || !strings.Contains(err.Error(), "CARDTRADER_TOKEN_BEARER") {
		t.Errorf("the sealed scraper was built without a bridge: %v", err)
	}
	_, err = cardmarketBridgedIndexScraper(cm.GameYuGiOh, cardtrader.GameYuGiOh)()
	if err == nil || !strings.Contains(err.Error(), "CARDTRADER_TOKEN_BEARER") {
		t.Errorf("the singles scraper was built without a bridge: %v", err)
	}
}
