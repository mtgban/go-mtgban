package abugames

import (
	"errors"
	"slices"
	"strconv"

	"github.com/mtgban/go-mtgban/matcher"
)

// matchCard uses catalog IDs to distinguish candidates the listing's wording
// leaves ambiguous. ABU often copies both IDs from another printing: Plains
// (38) names DDI #40 by ID, and The List listings carry the original's IDs.
// A successful text match therefore wins; IDs cannot override its distinctions
// or resurrect an unsupported listing.
func matchCard(b *matcher.Backend, card *ABUCard, in *matcher.InputCard) (string, error) {
	id, err := b.Match(in)
	var alias *matcher.AliasingError
	if !errors.As(err, &alias) {
		return id, err
	}
	candidates := alias.Probe()
	finish := matcher.FinishNonfoil
	if in.IsEtched() {
		finish = matcher.FinishEtched
	} else if in.Foil {
		finish = matcher.FinishFoil
	}
	var found string
	conflict := false
	consider := func(space matcher.IDSpace, external string) {
		base := b.ConvertID(space, external)
		if base == "" {
			return
		}
		candidate, lookupErr := b.MatchIDFinish(base, finish)
		if lookupErr != nil || !slices.Contains(candidates, candidate) {
			return
		}
		// The ambiguity path has not performed the final ID-path promo checks.
		probe := *in
		probe.ID = candidate
		probe.Finish = finish
		validated, lookupErr := b.ValidateID(probe, matcher.IDValidationOptions{})
		if lookupErr != nil || validated != candidate {
			return
		}
		if found != "" && found != candidate {
			conflict = true
		}
		found = candidate
	}
	for _, external := range card.ScryfallIDs {
		consider(matcher.IDSpaceScryfall, external)
	}
	for _, external := range card.TCGplayerIDs {
		if external > 0 {
			consider(matcher.IDSpaceTCGplayer, strconv.FormatInt(external, 10))
		}
	}
	if found != "" && !conflict {
		return found, nil
	}
	return id, err
}
