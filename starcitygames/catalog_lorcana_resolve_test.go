package starcitygames

import (
	"log"
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/lorcana"
)

// withGameDatastore loads another game's datastore for a test, skipped where
// that game's datastore is not configured, which is how the shared
// `go test ./...` run sees it.
func withGameDatastore(t *testing.T, game, env string) *mtgmatcher.Backend {
	t.Helper()
	path := os.Getenv(env)
	if path == "" {
		t.Skipf("Need %s set to run this test", env)
	}
	b, err := datastore.Read(game, path)
	if err != nil {
		t.Fatal(err)
	}
	b.Logger = log.New(os.Stderr, "", 0)
	return b
}

func withLorcana(t *testing.T) *mtgmatcher.Backend {
	t.Helper()
	return withGameDatastore(t, "lorcana", "LORCANA_PATH")
}

// requireSibling skips a case whose premise the installed datastore does not
// hold. The errata printings a sku marker reaches are recent rows, and the
// Lorcana file a checkout carries may predate them; against such a copy the
// marker has nothing to reach and the refusal is the rule working rather than
// the rule broken, so there is nothing here to assert either way.
func requireSibling(t *testing.T, b *mtgmatcher.Backend, name string) {
	t.Helper()
	uuids, err := b.SearchEquals(name)
	if err != nil || len(uuids) == 0 {
		t.Skipf("the installed Lorcana datastore does not carry %q", name)
	}
}

// TestResolveLorcanaMarkedPrinting covers the printing markers Star City Games
// hangs off a Lorcana sku's number. The datastore answers them two ways: it
// numbers the sibling with the same letter, or it numbers both the same and
// tells them apart in the name. Either way the marked sku must not land on the
// base card, whose price it is not - the errata printings sell for twenty
// times the common they were reprinting.
func TestResolveLorcanaMarkedPrinting(t *testing.T) {
	b := withLorcana(t)

	for _, tt := range []struct {
		name                                string
		product                             CatalogProduct
		needs                               string
		wantSet, wantNum, wantName, wantErr string
	}{
		{
			name: "the datastore's own variant letter resolves outright",
			product: CatalogProduct{
				SKU: "SGL-LOR-003-004c-ENN", Name: "Dalmatian Puppy - Tail Wagger",
				Set: "Into the Inklands", CollectorNumber: "004c",
				Finish: "Non-foil", FinishGroup: "Non-foil",
			},
			wantSet: "3", wantNum: "4c",
		},
		{
			name: "a marker the datastore spells in the name reaches that printing",
			product: CatalogProduct{
				SKU: "SGL-LOR-002-073b-ENC", Name: "Bucky - Squirrel Squeak Tutor",
				Set: "Rise of the Floodborn", CollectorNumber: "073b",
				Finish: "Foil", FinishGroup: "Foil",
			},
			needs:    "Bucky - Squirrel Squeak Tutor (Errata Version)",
			wantSet:  "2",
			wantNum:  "73",
			wantName: "Bucky - Squirrel Squeak Tutor (Errata Version)",
		},
		{
			name: "a marker only the sku carries reaches it too",
			product: CatalogProduct{
				SKU: "SGL-LOR-002-039M-ENN", Name: "Elsa - Gloves Off",
				Set: "Rise of the Floodborn", CollectorNumber: "039",
				Finish: "Non-foil", FinishGroup: "Non-foil",
			},
			needs:    "Elsa - Gloves Off (Errata Version)",
			wantSet:  "2",
			wantNum:  "39",
			wantName: "Elsa - Gloves Off (Errata Version)",
		},
		{
			name: "a jumbo print of the card is not the printing a marker names",
			product: CatalogProduct{
				SKU: "SGL-LOR-001-005M-ENC", Name: "Hades - King of Olympus",
				Set: "The First Chapter", CollectorNumber: "005",
				Finish: "Foil", FinishGroup: "Foil",
			},
			needs:   "Hades - King of Olympus (Oversized)",
			wantErr: "no printing beside 1 5 for the sku marker",
		},
		{
			// The jumbo is sold in one foil and nothing else, so adopting it
			// puts a non-foil listing's price on a foil printing as well.
			name: "a jumbo print is refused for a non-foil listing too",
			product: CatalogProduct{
				SKU: "SGL-LOR-001-118M-ENN", Name: "Mulan - Imperial Soldier",
				Set: "The First Chapter", CollectorNumber: "118",
				Finish: "Non-foil", FinishGroup: "Non-foil",
			},
			needs:   "Mulan - Imperial Soldier (Oversized)",
			wantErr: "no printing beside 1 118 for the sku marker",
		},
		{
			name: "a marker no printing answers is refused, not folded onto the base",
			product: CatalogProduct{
				SKU: "SGL-LOR-001-143M-ENC", Name: "Chief Tui - Respected Leader",
				Set: "The First Chapter", CollectorNumber: "143",
				Finish: "Foil", FinishGroup: "Foil",
			},
			wantErr: "no printing beside 1 143 for the sku marker",
		},
		{
			name: "an unmarked number keeps the base printing",
			product: CatalogProduct{
				SKU: "SGL-LOR-002-039-ENN", Name: "Elsa - Gloves Off",
				Set: "Rise of the Floodborn", CollectorNumber: "039",
				Finish: "Non-foil", FinishGroup: "Non-foil",
			},
			wantSet: "2", wantNum: "39",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.needs != "" {
				requireSibling(t, b, tt.needs)
			}
			id, err := resolveProduct(b, GameLorcana, tt.product)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("got id %q err %v, want error %q", id, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveProduct: %v", err)
			}
			co, cerr := b.GetUUID(id)
			if cerr != nil {
				t.Fatalf("GetUUID(%q): %v", id, cerr)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("got %s #%s (%s), want %s #%s", co.SetCode, co.Number, co.Name, tt.wantSet, tt.wantNum)
			}
			// The errata printing shares the base card's set and number and
			// is told apart only by its name, so the number alone cannot say
			// which of the two answered.
			if tt.wantName != "" && co.Name != tt.wantName {
				t.Errorf("got %q, want %q", co.Name, tt.wantName)
			}
		})
	}

	// The two Rise of the Floodborn errata printings the datastore does carry
	// are printings of their own, so the marked sku and the plain one must not
	// share a uuid.
	for _, tt := range []struct {
		needs         string
		plain, marked CatalogProduct
	}{
		{
			needs: "Bucky - Squirrel Squeak Tutor (Errata Version)",
			plain: CatalogProduct{
				SKU: "SGL-LOR-002-073-ENC", Name: "Bucky - Squirrel Squeak Tutor",
				Set: "Rise of the Floodborn", CollectorNumber: "073", Finish: "Foil", FinishGroup: "Foil",
			},
			marked: CatalogProduct{
				SKU: "SGL-LOR-002-073b-ENC", Name: "Bucky - Squirrel Squeak Tutor",
				Set: "Rise of the Floodborn", CollectorNumber: "073b", Finish: "Foil", FinishGroup: "Foil",
			},
		},
		{
			needs: "Elsa - Gloves Off (Errata Version)",
			plain: CatalogProduct{
				SKU: "SGL-LOR-002-039-ENC", Name: "Elsa - Gloves Off",
				Set: "Rise of the Floodborn", CollectorNumber: "039", Finish: "Foil", FinishGroup: "Foil",
			},
			marked: CatalogProduct{
				SKU: "SGL-LOR-002-039M-ENC", Name: "Elsa - Gloves Off",
				Set: "Rise of the Floodborn", CollectorNumber: "039", Finish: "Foil", FinishGroup: "Foil",
			},
		},
	} {
		t.Run("marked and plain stay apart: "+tt.marked.SKU, func(t *testing.T) {
			requireSibling(t, b, tt.needs)
			plainID, err := resolveProduct(b, GameLorcana, tt.plain)
			if err != nil {
				t.Fatalf("plain: %v", err)
			}
			markedID, err := resolveProduct(b, GameLorcana, tt.marked)
			if err != nil {
				t.Fatalf("marked: %v", err)
			}
			if plainID == markedID {
				t.Errorf("both skus resolved to %s", plainID)
			}
		})
	}
}

// TestResolveLorcanaPromoSeries covers the promo skus whose number segment
// opens with the series that issued the card. The series is a heading rather
// than part of the number, and reading it as one left the card with no number
// at all, so every printing of the name aliased.
func TestResolveLorcanaPromoSeries(t *testing.T) {
	b := withLorcana(t)

	for _, tt := range []struct {
		sku, name, finish, number, wantSet, wantNum string
	}{
		{"SGL-LOR-PRM-P3_031-ENC", "Beast - Snowfield Troublemaker", "Foil", "P3_031", "DLPC", "31"},
		{"SGL-LOR-PRM-P3_033-ENN", "Tinker Bell - Snowflake Collector", "Non-foil", "P3_033", "DLPC", "33"},
		{"SGL-LOR-PRM-P3_034-ENK", "Tinker Bell - Snowflake Collector", "Inkwash Foil", "P3_034", "DLPC", "34"},
		{"SGL-LOR-PRM-P02_025-ENA", "Lilo - Escape Artist", "Glimmer Foil", "P02_025", "DLPC", "25"},
		{"SGL-LOR-PRM-P03_036-ENA", "Scrooge McDuck - S.H.U.S.H. Agent", "Glimmer Foil", "P03_036", "DLPC", "36"},
	} {
		t.Run(tt.sku, func(t *testing.T) {
			group := "Alt Foil"
			if tt.finish == "Non-foil" {
				group = "Non-foil"
			}
			id, err := resolveProduct(b, GameLorcana, CatalogProduct{
				SKU: tt.sku, Name: tt.name, Set: "Promotional Cards",
				CollectorNumber: tt.number, Finish: tt.finish, FinishGroup: group,
			})
			if err != nil {
				t.Fatalf("resolveProduct: %v", err)
			}
			co, cerr := b.GetUUID(id)
			if cerr != nil {
				t.Fatalf("GetUUID(%q): %v", id, cerr)
			}
			if co.SetCode != tt.wantSet || co.Number != tt.wantNum {
				t.Errorf("got %s #%s, want %s #%s", co.SetCode, co.Number, tt.wantSet, tt.wantNum)
			}
		})
	}
}

// TestResolveLorcanaPromoTotal covers the collision a bare number cannot
// settle on its own: "Maleficent - Monstrous Dragon" is card 5 of both the
// P1 and the P3 promo pool, so the pool a sku's own prefix names has to reach
// the datastore as the total the card's face prints, or the two alias.
func TestResolveLorcanaPromoTotal(t *testing.T) {
	b := withLorcana(t)

	p01, err := resolveProduct(b, GameLorcana, CatalogProduct{
		SKU: "SGL-LOR-PRM-P01_005-ENK", Name: "Maleficent - Monstrous Dragon",
		Set: "Promotional Cards", CollectorNumber: "005",
		Finish: "Miscellaneous Foil", FinishGroup: "Alt Foil",
	})
	if err != nil {
		t.Fatalf("P01: %v", err)
	}
	p03, err := resolveProduct(b, GameLorcana, CatalogProduct{
		SKU: "SGL-LOR-PRM-P03_005-ENK", Name: "Maleficent - Monstrous Dragon",
		Set: "Promotional Cards", CollectorNumber: "005",
		Finish: "Miscellaneous Foil", FinishGroup: "Alt Foil",
	})
	if err != nil {
		t.Fatalf("P03: %v", err)
	}
	if p01 == p03 {
		t.Errorf("both pools resolved to %s", p01)
	}
	co, cerr := b.GetUUID(p03)
	if cerr != nil {
		t.Fatalf("GetUUID(%q): %v", p03, cerr)
	}
	if co.SetTotal != "P3" {
		t.Errorf("P03 resolved to a %s printing, want one printing P3", co.SetTotal)
	}
}

// TestResolveLorcanaRainbowFoil covers the one treatment a Lorcana printing is
// sold in beside its standard foil. The foil flag alone always picked the
// standard, so the two skus landed on one uuid and one price overwrote the
// other; the catalog's own name for it is what separates them.
func TestResolveLorcanaRainbowFoil(t *testing.T) {
	b := withLorcana(t)

	for _, tt := range []struct {
		name, sku, cardName, set, number, standard string
	}{
		{"a rainbow beside a silver", "SGL-LOR-009-015-ENA", "Ariel - Singing Mermaid", "Fabled", "015", "Foil"},
		{"a rainbow beside a set's own foil", "SGL-LOR-010-020-ENA", "Simba - King in the Making", "Whispers in the Well", "020", "Whisper Foil"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			rainbow, err := resolveProduct(b, GameLorcana, CatalogProduct{
				SKU: tt.sku, Name: tt.cardName, Set: tt.set, CollectorNumber: tt.number,
				Finish: "Rainbow Foil", FinishGroup: "Alt Foil",
			})
			if err != nil {
				t.Fatalf("rainbow: %v", err)
			}
			co, cerr := b.GetUUID(rainbow)
			if cerr != nil {
				t.Fatalf("GetUUID(%q): %v", rainbow, cerr)
			}
			// The datastore names the finishes the way TCGplayer sells
			// them, so the treatment past the standard foil is "holofoil";
			// "rainbowpillars" is the spelling that reaches it, not the
			// name it wears.
			if co.Finish != "holofoil" {
				t.Errorf("rainbow foil resolved to the %q printing, want holofoil", co.Finish)
			}

			standard, err := resolveProduct(b, GameLorcana, CatalogProduct{
				SKU: tt.sku, Name: tt.cardName, Set: tt.set, CollectorNumber: tt.number,
				Finish: tt.standard, FinishGroup: "Foil",
			})
			if err != nil {
				t.Fatalf("standard: %v", err)
			}
			if standard == rainbow {
				t.Errorf("both treatments resolved to %s", rainbow)
			}
		})
	}

	// A printing sold in one foil only is untouched by the name: it has no
	// sibling treatment to reach, so the name lands on the only foil it has.
	// Checked against the printing's own finishes rather than by naming one,
	// because that single foil is itself sold as a Holofoil - the finish a
	// two-foil card's treatment wears.
	id, err := resolveProduct(b, GameLorcana, CatalogProduct{
		SKU: "SGL-LOR-010b-242-ENA", Name: "Hades - Looking for a Deal",
		Set: "Whispers in the Well", CollectorNumber: "242",
		Finish: "Rainbow Foil", FinishGroup: "Alt Foil",
	})
	if err != nil {
		t.Fatalf("foil-only printing: %v", err)
	}
	co, cerr := b.GetUUID(id)
	if cerr != nil {
		t.Fatalf("foil-only printing: GetUUID(%q): %v", id, cerr)
	}
	for name, uuid := range co.FoilUUIDs {
		if uuid != id {
			t.Errorf("foil-only printing: %q reaches %s, but the name resolved to %s", name, uuid, id)
		}
	}
}
