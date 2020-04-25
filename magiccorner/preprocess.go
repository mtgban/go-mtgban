package magiccorner

import (
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/kodabb/go-mtgban/mtgdb"
)

var cardTable = map[string]string{
	"Ashiok's Forerummer":     "Ashiok's Forerunner",
	"Fire (Fire/Ice) ":        "Fire",
	"Fire (Fire/Ice)":         "Fire",
	"Fire/Ice":                "Fire",
	"Kingsbaile Skirmisher":   "Kinsbaile Skirmisher",
	"Rough (Rough/Tumble)":    "Rough",
	"Time Shrike":             "Tine Shrike",
	"Treasure Find":           "Treasured Find",
	"Wax/Wane":                "Wax",
	"Who,What,When,Where,Why": "Who",
	"Who/What/When/Where/Why": "Who",
}

func preprocess(card *MCCard, index int) (*mtgdb.Card, error) {
	cardName := card.Name
	edition := card.Set

	// Grab the image url and keep only the image name
	extra := strings.TrimSuffix(path.Base(card.Extra), path.Ext(card.Extra))

	// Skip any token or similar cards
	if strings.Contains(cardName, "Token") ||
		strings.Contains(cardName, "token") ||
		strings.Contains(cardName, "Art Series") ||
		strings.Contains(cardName, "Checklist") ||
		strings.Contains(cardName, "Check List") ||
		strings.Contains(cardName, "Check-List") ||
		strings.Contains(cardName, "Emblem") ||
		cardName == "Punch Card" ||
		cardName == "The Monarch" ||
		cardName == "Spirit" ||
		cardName == "City's Blessing" {
		return nil, fmt.Errorf("Skipping %s", cardName)
	}

	// Circle of Protection: Red in Revised EU FWB???
	if card.Variants[index].Id == 223958 ||
		// Excruciator RAV duplicate card
		card.Variants[index].Id == 108840 {
		return nil, fmt.Errorf("Skipping unsupported config %s/%s", cardName, edition)
	}

	isFoil := card.Variants[index].Foil == "Foil"

	cn, found := cardTable[cardName]
	if found {
		cardName = cn
	}

	variation := ""
	// Do not set variation for sets that have cards with parenthesis in their names
	if edition != "Unglued" && edition != "Unhinged" {
		variants := mtgdb.SplitVariants(cardName)
		if len(variants) > 1 {
			variation = variants[1]
		}
		cardName = variants[0]
		if variation == "" {
			variants = mtgdb.SplitVariants(card.OrigName)
			if len(variants) > 1 {
				variation = variants[1]
			}
		}
	}

	switch edition {
	case "War of the Spark: Japanese Alternate-Art Planeswalkers":
		if variation == "Version 2" {
			variation = "Japanese Prerelease"
			edition = "War of the Spark Promos"
		}
	// Use the tag information if available
	case "Promo":
		switch variation {
		case "War of the Spark: Extras", "Version1", "Version 2":
			variation = "Prerelease"
		case "Core 2020: Extras", "Version 1":
			variation = "Promo Pack"
		}
	// Use the number from extra if present, or keep the current version
	case "Unstable":
		if strings.HasPrefix(variation, "Version") && strings.HasPrefix(extra, "UST") {
			variation = strings.TrimLeft(extra[3:], "0")
		}
	case "Unsanctioned":
		if len(extra) > 3 {
			variation = strings.TrimLeft(extra[3:], "0")
		}
	// Handle wastes
	case "Giuramento dei Guardiani":
		if cardName == "Wastes" && len(extra) > 3 {
			variation = extra[3:]
		}
	// DDA correct edition is hidden in the extra
	case "Duel Decks Anthology":
		switch {
		case strings.HasPrefix(extra, "DDA0"):
			edition = "Duel Decks Anthology: Elves vs. Goblins"
		case strings.HasPrefix(extra, "DDA1"):
			edition = "Duel Decks Anthology: Jace vs. Chandra"
		case strings.HasPrefix(extra, "DDA2"):
			edition = "Duel Decks Anthology: Divine vs. Demonic"
		case strings.HasPrefix(extra, "DDA3"):
			edition = "Duel Decks Anthology: Garruk vs. Liliana"
		}
		variation = strings.TrimLeft(extra[4:], "0")
	}

	id := ""
	if variation == "" {
		switch edition {
		// Work around missing tags until (if) they add them
		case "Promo":
			if strings.HasPrefix(extra, "p2019ELD") || strings.HasPrefix(extra, "p2020THB") {
				internalNumber := strings.TrimLeft(extra[8:], "0")
				num, err := strconv.Atoi(internalNumber)
				if err == nil {
					if num < 69 {
						variation = "Prerelease"
					} else {
						variation = "Promo Pack"
					}
				}
			}
		// These editions contain numbers that can be used safely
		case "Throne of Eldraine: Extras", "Theros Beyond Death: Extras",
			"Gilde di Ravnica", "Fedeltà di Ravnica", "Unglued":
			variation = extra
			if len(extra) > 3 {
				internalNumber := strings.TrimLeft(extra[3:], "0")
				_, err := strconv.Atoi(internalNumber)
				if err == nil {
					variation = internalNumber
				}
			}
		// Image is scryfall ID
		case "Ikoria: Lair of Behemoths":
			id = extra
		// Only one of them
		case "Campioni di Kamigawa":
			if cardName == "Brothers Yamazaki" {
				variation = "Facing Left"
			}
		// These are the editions that need table lookup
		case "Antiquities", "Fallen Empires", "Chronicles",
			"Alleanze", "Rinascimento", "Origini":
			variation = extra
		// Same for this one, except the specifier is elsewhere
		case "Commander Anthology 2018":
			variation = card.URL
		// Full-art Zendikar lands
		case "Zendikar":
			if strings.HasPrefix(cardName, "Forest") ||
				strings.HasPrefix(cardName, "Island") ||
				strings.HasPrefix(cardName, "Mountain") ||
				strings.HasPrefix(cardName, "Plains") ||
				strings.HasPrefix(cardName, "Swamp") {
				s := strings.Fields(cardName)
				if len(s) > 1 {
					cardName = s[0]
					variation = s[1] + "a"
				}
			}
		default:
			switch cardName {
			// Try using the number, except the set code can randomly be
			// 2 or 3 characters.
			case "Plains", "Island", "Swamp", "Mountain", "Forest":
				for _, lengthToDrop := range []int{2, 3} {
					if len(extra) > lengthToDrop {
						internalNumber := strings.TrimLeft(extra[lengthToDrop:], "0")
						val, err := strconv.Atoi(internalNumber)
						if err == nil && val < 400 {
							variation = internalNumber
							break
						}
					}
				}
			}
		}
	}

	return &mtgdb.Card{
		Id:        id,
		Name:      cardName,
		Variation: variation,
		Edition:   edition,
		Foil:      isFoil,
	}, nil
}
