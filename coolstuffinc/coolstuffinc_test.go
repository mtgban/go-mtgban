package coolstuffinc

import (
	"slices"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgban"
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

// TestIsGraded pins which wordings mean the copy is not being sold at a
// condition tier. One Piece sells PSA slabs and the condition parser used to
// refuse them outright, so the listing was dropped rather than priced.
func TestIsGraded(t *testing.T) {
	for _, tt := range []struct {
		conditions string
		want       bool
	}{
		// The storefront repeats the wording in the row it is read from.
		{"PSA 10  PSA 10 ", true},
		{"BGS 9.5", true},
		{"Unique", true},
		{"Non-Foil", true},
		{"Near Mint", false},
		{"Foil Near Mint", false},
		{"Played", false},
		{"", false},
	} {
		t.Run(tt.conditions, func(t *testing.T) {
			if got := isGraded(tt.conditions); got != tt.want {
				t.Errorf("isGraded(%q) = %v, want %v", tt.conditions, got, tt.want)
			}
		})
	}
}

// TestMarketNamesCarryTheGradedSeller pins that every game can publish a
// graded price. The scraper files one under its own seller whatever the game,
// but the name used to be offered for Magic alone, so everywhere else the
// entry was written and then dropped by the split, which collects only the
// names a market answers with. A game holding no graded copy publishes
// nothing extra: the split skips a seller whose inventory is empty.
func TestMarketNamesCarryTheGradedSeller(t *testing.T) {
	for _, game := range []string{GameMagic, GameOnePiece, GamePokemon, GameYuGiOh,
		GameLorcana, GameRiftbound, GameGundam, GamePalworld} {
		t.Run(game, func(t *testing.T) {
			names := NewScraper(game).MarketNames()
			if !slices.Contains(names, "Cool Stuff Inc (unique)") {
				t.Errorf("MarketNames() = %v, want the graded seller among them", names)
			}
		})
	}
}

// TestUnfoldSkipsTheEmptyGradedSeller pins what makes it safe to answer with
// the graded seller for every game: a game holding no graded copy publishes
// nothing extra, because the split drops a seller whose inventory is empty.
// Palworld is that game today.
func TestUnfoldSkipsTheEmptyGradedSeller(t *testing.T) {
	csi := NewScraper(GamePalworld)
	csi.inventory["some-uuid"] = []mtgban.InventoryEntry{{
		Conditions: "NM",
		Price:      1,
		Quantity:   1,
		SellerName: "Cool Stuff Inc",
	}}

	sellers, _ := mtgban.UnfoldScrapers([]mtgban.Scraper{csi})

	var names []string
	for _, seller := range sellers {
		names = append(names, seller.Info().Name)
	}
	want := []string{"Cool Stuff Inc"}
	if !slices.Equal(names, want) {
		t.Errorf("UnfoldScrapers gave %v, want %v", names, want)
	}
}

// TestOfferConditionKeepsALeadingDigit pins the trim the count is cut with.
// It was a cutset, so a condition opening with a digit the count also
// carries lost it, and "1st Edition" was reported as "st Edition".
func TestOfferConditionKeepsALeadingDigit(t *testing.T) {
	got := offerCondition("1 1st Edition  1st Edition $19.99Add to Cart", "1", "")
	want := "1st Edition  1st Edition "
	if got != want {
		t.Errorf("offerCondition = %q, want %q", got, want)
	}
}
