package abugames

import (
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/matcher"
	"github.com/mtgban/go-mtgban/matcher/magic"
)

// identifiedCard reads IDs before text matching. An ID must name the card in
// the listing and support its language, finish and printing descriptors. Both
// namespaces may carry copied IDs, so an unresolved disagreement falls back to
// text rather than giving either source unconditional priority.
func identifiedCard(b *matcher.Backend, card *ABUCard, in matcher.InputCard) *matcher.InputCard {
	var found string
	conflict := false
	consider := func(space matcher.IDSpace, external string) {
		base := b.ConvertID(space, external)
		if base == "" {
			return
		}
		co, err := b.GetUUID(base)
		if err != nil {
			return
		}
		// Flavor-name misprints can have their own catalog entry (Luca/Luka
		// Stadium). A title spelling alone must not override its printed number.
		if (matcher.Equals(in.Name, co.FlavorName) || matcher.Equals(in.Name, co.FaceFlavorName)) && card.Number != "" && card.Number != co.Number {
			return
		}
		// Multiverse identifies a catalog printing, but older ABU shelves
		// include reprints and special editions with copied original IDs.
		if space == matcher.IDSpaceMultiverse {
			set, setErr := b.GetSetByName(card.Edition)
			if setErr != nil || set.Code != co.SetCode || (card.Number != "" && card.Number != co.Number) {
				return
			}
		}
		lang := in.Language
		if lang == "" {
			lang = "English"
		}
		if magic.IsJPN(&in) && lang != "Japanese" {
			return
		}
		finish := listingFinish(co, in.Foil, in.IsEtched())
		probe := in
		probe.Name = identifierName(in.Name)
		probe.ID, probe.Finish, probe.Language = base, finish, lang
		id, err := b.ValidateID(probe, matcher.IDValidationOptions{AllowFinishSiblings: true})
		if err != nil {
			return
		}
		co, err = b.GetUUID(id)
		if err != nil {
			return
		}
		if set, err := b.GetSetByName(card.Edition); err == nil && set.Code != co.SetCode &&
			!(in.Contains("The List") && co.SetCode == "PLST") {
			return
		}
		if !idDescribesPrinting(b, card, &in, co) {
			return
		}
		if found != "" && found != id {
			conflict = true
		}
		found = id
	}
	for _, id := range card.ScryfallIDs {
		consider(matcher.IDSpaceScryfall, id)
	}
	for _, id := range card.TCGplayerIDs {
		if id > 0 {
			consider(matcher.IDSpaceTCGplayer, strconv.FormatInt(id, 10))
		}
	}
	// Older listings sometimes supply only a Multiverse ID. Keep it in its
	// own numeric namespace and apply the same listing checks. Do not let it
	// override a disagreement between the two product-level sources.
	if len(card.ScryfallIDs) == 0 && len(card.TCGplayerIDs) == 0 {
		for _, id := range card.MultiverseIDs {
			if id > 0 {
				consider(matcher.IDSpaceMultiverse, strconv.FormatInt(id, 10))
			}
		}
	}
	if found == "" || conflict {
		return nil
	}
	co, _ := b.GetUUID(found)
	in.ID = found
	in.Foil = co.Foil || co.Etched
	if co.Etched {
		in.Finish = matcher.FinishEtched
	} else if co.Foil {
		in.Finish = matcher.FinishFoil
	} else {
		in.Finish = matcher.FinishNonfoil
	}
	return &in
}

// foreignIdentified takes the vendor ids of a listing in a language other than
// English when every one of them names the same printing, in that language,
// under the listing's name and finish. The shelf wording of such listings
// rarely describes the printing, so the ids are the only witness.
func foreignIdentified(b *matcher.Backend, card *ABUCard, name, lang string, foil bool) *matcher.InputCard {
	var found string
	var finish string
	consider := func(space matcher.IDSpace, external string) bool {
		base := b.ConvertID(space, external)
		if base == "" {
			return false
		}
		co, err := b.GetUUID(base)
		if err != nil {
			return false
		}
		finish = listingFinish(co, foil, false)
		id, err := b.ValidateID(matcher.InputCard{
			Name: identifierName(name), ID: base, Finish: finish, Foil: foil, Language: lang,
		}, matcher.IDValidationOptions{})
		if err != nil || (found != "" && found != id) {
			return false
		}
		found = id
		return true
	}
	for _, id := range card.ScryfallIDs {
		if !consider(matcher.IDSpaceScryfall, id) {
			return nil
		}
	}
	for _, id := range card.TCGplayerIDs {
		if id > 0 && !consider(matcher.IDSpaceTCGplayer, strconv.FormatInt(id, 10)) {
			return nil
		}
	}
	if found == "" {
		return nil
	}
	return &matcher.InputCard{Name: name, ID: found, Foil: foil, Finish: finish, Language: lang}
}

// listingFinish is the finish a listing asks of co. ABU calls both foil and
// etched-only printings FOIL.
func listingFinish(co *matcher.CardObject, foil, etched bool) string {
	if etched {
		return matcher.FinishEtched
	}
	if !foil {
		return matcher.FinishNonfoil
	}
	if !co.HasFinish(matcher.FinishFoil) && co.HasFinish(matcher.FinishEtched) {
		return matcher.FinishEtched
	}
	return matcher.FinishFoil
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
func idDescribesPrinting(b *matcher.Backend, card *ABUCard, in *matcher.InputCard, co *matcher.CardObject) bool {
	variation := in.Variation
	if number := describedArtwork(b, in.Name, card.Edition, variation); number != "" {
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
	if normalized := matcher.Normalize(variation); (normalized == "prerelease" || normalized == "promopack") && card.Number != "" && card.Number != co.Number {
		return false
	}
	probe := *in
	probe.Variation = variation
	return b.ValidatePrinting(probe, co.UUID)
}
