package channelfireball

import (
	"errors"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type cfbCard struct {
	URLId      string
	Key        string
	Name       string
	Edition    string
	Foil       bool
	Conditions string
	Price      float64
	Quantity   int
}

var tagsTable = []string{
	"Bundle Promo",
	"Buy-a-Box Promo",
	"DCI Judge Promo",
	"Dark Frame Promo",
	"Draft Weekend Promo",
	"Draft Weekend",
	"FNM 2017",
	"FNM 2019",
	"FNM Promo 2019",
	"Judge Academy Promo",
	"Magic League Promo",
	"Media Promo",
	"Open House Promo",
	"Planeswalker Deck Exclusive",
	"Planeswalker Weekend Promo",
	"SDCC 2019 Exclusive",
	"Store Championship Promo",
	"Treasure Map",
}

var cardTable = map[string]string{
	// Typos
	"Morbid Curiousity":                       "Morbid Curiosity",
	"Pir, Imaginitive Rascal (Release Promo)": "Pir, Imaginative Rascal (Release Promo)",
	"Essence Symbiont":                        "Essence Symbiote",
	"Quartzwood Crusher":                      "Quartzwood Crasher",
	"Souvernir Snatcher":                      "Souvenir Snatcher",

	// Tags that confuse mtgmatcher
	"Stocking Tiger (No Stamp Holiday 2013)": "Stocking Tiger (misprint)",
	"Beast in Show (A 2 Pink Bows)":          "Beast in Show (A Two Pink Bows)",
	"Beast in Show (D 7 Red Bows)":           "Beast in Show (D Seven Pink Bows)",
	"Lu Bu, Master-at-Arms (July 4, 1999)":   "Lu Bu, Master-at-Arms (July Prerelease)",
	"Plains (Portal 1)":                      "Plains (B)",

	// Name got cut during scraping
	"Path of Mettle // Metzali, To... (RIX Prerelease Foil)": "Path of Mettle (Prerelease)",

	// Funny cards
	"_________":                             "_____",
	"B.F.M. (Big Furry Monster Left side)":  "B.F.M. (28)",
	"B.F.M. (Big Furry Monster Right side)": "B.F.M. (29)",
	"Who/What/When/Where/Why":               "Who",
}

var card2setTable = map[string]string{
	"Cast Down (Japanese Promo)": "Magazine Inserts",

	"Demonic Tutor (Judge Foil)":          "Judge Gift Cards 2008",
	"Demonic Tutor (Judge Academy Promo)": "Judge Gift Cards 2020",
}

func preprocess(cardName, edition string) (string, string, error) {
	// Skip oversized card sets
	switch {
	case (strings.Contains(edition, "{") && strings.Contains(edition, "}")):
		return "", "", errors.New("skipping oversized card set")
	}

	// Quotes are not escaped
	if cardName == "" || strings.HasSuffix(cardName, ", ") {
		return "", "", errors.New("empty card name")
	}

	// Skip untracked cards
	switch cardName {
	case "Blaze (Alternate Art - Deck)",
		"Blaze (Alternate Art - Booster)",
		"Crystalline Sliver - Arena 2003":
		return "", "", errors.New("not tracked in mtgjson")
	}

	// Skip tokens and similar cards
	switch cardName {
	case "Experience Counter", "Poison Counter", "Experience Card",
		"Goblin", "Pegasus", "Sheep", "Soldier", "Squirrel", "Zombie",
		"Standard Placeholder", "Blank Card", "Splendid Genesis",
		"Black ": // Black "M" Filler Card
		return "", "", errors.New("not a real card")
	default:
		if strings.Contains(strings.ToLower(cardName), "token") ||
			strings.Contains(cardName, "Checklist") ||
			strings.Contains(cardName, "Filler") ||
			strings.Contains(cardName, "APAC Land Set") ||
			strings.Contains(cardName, "Emblem") {
			return "", "", errors.New("not a real card")
		}
	}

	// Skip non-english versions of this card
	if strings.HasPrefix(cardName, "Mana Crypt (Book Promo) (") {
		return "", "", errors.New("non-english card")
	}

	// Convert UTF-8 dash in ASHII dash
	cardName = strings.Replace(cardName, "–", "-", -1)
	// Strip stars indicating edition in preorder
	edition = strings.Replace(edition, "*", "", -1)

	// Drop pointeless tags
	cardName = strings.Replace(cardName, " - Foil", "", 1)
	cardName = strings.Replace(cardName, " (Masterpiece Foil)", "", 1)

	// Correctly put variants in the correct tag (within parenthesis)
	for _, tag := range tagsTable {
		cardName = strings.Replace(cardName, " "+tag, " ("+tag+")", 1)
	}

	// Make sure that variants are separated from the name
	parIndex := strings.Index(cardName, "(")
	if parIndex-1 > 0 && parIndex-1 < len(cardName) && cardName[parIndex-1] != ' ' {
		cardName = strings.Replace(cardName, "(", " (", 1)
	}

	// Keep original reference in case we need to reference it later
	orig := cardName

	// Split by () and by -, rebuild the cardname in a standardized way
	fields := mtgmatcher.SplitVariants(cardName)
	subfields := strings.Split(fields[0], " - ")
	cardName = subfields[0]
	for _, field := range fields[1:] {
		field = strings.Replace(field, " - ", " ", -1)
		cardName += " (" + field + ")"
	}
	for _, field := range subfields[1:] {
		cardName += " (" + field + ")"
	}
	cardName = strings.Replace(cardName, " - ", " ", 1)

	// Flatten the variants
	cardName = strings.Replace(cardName, ") (", " ", -1)

	// Fixup any expected errors
	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}
	ed, found := card2setTable[cardName]
	if found {
		edition = ed
	}

	if cardName == "Ogre Arsonist" && edition == "Promos: Book Inserts" {
		edition = "IDW Comics 2013"
	} else if strings.Contains(orig, "Ravnica Weekend") {
		edition = "GRN Ravnica Weekend"
		if strings.Contains(cardName, "B") {
			edition = "RNA Ravnica Weekend"
		}
	} else if strings.Contains(cardName, "APAC") {
		// Cut the "Set 1", "Set 2" tags that confuse the matcher
		cardName = mtgmatcher.Cut(cardName, "APAC")[0] + ")"
	} else if strings.Contains(cardName, "Euro Set") {
		// Cut the "Set 1", "Set 2" tags that confuse the matcher
		cardName = mtgmatcher.Cut(cardName, "Euro")[0] + ")"
	} else if strings.HasPrefix(edition, "Gift Boxes: ") {
		edition = strings.Replace(edition, "Gift Boxes: ", "", 1)
	}

	return cardName, edition, nil
}
