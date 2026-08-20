package pokemon

import (
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// promoTypeWords spells the words title-casing cannot put back. The builder
// folds a qualifier to lower case before it reaches the datastore, which
// leaves the game's own suffixes and its set-series codes looking like
// ordinary words: "charizard gx" title-cases to "Charizard Gx", and the
// series prefixes to "Bw" and "Hgss". Everything else the catalog writes is
// ordinary words - the artist names most of all - and title-casing spells
// those itself.
//
// The spellings are the catalog's own, which writes these upper case
// wherever it qualifies a name with them.
//
// A word missing from this list still reads as something - it is title-cased
// like any other - so a promo the catalog adds tomorrow shows up plainly
// spelled rather than not at all.
var promoTypeWords = map[string]string{
	"bw":    "BW",
	"dp":    "DP",
	"ex":    "EX",
	"gx":    "GX",
	"hgss":  "HGSS",
	"pop":   "POP",
	"sm":    "SM",
	"sv":    "SV",
	"swsh":  "SWSH",
	"us":    "US",
	"v":     "V",
	"vmax":  "VMAX",
	"vstar": "VSTAR",
	"xy":    "XY",
}

// promoTypeLabel spells a qualifier the way a reader should see it: the
// catalog's own words, title-cased, with the words above put back as they
// are written. Unlike Magic and Riftbound, whose tokens ran their words
// together with nothing to split them on, these qualifiers keep their
// spaces, so the words themselves survive the fold and only their case is
// lost.
func promoTypeLabel(promoType string) string {
	words := strings.Split(mtgmatcher.Title(promoType), " ")
	for i, word := range words {
		if spelled, found := promoTypeWords[strings.ToLower(word)]; found {
			words[i] = spelled
		}
	}
	return strings.Join(words, " ")
}
