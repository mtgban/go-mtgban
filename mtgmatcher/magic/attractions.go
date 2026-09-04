package magic

import (
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// attractionLightsID keys the lit bulbs of an Unfinity attraction among the
// printing's identifiers, which is where they belong twice over: they are
// what identifies the printing - four Balloon Stands share a name and a set
// and differ in nothing but which of the six lights are on - and no other
// game prints anything like them, so the card type every game shares should
// not carry a field for them.
const attractionLightsID = "attractionLights"

// attractionTag spells the lit bulbs the way a storefront writes them, so a
// listing's own wording can be compared against the printing it claims.
func attractionTag(lights []int) string {
	tag := make([]string, 0, len(lights))
	for _, light := range lights {
		tag = append(tag, strconv.Itoa(light))
	}
	return strings.Join(tag, "/")
}

// AttractionLights returns the lit bulbs of the printing as a storefront
// writes them ("2/6"), or an empty string where it is not an attraction.
func AttractionLights(card *mtgmatcher.Card) string {
	return card.Identifiers[attractionLightsID]
}

// hasAttractionPrinting reports whether the name is an attraction, asked of
// a listing whose number cannot be trusted to mean a collector number: an
// attraction's wording carries the lit bulbs where a number would go.
func hasAttractionPrinting(b *mtgmatcher.Backend, name string) bool {
	for _, card := range b.MatchInSet(name, "UNF") {
		if AttractionLights(&card) != "" {
			return true
		}
	}
	return false
}
