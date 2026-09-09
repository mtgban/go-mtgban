package onepiece

import (
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// promoTypeValues are a printing's promo types as the datastore says them.
// They are slugged into tokens by the caller and spelled out by
// promoTypeSpelling; the two are not the same string, and running one
// through the other renamed the token "sp" to "special".
//
// A datastore built after the builder slugged them says a slug
// ("tournamentpack"), which has no spaces left to read, so promoTypeLabels
// puts the words back. One built before says the words already ("tournament
// pack vol. 1"), and those are their own label.
//
// marked says whether this datastore publishes the mark at all, and it is
// asked of the datastore rather than of the card. A printing's promo types
// are not its whole wording: what a DON!! card pictures is no promotion and
// is published as a mark, and every DON!! card is named "DON!! Card" at one
// number, so a datastore that does not publish marks has said the character
// nowhere but the variant. Reading the published list there would take the
// only fact telling one from another.
func promoTypeValues(card *DatastoreCard, marked bool) []string {
	if !marked {
		if card.Variant == "" {
			return nil
		}
		return []string{card.Variant}
	}
	return card.PromoTypes
}

// datastoreMarks says whether a datastore publishes the mark saying which
// copy of a number a printing is. One that does has had its subjects,
// instalments and decks taken out of the variant and put there.
func datastoreMarks(cards []DatastoreCard) bool {
	for i := range cards {
		if cards[i].Watermark != "" {
			return true
		}
	}
	return false
}

// promoTypeSpelling is one promo type as the words it reads as, which is
// what a reader is shown and never what a query carries. A published value
// carrying anything a slug cannot - a space, a dot, a bang - is the words
// already; one that is its own slug is looked up, and falls back on the
// title-casing that cannot put spaces back where no spelling is kept.
func promoTypeSpelling(promoType string) string {
	if mtgmatcher.PromoTypeSlug(promoType) != strings.ToLower(promoType) {
		return mtgmatcher.Title(promoType)
	}
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
