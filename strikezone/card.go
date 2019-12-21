package strikezone

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

// SZCard contains a set of a generic card representation with the associated
// pricing data scaped from Strikezone Online.
type SZCard struct {
	Name string `json:"name"`
	Set  string `json:"set"`
	Foil bool   `json:"foil"`

	Pricing float64 `json:"price"`
	Qty     int     `json:"quantity"`
}

// LUT for SZ sets to MTGJSON sets.
var setTable = map[string]string{
	"10th Edition":                           "Tenth Edition",
	"4th Edition":                            "Fourth Edition",
	"5th Edition":                            "Fifth Edition",
	"7th Edition":                            "Seventh Edition",
	"8th Edition":                            "Eighth Edition",
	"9th Edition":                            "Ninth Edition",
	"Classic 6th Edition":                    "Classic Sixth Edition",
	"Commander 2013 Edition":                 "Commander 2013",
	"Commander 2014 Edition":                 "Commander 2014",
	"Commander 2016 Edition":                 "Commander 2016",
	"Commander":                              "Commander 2011",
	"Duel Decks: Kiora vs. Elspeth":          "Duel Decks: Elspeth vs. Kiora",
	"Duel Decks: Phyrexia vs. The Coalition": "Duel Decks: Phyrexia vs. the Coalition",
	"Futuresight":                            "Future Sight",
	"Guilds of Ravnica Mythic Edition":       "Mythic Edition",
	"Mystery Booster Test Print":             "Mystery Booster Playtest Cards",
	"Ravnica Allegiance Mythic Edition":      "Mythic Edition",
	"Ravnica":                                "Ravnica: City of Guilds",
	"Revised":                                "Revised Edition",
	"Shadows Over Innistrad":                 "Shadows over Innistrad",
	"Time Spiral Time Shifted":               "Time Spiral Timeshifted",
	"Ultimate Box Toppers":                   "Ultimate Box Topper",
	"Unlimited":                              "Unlimited Edition",
	"War of the Spark Mythic Edition":        "Mythic Edition",
}

// LUT for typos and variants in SZ card names.
var cardTable = map[string]string{
	"Ach!  Hans, Run!":               "\"Ach! Hans, Run!\"",
	"B.F.M. #28 (Big Furry Monster)": "B.F.M. (Big Furry Monster)",
	"B.F.M. #29 (Big Furry Monster)": "B.F.M. (Big Furry Monster) (b)",
	"Fire/Ice (Fire)":                "Fire Ice",
	"Who What When Where Why":        "Who",

	"Archangel Avacyn (Ayacyn the Purifier)": "Archangel Avacyn (Avacyn, the Purifier)",
}

var allVariants = map[string]map[string]string{
	"Arcane Denial": map[string]string{
		"1": "22a",
		"2": "22b",
	},
}

var atqVariants = map[string]map[string]string{
	"Strip Mine": map[string]string{
		"No Sky, No Tower":    "82a",
		"Sky, Even Terraces":  "82b",
		"No Sky wth Tower":    "82c",
		"Sky Uneven Terraces": "82d",
	},
	"Mishra's Factory": map[string]string{
		"Spring":        "80a",
		"Summer Green":  "80b",
		"Autumn Orange": "80c",
		"Winter Snow":   "80d",
	},
	"Urza's Mine": map[string]string{
		"Pulley": "83a",
		"Mouth":  "83b",
		"Sphere": "83c",
		"Tower":  "83d",
	},
	"Urza's Power Plant": map[string]string{
		"Sphere":  "84a",
		"Columns": "84b",
		"Bug":     "84c",
		"Pot":     "84d",
	},
	"Urza's Tower": map[string]string{
		"Forest":    "85a",
		"Shore":     "85b",
		"Plains":    "85c",
		"Mountains": "85d",
	},
}

var chrVariants = map[string]map[string]string{
	"Urza's Mine": map[string]string{
		"Mouth":  "114a",
		"Pulley": "114c",
		"Sphere": "114b",
		"Tower":  "114d",
	},
	"Urza's Power Plant": map[string]string{
		"Pot":     "115a",
		"Columns": "115b",
		"Sphere":  "115c",
		"Bug":     "115d",
	},
	"Urza's Tower": map[string]string{
		"Forest":    "116a",
		"Plains":    "116b",
		"Mountains": "116c",
		"Shore":     "116d",
	},
}

// CanonicalCard returns a generic Card representation.
func (c *SZCard) CanonicalCard(db mtgjson.MTGDB) (*mtgban.Card, error) {
	setName := c.Set
	cardName := c.Name

	// Adjust the Set information
	if strings.HasPrefix(setName, "Duel Decks: ") {
		setName = strings.Replace(setName, " vs ", " vs. ", 1)
	} else if strings.HasPrefix(setName, "Magic 20") {
		setName = strings.Replace(setName, " Core Set", "", 1)

		// Handle the post-Origins core sets
		s := strings.Split(setName, " ")
		if len(s) < 1 {
			return nil, fmt.Errorf("Invalid Core Set entry %s", setName)
		}
		year, err := strconv.Atoi(s[1])
		if err != nil {
			return nil, err
		}
		if year >= 2019 {
			setName = fmt.Sprintf("Core Set %d", year)
		}
	}

	// Separate anything within () for later use
	variants := mtgban.SplitVariants(cardName)

	// Look up the Set
	ed, found := setTable[setName]
	if found {
		setName = ed
	}

	// TODO
	/*
		    if setName == "Promotional Cards" && len(variants) > 1 {
				cardName = variants[0]
				setName = variants[1]
			}
	*/

	// Handle minor name variations
	variant, found := cardTable[cardName]
	if found {
		cardName = variant
	} else {
		// Work around different variants from ARN
		if setName == "Arabian Nights" {
			if strings.Contains(cardName, "dark circle") {
				cardName = strings.Replace(cardName, " (dark circle)", "", 1)
			} else if strings.Contains(cardName, "light circle") {
				cardName = strings.Replace(cardName, "(light circle)", "[variant]", 1)
			}
		}
	}

	n := mtgban.NewNormalizer()
	cardName = n.Normalize(cardName)

	// Loop over the DB
	for _, set := range db {
		if set.Name == setName {
			for _, card := range set.Cards {
				dbCardName := card.Name
				number := card.Number

				// If we are lucky, name and set match right away
				if n.Normalize(dbCardName) == cardName {
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

				// Adjust as needed then
				switch {
				// Append card Number to basic lands, sometimes appending full art too
				case strings.HasPrefix(card.Type, "Basic Land") &&
					set.Name != "Unstable" && set.Name != "Arabian Nights" && set.Name != "Battlebond":
					dbCardName = fmt.Sprintf("%s (%s)", dbCardName, card.Number)
					if card.IsFullArt {
						switch set.Name {
						case "Amonkhet", "Hour of Devastation", "Unhinged", "Unglued", "Zendikar":
						case "Oath of the Gatewatch":
							if card.Name == "Wastes" {
								dbCardName = fmt.Sprintf("%s (Full Art %s)", card.Name, card.Number)
							}
						default:
							dbCardName += " (Full Art)"
						}
					}

				// JPN Planeswalkers
				case set.Name == "War of the Spark" && len(card.ForeignData) == 1 && card.ForeignData[0].Language == mtgjson.LanguageJapanese:
					dbCardName = fmt.Sprintf("%s (JPN Alternate Art)", card.Name)

				// ARN light/dark variants
				case set.Name == "Arabian Nights":
					if strings.HasSuffix(card.Number, mtgjson.SuffixLightMana) {
						dbCardName += " [variant]"
					}

				// Cards with different art
				case (set.Name == "Antiquities" || set.Name == "Chronicles" || set.Name == "Alliances") && len(variants) > 1:
					versions := atqVariants
					if set.Name == "Chronicles" {
						versions = chrVariants
					} else if set.Name == "Alliances" {
						versions = allVariants
					}

					no, found := versions[variants[0]][variants[1]]
					if found {
						dbCardName = cardName
						number = no
					}

				// ELD-style variants (needs to be last)
				case len(card.Variations) > 0:
					switch card.FrameEffect {
					case mtgjson.FrameEffectShowcase:
						dbCardName = fmt.Sprintf("Showcase %s", card.Name)
					case mtgjson.FrameEffectExtendedArt:
						dbCardName = fmt.Sprintf("Extended Art %s", card.Name)
					default:
						if card.BorderColor == mtgjson.BorderColorBorderless {
							dbCardName = fmt.Sprintf("Borderless %s", card.Name)
						}
					}

				// Layout renaming
				default:
					switch card.Layout {
					case mtgjson.LayoutTransform:
						if set.Name != "Rivals of Ixalan" {
							dbCardName = fmt.Sprintf("%s (%s)", card.Names[0], card.Names[1])
						}
					case mtgjson.LayoutAftermath:
						dbCardName = fmt.Sprintf("%s to %s", card.Names[0], card.Names[1])
					case mtgjson.LayoutSplit:
						switch set.Name {
						case "Dissension", "Dragon's Maze", "Planar Chaos":
							dbCardName = fmt.Sprintf("%s / %s", card.Names[0], card.Names[1])
						case "Mystery Booster Playtest Cards":
							switch card.Names[0] {
							case "Smelt":
								dbCardName = fmt.Sprintf("%s / %s / %s", card.Names[0], card.Names[1], card.Names[2])
							case "Start":
								dbCardName = fmt.Sprintf("%s / %s", card.Names[0], card.Names[1])

							case "Bind":
								dbCardName = fmt.Sprintf("%s // %s", card.Names[0], card.Names[1])
							default:

							}
						default:
							dbCardName = fmt.Sprintf("%s %s", card.Names[0], card.Names[1])
						}
					}
				}

				dbCardName = n.Normalize(dbCardName)

				// Check for the (possibly) modified card name, and the card number variant
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

func (c *SZCard) Price() float64 {
	return c.Pricing
}
func (c *SZCard) TradePrice() float64 {
	return c.Pricing * 1.3
}
func (c *SZCard) Quantity() int {
	return c.Qty
}
func (c *SZCard) Conditions() string {
	return "NM"
}
func (c *SZCard) Market() string {
	return "Strikezone Online"
}
