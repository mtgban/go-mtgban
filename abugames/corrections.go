package abugames

// These listings copy another printing's shelf and number. The product image
// supplies the printed set and number; the title supplies the List designation.
// Match both the product and original wording so a revised listing is rechecked.
var listingCorrections = map[string]struct {
	title, edition, number            string
	correctedEdition, correctedNumber string
}{
	"8124578": {"Path of Ancestry (The List)", "Outlaws of Thunder Junction Commander", "310", "Modern Horizons 3 Commander", "363"},
	"8098901": {"Gempalm Incinerator (The List)", "Elves vs. Goblins", "37", "Duel Decks: Merfolk vs. Goblins", "39"},
	"8107290": {"Gempalm Incinerator (The List)", "Duel Decks Anthology: Elves vs. Goblins", "37", "Duel Decks: Merfolk vs. Goblins", "39"},
	"8149750": {"Mana Geyser (The List)", "Secrets of Strixhaven Commander", "247", "Commander 2021", "176"},
	// Both vendor IDs and the pictured WPN Premium artwork identify PW24 #1.
	"8111219": {"Serra Angel (WPN Borderless) - FOIL", "Wizards Play Network 2022", "1", "Wizards Play Network 2024", "1"},
}

func correctListing(card ABUCard) ABUCard {
	if fix, ok := listingCorrections[card.ProductID]; ok && card.DisplayTitle == fix.title && card.Edition == fix.edition && card.Number == fix.number {
		card.Edition, card.Number = fix.correctedEdition, fix.correctedNumber
	}
	return card
}
