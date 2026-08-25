package starcitygames

import (
	"io"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// withGameDatastore installs another game's datastore for the duration of a
// test and puts the Magic one back afterwards, since the package-level matcher
// holds a single datastore and every other test in this package is a Magic
// one. The test is skipped where that game's datastore is not configured,
// which is how the shared `go test ./...` run sees it.
func withGameDatastore(t *testing.T, env string, load func(io.Reader) (*mtgmatcher.Backend, error)) {
	t.Helper()
	path := os.Getenv(env)
	if path == "" {
		t.Skipf("Need %s set to run this test", env)
	}
	reader, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	ds, err := load(reader)
	if err != nil {
		t.Fatal(err)
	}
	mtgmatcher.SetGlobalDatastore(ds)

	t.Cleanup(func() {
		allPrintings, err := os.Open(os.Getenv("ALLPRINTINGS5_PATH"))
		if err != nil {
			t.Fatal(err)
		}
		defer allPrintings.Close()
		back, err := magic.Load(allPrintings)
		if err != nil {
			t.Fatal(err)
		}
		mtgmatcher.SetGlobalDatastore(back)
	})
}

func withLorcana(t *testing.T) {
	t.Helper()
	withGameDatastore(t, "LORCANA_PATH", lorcana.Load)
}

// TestResolveLorcanaPromoSeries covers the promo skus whose number segment
// opens with the series that issued the card. The series is a heading rather
// than part of the number, and reading it as one left the card with no number
// at all, so every printing of the name aliased.
func TestResolveLorcanaPromoSeries(t *testing.T) {
	withLorcana(t)

	for _, tt := range []struct {
		sku, name, finish, number, wantSet, wantNum string
	}{
		{"SGL-LOR-PRM-P3_031-ENC", "Beast - Snowfield Troublemaker", "Foil", "P3_031", "11", "31"},
		{"SGL-LOR-PRM-P3_033-ENN", "Tinker Bell - Snowflake Collector", "Non-foil", "P3_033", "11", "33"},
		{"SGL-LOR-PRM-P3_034-ENK", "Tinker Bell - Snowflake Collector", "Inkwash Foil", "P3_034", "11", "34"},
		{"SGL-LOR-PRM-P02_025-ENA", "Lilo - Escape Artist", "Glimmer Foil", "P02_025", "6", "25"},
		{"SGL-LOR-PRM-P03_036-ENA", "Scrooge McDuck - S.H.U.S.H. Agent", "Glimmer Foil", "P03_036", "10", "36"},
	} {
		t.Run(tt.sku, func(t *testing.T) {
			group := "Alt Foil"
			if tt.finish == "Non-foil" {
				group = "Non-foil"
			}
			id, err := resolveProduct(GameLorcana, CatalogProduct{
				SKU: tt.sku, Name: tt.name, Set: "Promotional Cards",
				CollectorNumber: tt.number, Finish: tt.finish, FinishGroup: group,
			})
			if err != nil {
				t.Fatalf("resolveProduct: %v", err)
			}
			co, cerr := mtgmatcher.GetUUID(id)
			if cerr != nil {
				t.Fatalf("GetUUID(%q): %v", id, cerr)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("got %s #%s, want %s #%s", co.SetCode, co.Number, tt.wantSet, tt.wantNum)
			}
		})
	}
}
