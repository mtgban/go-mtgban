package tcgplayer

import (
	"fmt"
	"slices"
	"strings"
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
	tcg := &Market{backend: &mtgmatcher.Backend{Game: mtgmatcher.GameMagic}}
	var got []string
	for _, name := range tcg.MarketNames() {
		got = append(got, tcg.InfoForScraper(name).Shorthand)
	}
	want := []string{"TCGPlayer", "TCGDirect", "TCGTrends"}
	if !slices.Equal(got, want) {
		t.Errorf("shorthands %v, want %v", got, want)
	}
}

// skuMissBackend holds the two nonfoil printings skuMissRequests price.
func skuMissBackend() *mtgmatcher.Backend {
	return &mtgmatcher.Backend{Game: mtgmatcher.GameMagic, UUIDs: map[string]*mtgmatcher.CardObject{
		"bolt":  {Card: mtgmatcher.Card{UUID: "bolt", Name: "Lightning Bolt"}},
		"shock": {Card: mtgmatcher.Card{UUID: "shock", Name: "Shock"}},
	}}
}

// skuMissRequests is one batch of two skus, asked for in this order.
var skuMissRequests = []marketChan{
	{UUID: "bolt", Condition: "NEAR MINT", Printing: "NON FOIL", ProductID: 1, SkuID: 10, Language: "ENGLISH"},
	{UUID: "shock", Condition: "NEAR MINT", Printing: "NON FOIL", ProductID: 2, SkuID: 20, Language: "ENGLISH"},
}

// skuMissResults answers skuMissRequests with a row for a sku neither asked
// for between them.
var skuMissResults = []tcgplayer.SKUPriceSet{
	{SKUID: 10, LowestListingPrice: 1.50},
	{SKUID: 99, LowestListingPrice: 500},
	{SKUID: 20, LowestListingPrice: 0.25},
}

// describeSKUEntries names each entry by the card it landed on, who sells
// it, its price and the sku that priced it.
func describeSKUEntries(outs []responseChan) []string {
	var got []string
	for _, out := range outs {
		got = append(got, fmt.Sprintf("%s %s %.2f sku %s", out.cardID, out.entry.SellerName, out.entry.Price, out.entry.InstanceID))
	}
	return got
}

// A price row for a sku the batch never asked for has no card to land on:
// it used to be priced onto the batch's last request.
func TestMarketResultEntriesDropsUnrequestedSKU(t *testing.T) {
	var logs []string
	tcg := &Market{backend: skuMissBackend(), logCallback: func(format string, a ...any) {
		logs = append(logs, fmt.Sprintf(format, a...))
	}}

	got := describeSKUEntries(tcg.resultEntries(skuMissResults, skuMissRequests))
	want := []string{"bolt TCG Player 1.50 sku 10", "shock TCG Player 0.25 sku 20"}
	if !slices.Equal(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	if len(logs) != 1 || !strings.Contains(logs[0], "sku 99") {
		t.Errorf("logs %q, want one naming sku 99", logs)
	}

	// Rows for exactly the skus asked, in any order, map as before
	got = describeSKUEntries(tcg.resultEntries([]tcgplayer.SKUPriceSet{skuMissResults[2], skuMissResults[0]}, skuMissRequests))
	want = []string{"shock TCG Player 0.25 sku 20", "bolt TCG Player 1.50 sku 10"}
	if !slices.Equal(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
}

// The sealed scraper's response had the same gap: an unrequested sku's
// price was filed under an empty card id.
func TestSealedResultEntriesDropsUnrequestedSKU(t *testing.T) {
	var logs []string
	tcg := &Sealed{logCallback: func(format string, a ...any) {
		logs = append(logs, fmt.Sprintf(format, a...))
	}}

	got := describeSKUEntries(tcg.resultEntries(skuMissResults, skuMissRequests))
	want := []string{"bolt  1.50 sku 10", "shock  0.25 sku 20"}
	if !slices.Equal(got, want) {
		t.Errorf("got  %q\nwant %q", got, want)
	}
	if len(logs) != 1 || !strings.Contains(logs[0], "sku 99") {
		t.Errorf("logs %q, want one naming sku 99", logs)
	}
}
