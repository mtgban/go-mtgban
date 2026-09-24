package onepiece

import (
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// promoTypeSpelling is one promo type as the words it reads as, which is
// what a reader is shown and never what a query carries. The datastore
// publishes the slug ("tournamentpack"), so the words are looked up, with the
// title-casing that cannot put spaces back where no spelling is kept.
func promoTypeSpelling(promoType string) string {
	if words := promoTypeLabels[promoType]; words != "" {
		return words
	}
	return mtgmatcher.Title(promoType)
}

// quotedRarities are the rarities this catalog writes in a product name as
// well as in the rarity field: the Treasure Rares, which arrive as "Vista
// (TR)". Every other rarity is only ever the field.
var quotedRarities = map[string]string{"TR": "TR"}

// quotedRarity is the rarity a listing may name, empty for one it never does.
func quotedRarity(rarity string) string {
	return quotedRarities[rarity]
}

// promoDate is the date a printing's label stated, written the way the
// catalog wrote it: a month and a year where the label named a month, and
// the bare year where it named only that. A date the set itself states is
// not published and so is not read here.
func promoDate(published string) string {
	if len(published) != 10 {
		return ""
	}
	if published[5:7] == "01" && published[8:] == "01" {
		return published[:4]
	}
	month := time.Month(int(published[5]-'0')*10 + int(published[6]-'0'))
	return month.String() + " " + published[:4]
}
