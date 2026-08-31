package vegassingles

import "testing"

// TestPokemonNumberFix pins the numbers the storefront states for the two
// twin sets, whose numbering is its own. The name is right and the number is
// a reordering of the set's, so the matcher - which reads the name first -
// finds no printing of that name at that number and drops the row.
//
// The key states the wrong number too, so a number the storefront corrects
// stops matching any entry and is read as written. The last case below is
// that: the same card with the number it actually carries.
func TestPokemonNumberFix(t *testing.T) {
	for _, tt := range []struct {
		desc, display, setName, want string
	}{
		{
			desc:    "a number the storefront states for another card's slot",
			display: "Throh 134/086  - Holofoil SV Black Bolt - Illustration Rare",
			setName: "SV: Black Bolt", want: "128/086",
		},
		{
			desc:    "and one in the main run",
			display: "Alomomola 027/086  - Holofoil SV Black Bolt - Uncommon",
			setName: "SV: Black Bolt", want: "024/086",
		},
		{
			desc:    "the other twin is corrected the same way",
			display: "Watchog 152/086  - Holofoil SV White Flare - Illustration Rare",
			setName: "SV: White Flare", want: "153/086",
		},
		{
			desc:    "a number the storefront already states correctly is left alone",
			display: "Throh 128/086  - Holofoil SV Black Bolt - Illustration Rare",
			setName: "SV: Black Bolt", want: "128/086",
		},
		{
			desc:    "and so is a set the table says nothing about",
			display: "Pikachu 025/165  - Holofoil SV 151 - Common",
			setName: "SV: 151", want: "025/165",
		},
	} {
		t.Run(tt.desc, func(t *testing.T) {
			product := VSProduct{
				DisplayName: tt.display,
				ProductData: VSProductData{SetName: tt.setName},
			}
			card, err := preprocessPokemon(product)
			if err != nil {
				t.Fatalf("preprocessPokemon(%q) = %v", tt.display, err)
			}
			if card.Variation != tt.want {
				t.Errorf("preprocessPokemon(%q) numbered %q, want %q", tt.display, card.Variation, tt.want)
			}
		})
	}
}
