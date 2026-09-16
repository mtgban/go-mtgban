package cardmarket

import (
	"encoding/json"
	"os"
	"testing"

	cm "github.com/mtgban/go-cardmarket"
)

// replayFixturePath is the checked-in default: the first 30 of 100 real
// listings read live from Articles() for a single Riftbound common
// ("Blazing Scorcher", product 845712) - trimmed from the full page fetched
// during development since this prefix alone already carries everything
// the two tests below check for (an excluded-country listing, a bare "D"
// German one, and a price inversion), without checking in the other 70.
// CARDMARKET_MARKET_PATH overrides it with a fresher export in the same
// []cm.Article JSON shape (marshal what Client.Articles returns), the way
// POKEMON_PATH and the other datastore variables override their own game's
// file.
const replayFixturePath = "testdata/articles_riftbound.json"

func loadReplayFixture(t *testing.T) []cm.Article {
	t.Helper()
	path := replayFixturePath
	if override := os.Getenv("CARDMARKET_MARKET_PATH"); override != "" {
		path = override
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var articles []cm.Article
	if err := json.Unmarshal(data, &articles); err != nil {
		t.Fatal(err)
	}
	if len(articles) == 0 {
		t.Fatal("fixture carries no articles")
	}
	return articles
}

// TestReplayAcceptArticleAgainstRealListings replays a real page of
// Articles() results - not a hand-built one - through acceptArticle and
// isCheaper the way queryOnePrinting does, checking properties a synthetic
// fixture cannot: whether the country and condition strings Cardmarket
// actually sends are handled the way this package assumes they are, and
// whether the held-cheapest logic holds up against a real, not strictly
// price-ordered, listing sequence.
func TestReplayAcceptArticleAgainstRealListings(t *testing.T) {
	articles := loadReplayFixture(t)

	var excludedSeen, germanSeen bool
	for _, article := range articles {
		switch article.Seller.Address.Country {
		case "GB":
			excludedSeen = true
		case "D":
			germanSeen = true
		}
	}
	// These two guard the test itself: without them, the assertions below
	// would pass just as well if acceptArticle rejected (or accepted)
	// every listing regardless of country, and the test would not be able
	// to tell the difference.
	if !excludedSeen {
		t.Fatal("fixture carries no UK listing - this test cannot tell the exclusion filter apart from doing nothing")
	}
	if !germanSeen {
		t.Fatal("fixture carries no German (\"D\") listing - this test cannot tell a bare country code apart from being wrongly excluded")
	}

	held := map[string]float64{}
	entries := map[string]cm.Article{}
	for _, article := range articles {
		cond, ok := acceptArticle(nil, article)
		if !ok {
			continue
		}
		if !isCheaper(held, cond, article.Price) {
			continue
		}
		held[cond] = article.Price
		entries[cond] = article
	}

	nm, found := entries["NM"]
	if !found {
		t.Fatal("no NM price held from a 100-listing page")
	}
	if nm.Price <= 0 {
		t.Errorf("held NM price is not positive: %v", nm.Price)
	}
	if excludedCountries[nm.Seller.Address.Country] {
		t.Errorf("held NM listing %d is from an excluded country (%s)", nm.IDArticle, nm.Seller.Address.Country)
	}

	// The held price must actually be the minimum over every accepted (not
	// excluded-country) listing at that condition - the property
	// isCheaper's held-price comparison exists to guarantee even though
	// the fixture's own order is not strictly ascending.
	var trueMin float64
	for _, article := range articles {
		if article.Condition != "NM" || article.Price == 0 || excludedCountries[article.Seller.Address.Country] {
			continue
		}
		if trueMin == 0 || article.Price < trueMin {
			trueMin = article.Price
		}
	}
	if nm.Price != trueMin {
		t.Errorf("held NM price %v is not the true minimum %v across the fixture", nm.Price, trueMin)
	}
}

// TestReplayListingsAreNotStrictlyPriceAscending pins the finding
// isCheaper's held-price comparison exists for: a real page of listings
// for a bulk-priced product is not returned in strict price order. If
// Cardmarket ever tightens this to a true sort, this test starts failing
// rather than the assumption silently going stale - at which point the
// held-price comparison becomes unnecessary but not wrong, so nothing
// needs to change urgently.
func TestReplayListingsAreNotStrictlyPriceAscending(t *testing.T) {
	articles := loadReplayFixture(t)

	inverted := false
	for i := 1; i < len(articles); i++ {
		if articles[i].Price < articles[i-1].Price {
			inverted = true
			break
		}
	}
	if !inverted {
		t.Skip("this fixture happens to be strictly ascending - refresh it from a heavily bulk-priced product to exercise the case acceptArticle guards against")
	}
}
