package mtgmatcher

import (
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

func GetUUIDs() map[string]CardObject {
	return backend.UUIDs
}

func GetUUID(uuid string) (*CardObject, error) {
	if backend.UUIDs == nil {
		return nil, ErrDatastoreEmpty
	}

	co, found := backend.UUIDs[uuid]
	if !found {
		return nil, ErrCardUnknownId
	}

	return &co, nil
}

func GetSets() map[string]*mtgjson.Set {
	return backend.Sets
}

func GetSet(code string) (*mtgjson.Set, error) {
	if backend.Sets == nil {
		return nil, ErrDatastoreEmpty
	}

	set, found := backend.Sets[strings.ToUpper(code)]
	if !found {
		return nil, ErrCardUnknownId
	}

	return set, nil
}

func GetSetByName(edition string, flags ...bool) (*mtgjson.Set, error) {
	if backend.Sets == nil {
		return nil, ErrDatastoreEmpty
	}

	card := &Card{
		Edition: edition,
	}
	if len(flags) > 0 {
		card.Foil = flags[0]
	}
	adjustEdition(card)

	for _, set := range backend.Sets {
		if Equals(set.Name, card.Edition) {
			return set, nil
		}
	}

	return nil, ErrCardUnknownId
}

func GetSetUUID(uuid string) (*mtgjson.Set, error) {
	if backend.UUIDs == nil || backend.Sets == nil {
		return nil, ErrDatastoreEmpty
	}

	co, found := backend.UUIDs[uuid]
	if !found {
		return nil, ErrCardUnknownId
	}

	set, found := backend.Sets[co.SetCode]
	if !found {
		return nil, ErrCardUnknownId
	}

	return set, nil
}

func Scryfall2UUID(id string) string {
	return backend.Scryfall[id]
}

func Tcg2UUID(id string) string {
	return backend.Tcgplayer[id]
}

func SearchEquals(name string) ([]string, error) {
	return searchEquals(name, backend.AllNames)
}

func SearchSealedEquals(name string) ([]string, error) {
	return searchEquals(name, backend.AllSealed)
}

func searchEquals(name string, slice []string) ([]string, error) {
	name = Normalize(name)
	for i := range slice {
		if slice[i] == name {
			return backend.Hashes[slice[i]], nil
		}
	}
	return nil, ErrCardDoesNotExist
}

func searchFunc(name string, slice []string, f func(string, string) bool) ([]string, error) {
	var hashes []string
	name = Normalize(name)
	for i := range slice {
		if f(slice[i], name) {
			hashes = append(hashes, backend.Hashes[slice[i]]...)
		}
	}
	if hashes == nil {
		return nil, ErrCardDoesNotExist
	}
	return hashes, nil
}

func SearchHasPrefix(name string) ([]string, error) {
	return searchFunc(name, backend.AllNames, strings.HasPrefix)
}

func SearchContains(name string) ([]string, error) {
	return searchFunc(name, backend.AllNames, strings.Contains)
}

func SearchSealedContains(name string) ([]string, error) {
	return searchFunc(name, backend.AllSealed, strings.Contains)
}

func Printings4Card(name string) ([]string, error) {
	entry, found := backend.Cards[Normalize(name)]
	if !found {
		return nil, ErrCardDoesNotExist
	}
	return entry.Printings, nil
}

func HasExtendedArtPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "frame_effect", mtgjson.FrameEffectExtendedArt, editions...)
}

func HasBorderlessPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "border_color", mtgjson.BorderColorBorderless, editions...)
}

func HasShowcasePrinting(name string, editions ...string) bool {
	return hasPrinting(name, "frame_effect", mtgjson.FrameEffectShowcase, editions...)
}

func HasReskinPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "promo_type", mtgjson.PromoTypeGodzilla, editions...)
}

func HasPromoPackPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "promo_type", mtgjson.PromoTypePromoPack, editions...)
}

func HasPrereleasePrinting(name string, editions ...string) bool {
	return hasPrinting(name, "promo_type", mtgjson.PromoTypePrerelease, editions...)
}

func HasRetroFramePrinting(name string, editions ...string) bool {
	return hasPrinting(name, "frame_version", "1997", editions...)
}

func HasNonfoilPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "finish", mtgjson.FinishNonfoil, editions...)
}

func HasFoilPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "finish", mtgjson.FinishFoil, editions...)
}

func HasEtchedPrinting(name string, editions ...string) bool {
	return hasPrinting(name, "finish", mtgjson.FinishEtched, editions...)
}

func hasPrinting(name, field, value string, editions ...string) bool {
	if backend.Sets == nil {
		return false
	}

	var checkFunc func(mtgjson.Card, string) bool
	switch field {
	case "promo_type":
		checkFunc = func(card mtgjson.Card, value string) bool {
			return card.HasPromoType(value)
		}
	case "frame_effect":
		checkFunc = func(card mtgjson.Card, value string) bool {
			return card.HasFrameEffect(value)
		}
	case "border_color":
		checkFunc = func(card mtgjson.Card, value string) bool {
			return card.BorderColor == value
		}
	case "frame_version":
		checkFunc = func(card mtgjson.Card, value string) bool {
			return card.FrameVersion == value
		}
	case "finish":
		checkFunc = func(card mtgjson.Card, value string) bool {
			return card.HasFinish(value)
		}
	default:
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
		var set *mtgjson.Set
		if len(editions) > 0 {
			set, found = backend.Sets[editions[0]]
			if !found {
				set, _ = GetSetByName(editions[0])
			}
		}
		if set == nil {
			set, found = backend.Sets[code]
			if !found {
				continue
			}
		}
		for _, in := range set.Cards {
			if (card.Name == in.Name) && checkFunc(in, value) {
				return true
			}
		}
	}

	return false
}
