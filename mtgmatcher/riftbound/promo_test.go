package riftbound

import (
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// promoFixture pins the promo-type machinery a datastore with clean promo
// names exercises: sibling variants share one name - and, for organized
// play, the main set's collector number - and are told apart by promo
// types against the storefront's own wording.
const promoFixture = `{
	"pageProps": {"page": {"blades": [{
		"type": "riftboundCardGallery",
		"sets": {"items": [
			{"id": "OGN", "name": "Origins", "collectorNumberMax": 298, "releaseDate": "2025-10-31"},
			{"id": "OPP", "name": "Riftbound Organized Play Promotional Cards", "type": "promo", "releaseDate": "2025-10-31"}
		]},
		"cards": {"items": [
			{
				"id": "ogn-139",
				"collectorNumber": 139,
				"name": "Fixture Blade",
				"publicCode": "OGN-139/298",
				"set": {"value": {"id": "OGN", "label": "Origins"}},
				"rarity": {"value": {"id": "rare"}},
				"finishes": ["foil"]
			},
			{
				"id": "opp-901",
				"collectorNumber": 139,
				"name": "Fixture Blade",
				"publicCode": "OPP-139/298",
				"set": {"value": {"id": "OPP", "label": "Riftbound Organized Play Promotional Cards"}},
				"rarity": {"value": {"id": "promo"}},
				"promoTypes": ["champion"],
				"finishes": ["foil"]
			},
			{
				"id": "opp-902",
				"collectorNumber": 139,
				"name": "Fixture Blade",
				"publicCode": "OPP-139/298",
				"set": {"value": {"id": "OPP", "label": "Riftbound Organized Play Promotional Cards"}},
				"rarity": {"value": {"id": "promo"}},
				"promoTypes": ["top 8"],
				"finishes": ["foil"]
			},
			{
				"id": "ogn-183",
				"collectorNumber": 183,
				"name": "Fixture Deck",
				"publicCode": "OGN-183/298",
				"set": {"value": {"id": "OGN", "label": "Origins"}},
				"rarity": {"value": {"id": "rare"}},
				"finishes": ["foil"]
			},
			{
				"id": "opp-903",
				"collectorNumber": 183,
				"name": "Fixture Deck",
				"publicCode": "OPP-183/298",
				"set": {"value": {"id": "OPP", "label": "Riftbound Organized Play Promotional Cards"}},
				"rarity": {"value": {"id": "promo"}},
				"promoTypes": ["prize wall"],
				"finishes": ["foil"]
			},
			{
				"id": "opp-904",
				"collectorNumber": 251,
				"name": "Fixture Cannon",
				"publicCode": "OPP-251/298",
				"set": {"value": {"id": "OPP", "label": "Riftbound Organized Play Promotional Cards"}},
				"rarity": {"value": {"id": "promo"}},
				"finishes": ["foil"]
			},
			{
				"id": "opp-905",
				"collectorNumber": 251,
				"name": "Fixture Cannon",
				"publicCode": "OPP-251b/298",
				"set": {"value": {"id": "OPP", "label": "Riftbound Organized Play Promotional Cards"}},
				"rarity": {"value": {"id": "promo"}},
				"promoTypes": ["metal", "best of"],
				"finishes": ["foil"]
			}
		]}
	}]}}
}`

func TestPromoTypeSelection(t *testing.T) {
	b, err := Load(strings.NewReader(promoFixture))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		desc string
		in   mtgmatcher.InputCard
		want string
		err  bool
	}{
		{
			desc: "the storefront wording picks the variant it describes",
			in:   mtgmatcher.InputCard{Name: "Fixture Blade (Champion Stamp)", Edition: "Promo", Variation: "139"},
			want: "opp-901_foil",
		},
		{
			desc: "a described variant outranks the plain promo sibling",
			in:   mtgmatcher.InputCard{Name: "Fixture Cannon (Metal) (Best Of)", Edition: "Promo"},
			want: "opp-905_foil",
		},
		{
			desc: "without a qualifier the plain promo answers, not a variant",
			in:   mtgmatcher.InputCard{Name: "Fixture Cannon", Edition: "Promo", Variation: "251"},
			want: "opp-904_foil",
		},
		{
			desc: "a bare number reaches a typed variant under a promo heading",
			in:   mtgmatcher.InputCard{Name: "Fixture Deck", Edition: "Promo", Variation: "183", Foil: true},
			want: "opp-903_foil",
		},
		{
			desc: "a per-set promo heading acts as the generic one",
			in:   mtgmatcher.InputCard{Name: "Fixture Deck", Edition: "Origins: Promos", Variation: "183", Foil: true},
			want: "opp-903_foil",
		},
		{
			desc: "the main edition still answers the main printing",
			in:   mtgmatcher.InputCard{Name: "Fixture Deck", Edition: "Origins", Variation: "183", Foil: true},
			want: "ogn-183_foil",
		},
		{
			desc: "a qualifier no variant carries refuses to pick one",
			in:   mtgmatcher.InputCard{Name: "Fixture Blade (Winner Stamp)", Edition: "Promo", Variation: "139"},
			err:  true,
		},
		{
			desc: "two variants both described stay ambiguous",
			in:   mtgmatcher.InputCard{Name: "Fixture Blade", Edition: "Promo", Variation: "139"},
			err:  true,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			in := test.in
			uuid, err := b.Match(&in)
			if test.err {
				if err == nil {
					t.Fatalf("Match = %q, want an error", uuid)
				}
				return
			}
			if err != nil {
				t.Fatalf("Match = %v, want %q", err, test.want)
			}
			if uuid != test.want {
				t.Errorf("Match = %q, want %q", uuid, test.want)
			}
		})
	}
}

func TestPromoSetReleaseDates(t *testing.T) {
	b, err := Load(strings.NewReader(promoFixture))
	if err != nil {
		t.Fatal(err)
	}
	set, found := b.Sets["OGN"]
	if !found {
		t.Fatal("OGN not loaded")
	}
	if set.ReleaseDate != "2025-10-31" {
		t.Errorf("ReleaseDate = %q, want %q", set.ReleaseDate, "2025-10-31")
	}
	if set.ReleaseDateTime.IsZero() {
		t.Error("ReleaseDateTime not parsed")
	}
}
