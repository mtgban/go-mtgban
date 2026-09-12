package magic

import (
	"slices"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Exact descriptions only: unknown compound descriptions require text matching.
var identifierPromoTypes = map[string]string{
	"prerelease":        PromoTypePrerelease,
	"promopack":         PromoTypePromoPack,
	"fnm":               PromoTypeFNM,
	"judge":             PromoTypeJudgeGift,
	"arena":             PromoTypeArenaLeague,
	"arenapromo":        PromoTypeArenaLeague,
	"surge":             PromoTypeSurgeFoil,
	"silverscreen":      PromoTypeSilverFoil,
	"releasepromo":      PromoTypeRelease,
	"intropackpromo":    PromoTypeIntroPack,
	"storechampionship": PromoTypeStoreChampionship,
	"wpn":               PromoTypeWPN,
	"playerrewards":     PromoTypePlayerRewards,
}

// ValidatePrinting checks a known printing without Match's best-effort fallback
// or lone-candidate shortcuts. Vendor spelling corrections belong in the caller.
func (Rules) ValidatePrinting(b *mtgmatcher.Backend, in *mtgmatcher.InputCard, co *mtgmatcher.CardObject) bool {
	variation := in.Variation
	if variation == "" {
		// Some plain listings carry the IDs of their extended or showcase
		// sibling. Keep the text path where the set also sells a plain card.
		for _, plain := range b.MatchInSet(in.Name, co.SetCode) {
			if plain.Number == co.Number || plain.Language != co.Language ||
				plain.HasPromoType(PromoTypeBoosterfun) || plain.BorderColor == "borderless" ||
				plain.HasFrameEffect("showcase") || plain.HasFrameEffect("extendedart") {
				continue
			}
			differentFrame := co.HasPromoType(PromoTypeBoosterfun) ||
				(len(co.FrameEffects) > 0 && !slices.Equal(co.FrameEffects, plain.FrameEffects))
			finishAvailable := plain.HasFinish(mtgmatcher.FinishNonfoil)
			if in.Foil {
				finishAvailable = plain.HasFinish(mtgmatcher.FinishFoil) || plain.HasFinish(mtgmatcher.FinishEtched)
			}
			if differentFrame && finishAvailable {
				return false
			}
		}
		return true
	}
	// Named edition aliases include NYCC 2024. A stated year must agree
	// with the identified printing, not merely its broad promo set.
	if set, err := b.GetSetByName(variation); err == nil && set.Code == co.SetCode {
		year := mtgmatcher.ExtractYear(variation)
		if year == "" || strings.HasPrefix(co.OriginalReleaseDate, year) {
			return true
		}
	}
	if promoType, ok := identifierPromoTypes[mtgmatcher.Normalize(variation)]; ok {
		return co.HasPromoType(promoType)
	}

	number := mtgmatcher.ExtractNumberAny(variation)
	if number != "" {
		// A title's own number outranks IDs copied from a sibling: Plains (38)
		// carries the IDs and card_number of #40; Secret Lair 2388 carries #41.
		if number != co.Number {
			return false
		}
		variation = strings.TrimSpace(strings.Replace(variation, number, "", 1))
	}
	switch mtgmatcher.Normalize(variation) {
	case "":
		return true
	case "borderless":
		return borderlessTreatment.carriedBy(&co.Card)
	case "extendedart":
		return extendedArtTreatment.carriedBy(&co.Card)
	case "showcase":
		return showcaseTreatment.carriedBy(&co.Card)
	case "retroframe":
		return co.FrameVersion == "1997"
	case "etched":
		return co.Etched
	case "thelist":
		return co.SetCode == "PLST"
	case "bundlepromo":
		return co.HasPromoType("bundle")
	case "secretlair":
		return number != "" && strings.Contains(co.Edition, "Secret Lair")
	case "magicfest":
		return strings.HasPrefix(co.Edition, "MagicFest ")
	default:
		return false
	}
}
