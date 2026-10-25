package cardgamecorner

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/matcher"
)

// listedCodeRe matches the collector code a One Piece name carries, ahead of
// the store's "(V.N)" index; some are written without the dash.
var listedCodeRe = regexp.MustCompile(`\(([A-Za-z]+[0-9]*-?[0-9]+[a-zA-Z]*)\)`)

// matchByProductImage answers a One Piece listing from the Cardmarket product
// id its image is named after: the id the datastore publishes, else the
// TCGplayer id the bridge links it to. The store numbers its versions the way
// Cardmarket does, which the datastore cannot derive. The answer is accepted
// only when the printing carries the collector code the listing's own name
// states, since the store also reuses one image across products.
func (cgc *Cardgamecorner) matchByProductImage(card *Card, foil bool) (string, bool) {
	image := mcmImageIDRe.FindStringSubmatch(card.Extra)
	code := listedCodeRe.FindStringSubmatch(card.Name)
	if image == nil || code == nil {
		return "", false
	}

	b := cgc.backend
	id := b.ConvertID(matcher.IDSpaceCardmarket, image[1])
	if id == "" {
		productID, err := strconv.Atoi(image[1])
		if err != nil {
			return "", false
		}
		tcgID, found := cgc.bridge[productID]
		if !found {
			return "", false
		}
		id = fmt.Sprint(tcgID)
	}

	cardID, err := b.MatchID(id, foil)
	if err != nil {
		return "", false
	}
	co, err := b.GetUUID(cardID)
	if err != nil || !strings.EqualFold(strings.ReplaceAll(co.Number, "-", ""), strings.ReplaceAll(code[1], "-", "")) {
		return "", false
	}
	return cardID, true
}
