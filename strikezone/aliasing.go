package strikezone

import (
	"slices"

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

// contradictions are the ways a listing rules out a printing the matcher
// could not tell from the others. Each reads the printing alone, so a row can
// be taken out of the table without touching the rest.
var contradictions = []contradiction{
	wearsUnnamedPremiumFoil,
	wearsUnnamedBorderless,
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

// resolveAliasing narrows an ambiguous match down to the one printing the
// listing does not rule out. It returns "" when the survivors do not reduce
// to exactly one, leaving the original error in place.
func resolveAliasing(b *mtgmatcher.Backend, l aliasedListing, probe []string) string {
	var keep []string
	for _, id := range probe {
		co, err := b.GetUUID(id)
		if err != nil {
			return ""
		}
		ruledOut := slices.ContainsFunc(contradictions, func(contradicts contradiction) bool {
			return contradicts(b, l, co)
		})
		if !ruledOut {
			keep = append(keep, id)
		}
	}
	if len(keep) == 1 {
		return keep[0]
	}
	return ""
}
