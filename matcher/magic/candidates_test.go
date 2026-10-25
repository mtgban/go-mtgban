package magic

import (
	"slices"
	"testing"
	"time"

	"github.com/mtgban/go-mtgban/matcher"
)

func TestMagicCandidateSets(t *testing.T) {
	for _, tt := range []struct {
		name      string
		in        matcher.InputCard
		printings []string
		want      []string
	}{
		{"single printing", matcher.InputCard{Edition: "Alpha", Variation: "Prerelease"}, []string{"A"}, []string{"A"}},
		{"exact", matcher.InputCard{Edition: "Alpha"}, []string{"A", "B"}, []string{"A"}},
		{"prerelease sibling outside printing list", matcher.InputCard{Edition: "Alpha", Variation: "Prerelease"}, []string{"A", "B"}, []string{"A", "PA"}},
		{"promo pack sibling", matcher.InputCard{Edition: "Alpha", Variation: "Promo Pack"}, []string{"A", "B"}, []string{"A", "PA"}},
		{"reverse sibling", matcher.InputCard{Edition: "Alpha Promos", Variation: "Prerelease"}, []string{"PA", "B"}, []string{"PA", "A"}},
		{"Japanese stays exact", matcher.InputCard{Edition: "Alpha", Variation: "Prerelease JPN", Language: "Japanese"}, []string{"A", "B"}, []string{"A"}},
		{"missing sibling", matcher.InputCard{Edition: "Beta", Variation: "Prerelease"}, []string{"A", "B"}, []string{"B"}},
		{"partial edition", matcher.InputCard{Edition: "Alph"}, []string{"A", "B"}, []string{"A"}},
		{"generic promo", matcher.InputCard{Edition: "Promos"}, []string{"A", "PA", "B"}, []string{"PA"}},
		{"unknown edition", matcher.InputCard{Edition: "Elsewhere"}, []string{"A", "B"}, []string{"A", "B"}},
		{"wildcard", matcher.InputCard{Edition: "Alpha", PromoWildcard: true}, []string{"A", "B"}, []string{"A", "B"}},
		{"Secret Lair skips exact selection", matcher.InputCard{Edition: "Secret Lair Drop"}, []string{"SLD", "A"}, []string{"SLD", "A"}},
		{"World Championship skips loose selection", matcher.InputCard{Edition: "World Championship"}, []string{"WC", "A"}, []string{"WC", "A"}},
		{"World Championship exact", matcher.InputCard{Edition: "World Championship 1999"}, []string{"WC", "A"}, []string{"WC"}},
		{"empty", matcher.InputCard{}, nil, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			b := &matcher.Backend{Sets: map[string]*matcher.Set{
				"A": {Name: "Alpha"}, "PA": {Name: "Alpha Promos"}, "B": {Name: "Beta"},
				"SLD": {Name: "Secret Lair Drop"}, "WC": {Name: "World Championship 1999"},
			}}
			got := (Rules{}).CandidateSets(b, &tt.in, tt.printings)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("candidates = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMagicCandidatePromoDateBoundaries(t *testing.T) {
	for _, promo := range []struct {
		name string
		date time.Time
	}{
		{"Buy-a-Box", BuyABoxInExpansionSetsDate}, {"Bundle", PromosForEverybodyYay},
	} {
		for _, delta := range []time.Duration{-time.Second, 0, time.Second} {
			t.Run(promo.name+delta.String(), func(t *testing.T) {
				b := &matcher.Backend{Sets: map[string]*matcher.Set{
					"A":  {Name: "Alpha", ReleaseDateTime: promo.date.Add(delta)},
					"PA": {Name: "Alpha Promos"}, "B": {Name: "Beta"},
				}}
				in := &matcher.InputCard{Edition: "Alpha", Variation: promo.name}
				want := []string{"A"}
				if delta > 0 {
					want = append(want, "PA")
				}
				got := (Rules{}).CandidateSets(b, in, []string{"A", "B"})
				if !slices.Equal(got, want) {
					t.Fatalf("exact candidates = %v, want %v", got, want)
				}
				// With no edition match, newer base sets qualify through the
				// same date rule; before and at the boundary, fall back to all.
				in.Edition = "Unknown"
				want = []string{"A", "B"}
				if delta > 0 {
					want = []string{"A"}
				}
				got = (Rules{}).CandidateSets(b, in, []string{"A", "B"})
				if !slices.Equal(got, want) {
					t.Fatalf("loose candidates = %v, want %v", got, want)
				}
			})
		}
	}
}

func TestMagicFinalizesWorldChampionshipCandidates(t *testing.T) {
	cards := []matcher.Card{{UUID: "first", Language: "French"}, {UUID: "second", Language: "English"}}
	for _, edition := range []string{"World Championship 1999", "Alpha"} {
		got := (Rules{}).FinalizeCandidates(nil, &matcher.InputCard{Edition: edition}, cards)
		want := 2
		if edition == "World Championship 1999" {
			want = 1
		}
		if len(got) != want || got[0].UUID != "first" {
			t.Fatalf("%s candidates = %v", edition, got)
		}
	}
}
