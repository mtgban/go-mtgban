package tcgplayer

import (
	"fmt"
	"slices"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-tcgplayer"
)

// TestMarketSKUEntries pins which sub-sellers a Magic sku's prices land
// on: the lowest listing always, Direct and the market price only where
// TCGplayer quotes one.
func TestMarketSKUEntries(t *testing.T) {
	b := realDatastore(t)
	uuid, err := b.Match(&mtgmatcher.InputCard{Name: "Lightning Bolt", Edition: "Magic 2010"})
	if err != nil {
		t.Fatal(err)
	}
	tcg := &Market{backend: b}
	req := marketChan{UUID: uuid, Condition: "NEAR MINT", Printing: "NON FOIL", ProductID: 1, SkuID: 376052, Language: "ENGLISH"}

	for _, tt := range []struct {
		desc   string
		result tcgplayer.SKUPriceSet
		want   []string
	}{
		{
			desc:   "no Direct quote",
			result: tcgplayer.SKUPriceSet{SKUID: 376052, LowestListingPrice: 28.61, MarketPrice: 21.65, LowPrice: 26.12},
			want:   []string{"TCG Player NM 28.61", "TCGplayer Trends NM 21.65"},
		},
		{
			desc:   "no sales",
			result: tcgplayer.SKUPriceSet{SKUID: 376052, LowestListingPrice: 28.61},
			want:   []string{"TCG Player NM 28.61"},
		},
		{
			desc:   "every price",
			result: tcgplayer.SKUPriceSet{SKUID: 376052, LowestListingPrice: 28.61, DirectLowPrice: 30, MarketPrice: 21.65},
			want:   []string{"TCG Player NM 28.61", "TCG Direct NM 30.00 +bl", "TCGplayer Trends NM 21.65"},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			var got []string
			for _, out := range tcg.skuEntries(tt.result, req) {
				line := fmt.Sprintf("%s %s %.2f", out.entry.SellerName, out.entry.Conditions, out.entry.Price)
				if out.bl != nil {
					line += " +bl"
				}
				got = append(got, line)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got  %q\nwant %q", got, tt.want)
			}
		})
	}
}

// TestMarketNames pins the shorthands the Magic sub-sellers dump under.
func TestMarketNames(t *testing.T) {
	tcg := &Market{}
	var got []string
	for _, name := range tcg.MarketNames() {
		got = append(got, tcg.InfoForScraper(name).Shorthand)
	}
	want := []string{"TCGPlayer", "TCGDirect", "TCGTrends"}
	if !slices.Equal(got, want) {
		t.Errorf("shorthands %v, want %v", got, want)
	}
}
