package cardkingdom

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

// CKCard contains a set of a generic card representation with the associated
// pricing data scaped from Card Kingdom.
type CKCard struct {
	Name string `json:"name"`
	Set  string `json:"set"`
	Foil bool   `json:"foil"`

	Pricing float64 `json:"price"`
	Qty     int     `json:"quantity"`

	Type string `json:"type"`
}

// Regualar expression for extracting year from WCD sets
var wcdExp = regexp.MustCompile(`.*[\w\s]+ (\d+) - [\w\s]+.*`)

// Regualar expression for extracting card number from some cards,
// in particular lands, JPN planeswalkers, and some random cards.
// The regex needs to process:
//   Wastes (1 A - Non-Full Art)
//   Mountain (023 B - Full Art)
//   Island (44)
//   Forest (123 C)
//   Island (6 - C)
//   Plains (D)
//   Huatli, the Sun's Heart (230 - JPN Alternate Art)
//   Nicol Bolas, Dragon-God (207 - JPN Alternate Art)
// but not:
//   Swamp (2017 Gift Pack)
var numberExp = regexp.MustCompile(`[ ,'\-\w]+ \(([0-9A-Z][0-9A-Z]?[0-9A-Z]?)[ \-A-Za-z]*\)`)

// Try to parse the card number/letter embedded in some cards
// (mostly basic lands and a few special cards)
func parseNumber(cardName string) (string, error) {
	if strings.HasPrefix(cardName, "Plains (") ||
		strings.HasPrefix(cardName, "Island (") ||
		strings.HasPrefix(cardName, "Swamp (") ||
		strings.HasPrefix(cardName, "Mountain (") ||
		strings.HasPrefix(cardName, "Forest (") ||
		strings.HasPrefix(cardName, "Wastes (") ||
		// These are not basic land types, but follow the same trend
		strings.Contains(cardName, "Guildgate (") ||
		strings.Contains(cardName, "Signet (") ||
		strings.Contains(cardName, "Simulacrum (") ||
		strings.Contains(cardName, "False God (") ||
		// The JPN planeswalker do too
		strings.Contains(cardName, "JPN Alternate Art") {

		// Extract number or letter prefix
		number := numberExp.ReplaceAllString(cardName, `$1`)

		// Drop the leading zeros from some lands, so that Atoi can process it
		number = strings.TrimLeft(number, "0")

		_, err := strconv.Atoi(number)
		switch {
		case err == nil:
			// Append the letter variant to the number for non-full art wastes
			if strings.HasPrefix(cardName, "Wastes") && strings.Contains(cardName, "Non-Full Art") {
				number += "a"
			} else if strings.Contains(cardName, "JPN Alternate Art") {
				// Rebuild the collector number to the expected one
				if strings.Contains(cardName, "Prerelease") {
					number += "s"
				}
				number += mtgjson.SuffixSpecial
			}
			return number, nil
		case len(number) > 0 && unicode.IsLetter(rune(number[0])):
			// Do nothing, this not an error
			return "", nil
		default:
			return "", fmt.Errorf("Unsupported card name in regexp '%s'", number)
		}
	}
	return "", nil
}

// CanonicalCard returns a generic Card representation.
func (c *CKCard) CanonicalCard(db mtgjson.MTGDB) (*mtgban.Card, error) {
	cardName := c.Name
	setName := parseSet(c.Name, c.Set, c.Type)

	// Function to determine whether we're parsing the correct set
	setCheck := func(set mtgjson.Set) bool {
		return set.Name == setName
	}

	// Handle minor name variations
	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	// Obtain as much information as possible from the card name
	specifier := ""
	variants := mtgban.SplitVariants(cardName)
	if len(variants) > 1 {
		specifier = variants[1]
	}

	// A few booleans to get accurate results later
	isSetVariant := strings.HasSuffix(c.Set, "Variants")
	isSetPromo := len(variants) > 1 && setName == "Promotional"
	isSetPrerelease := strings.Contains(specifier, "Prerelease")
	isSetPromoPack := strings.HasPrefix(specifier, "Promo Pack")

	// Adjust set name and comparison function for promotional sets
	if isSetPromo {
		ed, fun := parsePromotional(variants)
		if ed != "" {
			setName = ed
		}
		setCheck = fun
	}

	// Look up card number from the static table
	number := ""
	no, found := setVariants[setName][variants[0]][specifier]
	if found {
		number = no
		// Find card number in another way for these sets
	} else if strings.Contains(setName, "Ravnica Weekend") {
		s := strings.Split(variants[1], " ")
		number = s[3]
	} else {
		var err error
		// If not present, try parsing it from the card name
		no, err = parseNumber(cardName)
		if err == nil {
			number = no
		} else {
			return nil, fmt.Errorf("%s: %s (%s)", err, cardName, setName)
		}
	}

	// We gathered all information we could from the variants,
	// use the main name of the card in any further check.
	cardName = variants[0]

	// Only keep one of the split cards
	if strings.Contains(cardName, " // ") {
		s := strings.Split(cardName, " // ")
		cardName = s[0]
	}

	n := mtgban.NewNormalizer()
	cardName = n.Normalize(cardName)

	// Loop over the DB
	for _, set := range db {
		if setCheck(set) {
			for _, card := range set.Cards {
				dbCardName := n.Normalize(card.Name)

				// These sets sometimes have extra stuff that we stripped earlier
				if set.Type == "funny" {
					s := mtgban.SplitVariants(dbCardName)
					dbCardName = s[0]
				}

				// Check card name
				cardCheck := dbCardName == cardName

				// If card number is available, use it to narrow results
				if number != "" {
					cardCheck = cardCheck && (number == card.Number)
				}

				// ELD-style variant cards
				if isSetVariant {
					// This kind of sets store a lot of variants at the end,
					// make sure check where each card belongs
					num, err := strconv.Atoi(card.Number)
					if err == nil {
						switch specifier {
						case "Showcase", "Extended Art", "Borderless":
							cardCheck = cardCheck && num > set.BaseSetSize
						default:
							cardCheck = cardCheck && num <= set.BaseSetSize
						}
					}
					// Hack to indentify sets with too many cards at the end
				} else if len(variants) == 1 && set.TotalSetSize-set.BaseSetSize > 90 {
					num, _ := strconv.Atoi(card.Number)
					if num > 0 && set.BaseSetSize > 0 {
						cardCheck = cardCheck && num <= set.BaseSetSize
					}
				} else if isSetPromo {
					// This check applies only to certain set types
					isStarter := ((set.Type == "box" || set.Type == "starter" || set.Type == "funny") && card.IsStarter)
					cardCheck = cardCheck && (card.IsPromo || card.IsAlternative || isStarter)

					// Prerelease and Promo Pack often go toghether, and it's easy
					// to get confused, so each section checks for the opposite,
					// making sure there is no aliasing (when number is not available).
					if isSetPrerelease {
						extraCheck := card.IsDateStamped &&
							!strings.Contains(card.Number, "p")
						cardCheck = cardCheck && extraCheck
					} else if isSetPromoPack {
						extraCheck := !card.IsDateStamped &&
							!strings.Contains(card.Number, "s") &&
							// This check is against the JPN alternate cards from WAR
							!strings.Contains(card.Number, mtgjson.SuffixSpecial)
						cardCheck = cardCheck && extraCheck
					}

					// Likewise, a lot of the ELD-style variants exist alongside
					// Prerelease and Promo Pack, so make sure to eclude them too
					if isSetPrerelease || isSetPromoPack {
						extraCheck := card.FrameEffect != mtgjson.FrameEffectExtendedArt &&
							card.FrameEffect != mtgjson.FrameEffectShowcase &&
							card.BorderColor != mtgjson.BorderColorBorderless
						cardCheck = cardCheck && extraCheck
					}
				}

				if cardCheck {
					ret := mtgban.Card{
						Id:   card.UUID,
						Name: card.Name,
						Set:  set.Name,
						Foil: c.Foil,
					}

					// Make uuid really unique
					if c.Foil && card.HasNonFoil {
						ret.Id += "_f"
					}

					return &ret, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("Card not found '%s->%s' in '%s->%s' (foil=%q) [%q]", c.Name, cardName, c.Set, setName, c.Foil, variants)
}

func (c *CKCard) Price() float64 {
	return c.Pricing
}
func (c *CKCard) TradePrice() float64 {
	return c.Pricing * 1.3
}
func (c *CKCard) Quantity() int {
	return c.Qty
}
func (c *CKCard) Conditions() string {
	return "NM"
}
func (c *CKCard) Market() string {
	return "Card Kingdom"
}
