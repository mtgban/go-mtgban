package miniaturemarket

import (
	"fmt"
	"strings"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

type MMCard struct {
	Name string `json:"name"`
	Set  string `json:"set"`
	Foil bool   `json:"foil"`

	Pricing float64 `json:"price"`
}

func splitVariantsSquare(str string) []string {
	fields := strings.Split(str, " [")
	for i, _ := range fields {
		fields[i] = strings.TrimRight(fields[i], "]")
	}
	return fields
}

func (c *MMCard) CanonicalCard(db mtgjson.MTGDB) (*mtgban.Card, error) {
	cardName := c.Name
	setName := c.Set

	// Split name according to the content of () or []
	variants := mtgban.SplitVariants(cardName)
	if len(variants) == 1 {
		variants = splitVariantsSquare(cardName)
	}

	ed, found := setTable[setName]
	if found {
		setName = ed
	} else {
		switch setName {
		// Handle light/dark variations
		case "Arabian Nights":
			if strings.Contains(cardName, "Dark") {
				cardName = strings.Replace(cardName, " (Dark)", "", 1)
			} else if strings.Contains(cardName, "Light") {
				cardName = strings.Replace(cardName, "(Light)", "[variant]", 1)
			}
		// MM does not distinguish among DDA decks
		case "Duel Decks: Anthology":
			setName = "Duel Decks Anthology"
			if len(variants) > 1 {
				setName += ": " + variants[1]
				setName = strings.Replace(setName, " vs ", " vs. ", 1)
			}
			cardName = variants[0]
		// The "reminder text" variants come from a different Portal
		case "Portal":
			if len(variants) > 1 && variants[1] == "Reminder Text" {
				setName = "Portal Demo Game"
				cardName = variants[0]
			}
		}
	}

	// TODO
	/*
		    if setName == "Promo" && len(variants) > 1 {
				cardName = variants[0]
				setName = variants[1]
			}
	*/

	// Handle minor name variations
	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	n := mtgban.NewNormalizer()
	cardName = n.Normalize(cardName)

	// Loop over the DB
	for _, set := range db {
		// Need to use strings.Contains due to:
		// - "Duel Decks Anthology" containing every single deck version
		// - "Guild Kit" not specifying GRN or ARN
		// - "Secret Lair" listing both planewalker promos and drop series
		check := set.Name == setName
		if setName == "Secret Lair" || strings.HasPrefix(setName, "Duel Decks") {
			check = strings.HasPrefix(set.Name, setName)
		} else if strings.Contains(setName, "Guild Kit") {
			check = strings.Contains(set.Name, setName)
		}
		if check {
			for _, card := range set.Cards {
				dbCardName := card.Name
				number := card.Number

				switch {
				// Handle Unlimited basic lands, they have variants in []
				case strings.HasPrefix(card.Type, "Basic Land") && set.Name == "Unlimited Edition":
					no, found := unlVariants[c.Name]
					if found {
						dbCardName = cardName
						number = no
					}

				// Append number to basic lands (except for a few sets)
				case strings.HasPrefix(card.Type, "Basic Land") &&
					!strings.HasPrefix(set.Name, "Un") &&
					set.Name != "Arabian Nights" &&
					set.Name != "Battlebond":
					cardNo := card.Number
					if len(cardNo) == 2 {
						cardNo = "0" + cardNo
					}
					dbCardName = fmt.Sprintf("%s #%s", card.Name, cardNo)
					if card.IsFullArt {
						dbCardName += " (Full Art)"
					}

				// JPN Planeswalkers
				case set.Name == "War of the Spark" && len(card.ForeignData) == 1 && card.ForeignData[0].Language == mtgjson.LanguageJapanese:
					dbCardName = fmt.Sprintf("%s (Japanese Alternate Art)", card.Name)

				// ARN light/dark variants
				case set.Name == "Arabian Nights":
					if strings.HasSuffix(card.Number, mtgjson.SuffixLightMana) {
						dbCardName += " [variant]"
					}

				// Split cards and the like
				case card.Layout == mtgjson.LayoutTransform ||
					card.Layout == mtgjson.LayoutAftermath ||
					card.Layout == mtgjson.LayoutSplit ||
					card.Layout == mtgjson.LayoutFlip:
					// Skip "Curse of the Fire Penguin"
					if set.Name != "Unhinged" {
						dbCardName = fmt.Sprintf("%s / %s", card.Names[0], card.Names[1])
					}

				// Cards with different versions
				case len(variants) > 1:
					if set.Name == "Fallen Empires" || set.Name == "Commander Anthology Volume II" {
						// Fallen Empires and CMD Anthology Vol2 variants match their artist
						if card.Artist == variants[1] {
							dbCardName = cardName
						}
					} else if strings.HasPrefix(variants[1], "#") {
						// Some cards have their number embedded in the card name
						dbCardName = cardName
						number = variants[1][1:]
					} else {
						// Cards with different art
						no, found := setVariants[set.Name][variants[0]][variants[1]]
						if found {
							dbCardName = cardName
							number = no
						} else if len(card.Variations) > 0 {
							// ELD-style variants
							switch card.FrameEffect {
							case mtgjson.FrameEffectShowcase:
								dbCardName = fmt.Sprintf("%s (Showcase Art)", card.Name)
							case mtgjson.FrameEffectExtendedArt:
								dbCardName = fmt.Sprintf("%s (Extended Art)", card.Name)
							default:
								if card.BorderColor == mtgjson.BorderColorBorderless {
									dbCardName = fmt.Sprintf("%s (Alternate Art)", card.Name)
								}
							}
						}
					}
				}

				// Check for the (possibly) modified card name, and the card number variant
				dbCardName = n.Normalize(dbCardName)
				if dbCardName == cardName && number == card.Number {
					ret := mtgban.Card{
						Id:   card.UUID,
						Name: card.Name,
						Set:  set.Name,
						Foil: c.Foil,
					}
					if card.HasNonFoil && c.Foil {
						ret.Id += "_f"
					}
					return &ret, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("Card not found '%s' in '%s' (%v)", c.Name, setName, c.Foil)
}

func (c *MMCard) Conditions() string {
	return "NM"
}
func (c *MMCard) Market() string {
	return "Miniature Market"
}
func (c *MMCard) Price() float64 {
	return c.Pricing
}
func (c *MMCard) TradePrice() float64 {
	return c.Pricing * 1.3
}
func (c *MMCard) Quantity() int {
	return 0
}
