package fleshandblood

import (
	"errors"
	"strings"
	"testing"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// pitchNumberFixture is Dig In as the datastore published it until
// datastore-gen corrected the catalog's numbers: the Yellow filed at the
// Red's FAB384, the two pitches told apart by nothing a bare number carries.
const pitchNumberFixture = `{"data": {
	"game": "fleshandblood",
	"sets": {"PR": {"name": "Flesh and Blood: Promo Cards", "releaseDate": "2019-10-11"}},
	"cards": [
		{"color": "Red", "externalLinks": {"fabId": "FAB384", "tcgPlayerId": 657466}, "finish": "Rainbow Foil", "id": "fab384_657466_rainbowfoil", "image": "x", "name": "Dig In", "number": "FAB384", "rarity": "Promo", "setCode": "PR", "variant": "Red"},
		{"color": "Yellow", "externalLinks": {"fabId": "FAB385", "tcgPlayerId": 657467}, "finish": "Rainbow Foil", "id": "fab384_657467_rainbowfoil", "image": "x", "name": "Dig In", "number": "FAB384", "rarity": "Promo", "setCode": "PR", "variant": "Yellow FAB385"}
	]
}}`

// TestPitchVariantsOnABareNumberAlias pins that two pitches of a card at one
// number, with no pitch-less printing beside them, refuse a listing naming
// the number alone rather than guess. The golden suite pinned this on the
// published Dig In until its Yellow was given its own number.
func TestPitchVariantsOnABareNumberAlias(t *testing.T) {
	b, err := Load(strings.NewReader(pitchNumberFixture))
	if err != nil {
		t.Fatal(err)
	}
	got, err := b.Match(&mtgmatcher.InputCard{Name: "Dig In", Variation: "FAB384"})
	var alias *mtgmatcher.AliasingError
	if !errors.As(err, &alias) {
		t.Fatalf("Match(Dig In, FAB384) = %q, %v, want an aliasing error", got, err)
	}
	if got, err := b.Match(&mtgmatcher.InputCard{Name: "Dig In (Yellow)", Variation: "FAB384"}); err != nil || got != "fab384_657467_rainbowfoil" {
		t.Errorf("Match(Dig In (Yellow), FAB384) = %q, %v; the pitch still names its printing", got, err)
	}
}
