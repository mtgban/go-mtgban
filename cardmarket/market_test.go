package cardmarket

import (
	"context"
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

func TestNewScraperMarketUnsupportedGame(t *testing.T) {
	_, err := NewScraperMarket(&mtgmatcher.Backend{Game: "NotAGame"}, "token", "secret")
	if err == nil {
		t.Fatal("expected an error for an unsupported game")
	}
}

func TestNewScraperMarketWiresTheResolver(t *testing.T) {
	mkm, err := NewScraperMarket(&mtgmatcher.Backend{Game: "magic"}, "token", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if mkm.gameID != cm.GameMagic {
		t.Errorf("gameID = %d, want %d - the resolver embedding should have picked this up", mkm.gameID, cm.GameMagic)
	}
	if mkm.resolver.printf == nil {
		t.Fatal("resolver.printf was not wired, so every id-resolution log line is a silent no-op")
	}
}

func TestMarketLanguage(t *testing.T) {
	tests := []struct {
		language string
		// The raw numbers rather than the cm.Language constants: the
		// library pins those against its own documentation, and this
		// wants to notice if a bump ever moves one underneath us.
		want cm.Language
	}{
		// Every language the datastore carries that Cardmarket also
		// does. Nine of these reach cm.LanguageFromName rather than
		// the table, so a rename on the library's side has to fail
		// here instead of quietly pricing the card in English.
		{"English", 1},
		{"French", 2},
		{"German", 3},
		{"Spanish", 4},
		{"Italian", 5},
		{"Japanese", 7},
		{"Portuguese", 8},
		{"Russian", 9},
		{"Korean", 10},
		// The two the table exists for: same languages, other word order.
		{"Chinese Simplified", 6},
		{"Chinese Traditional", 11},
		// No clean match is English: a missing field, mtgban's fictional
		// languages, and the ones Cardmarket's table does not carry.
		{"", 1},
		{"Phyrexian", 1},
		{"Quenya", 1},
		{"Polish", 1},
		{"Klingon", 1},
	}
	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			if got := marketLanguage(tt.language); got != tt.want {
				t.Errorf("marketLanguage(%q) = %d, want %d", tt.language, got, tt.want)
			}
		})
	}
}

func TestMkmConditionMapping(t *testing.T) {
	// Every one of Cardmarket's seven grades must map onto one of mtgban's
	// five, and the three this scraper actually chases - NM, SP, MP - must
	// each own a distinct row rather than folding into one another: that
	// is the whole reason the early-stop rule can tell them apart.
	want := map[cm.Condition]string{
		"MT": "NM", "NM": "NM",
		"EX": "SP",
		"GD": "MP",
		"LP": "HP", "PL": "HP",
		"PO": "PO",
	}
	if len(mkmCondition) != len(want) {
		t.Fatalf("mkmCondition has %d rows, want %d", len(mkmCondition), len(want))
	}
	for grade, mtgbanGrade := range want {
		if got := mkmCondition[grade]; got != mtgbanGrade {
			t.Errorf("mkmCondition[%q] = %q, want %q", grade, got, mtgbanGrade)
		}
	}
}

func TestAcceptArticle(t *testing.T) {
	ukSeller := cm.Article{Price: 5, Condition: "NM"}
	ukSeller.Seller.Address.Country = "GB"

	tests := []struct {
		name     string
		flags    map[string]bool
		article  cm.Article
		wantOK   bool
		wantCond string
	}{
		{
			name:     "an ordinary NM listing is accepted",
			article:  cm.Article{Price: 5, Condition: "NM"},
			wantOK:   true,
			wantCond: "NM",
		},
		{
			name:    "a zero price is not a real listing",
			article: cm.Article{Price: 0, Condition: "NM"},
			wantOK:  false,
		},
		{
			name:    "a seller in an excluded country is dropped",
			article: ukSeller,
			wantOK:  false,
		},
		{
			name:    "a seller on vacation is dropped",
			article: cm.Article{Price: 5, Condition: "NM", Seller: cm.ArticleSeller{OnVacation: true}},
			wantOK:  false,
		},
		{
			name:    "an unrecognised condition string is dropped rather than guessed at",
			article: cm.Article{Price: 5, Condition: "XX"},
			wantOK:  false,
		},
		{
			name:    "a listing whose own flag disagrees with what was asked for is rejected",
			flags:   map[string]bool{"isFoil": true},
			article: cm.Article{Price: 5, Condition: "NM", IsFoil: false},
			wantOK:  false,
		},
		{
			name:     "a listing whose own flag agrees is accepted",
			flags:    map[string]bool{"isFoil": true},
			article:  cm.Article{Price: 5, Condition: "NM", IsFoil: true},
			wantOK:   true,
			wantCond: "NM",
		},
		{
			name:     "no flags to verify accepts regardless of the article's own - the filter fails open on a game or value it does not apply to, so nothing here can be trusted to narrow it either way",
			article:  cm.Article{Price: 5, Condition: "NM", IsFoil: false},
			wantOK:   true,
			wantCond: "NM",
		},
		{
			name:     "Pokemon's two flags are both verified independently",
			flags:    map[string]bool{"isFirstEd": true, "isReverseHolo": false},
			article:  cm.Article{Price: 5, Condition: "NM", IsFirstEd: true, IsReverseHolo: false},
			wantOK:   true,
			wantCond: "NM",
		},
		{
			name:    "Pokemon rejects a listing that agrees on one axis but not the other",
			flags:   map[string]bool{"isFirstEd": true, "isReverseHolo": false},
			article: cm.Article{Price: 5, Condition: "NM", IsFirstEd: true, IsReverseHolo: true},
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond, ok := acceptArticle(tt.flags, tt.article)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && cond != tt.wantCond {
				t.Errorf("cond = %q, want %q", cond, tt.wantCond)
			}
		})
	}
}

// TestIsCheaper pins the held-price comparison acceptArticle used to do
// itself, now factored out so each bucket (main, Powerseller) can apply it
// against its own held map independently - see TestPowersellerBucketIsIndependentOfMain
// for why that independence is the whole point.
func TestIsCheaper(t *testing.T) {
	tests := []struct {
		name  string
		held  map[string]float64
		cond  string
		price float64
		want  bool
	}{
		{"nothing held yet is always cheaper", map[string]float64{}, "NM", 5, true},
		{"strictly cheaper than what is held", map[string]float64{"NM": 5}, "NM", 3, true},
		{"equal to what is held is not cheaper - no reason to replace it", map[string]float64{"NM": 5}, "NM", 5, false},
		{"more expensive than what is held is not cheaper", map[string]float64{"NM": 5}, "NM", 9, false},
		{"a different condition being held does not block this one", map[string]float64{"SP": 1}, "NM", 100, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCheaper(tt.held, tt.cond, tt.price); got != tt.want {
				t.Errorf("isCheaper(%v, %q, %v) = %v, want %v", tt.held, tt.cond, tt.price, got, tt.want)
			}
		})
	}
}

// TestPowersellerBucketIsIndependentOfMain is the direct regression test
// for the bug a review of #629 caught: acceptArticle used to compare every
// listing against the *main* held map before either bucket ever saw it, so
// a Powerseller listing more expensive than an already-held cheap listing
// from an unrelated seller (a cheap French private seller, say) never
// reached the Powerseller bucket at all - that bucket could only ever
// fill when its own listing also happened to be the single global
// cheapest, not "cheapest among Powersellers", which is what it actually
// claims to be.
//
// Replays the same two-bucket decision queryOnePrinting's loop makes,
// article by article, using the same three functions it calls
// (acceptArticle, isPowerseller, isCheaper) - not the loop itself, which
// needs a live client to reach at all - so a regression in how those
// three compose would fail here the same way it would in production.
func TestPowersellerBucketIsIndependentOfMain(t *testing.T) {
	cheapFrenchPrivate := cm.Article{Price: 1, Condition: "NM"}
	cheapFrenchPrivate.Seller.Address.Country = "FR"

	pricierGermanPowerseller := cm.Article{Price: 5, Condition: "NM"}
	pricierGermanPowerseller.Seller.Address.Country = "D"
	pricierGermanPowerseller.Seller.IsCommercial = 2

	held := map[string]float64{}
	heldPS := map[string]float64{}

	for _, article := range []cm.Article{cheapFrenchPrivate, pricierGermanPowerseller} {
		cond, ok := acceptArticle(nil, article)
		if !ok {
			t.Fatalf("acceptArticle rejected a listing that should have been accepted: %+v", article)
		}
		if isCheaper(held, cond, article.Price) {
			held[cond] = article.Price
		}
		if isPowerseller(article) && isCheaper(heldPS, cond, article.Price) {
			heldPS[cond] = article.Price
		}
	}

	if held["NM"] != 1 {
		t.Errorf("main bucket NM = %v, want 1 (the cheap French private listing)", held["NM"])
	}
	if heldPS["NM"] != 5 {
		t.Errorf("Powerseller bucket NM = %v, want 5 (the German Powerseller listing) - "+
			"got the bug back if this is 0: the Powerseller bucket only fills when its own "+
			"listing also happens to be the single global cheapest", heldPS["NM"])
	}
}

// TestArticleFlagValue pins which article field each of the three
// Cardmarket finish parameters reads.
func TestArticleFlagValue(t *testing.T) {
	magic := cm.Article{IsFoil: true}
	firstEd := cm.Article{IsFirstEd: true}
	reverseHolo := cm.Article{IsReverseHolo: true}

	if !articleFlagValue("isFoil", &magic) {
		t.Error("isFoil should read IsFoil")
	}
	if !articleFlagValue("isFirstEd", &firstEd) {
		t.Error("isFirstEd should read IsFirstEd")
	}
	if !articleFlagValue("isReverseHolo", &reverseHolo) {
		t.Error("isReverseHolo should read IsReverseHolo")
	}
	if articleFlagValue("noSuchParam", &magic) {
		t.Error("an unknown parameter should never report a flag")
	}
}

// TestMarketFinishParamCoverage pins which games have a verified,
// single-axis finish signal - see marketFinishParam's own comment on how
// Lorcana and Riftbound's was confirmed (article.IsFoil, live-sampled, not
// just the request-side filter's documentation) and on why Pokemon is not
// here at all (queryPokemonPrintings, pokemonFinishPlan).
func TestMarketFinishParamCoverage(t *testing.T) {
	want := map[cm.Game]string{
		cm.GameMagic:     "isFoil",
		cm.GameYuGiOh:    "isFirstEd",
		cm.GameLorcana:   "isFoil",
		cm.GameRiftbound: "isFoil",
	}
	if len(marketFinishParam) != len(want) {
		t.Fatalf("marketFinishParam has %d games, want %d", len(marketFinishParam), len(want))
	}
	for gameID, param := range want {
		if got := marketFinishParam[gameID]; got != param {
			t.Errorf("marketFinishParam[%d] = %q, want %q", gameID, got, param)
		}
	}
}

// TestDenmarkIsNotExcluded pins excludedCountries to EU membership, not
// Nordic-ness: Sweden and Finland are EU members and are not excluded,
// same as Denmark; Norway and Iceland are EEA/EFTA, not EU, and stay
// excluded alongside the UK and Switzerland.
func TestDenmarkIsNotExcluded(t *testing.T) {
	for _, cc := range []string{"DK", "SE", "FI"} {
		if excludedCountries[cc] {
			t.Errorf("%s is an EU member and should not be excluded", cc)
		}
	}
	for _, cc := range []string{"GB", "CH", "NO", "IS"} {
		if !excludedCountries[cc] {
			t.Errorf("%s should still be excluded", cc)
		}
	}
}

// TestIsPowerseller pins isCommercial's documented values against the two
// real accounts this was confirmed against directly: 1 reads
// "Professional" on Cardmarket's own seller page, 2 reads "Powerseller" -
// and pins the two countries this scraper actually holds a Powerseller
// bucket for.
func TestIsPowerseller(t *testing.T) {
	tests := []struct {
		name         string
		isCommercial int
		country      string
		want         bool
	}{
		{"a private seller in Germany is not a Powerseller", 0, "D", false},
		{"a Professional seller in Germany is not a Powerseller", 1, "D", false},
		{"a Powerseller in Germany counts", 2, "D", true},
		{"a Powerseller in the Netherlands counts", 2, "NL", true},
		{"a Powerseller elsewhere does not count", 2, "FR", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			article := cm.Article{}
			article.Seller.IsCommercial = tt.isCommercial
			article.Seller.Address.Country = tt.country
			if got := isPowerseller(article); got != tt.want {
				t.Errorf("isPowerseller() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMarketNames pins the two sub-sellers Market splits its inventory
// into, the same shape cardtrader.Market's three storefronts use.
func TestMarketNames(t *testing.T) {
	mkm, err := NewScraperMarket(&mtgmatcher.Backend{Game: "magic"}, "token", "secret")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Cardmarket", "Cardmarket Powersellers"}
	got := mkm.MarketNames()
	if len(got) != len(want) {
		t.Fatalf("MarketNames() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("MarketNames()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestInfoForScraper pins each sub-seller's own Name/Shorthand, and that
// everything else (Game, CountryFlag, InventoryTimestamp) still comes
// through from the base Info().
func TestInfoForScraper(t *testing.T) {
	mkm, err := NewScraperMarket(&mtgmatcher.Backend{Game: "magic"}, "token", "secret")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name          string
		wantShorthand string
	}{
		{"Cardmarket", "MKM"},
		{"Cardmarket Powersellers", "MKMPS"},
	}
	for _, tt := range tests {
		info := mkm.InfoForScraper(tt.name)
		if info.Name != tt.name {
			t.Errorf("InfoForScraper(%q).Name = %q, want %q", tt.name, info.Name, tt.name)
		}
		if info.Shorthand != tt.wantShorthand {
			t.Errorf("InfoForScraper(%q).Shorthand = %q, want %q", tt.name, info.Shorthand, tt.wantShorthand)
		}
		if info.Game != mtgban.GameMagic {
			t.Errorf("InfoForScraper(%q).Game = %q, want %q", tt.name, info.Game, mtgban.GameMagic)
		}
	}
}

// TestShouldStopPaging pins the Powerseller fail-safe: once the main
// bucket is satisfied, keep paging up to marketPowersellerExtraPages more
// pages chasing a Powerseller listing, stopping the instant one is found
// or the extra budget runs out, whichever comes first - and never before
// the main bucket itself is satisfied at all.
func TestShouldStopPaging(t *testing.T) {
	tests := []struct {
		name             string
		mainDone         bool
		mainSatisfiedAt  int
		page             int
		foundPowerseller bool
		want             bool
	}{
		{"main not yet satisfied never stops, Powerseller or not", false, -1, 0, false, false},
		{"main not yet satisfied never stops even with one already found", false, -1, 5, true, false},
		{"main just satisfied, no Powerseller yet, keeps going", true, 3, 3, false, false},
		{"a Powerseller found the same page main is satisfied stops immediately", true, 3, 3, true, true},
		{"still within the extra budget, none found yet, keeps going", true, 3, 6, false, false},
		{"extra budget exactly spent with none found stops", true, 3, 7, false, true},
		{"a Powerseller found partway through the extra budget stops early", true, 3, 5, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shouldStopPaging(tt.mainDone, tt.mainSatisfiedAt, tt.page, tt.foundPowerseller)
			if got != tt.want {
				t.Errorf("shouldStopPaging(%v, %d, %d, %v) = %v, want %v",
					tt.mainDone, tt.mainSatisfiedAt, tt.page, tt.foundPowerseller, got, tt.want)
			}
		})
	}
}

// TestResolveExpansionEntry pins the fallback that fills a gap in
// mkm.catalog (MTGJSON's own CardmarketIdentifiers.json export, currently
// missing 88 of Magic's real expansions) from Cardmarket's live API
// instead of falling through to the unresolvable "expansion <id>"
// placeholder.
func TestResolveExpansionEntry(t *testing.T) {
	live := map[int]cm.Expansion{
		6493: {IDExpansion: 6493, Name: "Commander: Teenage Mutant Ninja Turtles: Extras", SetCode: "XTMC"},
	}

	tests := []struct {
		name        string
		entry       cm.CatalogExpansion
		expansionID int
		want        cm.CatalogExpansion
	}{
		{
			name:        "an entry Catalog already names is left alone, live is not consulted",
			entry:       cm.CatalogExpansion{Name: "Magic 2011", Code: "M11"},
			expansionID: 1197,
			want:        cm.CatalogExpansion{Name: "Magic 2011", Code: "M11"},
		},
		{
			name:        "a gap live covers is substituted with the real name and code",
			entry:       cm.CatalogExpansion{},
			expansionID: 6493,
			want:        cm.CatalogExpansion{Name: "Commander: Teenage Mutant Ninja Turtles: Extras", Code: "XTMC"},
		},
		{
			name:        "a gap live does not cover either is left empty, for the caller's own placeholder",
			entry:       cm.CatalogExpansion{},
			expansionID: 9999999,
			want:        cm.CatalogExpansion{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveExpansionEntry(tt.entry, tt.expansionID, live)
			if got != tt.want {
				t.Errorf("resolveExpansionEntry(%+v, %d, live) = %+v, want %+v", tt.entry, tt.expansionID, got, tt.want)
			}
		})
	}
}

// TestLiveExpansionsMemoizesWithoutRetrying pins the failure-mode contract
// banreviewer's review on this fix asked for: the live call is attempted
// at most once per Load, its outcome - success or failure - reused for
// every later gap rather than retried. mkm.client is left nil in both
// cases; if liveExpansions ever attempted a real call instead of trusting
// liveExpansionsTried, dereferencing it would panic, so a clean run here
// is itself proof no network call was attempted.
func TestLiveExpansionsMemoizesWithoutRetrying(t *testing.T) {
	t.Run("a cached success is returned without touching client", func(t *testing.T) {
		mkm := &Market{
			liveExpansionsTried: true,
			liveExpansionsCache: map[int]cm.Expansion{6493: {IDExpansion: 6493, Name: "cached"}},
		}
		got := mkm.liveExpansions(context.Background())
		if got[6493].Name != "cached" {
			t.Errorf("liveExpansions() = %+v, want the cached map returned as-is", got)
		}
	})

	t.Run("a remembered failure is not retried", func(t *testing.T) {
		mkm := &Market{liveExpansionsTried: true}
		got := mkm.liveExpansions(context.Background())
		if got != nil {
			t.Errorf("liveExpansions() = %+v, want nil - a prior failure should not be retried this run", got)
		}
	})
}
