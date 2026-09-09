package fleshandblood

import (
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// promoTypeLabels are the words behind a token. Decoration is this side's
// job - the datastore publishes a slug and nothing else, and a slug cannot
// give back the boundaries it dropped, so "extendedart" reads as
// "Extendedart" to a title-caser and has to be written down instead.
//
// Only the tokens a title-caser gets wrong are here. A single word it gets
// right on its own - "marvel", "golden", "treasure", "reverse", the pitch
// values and the elements - so listing those would only be a second place to
// keep them in step with the first.
var promoTypeLabels = map[string]string{
	"alternateart":         "Alternate Art",
	"bottomcenter":         "Bottom Center",
	"bottomleft":           "Bottom Left",
	"bottomright":          "Bottom Right",
	"cctag":                "CC Tag",
	"chinesealternateart":  "Chinese Alternate Art",
	"extendedart":          "Extended Art",
	"japanesealternateart": "Japanese Alternate Art",
	"japaneseexclusive":    "Japanese Exclusive",
	"middlecenter":         "Middle Center",
	"middleleft":           "Middle Left",
	"middleright":          "Middle Right",
	"pleiadessuperstar":    "Pleiades, Superstar",
	"topcenter":            "Top Center",
	"topleft":              "Top Left",
	"topright":             "Top Right",
}

// promoTypeLabel spells a token the way a reader should see it: the words
// above where a title-caser cannot work them out, and the title-cased token
// otherwise.
func promoTypeLabel(promoType string) string {
	if label, found := promoTypeLabels[promoType]; found {
		return label
	}
	return mtgmatcher.Title(promoType)
}
