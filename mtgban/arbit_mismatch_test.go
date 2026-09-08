package mtgban

import (
	"math"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// TestMismatchNormalisesTheGrades pins the correction that makes two shops
// comparable. The reference's own grade is undone before the probe's is
// applied, so a played reference is not compared against a rescaled copy of
// itself.
func TestMismatchNormalisesTheGrades(t *testing.T) {
	for _, tt := range []struct {
		desc                 string
		refCond, probeCond   string
		refPrice, probePrice float64
		wantDifference       float64
	}{
		{"two NM copies compare as they are", "NM", "NM", 10, 6, 4},
		// The reference is NM, the probe is played: the NM price is brought
		// down to the played grade before the two are compared.
		{"an NM reference is graded down to the probe", "NM", "MP", 10, 5, 10*0.6 - 5},
		// The reference is played, the probe is NM: undoing the reference's
		// own grade is what stops it being compared against itself rescaled.
		{"a played reference is graded up to the probe", "MP", "NM", 6, 8, 6/0.6 - 8},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			installCards(t, plainCard())
			entries := Mismatch(&ArbitOpts{MinDiff: -1000, MinSpread: -1000},
				sellerOf(InventoryRecord{
					"card": {{Conditions: tt.refCond, Price: tt.refPrice, Quantity: 1}},
				}, ScraperInfo{Name: "reference"}),
				sellerOf(InventoryRecord{
					"card": {{Conditions: tt.probeCond, Price: tt.probePrice, Quantity: 1}},
				}, ScraperInfo{Name: "probe"}))
			if len(entries) != 1 {
				t.Fatalf("Mismatch returned %d entries, want 1", len(entries))
			}
			if math.Abs(entries[0].Difference-tt.wantDifference) > 1e-9 {
				t.Errorf("Difference = %v, want %v", entries[0].Difference, tt.wantDifference)
			}
		})
	}
}

// A grade worth nothing cannot be divided by, and a grade the map does not
// know cannot be scaled at all: either way the pair is dropped rather than
// reported at an invented spread.
func TestMismatchSkipsAnUnusableGrade(t *testing.T) {
	for _, tt := range []struct{ desc, refCond, probeCond string }{
		{"a poor reference is worth zero", "PO", "NM"},
		{"a poor probe is too", "NM", "PO"},
		{"a grade the map does not know", "NM", "GEM-MT"},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			installCards(t, plainCard())
			entries := Mismatch(&ArbitOpts{MinDiff: -1000, MinSpread: -1000},
				sellerOf(InventoryRecord{
					"card": {{Conditions: tt.refCond, Price: 10, Quantity: 1}},
				}, ScraperInfo{Name: "reference"}),
				sellerOf(InventoryRecord{
					"card": {{Conditions: tt.probeCond, Price: 1, Quantity: 1}},
				}, ScraperInfo{Name: "probe"}))
			if len(entries) != 0 {
				t.Errorf("Mismatch returned %d entries, want none", len(entries))
			}
		})
	}
}

// The row carries both sides so a caller can show its working, and the
// quantity is what the smaller side allows.
func TestMismatchReportsBothSides(t *testing.T) {
	installCards(t, plainCard())

	entries := Mismatch(nil,
		sellerOf(InventoryRecord{
			"card": {{Conditions: "NM", Price: 10, Quantity: 2}},
		}, ScraperInfo{Name: "reference"}),
		sellerOf(InventoryRecord{
			"card": {{Conditions: "NM", Price: 4, Quantity: 5}},
		}, ScraperInfo{Name: "probe"}))
	if len(entries) != 1 {
		t.Fatalf("Mismatch returned %d entries, want 1", len(entries))
	}
	got := entries[0]
	if got.ReferenceEntry.Price != 10 || got.InventoryEntry.Price != 4 {
		t.Errorf("entry carries %v against %v, want 10 against 4",
			got.ReferenceEntry.Price, got.InventoryEntry.Price)
	}
	if got.Quantity != 2 {
		t.Errorf("Quantity = %d, want 2", got.Quantity)
	}
	if got.BuylistEntry.BuyPrice != 0 {
		t.Errorf("a mismatch filled in a buylist entry: %v", got.BuylistEntry)
	}
}

// TestArbitEntryString pins which side the printed line reads from: a buylist
// price where there is one, the reference price otherwise, and nothing at all
// for a card the datastore cannot name.
func TestArbitEntryString(t *testing.T) {
	installCards(t, plainCard())

	arbit := ArbitEntry{
		CardID:         "card",
		BuylistEntry:   BuylistEntry{BuyPrice: 15},
		InventoryEntry: InventoryEntry{Price: 10},
		Quantity:       3,
	}
	if got := arbit.String(); got == "" {
		t.Error("String() said nothing about a trade it can name")
	} else if want := "10.00 -> 15.00"; !contains(got, want) {
		t.Errorf("String() = %q, want it to carry %q", got, want)
	}

	mismatch := ArbitEntry{
		CardID:         "card",
		ReferenceEntry: InventoryEntry{Price: 12},
		InventoryEntry: InventoryEntry{Price: 10},
		Quantity:       1,
	}
	if want := "10.00 ~ 12.00"; !contains(mismatch.String(), want) {
		t.Errorf("String() = %q, want it to carry %q", mismatch.String(), want)
	}

	unknown := ArbitEntry{CardID: "nothing", InventoryEntry: InventoryEntry{Price: 1}}
	if got := unknown.String(); got != "" {
		t.Errorf("String() = %q for a card the datastore does not hold, want empty", got)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// Nil options are the ones that filter nothing, and the rate they imply is
// one: a zero would price every card at nothing.
func TestResolveOptsDefaults(t *testing.T) {
	r := resolveOpts(nil)
	if r.rate != 1.0 {
		t.Errorf("rate = %v, want 1", r.rate)
	}
	if r.minPrice != 0 || r.minDiff != 0 || r.minSpread != 0 {
		t.Errorf("nil options filtered something: %+v", r)
	}

	// An explicit zero rate means the same thing: unset.
	if got := resolveOpts(&ArbitOpts{Rate: 0}).rate; got != 1.0 {
		t.Errorf("rate = %v for an unset rate, want 1", got)
	}
	if got := resolveOpts(&ArbitOpts{Rate: 0.5}).rate; got != 0.5 {
		t.Errorf("rate = %v, want 0.5", got)
	}
}

// The stabilising constant is what keeps a cheap card from dominating the
// ranking, so it has to reach the denominator.
func TestProfitabilityConstantDampensCheapCards(t *testing.T) {
	installCards(t, plainCard())

	run := func(k float64) float64 {
		entries := Arbit(&ArbitOpts{ProfitabilityConstant: k},
			vendorOf(BuylistRecord{"card": {{Conditions: "NM", BuyPrice: 3}}}),
			sellerOf(InventoryRecord{"card": {{Conditions: "NM", Price: 1, Quantity: 1}}},
				ScraperInfo{Name: "seller"}))
		if len(entries) != 1 {
			t.Fatalf("Arbit returned %d entries, want 1", len(entries))
		}
		return entries[0].Profitability
	}
	plain, damped := run(0), run(10)
	if !(damped < plain) {
		t.Errorf("a constant of 10 gave %v against %v unconstrained, want it lower", damped, plain)
	}
	want := (2.0 / (1.0 + 10)) * math.Log10(1+200)
	if math.Abs(damped-want) > 1e-9 {
		t.Errorf("Profitability = %v, want %v", damped, want)
	}
}

// TestMismatchFilters walks the thresholds against a pair that clears them
// all by default, the way the arbitrage ones are walked.
func TestMismatchFilters(t *testing.T) {
	reference := func() Seller {
		return sellerOf(InventoryRecord{
			"card": {{Conditions: "NM", Price: 10, Quantity: 2}},
		}, ScraperInfo{Name: "reference"})
	}
	probe := func() Seller {
		return sellerOf(InventoryRecord{
			"card": {{Conditions: "NM", Price: 5, Quantity: 2}},
		}, ScraperInfo{Name: "probe"})
	}

	for _, tt := range []struct {
		desc string
		opts *ArbitOpts
		want int
	}{
		{"no options keeps the pair", &ArbitOpts{}, 1},
		{"an ignored condition drops it", &ArbitOpts{Conditions: []string{"NM"}}, 0},
		{"a price floor above both drops it", &ArbitOpts{MinPrice: 11}, 0},
		{"a quantity floor above the stock drops it", &ArbitOpts{MinQuantity: 3}, 0},
		{"a difference floor above the gap drops it", &ArbitOpts{MinDiff: 6}, 0},
		{"a spread floor above the spread drops it", &ArbitOpts{MinSpread: 101}, 0},
		{"a spread ceiling below the spread drops it", &ArbitOpts{MaxSpread: 99}, 0},
		{"a profitability floor above the index drops it", &ArbitOpts{MinProfitability: 100}, 0},
		{"an ignored edition drops it", &ArbitOpts{Editions: []string{"Alpha Set"}}, 0},
		{"a price filter can refuse the pair", &ArbitOpts{
			CustomPriceFilter: func(string, InventoryEntry) (float64, bool) { return 1, true },
		}, 0},
		{"a price filter can rescale the reference", &ArbitOpts{
			CustomPriceFilter: func(string, InventoryEntry) (float64, bool) { return 0.4, false },
		}, 0},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			installCards(t, plainCard())
			if got := len(Mismatch(tt.opts, reference(), probe())); got != tt.want {
				t.Errorf("Mismatch returned %d entries, want %d", got, tt.want)
			}
		})
	}
}

// A card only one shop carries has nothing to be compared against.
func TestMismatchNeedsBothShops(t *testing.T) {
	installCards(t, plainCard())

	entries := Mismatch(nil,
		sellerOf(InventoryRecord{"card": {{Conditions: "NM", Price: 10, Quantity: 1}}},
			ScraperInfo{Name: "reference"}),
		sellerOf(InventoryRecord{"other": {{Conditions: "NM", Price: 5, Quantity: 1}}},
			ScraperInfo{Name: "probe"}))
	if len(entries) != 0 {
		t.Errorf("Mismatch returned %d entries, want none", len(entries))
	}
}

// And a card the datastore cannot name is skipped before any of it.
func TestMismatchSkipsAnUnknownCard(t *testing.T) {
	installCards(t, map[string]*mtgmatcher.CardObject{})

	entries := Mismatch(nil,
		sellerOf(InventoryRecord{"card": {{Conditions: "NM", Price: 10, Quantity: 1}}},
			ScraperInfo{Name: "reference"}),
		sellerOf(InventoryRecord{"card": {{Conditions: "NM", Price: 5, Quantity: 1}}},
			ScraperInfo{Name: "probe"}))
	if len(entries) != 0 {
		t.Errorf("Mismatch returned %d entries, want none", len(entries))
	}
}

// TestMismatchFiltersTheProbeSide pins the checks the inner loop makes on its
// own. Each mirrors one the reference is held to, and a pair where only the
// probe offends would pass the outer check and reach them.
func TestMismatchFiltersTheProbeSide(t *testing.T) {
	for _, tt := range []struct {
		desc       string
		opts       *ArbitOpts
		probeCond  string
		probePrice float64
	}{
		{"a condition ignored only on the probe", &ArbitOpts{Conditions: []string{"SP"}}, "SP", 5},
		{"a price floor the probe alone falls under", &ArbitOpts{MinPrice: 5}, "NM", 1},
		{"a probe asking nothing", nil, "NM", 0},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			installCards(t, plainCard())
			entries := Mismatch(tt.opts,
				sellerOf(InventoryRecord{
					"card": {{Conditions: "NM", Price: 10, Quantity: 1}},
				}, ScraperInfo{Name: "reference"}),
				sellerOf(InventoryRecord{
					"card": {{Conditions: tt.probeCond, Price: tt.probePrice, Quantity: 1}},
				}, ScraperInfo{Name: "probe"}))
			if len(entries) != 0 {
				t.Errorf("Mismatch returned %d entries, want none", len(entries))
			}
		})
	}
}
