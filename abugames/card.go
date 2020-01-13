package abugames

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

type ABUCard struct {
	Name      string `json:"name"`
	Set       string `json:"set"`
	Foil      bool   `json:"foil"`
	Condition string `json:"conditions"`

	BuyPrice     float64 `json:"buy_price"`
	BuyQuantity  int     `json:"buy_quantity"`
	TradePricing float64 `json:"trade_price"`

	SellPrice    float64 `json:"sell_price"`
	SellQuantity int     `json:"sell_quantity"`

	FullName string `json:"full_name"`
	Artist   string `json:"artist"`
	Number   string `json:"number"`
	Layout   string `json:"layout"`
	Id       string `json:"rarity"`
}

var cardTable = map[string]string{
	"B.F.M. Big Furry Monster Left":     "B.F.M.",
	"B.F.M. Big Furry Monster Right":    "B.F.M.",
	"Erase (Not the Urza's Legacy One)": "Erase",
	"Hazmat Suit (Used)":                "Hazmat Suit",
	"No Name":                           "_____",
	"Rathi Berserker (Aerathi)":         "Aerathi Berserker",
	"Scholar of the Stars":              "Scholar of Stars",
	"Who What When Where Why":           "Who",
	"Absolute Longest Card Name Ever":   "Our Market Research Shows That Players Like Really Long Card Names So We Made this Card to Have the Absolute Longest Card Name Ever Elemental",
}

// This is the date (month) of release of Khans of Tarkir, any prerelease card
// published after this date will have 's' appended to its number
var newPrereleaseDate = time.Date(2014, time.September, 1, 0, 0, 0, 0, time.UTC)

func (c *ABUCard) CanonicalCard(db mtgjson.MTGDB) (*mtgban.Card, error) {
	cardName := c.Name
	number := ""

	lutName, found := cardTable[cardName]
	if found {
		cardName = lutName
	}

	// Split name according to the content of ()
	variants := mtgban.SplitVariants(cardName)

	specifier := ""
	if len(variants) > 1 {
		specifier = variants[1]
	}

	setName, setCheck := c.parseSet(cardName)
	number, numberCheck := c.parseNumber(setName, specifier)

	isSetPrerelease := specifier == "Prerelease"
	isSetPromoPack := specifier == "Promo Pack"
	isSetSomeFormOfPromo := strings.HasSuffix(specifier, "Intro Pack") ||
		strings.HasSuffix(specifier, "Into Pack") ||
		specifier == "Bundle"
	isSetExtendedArt := specifier == "Extended Art"
	isSetShowcase := specifier == "Showcase"
	isSetBorderless := specifier == "Borderless"
	if isSetPrerelease {
		switch cardName {
		case // These cards are not stamped, don't consider them as prerelease
			"Rukh Egg",                           //8ed
			"Reya Dawnbringer",                   //10e
			"Bloodlord of Vaasgoth",              //m12
			"Xathrid Gorgon",                     //m13
			"Mayor of Avabruck / Howlpack Alpha", //inn
			"Moonsilver Spear":                   //avr
			isSetPrerelease = false
		}
	}

	// Only keep one of the split cards
	if strings.Contains(cardName, " / ") {
		s := strings.Split(cardName, " / ")
		cardName = s[0]
	}

	if c.Layout != "" {
		if strings.Contains(cardName, " and ") {
			s := strings.Split(cardName, " and ")
			cardName = s[0]
		} else if strings.Contains(cardName, " to ") {
			s := strings.Split(cardName, " to ")
			cardName = s[0]
		}
	}

	n := mtgban.NewNormalizer()
	cardName = n.Normalize(cardName)

	// Loop over the DB
	for _, set := range db {
		if setCheck(set) {
			s := strings.Split(set.ReleaseDate, "-")
			setYear, _ := strconv.Atoi(s[0])
			setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)
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
				// WCD always have something that should be checked, but we can't trust
				// number alone (as it may or not be initialized in that case)
				if number != "" || c.Set == "World Championship" {
					cardCheck = cardCheck && numberCheck(card.Number)
				}

				// These three variants often fall in the same set, and cannot use
				// numbers because they are unreliable. Each section checks for the
				// opposite making sure there is no aliasing (when number is not available).
				if isSetPrerelease {
					extraCheck := card.IsDateStamped && !strings.Contains(card.Number, "p")
					if setDate.After(newPrereleaseDate) {
						extraCheck = extraCheck && strings.Contains(card.Number, "s")
					}
					cardCheck = cardCheck && extraCheck
				} else if isSetPromoPack {
					extraCheck := !card.IsDateStamped && !strings.Contains(card.Number, "s")
					if strings.HasSuffix(set.Name, "Promos") {
						extraCheck = extraCheck && strings.Contains(card.Number, "p")
					}
					if setYear >= 2019 {
						num := card.Number
						if num[len(num)-1] == 'p' {
							num = num[:len(num)-1]
						}
						no, err := strconv.Atoi(num)
						if err == nil {
							extraCheck = extraCheck && no > set.BaseSetSize
						}
					}
					cardCheck = cardCheck && extraCheck
				} else if isSetSomeFormOfPromo {
					extraCheck := !strings.Contains(card.Number, "s") &&
						!strings.Contains(card.Number, "p")
					cardCheck = cardCheck && extraCheck && card.IsPromo
				} else {
					if setYear >= 2015 && !strings.HasSuffix(set.Name, "Promos") {
						cardCheck = cardCheck && !strings.Contains(card.Number, "s")
					}
				}

				// Handle the ELD-style promo cards
				if isSetPrerelease || isSetPromoPack || isSetSomeFormOfPromo {
					cardCheck = cardCheck &&
						card.FrameEffect != mtgjson.FrameEffectExtendedArt &&
						card.FrameEffect != mtgjson.FrameEffectShowcase &&
						card.BorderColor != mtgjson.BorderColorBorderless
				}
				if isSetExtendedArt {
					extraCheck := card.FrameEffect == mtgjson.FrameEffectExtendedArt
					cardCheck = cardCheck && extraCheck
				} else if isSetShowcase {
					extraCheck := card.FrameEffect == mtgjson.FrameEffectShowcase
					cardCheck = cardCheck && extraCheck
				} else if isSetBorderless {
					extraCheck := card.BorderColor == mtgjson.BorderColorBorderless
					cardCheck = cardCheck && extraCheck
				} else {
					if setYear >= 2019 && !strings.HasSuffix(set.Name, "Promos") {
						cardCheck = cardCheck &&
							card.FrameEffect != mtgjson.FrameEffectExtendedArt &&
							card.FrameEffect != mtgjson.FrameEffectShowcase &&
							card.BorderColor != mtgjson.BorderColorBorderless
					}
				}

				if cardCheck {
					ret := mtgban.Card{
						Id:   card.UUID,
						Name: card.Name,
						Set:  set.Name,
						Foil: c.Foil,
					}
					if (card.HasNonFoil || card.IsAlternative) && c.Foil {
						ret.Id += "_f"
					}
					return &ret, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("Card not found '%s->%s' (%s) in '%s->%s' (foil=%v) {%s/%s} [%v]", c.Name, cardName, c.FullName, c.Set, setName, c.Foil, number, c.Number, c)
}

func (c *ABUCard) Conditions() string {
	return c.Condition
}
func (c *ABUCard) Market() string {
	return "ABU Games"
}
func (c *ABUCard) Price() float64 {
	if c.SellPrice != 0 {
		return c.SellPrice
	}
	return c.BuyPrice
}

func (c *ABUCard) TradePrice() float64 {
	return c.TradePricing
}
func (c *ABUCard) Quantity() int {
	if c.SellPrice != 0 {
		return c.SellQuantity
	}
	return c.BuyQuantity
}

func (c *ABUCard) Same(card interface{}) bool {
	cc, ok := card.(*ABUCard)
	if ok {
		return c.Id == cc.Id
	}
	return false
}
