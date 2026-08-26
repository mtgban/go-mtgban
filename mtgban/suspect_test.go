package mtgban

import "testing"

// TestSuspectPricings pins what the report says and, as much, what it stays
// quiet about: a shop buying at a fraction of its asking price is doing its
// job, and saying so would bury the pairings that are not.
func TestSuspectPricings(t *testing.T) {
	for _, tt := range []struct {
		desc      string
		inventory InventoryRecord
		buylist   BuylistRecord
		want      []string
	}{
		{
			desc: "a shop buying at a quarter of what it asks is left alone",
			inventory: InventoryRecord{
				"ordinary": {{Conditions: "NM", Price: 10}},
			},
			buylist: BuylistRecord{
				"ordinary": {{Conditions: "NM", BuyPrice: 2.50}},
			},
		},
		{
			desc: "buying at the asking price names the pairing",
			inventory: InventoryRecord{
				"textured": {{Conditions: "NM", Price: 0.75}},
			},
			buylist: BuylistRecord{
				"textured": {{Conditions: "NM", BuyPrice: 0.75}},
			},
			want: []string{"textured"},
		},
		{
			desc: "buying above the asking price too",
			inventory: InventoryRecord{
				"secret": {{Conditions: "NM", Price: 2.99}},
			},
			buylist: BuylistRecord{
				"secret": {{Conditions: "NM", BuyPrice: 65}},
			},
			want: []string{"secret"},
		},
		{
			desc: "the worst pairing is named first",
			inventory: InventoryRecord{
				"near": {{Conditions: "NM", Price: 1}},
				"over": {{Conditions: "NM", Price: 1}},
			},
			buylist: BuylistRecord{
				"near": {{Conditions: "NM", BuyPrice: 0.95}},
				"over": {{Conditions: "NM", BuyPrice: 4}},
			},
			want: []string{"over", "near"},
		},
		{
			desc: "a grade is read against its own, never another",
			inventory: InventoryRecord{
				"graded": {{Conditions: "MP", Price: 1}},
			},
			buylist: BuylistRecord{
				"graded": {{Conditions: "NM", BuyPrice: 10}},
			},
		},
		{
			desc: "and against its own it is read",
			inventory: InventoryRecord{
				"graded": {{Conditions: "NM", Price: 20}, {Conditions: "MP", Price: 1}},
			},
			buylist: BuylistRecord{
				"graded": {{Conditions: "NM", BuyPrice: 5}, {Conditions: "MP", BuyPrice: 4}},
			},
			want: []string{"graded"},
		},
		{
			desc: "a card the shop only sells says nothing",
			inventory: InventoryRecord{
				"sellonly": {{Conditions: "NM", Price: 10}},
			},
			buylist: BuylistRecord{},
		},
		{
			desc:      "nor one it only buys",
			inventory: InventoryRecord{},
			buylist: BuylistRecord{
				"buyonly": {{Conditions: "NM", BuyPrice: 10}},
			},
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got := SuspectPricings(tt.inventory, tt.buylist, SuspectRatioThreshold)
			if len(got) != len(tt.want) {
				t.Fatalf("named %d pairings %v, want %d %v", len(got), ids(got), len(tt.want), tt.want)
			}
			for i := range got {
				if got[i].CardID != tt.want[i] {
					t.Errorf("pairing %d is %q, want %q", i, got[i].CardID, tt.want[i])
				}
			}
		})
	}
}

func ids(list []SuspectPricing) []string {
	var out []string
	for _, s := range list {
		out = append(out, s.CardID)
	}
	return out
}
