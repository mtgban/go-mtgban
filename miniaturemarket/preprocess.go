package miniaturemarket

import (
	"errors"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

var sealedRenames = map[string]string{
	"Adventures in the Forgotten Realms - Commander Deck Set": "Adventures in the Forgotten Realms Commander Deck Display",

	"Kaldheim - Commander Deck Set (Set of 2)":           "Kaldheim Commander Decks - Set of 2",
	"Kaladesh - Planeswalker Deck Set (2 Theme Decks)":   "Kaladesh Planeswalker Decks - Set of 2",
	"Pioneer Challenger Deck Set (4)":                    "Pioneer Challenger Deck 2021 Set of 4",
	"Pioneer Challenger Deck Set 2022 (4)":               "Pioneer Challenger Deck 2022 Set of 4",
	"MtG Lost Caverns of Ixalan: Commander Deck Set (4)": "The Lost Caverns of Ixalan Commander Deck Case",

	"2022 Starter Kit": "2022 Arena Starter Kit",
}

func preprocessSealed(productName, edition string) (string, error) {
	switch {
	case strings.Contains(productName, "Jumpstart 2022"):
		edition = "Jumpstart 2022"
	case strings.Contains(edition, "2022 Starter Kit"):
		edition = "SNC"
	}

	// If edition is empty, do not return and instead loop through
	var setCode string
	set, err := mtgmatcher.GetSetByName(edition)
	if err == nil {
		setCode = set.Code
	}

	rename, found := sealedRenames[productName]
	if found {
		productName = rename
	}

	productName = strings.Replace(productName, "Deck Set (Set of 2)", "Decks - Set of 2", 1)
	productName = strings.Replace(productName, "Deck Set (2)", "Decks - Set of 2", 1)
	productName = strings.Replace(productName, "Deck Set (4)", "Decks - Set of 4", 1)
	productName = strings.Replace(productName, "Deck Set (5)", "Decks - Set of 5", 1)
	productName = strings.Replace(productName, "(Premium)", "Premium", 1)

	if strings.Contains(edition, "Lost Caverns") {
		edition = strings.Replace(edition, "MtG Lost Caverns", "The Lost Caverns", 1)
		productName = strings.Replace(productName, "MtG Lost Caverns", "The Lost Caverns", 1)
		if strings.Contains(productName, "Commander") {
			edition = strings.TrimPrefix(edition, "The ")
		}
	}

	edition = strings.Replace(edition, "Phyrexia - All Will Be One", "Phyrexia: All Will Be One", 1)
	productName = strings.Replace(productName, "Phyrexia - All Will Be One", "Phyrexia: All Will Be One", 1)

	if strings.Contains(edition, "Tales of Middle-earth") && !strings.Contains(edition, "The Lord of the Rings") {
		edition = "The Lord of the Rings " + edition
	}
	if strings.Contains(productName, "Tales of Middle-earth") && !strings.Contains(productName, "The Lord of the Rings") {
		productName = "The Lord of the Rings " + productName
	}

	productName = mtgmatcher.SplitVariants(productName)[0]

	switch {
	case strings.Contains(productName, "Land Station"),
		strings.Contains(productName, "Variety Pack"),
		strings.Contains(productName, "Scene Box"),
		strings.Contains(productName, "Transformers TCG"):
		return "", errors.New("unsupported")
	}

	var uuid string
	for _, set := range mtgmatcher.GetSets() {
		if setCode != "" && setCode != set.Code {
			continue
		}

		for _, sealedProduct := range set.SealedProduct {
			if mtgmatcher.SealedEquals(sealedProduct.Name, productName) {
				uuid = sealedProduct.UUID
				break
			}
		}

		if uuid == "" {
			for _, sealedProduct := range set.SealedProduct {
				// If not found, look if the a chunk of the name is present in the deck name
				switch {
				case strings.Contains(productName, "Archenemy"),
					strings.Contains(productName, "Duels of the Planeswalkers"),
					strings.Contains(productName, "Commander"),
					strings.Contains(productName, "Challenger Deck"),
					strings.Contains(productName, "Secret Lair"),
					strings.Contains(productName, "Planechase"):
					decks, found := sealedProduct.Contents["deck"]
					if found {
						for _, deck := range decks {
							// Work around internal names that are too long, like
							// "Teeth of the Predator - the Garruk Wildspeaker Deck"
							deckName := strings.Split(deck.Name, " - ")[0]
							if mtgmatcher.SealedContains(productName, deckName) {
								uuid = sealedProduct.UUID
								break
							}
							// Scret Lair may have
							deckName = strings.TrimSuffix(strings.ToLower(deckName), " foil")
							if mtgmatcher.SealedContains(productName, deckName) {
								uuid = sealedProduct.UUID
								break
							}
						}
					}
				}
				if uuid != "" {
					break
				}
			}
		}

		// Last chance (in case edition is known)
		if uuid == "" && setCode != "" && len(set.SealedProduct) == 1 {
			uuid = set.SealedProduct[0].UUID
		}

		if uuid != "" {
			break
		}

	}

	return uuid, nil
}
