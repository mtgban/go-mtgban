package magic

import (
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// CandidateSets selects Magic's editions and their promo siblings. Preserve
// the order of the exact, loose and all-editions passes: later filtering
// depends on which candidates the earlier pass admitted.
func (Rules) CandidateSets(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, printings []string) []string {
	var codes []string
	// Only one printing, it *has* to be it
	if len(printings) == 1 {
		codes = append(codes, printings[0])
	} else if !inCard.PromoWildcard && !isSecretLair(inCard) {
		// If multiple printing, try filtering to the closest name
		// described by the inCard.Edition.
		// This is skipped if we're in the wildcard Promo mode, as we
		// need as many editions as possible.
		b.Log("Several printings found, iterating over edition name")

		// First loop, search for a perfect match
		for _, setCode := range printings {
			// Perfect match, the card *has* to be present in the set
			if mtgmatcher.Equals(b.Sets[setCode].Name, inCard.Edition) {
				b.Log("Found a perfect match with", inCard.Edition, setCode)
				codes = append(codes, setCode)

				set := b.Sets[setCode]

				// In case it's a well known promo, consider the promo sets (or vice
				// versa for promo sets) in order to let filtering take care of them
				// JPN cards are skipped because they are well set usually
				if !IsJPN(inCard) && (IsPrerelease(inCard) || IsPromoPack(inCard) ||
					(isBundle(inCard) && set.ReleaseDateTime.After(PromosForEverybodyYay)) ||
					(isBaB(inCard) && set.ReleaseDateTime.After(BuyABoxInExpansionSetsDate))) {
					setName := b.Sets[setCode].Name
					if !strings.HasSuffix(setName, "Promos") {
						setCode = "P" + setCode
						set, found := b.Sets[setCode]
						if found {
							b.Log("Detected possible promo, adding edition", set.Name, setCode)
							codes = append(codes, setCode)
						}
					} else {
						setCode = strings.TrimPrefix(setCode, "P")
						set, found := b.Sets[setCode]
						if found {
							b.Log("Detected possible non-promo, adding edition", set.Name, setCode)
							codes = append(codes, setCode)
						}
					}
				}
			}
		}

		// Second loop, hope that a portion of the edition is in the set Name
		// This may result in false positives under certain circumstances.
		if len(codes) == 0 {
			b.Log("No perfect match found, trying with heuristics")
			for _, setCode := range printings {
				set := b.Sets[setCode]

				// Skip heuristics for WCD as short version would catch a lot
				if isWorldChamp(inCard) {
					break
				}

				if mtgmatcher.Contains(set.Name, inCard.Edition) ||
					// If a card is promotional, only consider promotional sets
					(isGenericPromo(b, inCard) && strings.HasSuffix(set.Name, "Promos")) ||
					// If it is Bundle or BaB, also consider base sets if recent enough
					(isBundle(inCard) && !strings.HasSuffix(set.Name, "Promos") && set.ReleaseDateTime.After(PromosForEverybodyYay)) ||
					(isBaB(inCard) && !strings.HasSuffix(set.Name, "Promos") && set.ReleaseDateTime.After(BuyABoxInExpansionSetsDate)) {
					b.Log("Found a possible match with", inCard.Edition, setCode)
					codes = append(codes, setCode)
				}
			}
		}
	}

	// Third loop, YOLO
	// Let's consider every edition and hope the second pass will filter
	// duplicates out. This may result in false positives of course.
	if len(codes) == 0 {
		b.Log("No loose match found, trying all")
		codes = append(codes, printings...)
	}

	return codes
}

// FinalizeCandidates preserves the historical choice of the first World
// Championship printing, and prefers a printing over a copy of it minted in
// the same language, before the pipeline checks its language.
func (Rules) FinalizeCandidates(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cards []mtgmatcher.Card) []mtgmatcher.Card {
	if len(cards) > 1 && isWorldChamp(inCard) {
		return cards[:1]
	}
	if len(cards) > 1 {
		cards = originalOverCopy(cards)
	}
	return cards
}

// originalOverCopy drops a copy minted from another candidate in the same
// language, which only a listing naming the copy's set tells apart from the
// original; the other language copies are left to the language check.
func originalOverCopy(cards []mtgmatcher.Card) []mtgmatcher.Card {
	languages := map[string]string{}
	for _, card := range cards {
		languages[card.UUID] = card.Language
	}
	var out []mtgmatcher.Card
	for _, card := range cards {
		base, tag, found := strings.Cut(card.UUID, "_")
		if found && langs[strings.ToUpper(tag)] != "" {
			language, isCopy := languages[base]
			if isCopy && language == card.Language {
				continue
			}
		}
		out = append(out, card)
	}
	return out
}
