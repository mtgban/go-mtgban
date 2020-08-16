package mtgmatcher

import (
	"github.com/kodabb/go-mtgmatcher/mtgmatcher/mtgjson"
)

func GetUUIDs() map[string]cardobject {
	return uuids
}

func GetSets() map[string]mtgjson.Set {
	return sets
}

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
			if (card.Name == in.Name) && in.HasPromoType(mtgjson.PromoTypePromoPack) {
				return true
			}
		}
	}

	return false
}
