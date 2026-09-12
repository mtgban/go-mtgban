package abugames

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// These Promo-shelf listings carry numbers and external IDs that agree on
// PF24 and PURL. Neither is the English PMEI manga insert. Keep the whole
// preprocessing path covered: dropping the MagicFest number loses its year.
func TestCounterspellPromoListings(t *testing.T) {
	realDatastore(t)
	for _, tt := range []struct {
		title, number, scryfall, set string
	}{
		{"Counterspell (MagicFest) - FOIL", "1", "8916e24f-9c74-4b6c-9894-d60669854f35", "PF24"},
		{"Counterspell (NYCC 2024) - FOIL", "2", "f2a7042f-a6f0-4e77-86a2-5eb0d2587363", "PURL"},
	} {
		t.Run(tt.title, func(t *testing.T) {
			in, err := preprocess(&ABUCard{
				DisplayTitle: tt.title, SimpleTitle: "Counterspell", Edition: "Promo",
				Number: tt.number, Language: []string{"English"},
			})
			if err != nil {
				t.Fatal(err)
			}
			id, err := mtgmatcher.Match(in)
			if err != nil {
				t.Fatal(err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != tt.set || co.Number != tt.number || !co.Foil {
				t.Fatalf("got %s, want %s #%s foil", co, tt.set, tt.number)
			}
			want, err := mtgmatcher.MatchID(tt.scryfall, true, false)
			if err != nil {
				t.Fatal(err)
			}
			if id != want {
				t.Errorf("wording resolved to %s, ABU's Scryfall ID resolves to %s", id, want)
			}
		})
	}
}
