package cardmarket

// onePieceShelves spells the shelves Cardmarket names apart from the set of
// ours they sell.
var onePieceShelves = map[string]string{
	"Reprints":   "Revision Pack Cards",
	"Demo Decks": "One Piece Demo Deck Cards",
}

// onePieceShelfLabels names the event the products of a promo shelf were
// handed out at, which the datastore labels the printing with.
var onePieceShelfLabels = map[string]string{
	"Store Tournament Promos": "Tournament Pack",
	"Winner Cards":            "Winner Pack",
}
