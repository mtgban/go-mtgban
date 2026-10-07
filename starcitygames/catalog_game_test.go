package starcitygames

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestGameFromCatalog pins the catalog's own spelling of each game. The
// mapping is the only thing standing between a product and the scraper that
// wants it, and an unrecognized name is indistinguishable from a game we do
// not carry: it maps to 0, the product is skipped, and a scraper configured
// for that game simply finds nothing.
func TestGameFromCatalog(t *testing.T) {
	tests := []struct {
		catalog string
		want    int
	}{
		{"Magic: The Gathering", GameMagic},
		{"Flesh and Blood", GameFleshAndBlood},
		{"Lorcana", GameLorcana},
		{"Disney Lorcana", GameLorcana},
		{"Riftbound", GameRiftbound},
		{"Riftbound: League of Legends TCG", GameRiftbound},
		// Shapes the catalog does not use, kept to show the mapping is exact
		// rather than prefix- or substring-based.
		{"Magic", 0},
		{"Flesh And Blood", 0},
		{"", 0},
	}
	for _, test := range tests {
		t.Run(test.catalog, func(t *testing.T) {
			if got := gameFromCatalog(test.catalog); got != test.want {
				t.Errorf("gameFromCatalog(%q) = %d, want %d", test.catalog, got, test.want)
			}
		})
	}
}

// A catalog naming no product of the scraper's game is a spelling
// gameFromCatalog misses, and must fail the run rather than publish an empty
// dump. Any product type of the game is enough to pass.
func TestLoadRefusesACatalogWithoutItsGame(t *testing.T) {
	tests := []struct {
		name    string
		catalog string
		wantErr bool
	}{
		{"renamed", `[{"sku":"SGL-1","game":"Lorcana TCG","product_type":"Singles"}]`, true},
		{"empty", `[]`, true},
		{"supplies", `[{"sku":"SUP-1","game":"Lorcana","product_type":"Supplies"}]`, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/sets" {
					_, _ = w.Write([]byte(`{"hits":[]}`))
					return
				}
				_, _ = w.Write([]byte(test.catalog))
			}))
			defer srv.Close()

			b := &mtgmatcher.Backend{Game: mtgmatcher.GameLorcana}
			singles, err := NewScraper(b, "")
			if err != nil {
				t.Fatal(err)
			}
			sealed, err := NewScraperSealed(b, "")
			if err != nil {
				t.Fatal(err)
			}
			for _, client := range []*SCGClient{singles.client, sealed.client} {
				client.catalogURL = srv.URL + "/catalog"
				client.setsURL = srv.URL + "/sets"
			}

			for kind, scraper := range map[string]mtgban.Scraper{"singles": singles, "sealed": sealed} {
				err := scraper.Load(context.Background())
				if (err != nil) != test.wantErr {
					t.Errorf("%s Load() = %v, want error %v", kind, err, test.wantErr)
				}
			}
		})
	}
}
