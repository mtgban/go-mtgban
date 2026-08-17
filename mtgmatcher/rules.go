package mtgmatcher

// GameRules abstracts the game-specific steps of the Match pipeline so that a
// datastore loaded for a non-Magic game can supply its own card-identification
// logic. Magic implements these hooks in the mtgmatcher/magic sub-package and
// Lorcana in mtgmatcher/lorcana; a datastore loader attaches its implementation
// via SetRules when it builds a Backend, and Match dispatches the major
// pipeline stages through the stored rules. Name preprocessing belongs to the
// game: Prefilter is what splits bracketed editions and parenthesized or
// dashed variants off the name. What stays in Match itself is the language
// handling (normalizing the requested language, then dropping candidates that
// do not carry it) and the token/oversize gate on an unresolved name, so a
// hook implementation must expect the InputCard to have been mutated by those
// steps before it runs.
//
// Hooks receive the InputCard by pointer and may mutate it; mutations persist
// for the rest of the pipeline and are also visible to the caller after Match
// returns. AdjustEdition in particular may set flags on the InputCard (the
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
	// MissingPromoTag reports whether the input claims a promo treatment the
	// resolved card does not carry; a claimed-but-absent tag means the card
	// is unsupported rather than mismatched. Games without tagged promos
	// return false.
	MissingPromoTag(b *Backend, inCard *InputCard, co *CardObject) bool
}

// finishResolver is an optional extension of GameRules, for a game whose
// printings are sold in named foil sub-types beyond the three finishes the
// caller's flags have a bit for. Match type-asserts it off the attached rules,
// so a game that registers no sub-type is untouched by it - and by
// construction never reaches the question either, since only a sub-type key in
// FoilUUIDs raises it.
type finishResolver interface {
	// ResolveFinish returns the uuid of the sub-type the input's wording names
	// for the given printing, or an empty string when it names none. Match
	// asks before the pipeline's name preprocessing has run, so an
	// implementation has to expect the wording wherever the caller put it.
	ResolveFinish(b *Backend, inCard *InputCard, co *CardObject) string
}

// SetRules attaches the game-specific identification hooks used by Match. A
// game's datastore loader calls this when it builds a Backend.
func (b *Backend) SetRules(r GameRules) {
	b.rules = r
}
