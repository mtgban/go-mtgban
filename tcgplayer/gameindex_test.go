package tcgplayer

import (
	"net/url"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"

	"github.com/mtgban/go-tcgplayer"
)

// TestNewScraperGameIndexSealed pins the sealed index mode's setup. The
// product types are load-bearing twice over: they select what the pages
// list, and Load counts the very same types, so a hardcoded singles type
// there would walk a different result set than the count promised.
func TestNewScraperGameIndexSealed(t *testing.T) {
	cases := []struct {
		name       string
		game       string
		wantSealed bool
		wantTypes  []string
	}{
		{"pokemon_sealed", mtgban.GamePokemon, true, tcgplayer.ProductTypesSealed},
		{"yugioh_sealed", mtgban.GameYuGiOh, true, tcgplayer.ProductTypesSealed},
		{"fleshandblood_sealed", mtgban.GameFleshAndBlood, true, tcgplayer.ProductTypesSealed},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tcg, err := NewScraperGameIndexSealed(tc.game, "public", "private")
			if err != nil {
				t.Fatalf("NewScraperGameIndexSealed(%q) returned %v", tc.game, err)
			}
			if tcg.sealed != tc.wantSealed {
				t.Errorf("sealed = %v, want %v", tcg.sealed, tc.wantSealed)
			}
			if !slices.Equal(tcg.productTypes, tc.wantTypes) {
				t.Errorf("productTypes = %v, want %v", tcg.productTypes, tc.wantTypes)
			}
			if slices.Contains(tcg.productTypes, tcgplayer.ProductTypesSingles[0]) {
				t.Errorf("productTypes = %v, must not carry the singles type", tcg.productTypes)
			}
		})
	}

	t.Run("singles_stays_singles", func(t *testing.T) {
		tcg, err := NewScraperGameIndex(mtgban.GamePokemon, "public", "private")
		if err != nil {
			t.Fatalf("NewScraperGameIndex returned %v", err)
		}
		if tcg.sealed {
			t.Error("sealed = true, want false")
		}
		if !slices.Equal(tcg.productTypes, tcgplayer.ProductTypesSingles) {
			t.Errorf("productTypes = %v, want %v", tcg.productTypes, tcgplayer.ProductTypesSingles)
		}
	})

	t.Run("unsupported_game", func(t *testing.T) {
		_, err := NewScraperGameIndexSealed("tiddlywinks", "public", "private")
		if err == nil {
			t.Error("NewScraperGameIndexSealed on an unsupported game returned no error")
		}
	})
}

// TestGameIndexSealedCardID pins the sealed resolution: the product id is
// the entry's whole identity, and only an id the datastore names exactly
// once can be priced.
func TestGameIndexSealedCardID(t *testing.T) {
	sealedMap := map[int][]string{
		91604:  {"xy-91604"},
		207000: {"first-207000", "second-207000"},
		207001: {},
	}

	cases := []struct {
		name      string
		productID int
		wantID    string
		wantFound bool
	}{
		{"single_uuid", 91604, "xy-91604", true},
		{"ambiguous_id", 207000, "", false},
		{"empty_entry", 207001, "", false},
		{"absent_id", 999999, "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tcg := &TCGGameIndex{sealed: true, sealedMap: sealedMap}
			gotID, gotFound := tcg.sealedCardID(tc.productID)
			if gotID != tc.wantID || gotFound != tc.wantFound {
				t.Errorf("sealedCardID(%d) = (%q, %v), want (%q, %v)",
					tc.productID, gotID, gotFound, tc.wantID, tc.wantFound)
			}
		})
	}
}

// TestGameIndexPriceEntries pins the keep rule that separates this scraper
// from the retail sealed one: any single non-zero index price is enough to
// publish, so a product TCGplayer still quotes but nobody currently lists
// is reported instead of dropped. Each price lands under its own statistic
// name and never as stock.
func TestGameIndexPriceEntries(t *testing.T) {
	cases := []struct {
		name        string
		printing    string
		result      tcgplayer.ProductPriceSet
		wantSellers []string
		wantPrices  []float64
		wantBundle  []bool
	}{
		{
			// pokemon 91604, dropped by the retail sealed scraper
			name:     "market_price_only",
			printing: "",
			result: tcgplayer.ProductPriceSet{
				ProductID: 91604, MarketPrice: 29.98, SubTypeName: "Normal",
			},
			wantSellers: []string{"TCG Market"},
			wantPrices:  []float64{29.98},
			wantBundle:  []bool{false},
		},
		{
			name:     "no_price_at_all",
			printing: "",
			result: tcgplayer.ProductPriceSet{
				ProductID: 91598, SubTypeName: "Normal",
			},
			wantSellers: nil,
			wantPrices:  nil,
			wantBundle:  nil,
		},
		{
			name:     "every_price",
			printing: "Normal",
			result: tcgplayer.ProductPriceSet{
				ProductID:      57163,
				LowPrice:       1.5,
				MarketPrice:    2.5,
				MidPrice:       3.5,
				DirectLowPrice: 4.5,
				SubTypeName:    "Normal",
			},
			wantSellers: []string{"TCG Low", "TCG Market", "TCG Mid", "TCG Direct Low"},
			wantPrices:  []float64{1.5, 2.5, 3.5, 4.5},
			wantBundle:  []bool{false, false, false, true},
		},
		{
			name:     "gap_in_the_middle",
			printing: "",
			result: tcgplayer.ProductPriceSet{
				ProductID: 47605, LowPrice: 60, MidPrice: 70, SubTypeName: "Normal",
			},
			wantSellers: []string{"TCG Low", "TCG Mid"},
			wantPrices:  []float64{60, 70},
			wantBundle:  []bool{false, false},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tcg := &TCGGameIndex{sealed: true}
			entries := tcg.priceEntries("some-uuid", tc.printing, tc.result)
			if len(entries) != len(tc.wantSellers) {
				t.Fatalf("got %d entries, want %d", len(entries), len(tc.wantSellers))
			}
			for i, entry := range entries {
				if entry.key != "some-uuid" {
					t.Errorf("entry %d key = %q, want %q", i, entry.key, "some-uuid")
				}
				if entry.entry.SellerName != tc.wantSellers[i] {
					t.Errorf("entry %d seller = %q, want %q", i, entry.entry.SellerName, tc.wantSellers[i])
				}
				if entry.entry.Price != tc.wantPrices[i] {
					t.Errorf("entry %d price = %v, want %v", i, entry.entry.Price, tc.wantPrices[i])
				}
				if entry.entry.Bundle != tc.wantBundle[i] {
					t.Errorf("entry %d bundle = %v, want %v", i, entry.entry.Bundle, tc.wantBundle[i])
				}
				// An index price is a statistic, not a listing: nothing
				// downstream may read it as a graded, buyable copy
				if entry.entry.Conditions != "" {
					t.Errorf("entry %d carries condition %q, want none", i, entry.entry.Conditions)
				}
			}
		})
	}
}

// TestGameIndexPriceEntriesURL pins that a sealed product's link does not
// ask for a card finish, since it has none.
func TestGameIndexPriceEntriesURL(t *testing.T) {
	cases := []struct {
		name         string
		printing     string
		wantPrinting string
	}{
		{"sealed_has_no_finish", "", ""},
		{"singles_keep_the_finish", "Foil", "Foil"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tcg := &TCGGameIndex{}
			result := tcgplayer.ProductPriceSet{ProductID: 91604, MarketPrice: 1}
			entries := tcg.priceEntries("some-uuid", tc.printing, result)
			if len(entries) != 1 {
				t.Fatalf("got %d entries, want 1", len(entries))
			}
			link, err := url.Parse(entries[0].entry.URL)
			if err != nil {
				t.Fatalf("parsing %q returned %v", entries[0].entry.URL, err)
			}
			got := link.Query().Get("Printing")
			if got != tc.wantPrinting {
				t.Errorf("Printing = %q, want %q", got, tc.wantPrinting)
			}
		})
	}
}

// TestGameIndexSealedInfo pins that the sealed run publishes under its own
// shorthands. It serves the same three statistics as the singles run, and a
// shared shorthand would have one overwrite the other's output.
func TestGameIndexSealedInfo(t *testing.T) {
	singles := &TCGGameIndex{game: mtgban.GamePokemon}
	sealed := &TCGGameIndex{game: mtgban.GamePokemon, sealed: true}

	if !sealed.Info().SealedMode {
		t.Error("sealed Info().SealedMode = false, want true")
	}
	if singles.Info().SealedMode {
		t.Error("singles Info().SealedMode = true, want false")
	}
	if sealed.Info().Shorthand == singles.Info().Shorthand {
		t.Errorf("both modes report shorthand %q", sealed.Info().Shorthand)
	}

	for _, name := range sealed.MarketNames() {
		t.Run(name, func(t *testing.T) {
			sealedShort := sealed.InfoForScraper(name).Shorthand
			singlesShort := singles.InfoForScraper(name).Shorthand
			if sealedShort == "" {
				t.Fatalf("sealed sub-seller %q has no shorthand", name)
			}
			if sealedShort == singlesShort {
				t.Errorf("sealed and singles sub-seller %q share shorthand %q", name, sealedShort)
			}
			if !sealed.InfoForScraper(name).SealedMode {
				t.Errorf("sealed sub-seller %q is not marked sealed", name)
			}
		})
	}
}
