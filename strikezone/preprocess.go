package strikezone

import (
	"fmt"
	"strings"

	"github.com/kodabb/go-mtgban/mtgdb"
)

func preprocess(cardName, cardSet, notes string) (*mtgdb.Card, error) {
	var variation string

	// skip tokens, too many variations
	if strings.Contains(cardName, "Token") {
		return nil, fmt.Errorf("Skipping %s", cardName)
	}

	isFoil := strings.Contains(notes, "Foil")

	cn, found := cardTable[cardName]
	if found {
		cardName = cn
	}

	// Sometimes the buylist specifies tags at the end of the card name,
	// but without parenthesis, so make sure they are present.
	for _, tag := range tagsTable {
		if strings.HasSuffix(cardName, tag) {
			cardName = strings.Replace(cardName, tag, "("+tag+")", 1)
			break
		}
	}

	cardName = strings.Replace(cardName, "[", "(", 1)
	cardName = strings.Replace(cardName, "]", ")", 1)

	switch {
	// Tyop
	case strings.HasPrefix(cardName, "Snow-Cover "):
		cardName = strings.Replace(cardName, "Snow-Cover ", "Snow-Covered ", 1)
	// Found at beginning, move it to variation
	case strings.HasPrefix(cardName, "Borderless"):
		cardName = strings.TrimPrefix(cardName, "Borderless ")
		variation = "Borderless"
	// Found at beginning, move it to variation
	case strings.HasPrefix(cardName, "Showcase"):
		cardName = strings.TrimPrefix(cardName, "Showcase ")
		variation = "Showcase"
	// Found at beginning, move it to variation
	case strings.HasPrefix(cardName, "Extended Art"):
		cardName = strings.TrimPrefix(cardName, "Extended Art ")
		variation = "Extended Art"
	// APAC and EURO lands, drop specifier
	case strings.Contains(cardName, "APAC") || strings.Contains(cardName, "EURO"):
		cardSet = "European Land Program"
		if strings.Contains(cardName, "APAC") {
			cardSet = "Asia Pacific Land Program"
		}
		variants := mtgdb.SplitVariants(cardName)
		cardName = variants[0]
		fields := strings.Fields(cardName)
		cardName = fields[0]
		variation = variants[1]
	// Ravnica weekend lands, move to variation
	case strings.Contains(cardName, "Ravnica Weekend"):
		fields := strings.Fields(cardName)
		cardName = fields[0]
		variation = strings.Join(fields[1:], " ")
	case strings.HasPrefix(cardName, "B.F.M.") && strings.Contains(cardName, "#"):
		for _, field := range strings.Fields(cardName) {
			if strings.HasPrefix(field, "#") {
				cardName = "B.F.M."
				variation = field[1:]
				break
			}
		}
	}

	ed, found := card2setTable[cardName]
	if found {
		cardSet = ed
	}

	variants := mtgdb.SplitVariants(cardName)
	cardName = variants[0]
	if len(variants) > 1 {
		variation = variants[1]
	}

	switch variation {
	case "6E", "VI DCI", "DCI", "US":
		switch cardName {
		case "Crusade",
			"Lord of Atlantis",
			"Serra Avatar",
			"Thran Quarry",
			"Two-Headed Dragon":
			cardSet = "Junior Super Series"
		case "Forest",
			"Island",
			"Mountain",
			"Swamp",
			"Plains":
			cardSet = "Arena League 1999"
		case "Argothian Enchantress",
			"Balance",
			"Ball Lightning",
			"Gaea's Cradle",
			"Intuition",
			"Living Death",
			"Memory Lapse",
			"Oath of Druids",
			"Stroke of Genius",
			"Tradewind Rider":
			variation = "Judge"
		case "Vampiric Tutor":
			variation = "Judge 2000"
		case "Arc Lightning",
			"Chill",
			"Duress",
			"Enlightened Tutor",
			"Karn, Silver Golem",
			"Mana Leak",
			"Rewind",
			"Serum Visions",
			"Stupor":
			variation = "Arena"
		case "Powder Keg",
			"Voidmage Prodigy",
			"Wasteland":
			variation = "Magic Player Rewards"
		default:
			variation = "FNM"
		}
	case "DotP 2012 - Xbox", "X Box Promo 2013", "X Box Promo",
		"Playstation", "PS3 Promo",
		"Duels of the Planeswalkers - PC",
		"Duel of the Planeswalkers":
		variation = "Duels"
	}

	switch {
	case strings.HasPrefix(variation, "The "):
		cardSet = "Magic Premiere Shop 2005"
	case strings.Contains(variation, "Holiday"):
		cardSet = "Happy Holidays"
	}

	if cardSet == "Promotional Cards" && variation == "" {
		ed, found = promo2setTable[cardName]
		if found {
			cardSet = ed
		}
	}

	return &mtgdb.Card{
		Name:      cardName,
		Variation: variation,
		Edition:   cardSet,
		Foil:      isFoil,
	}, nil
}
