package yugioh

import (
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// promoTypeWords spells the words title-casing cannot put back. The builder
// folds a qualifier to lower case before it reaches the datastore, which
// leaves an acronym looking like an ordinary word: "ots stamp" title-cases to
// "Ots Stamp", and the rarity and set codes to "Scr" and "Dl18". Everything
// else the catalog writes is ordinary words, and title-casing spells those
// itself.
//
// A word missing from this list still reads as something - it is title-cased
// like any other - so a promo the catalog adds tomorrow shows up plainly
// spelled rather than not at all.
var promoTypeWords = map[string]string{
	"blu-ray":          "Blu-ray",
	"dl18":             "DL18",
	"dvd":              "DVD",
	"ensp1":            "ENSP1",
	"eu":               "EU",
	"gx":               "GX",
	"m24-en052":        "M24-EN052",
	"ots":              "OTS",
	"pre-registration": "Pre-registration",
	"pur":              "PUR",
	"scr":              "SCR",
	"se":               "SE",
	"sho":              "SHO",
	"sr":               "SR",
	"uds":              "UDS",
	"wb":               "WB",
	"ycs":              "YCS",
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
