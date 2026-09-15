package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgban"
)

func TestNewScraperMarketUnsupportedGame(t *testing.T) {
	_, err := NewScraperMarket(mtgban.Game("NotAGame"), "token", "secret")
	if err == nil {
		t.Fatal("expected an error for an unsupported game")
	}
}

func TestNewScraperMarketWiresTheResolver(t *testing.T) {
	mkm, err := NewScraperMarket(mtgban.GameMagic, "token", "secret")
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
		want     int
	}{
		{"", 1},
		{"English", 1},
		{"Japanese", 7},
		{"Chinese Simplified", 6},
		{"Chinese Traditional", 11},
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
	want := map[string]string{
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
		name       string
		gameID     int
		wantFinish bool
		verifiable bool
		article    cm.Article
		wantOK     bool
		wantCond   string
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
			name:    "an unrecognised condition string is dropped rather than guessed at",
			article: cm.Article{Price: 5, Condition: "XX"},
			wantOK:  false,
		},
		{
			name:       "a verifiable game rejects a listing whose own flag disagrees with what was asked for",
			gameID:     cm.GameMagic,
			wantFinish: true,
			verifiable: true,
			article:    cm.Article{Price: 5, Condition: "NM", IsFoil: false},
			wantOK:     false,
		},
		{
			name:       "a verifiable game accepts a listing whose own flag agrees",
			gameID:     cm.GameMagic,
			wantFinish: true,
			verifiable: true,
			article:    cm.Article{Price: 5, Condition: "NM", IsFoil: true},
			wantOK:     true,
			wantCond:   "NM",
		},
		{
			name:       "an unverifiable game accepts regardless of the article's own flag - the filter fails open, so nothing here can be trusted to narrow it either way",
			gameID:     cm.GameFleshAndBlood,
			wantFinish: true,
			verifiable: false,
			article:    cm.Article{Price: 5, Condition: "NM", IsFoil: false},
			wantOK:     true,
			wantCond:   "NM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond, ok := acceptArticle(tt.gameID, tt.wantFinish, tt.verifiable, tt.article)
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
		cond, ok := acceptArticle(cm.GameMagic, false, false, article)
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

func TestMarketArticleFinish(t *testing.T) {
	magic := cm.Article{IsFoil: true}
	pokemon := cm.Article{IsReverseHolo: true}
	yugioh := cm.Article{IsFirstEd: true}
	lorcana := cm.Article{IsFoil: true}
	riftbound := cm.Article{IsFoil: true}

	if !marketArticleFinish(cm.GameMagic, &magic) {
		t.Error("Magic should read IsFoil")
	}
	if !marketArticleFinish(cm.GamePokemon, &pokemon) {
		t.Error("Pokemon should read IsReverseHolo")
	}
	if !marketArticleFinish(cm.GameYuGiOh, &yugioh) {
		t.Error("YuGiOh should read IsFirstEd")
	}
	if !marketArticleFinish(cm.GameLorcana, &lorcana) {
		t.Error("Lorcana should read IsFoil")
	}
	if !marketArticleFinish(cm.GameRiftbound, &riftbound) {
		t.Error("Riftbound should read IsFoil")
	}
	if marketArticleFinish(cm.GameFleshAndBlood, &magic) {
		t.Error("a game with no known signal should never report a finish")
	}
}

// TestMarketFinishParamCoverage pins which games have a verified finish
// signal - see marketFinishParam's own comment on how Lorcana and
// Riftbound's was confirmed (article.IsFoil, live-sampled, not just the
// request-side filter's documentation).
func TestMarketFinishParamCoverage(t *testing.T) {
	want := map[int]string{
		cm.GameMagic:     "isFoil",
		cm.GamePokemon:   "isReverseHolo",
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

// TestDenmarkIsNotExcluded pins that Denmark stays a fine country - the
// other four Nordic ones (Norway, Sweden, Finland, Iceland) and the UK and
// Switzerland are still excluded, but Denmark on its own is not one of
// them.
func TestDenmarkIsNotExcluded(t *testing.T) {
	if excludedCountries["DK"] {
		t.Error("Denmark should not be excluded")
	}
	for _, cc := range []string{"GB", "CH", "NO", "SE", "FI", "IS"} {
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
	mkm, err := NewScraperMarket(mtgban.GameMagic, "token", "secret")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Cardmarket Market", "Cardmarket Powersellers"}
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
	mkm, err := NewScraperMarket(mtgban.GameMagic, "token", "secret")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name          string
		wantShorthand string
	}{
		{"Cardmarket Market", "MKM"},
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
