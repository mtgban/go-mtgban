package mtgmatcher

// The id spaces the loaders file external identifiers under. A space names
// who assigned the id, not an mtgjson tag: TCGplayer's product and etched
// product ids are one space, because they are disjoint integers from one
// vendor and a caller holds "the TCGplayer id" without knowing which flavor.
const (
	IDSpaceMTGJSON   = "mtgjson"
	IDSpaceScryfall  = "scryfall"
	IDSpaceTCGplayer = "tcgplayer"

	// The multiverse ids are integers like TCGplayer's, so their space
	// stays out of idSpaceOrder: only ConvertID reaches it, by name.
	IDSpaceMultiverse = "multiverse"
)

// idSpaceOrder is the chain the space-blind lookups walk, in order. A space
// left out is reachable only through ConvertID with its name: an id space
// whose integers collide with another's must never answer a bare integer.
var idSpaceOrder = []string{
	IDSpaceMTGJSON,
	IDSpaceScryfall,
	IDSpaceTCGplayer,
}

// ConvertID answers the uuid an external identifier maps to, in the one id
// space the caller names, and an empty string when the space carries no such
// id. The uuid is the printing's base sibling - the nonfoil, or the foil
// where no nonfoil was sold - so a caller that also knows the finish walks
// there with MatchID, which reports the miss the empty string carries.
func (b *Backend) ConvertID(space, inputID string) string {
	return b.ExternalIdentifiers[space][inputID]
}

// ConvertID resolves an external identifier in one named id space against the
// default datastore. See the method.
func ConvertID(space, inputID string) string {
	return defaultBackend.ConvertID(space, inputID)
}
