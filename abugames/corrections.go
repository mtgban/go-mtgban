package abugames

// These listings copy another printing's shelf and number, misspell the
// title, or copy an id. The product image supplies the printed set/number/
// title; match it and the original wording so a revised listing is
// rechecked. An empty corrected field leaves that part of the listing as
// the vendor wrote it.
var listingCorrections = map[string]struct {
	title, edition, number                            string
	correctedTitle, correctedEdition, correctedNumber string
	ignoreIDs                                         bool
}{
	"8124578": {title: "Path of Ancestry (The List)", edition: "Outlaws of Thunder Junction Commander", number: "310", correctedEdition: "Modern Horizons 3 Commander", correctedNumber: "363"},
	"8098901": {title: "Gempalm Incinerator (The List)", edition: "Elves vs. Goblins", number: "37", correctedEdition: "Duel Decks: Merfolk vs. Goblins", correctedNumber: "39"},
	"8107290": {title: "Gempalm Incinerator (The List)", edition: "Duel Decks Anthology: Elves vs. Goblins", number: "37", correctedEdition: "Duel Decks: Merfolk vs. Goblins", correctedNumber: "39"},
	"8149750": {title: "Mana Geyser (The List)", edition: "Secrets of Strixhaven Commander", number: "247", correctedEdition: "Commander 2021", correctedNumber: "176"},
	// The pictured WPN Premium artwork identifies PW24 #1, not #22.
	"8111219": {title: "Serra Angel (WPN Borderless) - FOIL", edition: "Wizards Play Network 2022", number: "1", correctedEdition: "Wizards Play Network 2024", correctedNumber: "1"},
	// The copied ids name the untagged schematic twin; the image doesn't.
	"7842864": {title: "Pristine Talisman", edition: "The Brothers' War Retro Artifacts", number: "106", correctedNumber: "43", ignoreIDs: true},
	"7843549": {title: "Pristine Talisman - FOIL", edition: "The Brothers' War Retro Artifacts", number: "106", correctedNumber: "43", ignoreIDs: true},
	// The copied ids and card_number both name the Extended Art twin.
	"7918400": {title: "Cirdan the Shipwright", edition: "The Lord of the Rings: Tales of Middle-earth Commander", number: "133", correctedNumber: "50", ignoreIDs: true},
	// card_number (208) is the set's Tutorial Card, not this one's own 218.
	"8114685": {title: "Sledding Otter-Penguin - FOIL", edition: "Avatar: The Last Airbender Eternal", number: "208", correctedNumber: "218"},
	// The title's parenthetical is a typo for the halofoil card_number.
	"7895279": {title: "Harnessed Snubhorn (HALO 168) - FOIL", edition: "March of the Machine: The Aftermath", number: "188", correctedTitle: "Harnessed Snubhorn (HALO 188) - FOIL"},
	// Filed as "(Prerelease)"; the catalog only carries the release promo.
	"162808": {title: "Reya Dawnbringer (Prerelease) - FOIL", edition: "Tenth Edition Promos", number: "35", correctedTitle: "Reya Dawnbringer (Release) - FOIL"},
	"206639": {title: "Reya Dawnbringer (Prerelease) - FOIL", edition: "Tenth Edition Promos", number: "35", correctedTitle: "Reya Dawnbringer (Release) - FOIL"},
	"206640": {title: "Reya Dawnbringer (Prerelease) - FOIL", edition: "Tenth Edition Promos", number: "35", correctedTitle: "Reya Dawnbringer (Release) - FOIL"},
}

func correctListing(card ABUCard) ABUCard {
	fix, ok := listingCorrections[card.ProductID]
	if !ok || card.DisplayTitle != fix.title || card.Edition != fix.edition || card.Number != fix.number {
		return card
	}
	if fix.correctedTitle != "" {
		card.DisplayTitle = fix.correctedTitle
	}
	if fix.correctedEdition != "" {
		card.Edition = fix.correctedEdition
	}
	if fix.correctedNumber != "" {
		card.Number = fix.correctedNumber
	}
	if fix.ignoreIDs {
		card.ScryfallIDs, card.TCGplayerIDs, card.MultiverseIDs = nil, nil, nil
	}
	return card
}
