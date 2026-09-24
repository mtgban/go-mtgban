package coolstuffinc

import "strings"

// lorcanaSpellings corrects the Lorcana names this storefront misspells.
// Each key is a name no set in the game has and each value is the card it
// means; left as typed they look up nothing at all and the listing goes
// unpriced.
//
// The storefront's own catalog is what says it is a slip rather than a
// name: it files "Basil - Perspective Investigator" at 140/204 of Rise of
// the Floodborn, under its own image sku dl_RotFB_140, and that set's
// number 140 is "Basil - Perceptive Investigator". Everything but the one
// word agrees.
//
// Spelled out rather than found by nearest match, for the reason
// csiSpellings gives - and the more so here, where a name is a character
// and a title joined by a dash and the titles are the whole of what tells
// one printing of a character from another. This set alone sells Basil
// under three of them at consecutive numbers, 138 through 140.
var lorcanaSpellings = map[string]string{
	"Basil - Perspective Investigator": "Basil - Perceptive Investigator",
}

// lorcanaSpelling spells a Lorcana name the way the catalog does, where
// this storefront has typed it wrong.
func lorcanaSpelling(name string) string {
	if spelled, found := lorcanaSpellings[name]; found {
		return spelled
	}
	return name
}

// lorcanaVariation spells "Rainbow Foil" the way TCGplayer names the printing
// it is, "Holofoil" - a Starter Deck Exclusive's rainbow foil otherwise names
// no finish the matcher knows and lands on the set's ordinary cold foil.
func lorcanaVariation(variation string) string {
	return strings.ReplaceAll(variation, "Rainbow Foil", "Holofoil")
}
