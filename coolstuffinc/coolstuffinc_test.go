package coolstuffinc

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/matcher"
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

// TestIsUnique pins which wordings mean the copy is not being sold at a
// condition tier, and that a slab is not one of them.
func TestIsUnique(t *testing.T) {
	for _, tt := range []struct {
		conditions string
		want       bool
	}{
		{"Unique \u00a0See image for details.\u00a0", true},
		{"Non-Foil", true},
		{"PSA 10 \u00a0PSA 10\u00a0", false},
		{"Near Mint", false},
		{"Foil Near Mint", false},
		{"Played", false},
		{"", false},
	} {
		t.Run(tt.conditions, func(t *testing.T) {
			got := isUnique(tt.conditions)
			if got != tt.want {
				t.Errorf("isUnique(%q) = %v, want %v", tt.conditions, got, tt.want)
			}
		})
	}
}

// TestSlabCondition pins how a slab is read off the storefront's own rows,
// that one it cannot read is kept as NM, and that any other wording is raw.
func TestSlabCondition(t *testing.T) {
	type slab = mtgban.Slab
	for _, tt := range []struct {
		fullRow string
		want    mtgban.Condition
		slab    *mtgban.Slab
		unread  string
	}{
		{"1 PSA 10 \u00a0PSA 10\u00a0$299.99Add to Cart", mtgban.NM, &slab{Grader: "PSA", Score: 10}, ""},
		{"1 BGS 8.5 \u00a0BGS 8.5\u00a0$149.99Add to Cart", mtgban.SP, &slab{Grader: "BGS", Score: 8.5}, ""},
		{"1 BGS 9.5 \u00a0BGS 9.5\u00a0Was\u00a0$349.99 Sale $314.99Add to Cart", mtgban.NM, &slab{Grader: "BGS", Score: 9.5}, ""},
		{"1 CGC 8.5 \u00a0CGC 8.5\u00a0$24.99Add to Cart", mtgban.NM, &slab{Grader: "CGC", Score: 8.5}, ""},
		{"1 CGC 10 Pristine \u00a0CGC 10 Pristine\u00a0$99.99Add to Cart", mtgban.NM, &slab{Grader: "CGC", Score: 10, Label: "Pristine"}, ""},
		{"1 PSA 9 Pokemon Center \u00a0PSA 9 Pokemon Center\u00a0$299.99Add to Cart", mtgban.NM, &slab{Grader: "PSA", Score: 9}, ""},
		{"1 PRE-ERRATA BGS 10 \u00a0PRE-ERRATA BGS 10\u00a0$299.99Add to Cart", mtgban.NM, &slab{Grader: "BGS", Score: 10}, ""},
		{"1 TAG 10 \u00a0TAG 10\u00a0Was\u00a0$99.99 Sale $89.99Add to Cart", mtgban.NM, &slab{Grader: "TAG", Score: 10}, ""},

		// Wordings other stores print, which the storefront has not yet
		{"1 CGC Pristine \u00a0CGC Pristine\u00a0$99.99Add to Cart", mtgban.NM, &slab{Grader: "CGC", Score: 10, Label: "Pristine"}, ""},
		{"1 BGS Quad 10 \u00a0BGS Quad 10\u00a0$99.99Add to Cart", mtgban.NM, &slab{Grader: "BGS", Score: 10, Label: "Black Label"}, ""},
		{"1 PSA Authentic \u00a0PSA Authentic\u00a0$99.99Add to Cart", mtgban.NM, &slab{Grader: "PSA", Label: "PSA Authentic"}, "PSA Authentic"},
		{"1 TAG \u00a0TAG\u00a0$99.99Add to Cart", mtgban.NM, &slab{Grader: "TAG", Label: "TAG"}, "TAG"},
		{"1 SGC 10 \u00a0SGC 10\u00a0$99.99Add to Cart", mtgban.NM, &slab{Label: "SGC 10"}, "SGC 10"},

		{"1 Unique \u00a0See image for details.\u00a0$59.99Add to Cart", "", nil, ""},
		{"1 PRE-ERRATA \u00a0PRE-ERRATA\u00a0$59.99Add to Cart", "", nil, ""},
		{"1 Near Mint $9.99Add to Cart", "", nil, ""},
	} {
		t.Run(tt.fullRow, func(t *testing.T) {
			got, read, unread := slabCondition(offerCondition(tt.fullRow, "1", ""))
			same := (read == nil) == (tt.slab == nil)
			if same && read != nil {
				same = *read == *tt.slab
			}
			if got != tt.want || !same || unread != tt.unread {
				t.Errorf("slabCondition(%q) = %q, %+v, %q; want %q, %+v, %q", tt.fullRow, got, read, unread, tt.want, tt.slab, tt.unread)
			}
		})
	}
}

// TestGradedMode pins that only the graded sub-seller is flagged as one.
func TestGradedMode(t *testing.T) {
	csi, err := NewScraper(&matcher.Backend{Game: matcher.GameMagic})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range csi.MarketNames() {
		info := csi.InfoForScraper(name)
		graded := name == "Cool Stuff Inc Graded"
		if info.GradedMode != graded {
			t.Errorf("InfoForScraper(%q).GradedMode = %v", name, info.GradedMode)
		}
		wantName := name
		if graded {
			wantName = "Cool Stuff Inc"
		}
		if info.Name != wantName {
			t.Errorf("InfoForScraper(%q).Name = %q, want %q", name, info.Name, wantName)
		}
	}
}

// TestGradedSeller walks a listing's plain, printing, unique and slab rows:
// slabs land on the graded seller only, each with its Slab, and one that
// cannot be read is published there as NM and logged.
func TestGradedSeller(t *testing.T) {
	b := readGameDatastore(t, "onepiece", "ONEPIECE_PATH")
	csi, err := NewScraper(b)
	if err != nil {
		t.Fatal(err)
	}
	var unread []string
	csi.logCallback = func(format string, a ...any) {
		if strings.HasSuffix(format, mtgban.UnreadSlabFormat) {
			unread = append(unread, fmt.Sprint(a...))
		}
	}
	searchFixture(t, "testdata/onepiece_graded_search.html")

	type want struct {
		seller string
		slab   mtgban.Slab
	}
	wants := map[string]want{
		"8404234":  {"Cool Stuff Inc", mtgban.Slab{}},
		"10776981": {"Cool Stuff Inc", mtgban.Slab{}},
		"12598564": {"Cool Stuff Inc Graded", mtgban.Slab{Grader: "BGS", Score: 10}},
		"4":        {"Cool Stuff Inc (unique)", mtgban.Slab{}},
		"5":        {"Cool Stuff Inc Graded", mtgban.Slab{Grader: "PSA", Label: "PSA Authentic"}},
	}
	got := searchResults(t, csi, "OP01 - Romance Dawn")
	if len(got) != len(wants) {
		t.Errorf("sent %d offers, want %d", len(got), len(wants))
	}
	for _, r := range got {
		entry := r.invEntry
		w := wants[entry.InstanceID]
		var slab mtgban.Slab
		if entry.Slab != nil {
			slab = *entry.Slab
		}
		if entry.SellerName != w.seller || entry.Conditions != mtgban.NM || slab != w.slab {
			t.Errorf("%s = %s %s %+v, want %s NM %+v", entry.InstanceID, entry.SellerName, entry.Conditions, slab, w.seller, w.slab)
		}
		if (entry.Slab != nil) != csi.InfoForScraper(entry.SellerName).GradedMode {
			t.Errorf("%s in %s with slab %+v", entry.InstanceID, entry.SellerName, entry.Slab)
		}
	}
	if !slices.Equal(unread, []string{"PSA Authentic"}) {
		t.Errorf("logged unread slabs %q", unread)
	}
}

// TestMarketNamesCarryTheGradedSeller pins that every game can publish a
// graded price. The scraper files one under its own seller whatever the game,
// and the split collects only the names a market answers with, so a game
// whose names leave the seller out writes the entry and then drops it. A game
// holding no graded copy publishes nothing extra: the split skips a seller
// whose inventory is empty.
func TestMarketNamesCarryTheGradedSeller(t *testing.T) {
	for _, game := range []matcher.Game{matcher.GameMagic, matcher.GameOnePiece, matcher.GamePokemon, matcher.GameYuGiOh,
		matcher.GameLorcana, matcher.GameRiftbound, matcher.GameGundam, matcher.GamePalworld} {
		t.Run(fmt.Sprintf("%q", game), func(t *testing.T) {
			csi, err := NewScraper(&matcher.Backend{Game: game})
			if err != nil {
				t.Fatalf("NewScraper(%q) = %v", game, err)
			}
			names := csi.MarketNames()
			for _, want := range []string{"Cool Stuff Inc (unique)", "Cool Stuff Inc Graded"} {
				if !slices.Contains(names, want) {
					t.Errorf("MarketNames() = %v, want %q among them", names, want)
				}
			}
		})
	}
}

// TestUnfoldSkipsTheEmptyGradedSeller pins what makes it safe to answer with
// the graded seller for every game: a game holding no graded copy publishes
// nothing extra, because the split drops a seller whose inventory is empty.
// Palworld is that game today.
func TestUnfoldSkipsTheEmptyGradedSeller(t *testing.T) {
	csi, err := NewScraper(&matcher.Backend{Game: matcher.GamePalworld})
	if err != nil {
		t.Fatalf("NewScraper(GamePalworld) = %v", err)
	}
	csi.inventory["some-uuid"] = []mtgban.InventoryEntry{{
		Conditions: mtgban.NM,
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
// As a cutset it would also take a leading digit the count carries off the
// condition, reporting "1st Edition" as "st Edition".
func TestOfferConditionKeepsALeadingDigit(t *testing.T) {
	got := offerCondition("1 1st Edition  1st Edition $19.99Add to Cart", "1", "")
	want := "1st Edition  1st Edition "
	if got != want {
		t.Errorf("offerCondition = %q, want %q", got, want)
	}
}

// TestIsSkippedCondition pins the wordings refused on purpose rather than
// reported: they name a copy the catalog has no row for.
func TestIsSkippedCondition(t *testing.T) {
	for _, tt := range []struct {
		conditions string
		want       bool
	}{
		{"Asian Near Mint", true},
		{"Near Mint", false},
		{"Foil Near Mint", false},
		{"Played", false},
	} {
		t.Run(tt.conditions, func(t *testing.T) {
			if got := isSkippedCondition(tt.conditions); got != tt.want {
				t.Errorf("isSkippedCondition(%q) = %v, want %v", tt.conditions, got, tt.want)
			}
		})
	}
}

// TestIsUniqueTakesTheSingleCopyPrintings pins the copies a storefront sells
// on another printing's product. The shadowless Venusaur is offered on the
// unlimited Venusaur's page at $299.99 beside a $99.99 unique copy, so its
// price is not the product's and must not sit with the ordinary conditions.
func TestIsUniqueTakesTheSingleCopyPrintings(t *testing.T) {
	for _, conditions := range []string{
		"Shadowless  Shadowless ",
		"No Set Symbol  No Set Symbol ",
	} {
		t.Run(conditions, func(t *testing.T) {
			if !isUnique(conditions) {
				t.Errorf("isUnique(%q) = false, want true", conditions)
			}
		})
	}
	for _, conditions := range []string{"Near Mint", "Foil Near Mint", "Played"} {
		t.Run(conditions, func(t *testing.T) {
			if isUnique(conditions) {
				t.Errorf("isUnique(%q) = true, want false", conditions)
			}
		})
	}
}

// TestCSIAmbiguousRefusal pins which refusals a "Can be" note keeps quiet:
// only the ones its own ambiguity causes.
func TestCSIAmbiguousRefusal(t *testing.T) {
	for _, tt := range []struct {
		notes string
		err   error
		want  bool
	}{
		{"Can be (STAS-EN020) OR (STAS-EN039)", matcher.NewAliasingError("a", "b"), true},
		{"Can Be Normal or 25th Anniversary Stamp", matcher.ErrCardWrongVariant, true},
		{"Can be Autumn 2011, Spring 2012", matcher.ErrCardDoesNotExist, false},
		{"Can be Autumn 2011, Spring 2012", matcher.ErrCardNotInEdition, false},
		{"Alt-Art", matcher.NewAliasingError("a", "b"), false},
	} {
		got := csiAmbiguousRefusal(tt.notes, tt.err)
		if got != tt.want {
			t.Errorf("csiAmbiguousRefusal(%q, %v) = %v, want %v", tt.notes, tt.err, got, tt.want)
		}
	}
}
