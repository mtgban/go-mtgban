package cardtrader

import "testing"

// numberShapes are the collector numbers One Piece writes, and whether the
// game's matcher reads one as a number. mtgmatcher/onepiece owns the shape
// - its fullNumberRe decides the same table, in TestFullNumberShapes -
// and collectorNumberRe below only has to agree with it: a shape the
// matcher gains and this gate does not would send the blueprint's Version
// riding along on a number nothing can parse.
var numberShapes = map[string]bool{
	"OP01-001":   true,
	"P-043":      true,
	"OP01-001a":  true,
	"OP07-047P2": false,
	"P-L":        false,
	"2024":       false,
	"":           false,
}

func TestCollectorNumberShapes(t *testing.T) {
	for number, want := range numberShapes {
		if got := collectorNumberRe.MatchString(number); got != want {
			t.Errorf("collectorNumberRe.MatchString(%q) = %v, want %v", number, got, want)
		}
	}
}

// TestGameVariation pins the gate: the Version names the printing only for
// One Piece and Riftbound, and only behind a number - a readable one for One
// Piece - since its wording is full of the years and volume numbers that
// would answer as a collector number in its place.
func TestGameVariation(t *testing.T) {
	tests := []struct {
		name    string
		gameID  int
		version string
		number  string
		want    string
	}{
		{"one piece appends the version", GameOnePiece, "OP16 Release Event", "P-135", "P-135 OP16 Release Event"},
		{"a letter-tailed number takes it too", GameOnePiece, "Winner Pack 2026 Vol.3", "OP01-001a", "OP01-001a Winner Pack 2026 Vol.3"},
		{"an unreadable number keeps the version out", GameOnePiece, "Winner Pack 2026 Vol.3", "OP07-047P2", "OP07-047P2"},
		{"so does a number with no digits to read", GameOnePiece, "Premium Card Collection", "P-L", "P-L"},
		{"an empty version leaves the number alone", GameOnePiece, "", "OP01-001", "OP01-001"},
		{"lorcana keeps its own number", GameLorcana, "Enchanted", "OP01-001", "OP01-001"},
		{"riftbound appends the version too", GameRiftbound, "Summoner Skirmish | Champion", "058c", "058c Summoner Skirmish | Champion"},
		{"a numberless riftbound blueprint keeps the version out", GameRiftbound, "6 Card Set", "", ""},
		{"magic keeps its own number", GameMagic, "Retro Frame", "OP01-001", "OP01-001"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			bp := Blueprint{Version: test.version}
			got := gameVariation(test.gameID, &bp, test.number)
			if got != test.want {
				t.Errorf("gameVariation(%d, %q, %q) = %q, want %q",
					test.gameID, test.version, test.number, got, test.want)
			}
		})
	}
}

func TestPriceToUSD(t *testing.T) {
	// Keyed as the feed writes them, which is how the table arrives.
	rates := map[string]float64{"eur": 1.10, "gbp": 1.25, "aud": 0.70, "chf": 1.25}

	tests := []struct {
		name     string
		cents    int
		currency string
		want     float64
		wantErr  bool
	}{
		{"dollars pass through", 1234, "USD", 12.34, false},
		{"dollars need no rate at all", 500, "USD", 5, false},
		{"euros convert", 1000, "EUR", 11, false},
		{"pounds convert", 1000, "GBP", 12.5, false},
		// The currency an expansion went unpriced for, and the one after
		// it: the table answers whatever the marketplace quotes, so
		// neither had to be named in advance.
		{"australian dollars convert", 1000, "AUD", 7, false},
		{"francs convert", 1000, "CHF", 12.5, false},
		// The marketplace spells a currency in capitals and the feed in
		// lower case, so the lookup folds rather than missing.
		{"the marketplace's spelling finds the feed's", 1000, "aud", 7, false},
		{"a currency the feed does not quote is refused", 1000, "XYZ", 0, true},
		{"an empty currency is refused", 1000, "", 0, true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			price, err := priceToUSD(test.cents, test.currency, rates)
			if test.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %f", price)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
			if price != test.want {
				t.Errorf("expected %f, got %f", test.want, price)
			}
		})
	}
}

// A nil map is what a scraper holds before Load runs; dollars still have to
// convert, since that path never consults a rate.
func TestPriceToUSDWithoutRates(t *testing.T) {
	price, err := priceToUSD(1234, "USD", nil)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if price != 12.34 {
		t.Errorf("expected 12.34, got %f", price)
	}
	if _, err = priceToUSD(1234, "EUR", nil); err == nil {
		t.Error("expected an error converting euros with no rates")
	}
}

// TestListingPrice pins which field answers, and that the currency comes
// from the same field as the amount: reading an amount from one and a
// currency from another is how a price gets converted by the wrong rate.
func TestListingPrice(t *testing.T) {
	tests := []struct {
		name         string
		product      Product
		wantCents    int
		wantCurrency string
	}{
		{
			"an export quotes price_cents",
			Product{PriceCents: 1000, PriceCurrency: "EUR"},
			1000, "EUR",
		},
		{
			"the marketplace quotes price",
			Product{Price: CTPrice{Cents: 250, Currency: "GBP"}},
			250, "GBP",
		},
		{
			"an order quotes buyer_price",
			Product{BuyerPrice: CTPrice{Cents: 750, Currency: "AUD"}},
			750, "AUD",
		},
		{
			"price_cents wins when more than one is filled",
			Product{
				PriceCents:    1000,
				PriceCurrency: "USD",
				Price:         CTPrice{Cents: 250, Currency: "GBP"},
				BuyerPrice:    CTPrice{Cents: 750, Currency: "AUD"},
			},
			1000, "USD",
		},
		{
			"a listing with no price at all reads as none",
			Product{},
			0, "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cents, currency := listingPrice(test.product)
			if cents != test.wantCents || currency != test.wantCurrency {
				t.Errorf("listingPrice() = (%d, %q), want (%d, %q)",
					cents, currency, test.wantCents, test.wantCurrency)
			}
		})
	}
}
