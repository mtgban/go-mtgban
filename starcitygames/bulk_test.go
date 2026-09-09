package starcitygames

import (
	"slices"
	"testing"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/fleshandblood"
)

// TestBulkBuyRates pins every rate the table quotes. The numbers are SCG's,
// read off the sell-your-cards index: for each tier, the one price its
// buying_in_bulk products all share. Changing one here has to be a deliberate
// answer to the shop changing one there.
func TestBulkBuyRates(t *testing.T) {
	for _, tt := range []struct {
		desc           string
		game           int
		rarity, finish string
		want           []float64
	}{
		// Magic. Rare and Mythic Rare part company unfoiled and meet again
		// in foil, which is how the shop files them.
		{"a rare is eight cents, or Unstable's tenth of a cent", GameMagic, "Rare", "Non-foil", []float64{0.001, 0.08}},
		{"in foil it is a dime", GameMagic, "Rare", "Foil", []float64{0.1}},
		{"as is a mythic in foil", GameMagic, "Mythic Rare", "Foil", []float64{0.1}},
		{"an unfoiled mythic is a quarter, and Unstable again", GameMagic, "Mythic Rare", "Non-foil", []float64{0.001, 0.25}},
		{"commons and uncommons share one tier", GameMagic, "Common", "Non-foil", []float64{0.006, 0.007, 0.008}},
		{"and another in foil", GameMagic, "Uncommon", "Foil", []float64{0.006, 0.02}},
		{"special is filed with them", GameMagic, "Special", "Non-foil", []float64{0.006, 0.007, 0.008}},
		{"a promo is three cents either way", GameMagic, "Promo", "Non-foil", []float64{0.03}},
		{"a token is a tenth of a cent", GameMagic, "Token", "Foil", []float64{0.001}},
		{"basic lands have two rates unfoiled", GameMagic, "Basic Land", "Non-foil", []float64{0.01, 0.015}},
		{"and one in foil", GameMagic, "Basic Land", "Foil", []float64{0.01}},

		// Any treatment that is not "Non-foil" is foil to the tiers, so a
		// finish coined after this table was written still lands in one.
		{"an exotic foil is still foil", GameMagic, "Rare", "Surge Foil", []float64{0.1}},
		{"including one nobody has coined yet", GameMagic, "Rare", "Moonlight Foil", []float64{0.1}},

		// Flesh and Blood prices most tiers per card; only these have rates.
		{"a cold foil is a quarter whatever it is", GameFleshAndBlood, "Common", "Cold Foil", []float64{0.25}},
		{"a rare cold foil too, where most of them are", GameFleshAndBlood, "Rare", "Cold Foil", []float64{0.25}},
		{"but a promo cold foil is priced per card, not in bulk", GameFleshAndBlood, "Promo", "Cold Foil", nil},
		{"so is a rainbow foil majestic", GameFleshAndBlood, "Majestic", "Rainbow Foil", []float64{0.25}},
		{"an unfoiled majestic is a nickel", GameFleshAndBlood, "Majestic", "Non-foil", []float64{0.05}},
		{"an ordinary common has no rate at all", GameFleshAndBlood, "Common", "Non-foil", nil},

		{"a Lorcana common is half a cent", GameLorcana, "Common", "Non-foil", []float64{0.005}},
		{"two cents in foil", GameLorcana, "Uncommon", "Foil", []float64{0.02}},
		{"a rare is a nickel", GameLorcana, "Rare", "Non-foil", []float64{0.05}},
		{"a dime in foil", GameLorcana, "Super Rare", "Foil", []float64{0.1}},
		{"a foil legendary is half a dollar", GameLorcana, "Legendary", "Foil", []float64{0.5}},
		{"unfoiled it is a dime", GameLorcana, "Legendary", "Non-foil", []float64{0.1}},
		{"a foil promo is two cents", GameLorcana, "Promo", "Foil", []float64{0.02}},
		{"an epic foil is a quarter", GameLorcana, "Epic", "Epic Foil", []float64{0.25}},
		{"an unfoiled promo has no rate", GameLorcana, "Promo", "Non-foil", nil},

		{"a Riftbound showcase is a quarter", GameRiftbound, "Showcase", "Overnumber Foil", []float64{0.25}},
		{"a rare is a dime either way", GameRiftbound, "Rare", "Non-foil", []float64{0.1}},
		{"a foil uncommon is three cents", GameRiftbound, "Uncommon", "Foil", []float64{0.03}},
		{"unfoiled it has no rate", GameRiftbound, "Uncommon", "Non-foil", nil},

		// Nothing recognised means nothing dropped.
		{"an unknown rarity has no rate", GameMagic, "Masterpiece", "Foil", nil},
		{"nor does a game this scraper does not carry", 99, "Rare", "Non-foil", nil},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got := bulkBuyRates(tt.game, tt.rarity, tt.finish)
			if !slices.Equal(got, tt.want) {
				t.Errorf("bulkBuyRates(%d, %q, %q) = %v, want %v", tt.game, tt.rarity, tt.finish, got, tt.want)
			}
		})
	}
}

// TestBuylistPrice pins which figures are an offer. A price is dropped only
// when it is its own tier's rate: the same $0.08 is a bulk rate on a rare and
// a real bid on a common, which is what a flat threshold could not express.
func TestBuylistPrice(t *testing.T) {
	for _, tt := range []struct {
		desc           string
		game           int
		rarity, finish string
		in             string
		want           float64
		priced, bulk   bool
	}{
		{"a rare at its tier's rate is not an offer", GameMagic, "Rare", "Non-foil", "0.0800", 0, false, true},
		{"the same figure on a common is", GameMagic, "Common", "Non-foil", "0.0800", 0.08, true, false},
		{"a common at its own rate is not", GameMagic, "Common", "Non-foil", "0.006", 0, false, true},
		{"half a cent is under no Magic rate but under the floor", GameMagic, "Common", "Non-foil", "0.005", 0, false, false},
		{"a whole cent clears it", GameMagic, "Common", "Non-foil", "0.01", 0.01, true, false},
		{"a tenth of a cent on a rare is Unstable's rate", GameMagic, "Rare", "Non-foil", "0.001", 0, false, true},
		{"a promo cold foil at the cold foil rate is still an offer", GameFleshAndBlood, "Promo", "Cold Foil", "0.25", 0.25, true, false},
		{"a real bid is kept", GameMagic, "Rare", "Non-foil", "15.00", 15, true, false},
		{"a dollar sign and a comma are read", GameMagic, "Rare", "Non-foil", "$1,250.00", 1250, true, false},
		{"a nickel is a rate for a Lorcana rare", GameLorcana, "Rare", "Non-foil", "0.05", 0, false, true},
		{"and an offer on a Lorcana legendary", GameLorcana, "Legendary", "Non-foil", "0.05", 0.05, true, false},
		{"trailing zeroes do not defeat the comparison", GameFleshAndBlood, "Majestic", "Cold Foil", "0.2500", 0, false, true},

		// Neither an offer nor a rate.
		{"an outright zero is neither", GameMagic, "Rare", "Non-foil", "0", 0, false, false},
		{"nor is an empty field", GameMagic, "Rare", "Non-foil", "", 0, false, false},
		{"nor is a word", GameMagic, "Rare", "Non-foil", "N/A", 0, false, false},
		{"nor is a negative", GameMagic, "Rare", "Non-foil", "-1.00", 0, false, false},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			p := CatalogProduct{Rarity: tt.rarity, Finish: tt.finish}
			got, priced, bulk := buylistPrice(tt.game, p, tt.in)
			if got != tt.want || priced != tt.priced || bulk != tt.bulk {
				t.Errorf("buylistPrice(%d, {%s/%s}, %q) = (%v, %v, %v), want (%v, %v, %v)",
					tt.game, tt.rarity, tt.finish, tt.in, got, priced, bulk, tt.want, tt.priced, tt.bulk)
			}
		})
	}
}

// TestCatalogDropsBulkBuyPrice drives the scraper itself, to show the rate is
// dropped from the buylist without costing the card its retail listing.
func TestCatalogDropsBulkBuyPrice(t *testing.T) {
	withGameDatastore(t, "fleshandblood", "FLESHANDBLOOD_PATH")

	product := func(sellList string) CatalogProduct {
		return CatalogProduct{
			SKU: "SGL-FAB-DYN-092-ENC", Name: "Construct Nitro Mechanoid // Nitro Mechanoid",
			Game: "Flesh and Blood", Set: "Dynasty", ProductType: ProductTypeSingles,
			CollectorNumber: "092", Finish: "Cold Foil", FinishGroup: "Alt Foil",
			Language: "English", Rarity: "Majestic",
			Variants: []CatalogVariant{{
				SKU: "SGL-FAB-DYN-092-ENC-NM", Condition: "Near Mint", Qty: 3,
				Price: "9.99", SellListPrice: sellList,
			}},
		}
	}

	for _, tt := range []struct {
		desc          string
		sellList      string
		wantBuylist   int
		wantBulkRated int
	}{
		{"the cold foil tier's rate is dropped", "0.25", 0, 1},
		{"a figure under a cent is not an offer either", "0.006", 0, 0},
		{"one above the floor and off the rate is", "0.05", 1, 0},
		{"as is a real one", "4.00", 1, 0},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			scg := NewScraper(GameFleshAndBlood, "")
			scg.LogCallback = nil
			scg.processProduct(product(tt.sellList))

			if got := len(scg.inventory); got != 1 {
				t.Fatalf("inventory has %d cards, want 1 — the retail side must be unaffected", got)
			}
			if got := len(scg.buylist); got != tt.wantBuylist {
				t.Errorf("buylist has %d cards, want %d", got, tt.wantBuylist)
			}
			if scg.bulkRated != tt.wantBulkRated {
				t.Errorf("bulkRated = %d, want %d", scg.bulkRated, tt.wantBulkRated)
			}
		})
	}
}
