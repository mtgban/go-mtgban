package fleshandblood

import (
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// promoTypeWords spells the words title-casing cannot put back. The builder
// folds a qualifier to lower case before it reaches the datastore, which
// leaves an acronym looking like an ordinary word: "cc tag" title-cases to
// "Cc Tag", and the set codes to "Aur" and "Fab362". Everything
// else the catalog writes is ordinary words, and title-casing spells those
// itself.
//
// A word missing from this list still reads as something - it is title-cased
// like any other - so a promo the catalog adds tomorrow shows up plainly
// spelled rather than not at all.
var promoTypeWords = map[string]string{
	"aur":    "AUR",
	"cc":     "CC",
	"fab362": "FAB362",
	"fab385": "FAB385",
	"fab442": "FAB442",
	"her052": "HER052",
	"jdg070": "JDG070",
	"jdg082": "JDG082",
	"jpn":    "JPN",
	"lgs247": "LGS247",
	"lgs391": "LGS391",
	"lgs423": "LGS423",
	"ter":    "TER",
	"tnp020": "TNP020",
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
