package strikezone

import (
	"slices"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
	"github.com/mtgban/go-mtgban/mtgmatcher/magic"
)

// aliasedListing is a row the matcher answered with several printings: the
// product's name as the store writes it, the shelf it sits on, and the card
// preprocess made of them.
type aliasedListing struct {
	name  string
	shelf string
	card  *mtgmatcher.InputCard
}

// contradiction reports whether a listing's wording rules out a printing.
type contradiction func(b *mtgmatcher.Backend, l aliasedListing, co *mtgmatcher.CardObject) bool

// contradictions are the ways a Magic listing rules out a printing the
// matcher could not tell from the others. Each reads the printing alone, so a
// row can be taken out of the table without touching the rest.
var contradictions = []contradiction{
	wearsUnnamedPremiumFoil,
	wearsUnnamedBorderless,
	wearsUnnamedFlavor,
	notPrintedInFinish,
}

// wearsUnnamedPremiumFoil reports whether the printing wears a premium foil
// treatment the listing never names, which this storefront spells out on
// every one of them.
func wearsUnnamedPremiumFoil(_ *mtgmatcher.Backend, l aliasedListing, co *mtgmatcher.CardObject) bool {
	for _, treatment := range premiumFoilTreatments {
		if co.HasPromoType(treatment.promoType) && !mtgmatcher.Contains(l.card.Variation, treatment.tag) {
			return true
		}
	}
	return false
}

// wearsUnnamedBorderless reports whether the printing is a borderless one the
// listing never calls borderless, which the store writes on every one of them.
func wearsUnnamedBorderless(_ *mtgmatcher.Backend, l aliasedListing, co *mtgmatcher.CardObject) bool {
	return co.HasPromoType(magic.PromoTypeBorderless) && !mtgmatcher.Contains(l.card.Variation, "Borderless")
}

// wearsUnnamedFlavor reports whether the printing goes by a flavor name the
// listing does not write, a name the store gives in the product's own.
func wearsUnnamedFlavor(_ *mtgmatcher.Backend, l aliasedListing, co *mtgmatcher.CardObject) bool {
	return co.FlavorName != "" && !mtgmatcher.Contains(l.name, co.FlavorName)
}

// notPrintedInFinish reports whether the printing was never made in the
// finish the listing is on sale in.
func notPrintedInFinish(_ *mtgmatcher.Backend, l aliasedListing, co *mtgmatcher.CardObject) bool {
	return !co.HasFinish(requestedFinish(l.card.Foil, l.card.Variation))
}

// resolveAliasing narrows an ambiguous match down to the one printing the
// listing does not rule out, or among several, the one in the set the shelf
// is named for. It returns "" when the survivors do not reduce to exactly
// one, leaving the original error in place.
//
// A promo shelf holds promos only, so a printing that is not one is not the
// answer however few survive: a lone survivor there is what the rest was
// ruled out for, not what the listing says.
func resolveAliasing(b *mtgmatcher.Backend, l aliasedListing, probe []string) string {
	var keep []string
	for _, id := range probe {
		co, err := b.GetUUID(id)
		if err != nil {
			return ""
		}
		ruledOut := b.Game == mtgmatcher.GameMagic && slices.ContainsFunc(contradictions, func(contradicts contradiction) bool {
			return contradicts(b, l, co)
		})
		if !ruledOut {
			keep = append(keep, id)
		}
	}
	if len(keep) > 1 {
		keep = slices.DeleteFunc(keep, func(id string) bool {
			co, err := b.GetUUID(id)
			return err != nil || reprintedAfterPromoPack(b, l, co)
		})
	}
	if len(keep) > 1 {
		keep = slices.DeleteFunc(keep, func(id string) bool {
			co, err := b.GetUUID(id)
			return err != nil || !namesSet(b, l, co)
		})
	}
	if len(keep) != 1 {
		return ""
	}
	co, err := b.GetUUID(keep[0])
	if err != nil || (isPromoShelf(l.shelf) && !co.IsPromo) {
		return ""
	}
	return keep[0]
}

// namesSet reports whether the shelf, or the edition preprocess made of it, is
// the name of the printing's set.
func namesSet(b *mtgmatcher.Backend, l aliasedListing, co *mtgmatcher.CardObject) bool {
	set, found := b.Sets[co.SetCode]
	return found && (mtgmatcher.Equals(set.Name, l.shelf) || mtgmatcher.Equals(set.Name, l.card.Edition))
}

// reprintedAfterPromoPack reports whether the printing sits in a set released
// after the one a "Promo Pack: <Set>" shelf is named for. The store stocks
// the packs of that release only, so a later reprint of the same land is not
// what the shelf holds, however the set's own promo pack came to share it.
func reprintedAfterPromoPack(b *mtgmatcher.Backend, l aliasedListing, co *mtgmatcher.CardObject) bool {
	name, found := strings.CutPrefix(l.shelf, "Promo Pack: ")
	if !found {
		return false
	}
	shelfSet, err := b.GetSetByName(name)
	if err != nil {
		return false
	}
	set, err := b.GetSet(co.SetCode)
	return err == nil && set.ReleaseDateTime.After(shelfSet.ReleaseDateTime)
}

// isPromoShelf reports whether the shelf is one of the promo categories.
func isPromoShelf(shelf string) bool {
	return strings.HasPrefix(shelf, "Promos:") || strings.HasPrefix(shelf, "Promo Pack:")
}
