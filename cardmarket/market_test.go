package cardmarket

import (
	"testing"

	cm "github.com/mtgban/go-cardmarket"

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
		held       map[string]float64
		wantOK     bool
		wantCond   string
	}{
		{
			name:     "an ordinary NM listing is accepted",
			article:  cm.Article{Price: 5, Condition: "NM"},
			held:     map[string]float64{},
			wantOK:   true,
			wantCond: "NM",
		},
		{
			name:    "a zero price is not a real listing",
			article: cm.Article{Price: 0, Condition: "NM"},
			held:    map[string]float64{},
			wantOK:  false,
		},
		{
			name:    "a seller in an excluded country is dropped",
			article: ukSeller,
			held:    map[string]float64{},
			wantOK:  false,
		},
		{
			name:    "an unrecognised condition string is dropped rather than guessed at",
			article: cm.Article{Price: 5, Condition: "XX"},
			held:    map[string]float64{},
			wantOK:  false,
		},
		{
			name:    "a listing no cheaper than what is already held is passed over",
			article: cm.Article{Price: 9, Condition: "NM"},
			held:    map[string]float64{"NM": 5},
			wantOK:  false,
		},
		{
			name:    "a listing at exactly the held price is passed over too - no reason to replace it with an equal one",
			article: cm.Article{Price: 5, Condition: "NM"},
			held:    map[string]float64{"NM": 5},
			wantOK:  false,
		},
		{
			name:     "a listing cheaper than what is held replaces it - listings are not strictly price-ascending in practice",
			article:  cm.Article{Price: 3, Condition: "NM"},
			held:     map[string]float64{"NM": 5},
			wantOK:   true,
			wantCond: "NM",
		},
		{
			name:       "a verifiable game rejects a listing whose own flag disagrees with what was asked for",
			gameID:     cm.GameMagic,
			wantFinish: true,
			verifiable: true,
			article:    cm.Article{Price: 5, Condition: "NM", IsFoil: false},
			held:       map[string]float64{},
			wantOK:     false,
		},
		{
			name:       "a verifiable game accepts a listing whose own flag agrees",
			gameID:     cm.GameMagic,
			wantFinish: true,
			verifiable: true,
			article:    cm.Article{Price: 5, Condition: "NM", IsFoil: true},
			held:       map[string]float64{},
			wantOK:     true,
			wantCond:   "NM",
		},
		{
			name:       "an unverifiable game accepts regardless of the article's own flag - the filter fails open, so nothing here can be trusted to narrow it either way",
			gameID:     cm.GameFleshAndBlood,
			wantFinish: true,
			verifiable: false,
			article:    cm.Article{Price: 5, Condition: "NM", IsFoil: false},
			held:       map[string]float64{},
			wantOK:     true,
			wantCond:   "NM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cond, ok := acceptArticle(tt.gameID, tt.wantFinish, tt.verifiable, tt.article, tt.held)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && cond != tt.wantCond {
				t.Errorf("cond = %q, want %q", cond, tt.wantCond)
			}
		})
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
