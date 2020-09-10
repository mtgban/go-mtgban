package mtgmatcher

import (
	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

func GetUUIDs() map[string]cardobject {
	return backend.UUIDs
}

func GetSets() map[string]mtgjson.Set {
	return backend.Sets
}

func Unmatch(cardId string) (*Card, error) {
	if backend.UUIDs == nil {
		return nil, ErrDatastoreEmpty
	}

	co, found := backend.UUIDs[cardId]
	if !found {
		return nil, ErrCardUnknownId
	}

	out := &Card{
		Id:      cardId,
		Name:    co.Card.Name,
		Edition: co.Edition,
		Foil:    co.Foil,
		Number:  co.Card.Number,
	}
	return out, nil
}

func HasPromoPackPrinting(name string) bool {
	if backend.Sets == nil {
		return false
	}

	card, found := backend.Cards[Normalize(name)]
	if !found {
		cc := &Card{
			Name: name,
		}
		adjustName(cc)
		card, found = backend.Cards[Normalize(cc.Name)]
		if !found {
			return false
		}
	}
	for _, code := range card.Printings {
		set, found := backend.Sets[code]
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
