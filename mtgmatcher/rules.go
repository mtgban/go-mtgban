package mtgmatcher

// GameRules abstracts the game-specific steps of the Match pipeline so that a
// datastore loaded for a non-Magic game can supply its own card-identification
// logic. Each method wraps logic that, for Magic, currently lives in core
// mtgmatcher; this is a seam only, no data or logic has moved yet.
type GameRules interface {
	// Prefilter mutates the input card before the canonical-name lookup.
	Prefilter(b *Backend, inCard *InputCard)
	// AdjustName fixes up the input card name to match a canonical name.
	AdjustName(b *Backend, inCard *InputCard)
	// AdjustEdition fixes up the input card edition to a known set.
	AdjustEdition(b *Backend, inCard *InputCard)
	// FilterPrintings narrows the candidate editions for the input card.
	FilterPrintings(b *Backend, inCard *InputCard, editions []string) []string
	// FilterCards narrows the candidate cards for the input card.
	FilterCards(b *Backend, inCard *InputCard, cardSet map[string][]Card) []Card
}

// MagicRules is the GameRules implementation for Magic: the Gathering. The
// mtgjson loader attaches it to every Backend it builds via SetRules. Every
// method delegates to the existing Magic logic in core mtgmatcher, so the seam
// leaves behavior unchanged.
var MagicRules GameRules = magicRules{}

type magicRules struct{}

func (magicRules) Prefilter(b *Backend, inCard *InputCard) { b.prefilter(inCard) }

func (magicRules) AdjustName(b *Backend, inCard *InputCard) { b.adjustName(inCard) }

func (magicRules) AdjustEdition(b *Backend, inCard *InputCard) { b.adjustEdition(inCard) }

func (magicRules) FilterPrintings(b *Backend, inCard *InputCard, editions []string) []string {
	return b.filterPrintings(inCard, editions)
}

func (magicRules) FilterCards(b *Backend, inCard *InputCard, cardSet map[string][]Card) []Card {
	return b.filterCards(inCard, cardSet)
}

// SetRules attaches the game-specific identification hooks used by Match. A
// game's datastore loader calls this when it builds a Backend.
func (b *Backend) SetRules(r GameRules) {
	b.rules = r
}
