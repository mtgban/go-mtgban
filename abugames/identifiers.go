package abugames

import (
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// identifiedCard reads IDs before text matching. An ID must name the card in
// the listing and support its language, finish and printing descriptors. Both
// namespaces may carry copied IDs, so an unresolved disagreement falls back to
// text rather than giving either source unconditional priority.
func identifiedCard(card *ABUCard, in mtgmatcher.InputCard) *mtgmatcher.InputCard {
	var found string
	conflict := false
	consider := func(space, external string) {
		base := mtgmatcher.ConvertID(space, external)
		if base == "" {
			return
		}
		co, err := mtgmatcher.GetUUID(base)
		if err != nil {
			return
		}
		// Flavor-name misprints can have their own catalog entry (Luca/Luka
		// Stadium). A title spelling alone must not override its printed number.
		if (mtgmatcher.Equals(in.Name, co.FlavorName) || mtgmatcher.Equals(in.Name, co.FaceFlavorName)) && card.Number != "" && card.Number != co.Number {
			return
		}
		// Multiverse identifies a catalog printing, but older ABU shelves
		// include reprints and special editions with copied original IDs.
		if space == mtgmatcher.IDSpaceMultiverse {
			set, setErr := mtgmatcher.GetSetByName(card.Edition)
			if setErr != nil || set.Code != co.SetCode || (card.Number != "" && card.Number != co.Number) {
				return
			}
		}
		lang := in.Language
		if lang == "" {
			lang = "English"
		}
		if in.IsJPN() && lang != "Japanese" {
			return
		}
		finish := mtgmatcher.FinishNonfoil
		if in.IsEtched() {
			finish = mtgmatcher.FinishEtched
		} else if in.Foil {
			finish = mtgmatcher.FinishFoil
			// ABU calls both foil and etched-only printings FOIL.
			if !co.HasFinish(finish) && co.HasFinish(mtgmatcher.FinishEtched) {
				finish = mtgmatcher.FinishEtched
			}
		}
		probe := in
		probe.Name = identifierName(in.Name)
		probe.ID, probe.Finish, probe.Language = base, finish, lang
		id, err := mtgmatcher.ValidateID(probe, mtgmatcher.IDValidationOptions{AllowFinishSiblings: true})
		if err != nil {
			return
		}
		co, err = mtgmatcher.GetUUID(id)
		if err != nil {
			return
		}
		if set, err := mtgmatcher.GetSetByName(card.Edition); err == nil && set.Code != co.SetCode &&
			!(in.Contains("The List") && co.SetCode == "PLST") {
			return
		}
		if !idDescribesPrinting(card, &in, co) {
			return
		}
		if found != "" && found != id {
			conflict = true
		}
		found = id
	}
	for _, id := range card.ScryfallIDs {
		consider(mtgmatcher.IDSpaceScryfall, id)
	}
	for _, id := range card.TCGplayerIDs {
		if id > 0 {
			consider(mtgmatcher.IDSpaceTCGplayer, strconv.FormatInt(id, 10))
		}
	}
	// Older listings sometimes supply only a Multiverse ID. Keep it in its
	// own numeric namespace and apply the same listing checks. Do not let it
	// override a disagreement between the two product-level sources.
	if len(card.ScryfallIDs) == 0 && len(card.TCGplayerIDs) == 0 {
		for _, id := range card.MultiverseIDs {
			if id > 0 {
				consider(mtgmatcher.IDSpaceMultiverse, strconv.FormatInt(id, 10))
			}
		}
	}
	if found == "" || conflict {
		return nil
	}
	co, _ := mtgmatcher.GetUUID(found)
	in.ID = found
	in.Foil = co.Foil || co.Etched
	if co.Etched {
		in.Finish = mtgmatcher.FinishEtched
	} else if co.Foil {
		in.Finish = mtgmatcher.FinishFoil
	} else {
		in.Finish = mtgmatcher.FinishNonfoil
	}
	return &in
}

func identifierName(name string) string {
	name = strings.ReplaceAll(name, " / ", " // ")
	if corrected, ok := cardTable[name]; ok {
		name = corrected
	}
	return name
}

// idDescribesPrinting checks the distinctions the ID must retain. The full
// catalog audit found copied IDs on The List, lettered artwork, marked-number
// variants and special finishes. Those descriptions keep their existing text
// rules unless the candidate explicitly establishes the distinction.
func idDescribesPrinting(card *ABUCard, in *mtgmatcher.InputCard, co *mtgmatcher.CardObject) bool {
	variation := in.Variation
	if number := describedArtwork(in.Name, card.Edition, variation); number != "" {
		return co.Number == number
	}
	// A marked collector number names a printing the vendor's base ID
	// cannot replace (for example Strict Proctor 33★).
	if wearsMark(card.Number) && card.Number != co.Number && sharesNumber(co.Number, card.Number) {
		return false
	}
	if variation == "" {
		// These LCC nonfoil listings picture #69-100 but copy the IDs of
		// #37-68. Their titles omit the distinction; retain the existing
		// normalization for this audited family (Bronzebeak Foragers et al.).
		if co.SetCode == "LCC" && !in.Foil {
			return false
		}
	}
	// ABU's Fifth Edition letters differ from the matcher's shared letter
	// table. Its independent collector numbers and IDs agree with its images
	// (Forest B Night is 447, not the table's autumn artwork at 448).
	if co.SetCode == "5ED" && artworkLetter(in.Name, variation) != "" && card.Number == co.Number {
		return true
	}
	// preprocess appends the shelf to prerelease and promo-pack descriptions.
	// Recognized shelves were checked above; do not parse a year
	// or set number in it as a collector number.
	if strings.HasPrefix(variation, "Prerelease ") || strings.HasPrefix(variation, "Promo Pack ") {
		variation = strings.TrimSuffix(variation, " "+card.Edition)
	}
	if normalized := mtgmatcher.Normalize(variation); (normalized == "prerelease" || normalized == "promopack") && card.Number != "" && card.Number != co.Number {
		return false
	}
	probe := *in
	probe.Variation = variation
	return mtgmatcher.ValidatePrinting(probe, co.UUID)
}
