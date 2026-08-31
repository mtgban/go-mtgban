package mtgban

import "testing"

// TestCollapsedPricings pins what the report names and what it stays quiet
// about. Every case below is a shape the arbitrage audit found in a live
// buylist: the four Neon Ink colours a shop buys at $300, $40, $18 and $17
// under one id, and the cash-and-credit pair that looks the same and is not.
func TestCollapsedPricings(t *testing.T) {
	for _, tt := range []struct {
		desc    string
		buylist BuylistRecord
		want    []string
	}{
		{
			desc: "four products on one id name the card",
			buylist: BuylistRecord{
				"neonink": {
					{Conditions: "NM", BuyPrice: 300, VendorName: "Strike Zone"},
					{Conditions: "NM", BuyPrice: 40, VendorName: "Strike Zone"},
					{Conditions: "NM", BuyPrice: 18, VendorName: "Strike Zone"},
					{Conditions: "NM", BuyPrice: 17, VendorName: "Strike Zone"},
				},
			},
			want: []string{"neonink"},
		},
		{
			desc: "one price at one grade is a shop doing its job",
			buylist: BuylistRecord{
				"ordinary": {
					{Conditions: "NM", BuyPrice: 10, VendorName: "Strike Zone"},
					{Conditions: "SP", BuyPrice: 8, VendorName: "Strike Zone"},
					{Conditions: "MP", BuyPrice: 5, VendorName: "Strike Zone"},
				},
			},
		},
		{
			// ABU Games publishes cash and store credit as two vendors in
			// one record. Both are the same card at the same grade, and
			// naming them would bury every pairing that is not.
			desc: "cash beside store credit is two vendors, not two products",
			buylist: BuylistRecord{
				"credit": {
					{Conditions: "NM", BuyPrice: 10, VendorName: "ABU Games (credit)"},
					{Conditions: "NM", BuyPrice: 4, VendorName: "ABU Games"},
				},
			},
		},
		{
			desc: "a quantity tier moves a price honestly and stays quiet",
			buylist: BuylistRecord{
				"tiered": {
					{Conditions: "NM", BuyPrice: 5.50, Quantity: 4, VendorName: "Card Kingdom"},
					{Conditions: "NM", BuyPrice: 5, Quantity: 20, VendorName: "Card Kingdom"},
				},
			},
		},
		{
			desc: "a grade with a second price is named on its own",
			buylist: BuylistRecord{
				"onegrade": {
					{Conditions: "NM", BuyPrice: 9, VendorName: "Hareruya"},
					{Conditions: "SP", BuyPrice: 120, VendorName: "Hareruya"},
					{Conditions: "SP", BuyPrice: 6, VendorName: "Hareruya"},
				},
			},
			want: []string{"onegrade"},
		},
		{
			desc: "a price of zero says nothing to compare against",
			buylist: BuylistRecord{
				"zeroed": {
					{Conditions: "NM", BuyPrice: 40, VendorName: "Magic Corner"},
					{Conditions: "NM", BuyPrice: 0, VendorName: "Magic Corner"},
				},
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got := CollapsedPricings(tt.buylist, CollapsedRatioThreshold)
			if len(got) != len(tt.want) {
				t.Fatalf("named %d cards %v, want %d %v", len(got), collapsedIDs(got), len(tt.want), tt.want)
			}
			for i := range got {
				if got[i].CardID != tt.want[i] {
					t.Errorf("card %d is %q, want %q", i, got[i].CardID, tt.want[i])
				}
			}
		})
	}
}

// TestCollapsedPricingsReport pins the figures the report carries, which are
// what a reader acts on: how far apart the prices are, how many there are,
// and the two listings that publish the wording telling them apart.
func TestCollapsedPricingsReport(t *testing.T) {
	buylist := BuylistRecord{
		"neonink": {
			{Conditions: "NM", BuyPrice: 300, VendorName: "Strike Zone", URL: "high"},
			{Conditions: "NM", BuyPrice: 40, VendorName: "Strike Zone", URL: "middle"},
			{Conditions: "NM", BuyPrice: 17, VendorName: "Strike Zone", URL: "low"},
		},
	}
	got := CollapsedPricings(buylist, CollapsedRatioThreshold)
	if len(got) != 1 {
		t.Fatalf("named %d cards, want 1", len(got))
	}
	entry := got[0]
	if entry.High != 300 || entry.Low != 17 {
		t.Errorf("prices are $%.2f and $%.2f, want $300.00 and $17.00", entry.High, entry.Low)
	}
	if entry.Count != 3 {
		t.Errorf("counted %d prices, want 3", entry.Count)
	}
	if entry.HighURL != "high" || entry.LowURL != "low" {
		t.Errorf("listings are %q and %q, want %q and %q", entry.HighURL, entry.LowURL, "high", "low")
	}
	if entry.VendorName != "Strike Zone" {
		t.Errorf("vendor is %q, want %q", entry.VendorName, "Strike Zone")
	}
}

// TestAddUnique pins the refusal a scraper opts into when its feed states one
// price per printing: Add keeps a second price because the two entries differ
// in exactly the field that is the symptom.
func TestAddUnique(t *testing.T) {
	first := BuylistEntry{Conditions: "NM", BuyPrice: 17, VendorName: "Strike Zone"}
	second := BuylistEntry{Conditions: "NM", BuyPrice: 300, VendorName: "Strike Zone"}

	kept := BuylistRecord{}
	if err := kept.Add("neonink", &first); err != nil {
		t.Fatal(err)
	}
	if err := kept.Add("neonink", &second); err != nil {
		t.Errorf("Add refused a second price: %v", err)
	}
	if len(kept["neonink"]) != 2 {
		t.Errorf("Add kept %d entries, want 2", len(kept["neonink"]))
	}

	refused := BuylistRecord{}
	if err := refused.AddUnique("neonink", &first); err != nil {
		t.Fatal(err)
	}
	err := refused.AddUnique("neonink", &second)
	if err == nil {
		t.Fatal("AddUnique accepted a second price at one grade")
	}
	if len(refused["neonink"]) != 1 {
		t.Errorf("AddUnique kept %d entries, want 1", len(refused["neonink"]))
	}

	// Another grade is another price, and another vendor is another shop.
	graded := BuylistEntry{Conditions: "SP", BuyPrice: 12, VendorName: "Strike Zone"}
	if err := refused.AddUnique("neonink", &graded); err != nil {
		t.Errorf("AddUnique refused another grade: %v", err)
	}
	credit := BuylistEntry{Conditions: "NM", BuyPrice: 21, VendorName: "Strike Zone (credit)"}
	if err := refused.AddUnique("neonink", &credit); err != nil {
		t.Errorf("AddUnique refused another vendor: %v", err)
	}
}

func collapsedIDs(list []CollapsedPricing) []string {
	var out []string
	for _, entry := range list {
		out = append(out, entry.CardID)
	}
	return out
}
