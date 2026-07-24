package mtgmatcher

// GameRules abstracts the game-specific steps of the Match pipeline so that a
// datastore loaded for a non-Magic game can supply its own card-identification
// logic. Magic implements these hooks in the mtgmatcher/magic sub-package and
// Lorcana in mtgmatcher/lorcana; a datastore loader attaches its implementation
// via SetRules when it builds a Backend, and Match dispatches the major
// pipeline stages through the stored rules. Some cross-cutting steps remain
// hardcoded in Match itself — name preprocessing (bracketed editions,
// parenthesized and dashed variants), token/oversize gates, promo-type
// validation, and the language filter — so a hook implementation must expect
// the InputCard to have been mutated by those steps before it runs.
//
// Hooks receive the InputCard by pointer and may mutate it; mutations persist
// for the rest of the pipeline and are also visible to the caller after Match
// returns. FilterPrintings in particular may set flags on the InputCard (the
// Magic rules set PromoWildcard and BeyondBaseSet) that Match and later hooks
// read.
type GameRules interface {
	// Prefilter mutates the input card before the canonical-name lookup.
	Prefilter(b *Backend, inCard *InputCard)
	// AdjustName fixes up the input card name to match a canonical name.
	AdjustName(b *Backend, inCard *InputCard)
	// AdjustEdition fixes up the input card edition to a known set.
	AdjustEdition(b *Backend, inCard *InputCard)
	// FilterPrintings narrows the candidate editions for the input card.
	FilterPrintings(b *Backend, inCard *InputCard, editions []string) []string
	// FilterCards narrows the candidate cards for the input card. The cardSet
	// map iterates in random order; implementations are responsible for
	// producing deterministic output ordering when more than one candidate
	// survives, since the result feeds user-visible aliasing diagnostics.
	FilterCards(b *Backend, inCard *InputCard, cardSet map[string][]Card) []Card
	// IsUnsupported reports whether the input card belongs to an unsupported
	// set, checked before name resolution.
	IsUnsupported(b *Backend, inCard *InputCard) bool
	// IsSpecificUnsupported reports whether the input card is a specific
	// unsupported card, checked after edition resolution.
	IsSpecificUnsupported(b *Backend, inCard *InputCard) bool
}

// SetRules attaches the game-specific identification hooks used by Match. A
// game's datastore loader calls this when it builds a Backend.
func (b *Backend) SetRules(r GameRules) {
	b.rules = r
}
