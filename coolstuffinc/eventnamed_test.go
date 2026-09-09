package coolstuffinc

import (
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestEventNamed pins the events this storefront gives its own name to. The
// catalog files the card under the pack it was handed out in and the listing
// goes up under the art, so the wording named nothing and the base printing
// answered - $1400 on one of them.
func TestEventNamed(t *testing.T) {
	withGameDatastore(t, "onepiece", "ONEPIECE_PATH")

	tests := []struct {
		desc       string
		name       string
		edition    string
		number     string
		wantSet    string
		wantPromos [][]string
	}{
		{
			desc: "the promo named after the art it shows",
			name: "Monkey.D.Luffy (073) (Afro Luffy Promo)", number: "OP07-073",
			edition: "OP07 - 500 Years in the Future",
			wantSet: "OP-PR",
			// The season is a tag of its own where the datastore takes it
			// off the fest's name and files it as the mark it is.
			wantPromos: [][]string{{"bandaicardgamesfest2526"}, {"bandaicardgamesfest", "2526"}},
		},
		{
			desc: "the promo named after the team it was handed out by",
			name: "Monkey.D.Luffy (EB02-010) (L.A. Dodgers Promo)", number: "EB02-010",
			edition: "EB02 - Anime 25th Collection",
			wantSet: "OP-PR", wantPromos: [][]string{{"dodgersxonepiece"}},
		},
		{
			desc: "and the plain listing beside them is unmoved",
			name: "Monkey.D.Luffy (073)", number: "OP07-073",
			edition: "OP07 - 500 Years in the Future",
			wantSet: "OP07",
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			card := &mtgmatcher.InputCard{
				Name:      test.name,
				Edition:   test.edition,
				Variation: eventNamed(test.number + " " + nameQualifiers(test.name)),
				Foil:      true,
			}
			id, err := mtgmatcher.Match(card)
			if err != nil {
				t.Fatalf("Match(%q) = %v", card, err)
			}
			co, err := mtgmatcher.GetUUID(id)
			if err != nil {
				t.Fatal(err)
			}
			if co.SetCode != test.wantSet {
				t.Errorf("Match(%q) = set %s, want %s", card, co.SetCode, test.wantSet)
			}
			if len(test.wantPromos) == 0 {
				if len(co.PromoTypes) != 0 {
					t.Errorf("Match(%q) = %v, want no label", card, co.PromoTypes)
				}
				return
			}
			if !promoTypesSpell(co.PromoTypes, test.wantPromos...) {
				t.Errorf("Match(%q) = %v, want %v spelled among them",
					card, co.PromoTypes, test.wantPromos)
			}
		})
	}
}
