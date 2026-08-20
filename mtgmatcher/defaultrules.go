package mtgmatcher

// DefaultRules answers the GameRules hooks a game has no use for, so a
// loader writes only the ones that do something. Embed it in a game's Rules
// and override whatever that game actually needs:
//
//	type Rules struct{ mtgmatcher.DefaultRules }
//
// The answers are the ones that change nothing: every candidate edition
// survives, nothing is unsupported, and no promo tag is missing. A game that means any of those has to say so itself, which is
// the point - a hook nobody wrote is a hook nobody has to read.
//
// Magic implements all three for real and embeds nothing.
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
