package mtgmatcher

// DefaultRules answers the GameRules hooks a game has no use for, so a
// loader writes only the ones that do something. Embed it in a game's Rules
// and override whatever that game actually needs:
//
//	type Rules struct{ mtgmatcher.DefaultRules }
//
// The answers are the ones that change nothing: every candidate edition
// survives, nothing is unsupported, and no promo tag is missing. A game that
// means any of those has to say so itself, which is the point - a hook
// nobody wrote is a hook nobody has to read.
//
// Magic implements its policies explicitly and embeds nothing.
type DefaultRules struct{}

// FilterPrintings keeps every candidate edition.
func (DefaultRules) FilterPrintings(b *Backend, inCard *InputCard, editions []string) []string {
	return editions
}

// IsUnsupported reports that no input belongs to an unsupported set, which
// is the answer for a game whose catalog holds nothing it cannot sell. A
// game that files something else under its card products - Lorcana's
// puzzle-piece inserts and multi-card lots - says so itself.
func (DefaultRules) IsUnsupported(b *Backend, inCard *InputCard) bool {
	return false
}

// IsToken reports that the game knows no token names beyond the ones its
// datastore already lists.
func (DefaultRules) IsToken(b *Backend, name string) bool {
	return false
}

// IsSpecificUnsupported reports that no single card is unsupported on its
// own account.
func (DefaultRules) IsSpecificUnsupported(b *Backend, inCard *InputCard) bool {
	return false
}

// MissingPromoTag reports that no input claims a promo treatment its card
// does not carry, which is the answer for a game whose promos are not
// tagged.
func (DefaultRules) MissingPromoTag(b *Backend, inCard *InputCard, co *CardObject) bool {
	return false
}

// CandidateSets prefers an exact edition name, then a partial name. When
// neither identifies a set, or PromoWildcard requests a wider search, all
// printings reach the game's card filter.
func (DefaultRules) CandidateSets(b *Backend, inCard *InputCard, editions []string) []string {
	if len(editions) <= 1 || inCard.PromoWildcard {
		return editions
	}
	var exact, loose []string
	for _, code := range editions {
		set := b.Sets[code]
		if Equals(set.Name, inCard.Edition) {
			exact = append(exact, code)
		} else if Contains(set.Name, inCard.Edition) {
			loose = append(loose, code)
		}
	}
	if len(exact) > 0 {
		return exact
	}
	if len(loose) > 0 {
		return loose
	}
	return editions
}

// FinalizeCandidates leaves ambiguity for the shared pipeline to report.
func (DefaultRules) FinalizeCandidates(b *Backend, inCard *InputCard, cards []Card) []Card {
	return cards
}
