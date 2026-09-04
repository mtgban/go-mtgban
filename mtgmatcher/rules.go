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
	// AliasEdition spells an edition string the way the datastore names its
	// set, using only the string itself - the card-free part of
	// AdjustEdition. GetSetByName's last resort is this hook, never the
	// full fixup, so an implementation must not route back through it.
	AliasEdition(b *Backend, edition string) string
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
	// IsToken reports whether a name is one this game knows as a token
	// without its carrying a token type of its own - the rules tips, the
	// checklists, the storefront spellings that only ever name a token. The
	// datastore's own token list is checked before this is asked, so a game
	// with nothing to add answers false.
	IsToken(b *Backend, name string) bool
	// CanonicalFinish spells a finish name the way this game names it: its
	// own name for every finish it has, the vendor aliases that reach them,
	// and CanonicalFinish (the package function) for the finishes every game
	// shares. The game owns this vocabulary - it is the one place a source's
	// spelling turns into the name the loaders key Card.FoilUUIDs and stamp
	// CardObject.Finish with, and the one place a new vendor spelling is
	// added. A name the game cannot place yields "", so a caller pricing it
	// is told rather than handed another finish's uuid; a game whose finish
	// names are data rather than a fixed list may instead hand back the
	// normalized name and let the lookup fail.
	CanonicalFinish(name string) string
}

// SetRules attaches the game-specific identification hooks used by Match. A
// game's datastore loader calls this when it builds a Backend.
func (b *Backend) SetRules(r GameRules) {
	b.rules = r

	// The finishes this datastore actually sells, which is what tells a
	// vendor spelling nobody has taught the game yet from a name that names
	// a real finish this one printing is not sold in. A game whose rules
	// place any name (Lorcana's foil types are data and a new one arrives
	// every set) has no other way to say "I have never heard of this".
	b.knownFinishes = map[string]bool{}
	for _, co := range b.UUIDs {
		for key := range co.FoilUUIDs {
			b.knownFinishes[key] = true
		}
		for name, key := range co.FinishAliases {
			b.knownFinishes[name] = true
			b.knownFinishes[key] = true
		}
	}
}
