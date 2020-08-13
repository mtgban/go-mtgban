package mtgmatcher

import (
	"strings"

	"github.com/kodabb/go-mtgmatcher/mtgmatcher/mtgjson"
)

func HasPromoPackPrinting(name string) bool {
	if sets == nil {
		return false
	}

	card, found := cards[Normalize(name)]
	if !found {
		cc := &Card{
			Name: name,
		}
		adjustName(cc)
		card, found = cards[Normalize(cc.Name)]
		if !found {
			return false
		}
	}
	for _, code := range card.Printings {
		set, found := sets[code]
		if !found || set.IsOnlineOnly {
			continue
		}
		for _, in := range set.Cards {
			if (card.Name == in.Name) &&
				(strings.HasSuffix(in.Number, "p") ||
					in.HasFrameEffect(mtgjson.FrameEffectInverted) ||
					IsBasicLand(name)) {
				return true
			}
		}
	}

	return false
}
