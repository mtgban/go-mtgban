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

// onePieceDons spells, by product id, the DON!! cards whose product name
// reads apart from the datastore's label.
var onePieceDons = map[int]struct{ edition, label string }{
	873734: {"", "Nico Robin"},
	873735: {"", "Nico Robin Gold"},
	873738: {"", "Boa Hancock"},
	873739: {"", "Boa Hancock Gold"},
	877822: {"One Piece Promotion Cards", "Japanese Version 3rd Anniversary Set"},
	882449: {"One Piece Promotion Cards", "Japanese Version 3rd Anniversary Set"},
	906864: {"One Piece Promotion Cards", "English Version 3rd Anniversary Set Ace, Luffy, Sabo"},
}
