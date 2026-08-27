package coolstuffinc

import (
	"strings"
	"testing"
)

// TestBuylistVariation pins what the buylist tells the matcher about a
// printing beyond its name. The qualifier lives in a free-text note the sell
// listing spends its variation on, and the numbers written in that note name
// other products rather than this one.
func TestBuylistVariation(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		product CSIPriceEntry
		want    string
	}{
		{"the number alone when there is no note",
			CSIPriceEntry{Number: "001/072"}, "001/072"},
		{"the note carries the qualifier",
			CSIPriceEntry{Number: "074/217", Notes: "Love Ball Foil"}, "074/217 Love Ball Foil"},
		{"a stamped promo says so in the note",
			CSIPriceEntry{Number: "SM198", Notes: "Detective Pikachu Stamped"}, "SM198 Detective Pikachu Stamped"},
		{"the note's own numbers name other products",
			CSIPriceEntry{Number: "SVP107", Notes: "Can be Pikachu 2, 19, 41, or 45"}, "SVP107 Can be Pikachu or"},
		{"a note and no number of its own",
			CSIPriceEntry{Notes: "Prerelease"}, "Prerelease"},
		{"a note describing a reprint names the other set",
			CSIPriceEntry{Number: "24/53", Notes: "25th Anniversary Stamp WOTC Black Star Promo Reprint"}, "24/53"},
		{"and so does one that names a real set",
			CSIPriceEntry{Number: "4/102", Notes: "25th Anniversary Stamp Base Set Reprint"}, "4/102"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := buylistVariation(tt.product); got != tt.want {
				t.Errorf("buylistVariation(%q, %q) = %q, want %q", tt.product.Number, tt.product.Notes, got, tt.want)
			}
		})
	}
}

// TestOfferCondition pins the condition read off an offer row whose tail
// carries the promotions the row is running. A row may run several at once,
// and the flags are laid out in the page's order, not the parser's.
func TestOfferCondition(t *testing.T) {
	const bundle = "Buy 1 get 3 free!"
	for _, tt := range []struct {
		desc                     string
		fullRow, qtyStr, bundleS string
		want                     string
	}{
		{"a plain row is the condition alone",
			"20+ Near Mint$0.29Add to Cart", "20", "", "Near Mint"},
		{"a foil row keeps its finish",
			"6 Foil Near Mint$1.99Add to Cart", "6", "", "Foil Near Mint"},
		{"a sale row drops the price column's lead-in",
			"20+ Near MintWas\u00a0$0.29 Sale\u00a0$0.26Add to Cart", "20", "", "Near Mint"},
		{"a bundle row drops the flag",
			"17Near Mint" + bundle + "$0.49Add to Cart", "17", bundle, "Near Mint"},
		{"a row running both drops both",
			"20+ Near Mint" + bundle + "Was\u00a0$0.29 Sale\u00a0$0.26Add to Cart", "20", bundle, "Near Mint"},
		{"and so does a played foil running both",
			"1 Foil Played" + bundle + "Was\u00a0$1.25 Sale\u00a0$1.13Add to Cart", "1", bundle, "Foil Played"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			if got := offerCondition(tt.fullRow, tt.qtyStr, tt.bundleS); got != tt.want {
				t.Errorf("offerCondition(%q, %q, %q) = %q, want %q", tt.fullRow, tt.qtyStr, tt.bundleS, got, tt.want)
			}
		})
	}
}

// TestBundledCopies pins the price divisor the bundle promotion carries:
// the listed price buys the bought copy and the free ones together, and the
// wording says how many that is.
func TestBundledCopies(t *testing.T) {
	for _, tt := range []struct {
		bundleStr string
		want      int
	}{
		{"Buy 1 get 3 free!", 4},
		{"Buy 1 get 2 free!", 3},
		{"Buy 1 get 1 free!", 2},
		{"", 1},
		{"Was\u00a0", 1},
		{"Buy 2 get 3 free!", 1},
	} {
		if got := bundledCopies(tt.bundleStr); got != tt.want {
			t.Errorf("bundledCopies(%q) = %d, want %d", tt.bundleStr, got, tt.want)
		}
	}
}

// TestBuylistYuGiOhNamesTheRarity pins the rarity onto the buylist
// variation. The tier is the only thing telling apart printings a set
// files at one number, and the sell listing already says it, so a buy
// listing whose note is empty must not ask a narrower question.
func TestBuylistYuGiOhNamesTheRarity(t *testing.T) {
	for _, tt := range []struct {
		desc string
		in   CSIPriceEntry
		want string
	}{
		{
			desc: "an empty note still names the tier",
			in:   CSIPriceEntry{Number: "BP01-EN030", RarityName: "Rare"},
			want: "BP01-EN030 Rare",
		},
		{
			desc: "the storefront's spelling reaches the catalog's",
			in:   CSIPriceEntry{Number: "ANGU-EN043", RarityName: "Collector Rare"},
			want: "ANGU-EN043 Collector's Rare",
		},
		{
			desc: "a note that says something keeps saying it",
			in:   CSIPriceEntry{Number: "BP02-EN045", Notes: "Mosaic Rare", RarityName: "Mosaic Rare"},
			want: "BP02-EN045 Mosaic Rare Mosaic Rare",
		},
		{
			desc: "a reprint note drops to the number and still names the tier",
			in:   CSIPriceEntry{Number: "SDK-001", Notes: "Reprints LOB-001", RarityName: "Ultra Rare"},
			want: "SDK-001 Ultra Rare",
		},
		{
			desc: "a row with no rarity at all says only what it did before",
			in:   CSIPriceEntry{Number: "LOB-005"},
			want: "LOB-005",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got := strings.TrimSpace(buylistVariation(tt.in) + " " + catalogRarity(tt.in.RarityName))
			if got != tt.want {
				t.Errorf("buylist variation = %q, want %q", got, tt.want)
			}
		})
	}
}
