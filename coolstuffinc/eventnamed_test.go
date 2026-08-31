package coolstuffinc

import (
	"os"
	"testing"

	"github.com/mtgban/go-mtgban/internal/datastore"
	"github.com/mtgban/go-mtgban/mtgmatcher"

	_ "github.com/mtgban/go-mtgban/mtgmatcher/games"
)

// TestEventNamed pins the events this storefront gives its own name to. The
// catalog files the card under the pack it was handed out in and the listing
// goes up under the art, so the wording named nothing and the base printing
// answered - $1400 on one of them.
func TestEventNamed(t *testing.T) {
	path := os.Getenv("ONEPIECE_PATH")
	if path == "" {
		t.Skip("Need ONEPIECE_PATH variable set to run this test")
	}
	if err := datastore.Load(path); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc      string
		name      string
		edition   string
		number    string
		wantSet   string
		wantPromo string
	}{
		{
			desc: "the promo named after the art it shows",
			name: "Monkey.D.Luffy (073) (Afro Luffy Promo)", number: "OP07-073",
			edition: "OP07 - 500 Years in the Future",
			wantSet: "OP-PR", wantPromo: "bandaicardgamesfest2526",
		},
		{
			desc: "the promo named after the team it was handed out by",
			name: "Monkey.D.Luffy (EB02-010) (L.A. Dodgers Promo)", number: "EB02-010",
			edition: "EB02 - Anime 25th Collection",
			wantSet: "OP-PR", wantPromo: "dodgersxonepiece",
		},
		{
			desc: "and the plain listing beside them is unmoved",
			name: "Monkey.D.Luffy (073)", number: "OP07-073",
			edition: "OP07 - 500 Years in the Future",
			wantSet: "OP07", wantPromo: "",
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
			if test.wantPromo == "" {
				if len(co.PromoTypes) != 0 {
					t.Errorf("Match(%q) = %v, want no label", card, co.PromoTypes)
				}
				return
			}
			var found bool
			for _, promoType := range co.PromoTypes {
				if promoType == test.wantPromo {
					found = true
				}
			}
			if !found {
				t.Errorf("Match(%q) = %v, want one of them to be %q", card, co.PromoTypes, test.wantPromo)
			}
		})
	}
}
