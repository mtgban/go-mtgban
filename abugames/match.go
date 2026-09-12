package abugames

import (
	"errors"
	"slices"
	"strconv"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// matchCard uses catalog IDs to distinguish candidates the listing's wording
// leaves ambiguous. ABU often copies both IDs from another printing: Plains
// (38) names DDI #40 by ID, and The List listings carry the original's IDs.
// A successful text match therefore wins; IDs cannot override its distinctions
// or resurrect an unsupported listing.
func matchCard(card *ABUCard, in *mtgmatcher.InputCard) (string, error) {
	id, err := mtgmatcher.Match(in)
	var alias *mtgmatcher.AliasingError
	if !errors.As(err, &alias) {
		return id, err
	}
	candidates := alias.Probe()
	finish := mtgmatcher.FinishNonfoil
	if in.IsEtched() {
		finish = mtgmatcher.FinishEtched
	} else if in.Foil {
		finish = mtgmatcher.FinishFoil
	}
	var found string
	conflict := false
	consider := func(space, external string) {
		base := mtgmatcher.ConvertID(space, external)
		if base == "" {
			return
		}
		candidate, lookupErr := mtgmatcher.MatchIDFinish(base, finish)
		if lookupErr != nil || !slices.Contains(candidates, candidate) {
			return
		}
		// The ambiguity path has not performed the final ID-path promo checks.
		probe := *in
		probe.ID = candidate
		probe.Finish = finish
		validated, lookupErr := mtgmatcher.ValidateID(probe, mtgmatcher.IDValidationOptions{})
		if lookupErr != nil || validated != candidate {
			return
		}
		if found != "" && found != candidate {
			conflict = true
		}
		found = candidate
	}
	for _, external := range card.ScryfallIDs {
		consider(mtgmatcher.IDSpaceScryfall, external)
	}
	for _, external := range card.TCGplayerIDs {
		if external > 0 {
			consider(mtgmatcher.IDSpaceTCGplayer, strconv.FormatInt(external, 10))
		}
	}
	if found != "" && !conflict {
		return found, nil
	}
	return id, err
}
