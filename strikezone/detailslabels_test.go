package strikezone

import "testing"

// TestPreprocessDetailsLabels pins what comes off a Pokemon row's name. The
// storefront writes the number as the card's share of the set and hangs the
// promo's labels behind it, and those labels name a printing the catalog
// files in a promo set of its own - so the row's own Set column names the set
// that numbers the card and excludes the printing being priced.
func TestPreprocessDetailsLabels(t *testing.T) {
	const details = "Near Mint Normal 1st Edition English"
	for _, tt := range []struct {
		desc                        string
		cardName, edition, number   string
		wantName, wantEd, wantVaria string
	}{
		{
			desc:     "the labels behind a share-of-set number come off",
			cardName: "Hop's Snorlax - 117/159 (GameStop) (Cosmos Holo)",
			edition:  "SV09: Journey Together", number: "117G",
			wantName: "Hop's Snorlax", wantEd: "",
			wantVaria: "117 (GameStop) (Cosmos Holo) 1st Edition",
		},
		{
			desc:     "a bare share-of-set number leaves the set standing",
			cardName: "Hop's Snorlax - 117/159",
			edition:  "SV09: Journey Together", number: "117",
			wantName: "Hop's Snorlax", wantEd: "SV09: Journey Together",
			wantVaria: "117 1st Edition",
		},
		{
			// The catalog labels this printing rather than naming it, so
			// the wording is a label like any other and the shelf it names
			// is the one that numbers the card, not the one that stamped it.
			desc:     "a promo's own wording is a label",
			cardName: "Gengar - SWSH241 (Prerelease)",
			edition:  "SWSH: Sword & Shield Promo Cards", number: "241",
			wantName: "Gengar", wantEd: "",
			wantVaria: "SWSH241 (Prerelease) 1st Edition",
		},
		{
			// The storefront closes a number with a letter of its own where
			// it files a promo under a number already taken, and that letter
			// is not the catalog's, so the name's fuller form is the one to
			// keep.
			desc:     "a marked number defers to the one in the name",
			cardName: "Bulbasaur - SM198 (Detective Pikachu Stamped)",
			edition:  "SM Promos", number: "198D",
			wantName: "Bulbasaur", wantEd: "",
			wantVaria: "SM198 (Detective Pikachu Stamped) 1st Edition",
		},
		{
			desc:     "a fuller promo number still replaces the column",
			cardName: "Dragapult (Prime) - SWSH132",
			edition:  "SWSH: Black Star Promos", number: "132",
			wantName: "Dragapult (Prime)", wantEd: "SWSH: Black Star Promos",
			wantVaria: "SWSH132 1st Edition",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			got, err := preprocessDetails(GamePokemon, tt.cardName, tt.edition, tt.number, details)
			if err != nil {
				t.Fatal(err)
			}
			if got.Name != tt.wantName || got.Edition != tt.wantEd || got.Variation != tt.wantVaria {
				t.Errorf("\n got  name=%q edition=%q variation=%q\n want name=%q edition=%q variation=%q",
					got.Name, got.Edition, got.Variation, tt.wantName, tt.wantEd, tt.wantVaria)
			}
		})
	}
}
