package strikezone

import (
	"fmt"

	"github.com/kodabb/go-mtgban/mtgban"
)

type szCard struct {
	Name    string
	Edition string
	IsFoil  bool

	Conditions string
	Price      float64
	Quantity   int

	Notes string
}

// LUT for typos and variants in SZ card names.
var cardTable = map[string]string{
	// Prerelease (that could could not have been handled earlier)
	"Crucible of Worlds M19 Prerelease":                   "Crucible of Worlds (Prerelease)",
	"Arcades the Strategist Prerelease Promo":             "Arcades the Strategist (Prerelease)",
	"Drowned Catacomb Prerelease Ixalan":                  "Drowned Catacomb (Prerelease)",
	"Legion s Landing Adanto the First Fort (Prerelease)": "Legion's Landing (Prerelease)",

	// Other promos
	"Flusterstorm Judge Promo (DCI)":                "Flusterstorm (Judge Promo)",
	"Sol Ring GP MagicFest Promo (Foil)":            "Sol Ring (MagicFest 2019)",
	"Sol Ring GP MagicFest Promo (Non Foil)":        "Sol Ring (MagicFest 2019)",
	"Shriekmaw (Lorwyn Release Event)":              "Shriekmaw (Release Event)",
	"Karador Ghost Chieftan (Judge Promo)":          "Karador Ghost Chieftain (Judge Promo)",
	"Some Disassembly Require (2017 Holiday Promo)": "Some Disassembly Required (Holiday Promo)",
	"Primordial Hydra X Box Promo 2013":             "Primordial Hydra (Duel of the Planeswalkers)",
	"Zombie Apocalypse Full Art Game Day Promo":     "Zombie Apocalypse (Full Art Game Day Promo)",
	"The Haunt of Hightower BIBB":                   "The Haunt of Hightower (BIBB)",
	"Mountain Grand Prix 2018":                      "Mountain (MagicFest 2019)",
	"Naughty Nice Holiday Promo":                    "Naughty (Holiday Promo)",
	"Island Ravnica Weekend A003":                   "Island Ravnica Weekend A03",
	"Spellseeker Judge":                             "Spellseeker (Judge)",

	"Growing Rites of Ittlimoc (Itlimoc Cradle of the Sun BIBB Alt Art)": "Growing Rites of Itlimoc (BIBB Alt Art)",

	// All the SDCC
	"Ajani Caller of the Pride 2013 SDCC Comicon Promo":  "Ajani Caller of the Pride (2013 SDCC)",
	"Ajani Steadfast (2104 SDCC)":                        "Ajani Steadfast (2014 SDCC)",
	"Chandra Fire of Kaladesh SDCC 2015":                 "Chandra Fire of Kaladesh (2015 SDCC)",
	"Chandra Pyromaster SDCC Comicon Promo (2013)":       "Chandra Pyromaster (2013 SDCC)",
	"Chandra Torch of Defiance SDCC 2017":                "Chandra Torch of Defiance (2017 SDCC)",
	"Chandra Torch of Defiance SDCC 2018":                "Chandra Torch of Defiance (2018 SDCC)",
	"Garruk Apex Predator SDCC 2014":                     "Garruk Apex Predator (2014 SDCC)",
	"Garruk Caller of Beast 2013 SDCC Comicon Promo":     "Garruk Caller of Beasts (2013 SDCC)",
	"Jace Memory Adept 2013 SDCC Comicon Promo":          "Jace Memory Adept (2013 SDCC)",
	"Jace Vryns Prodigy SDCC 2015":                       "Jace Vryns Prodigy (2015 SDCC)",
	"Liliana Heretical Healer SDCC 2015":                 "Liliana Heretical Healer (2015 SDCC)",
	"Liliana of the Dark Realms 2013 SDCC Comicon Promo": "Liliana of the Dark Realms (2013 SDCC)",
	"Nissa Vastwood Seer SDCC 2015":                      "Nissa Vastwood Seer (2015 SDCC)",
	"Nissa Vital Force 2018 SDCC":                        "Nissa Vital Force (2018 SDCC)",

	// Real typos
	"Faithless Lootoing (IDW Promo)":            "Faithless Looting (IDW)",
	"Geist of Sain Traft (WMCQ)":                "Geist of Saint Traft (WMCQ)",
	"Goldari Thug":                              "Golgari Thug",
	"Grafdiffer s Cage (Prerelease)":            "Grafdigger's Cage (Prerelease)",
	"Honor the Pure (M10 Release Event Promo)":  "Honor of the Pure (M10 Release Event Promo)",
	"Ink-Eyes Servan of Oni":                    "Ink-Eyes Servant of Oni",
	"Jace, Weilder of Mysteries Stained Glass":  "Jace, Wielder of Mysteries Stained Glass",
	"Laughing Hyenas":                           "Laughing Hyena",
	"Ox of Agona (Prerelease)":                  "Ox of Agonas (Prerelease)",
	"Polukranos Unchanied (Prerelease)":         "Polukranos Unchained (Prerelease)",
	"Selfless Spirt (Eldritch Moon Prerelease)": "Selfless Spirit (Eldritch Moon Prerelease)",
	"Showcase Aanax, Hardened in the Forge":     "Showcase Anax, Hardened in the Forge",
	"SwampMagic Fest 2019":                      "Swamp (MagicFest 2019)",

	// Split cards
	"Discovery Dispersal": "Discovery",
	"Expansion Explosion": "Expansion",
	"Find Finality":       "Find",
	"Fire/Ice (Fire)":     "Fire",
	"Revival Revenge":     "Revival",
	"Thrash Threat":       "Thrash",

	// Funny cards
	"Ach!  Hans, Run!":               "\"Ach! Hans, Run!\"",
	"B.F.M. #28 (Big Furry Monster)": "B.F.M. (Big Furry Monster)",
	"B.F.M. #29 (Big Furry Monster)": "B.F.M. (Big Furry Monster) (b)",
	"Who What When Where Why":        "Who",
}

func (sz *Strikezone) convert(c *szCard) (*mtgban.Card, error) {
	setName, setCheck := sz.parseSet(c)
	cardName, numberCheck := sz.parseNumber(c, setName)

	cardName = sz.norm.Normalize(cardName)

	// Loop over the DB
	for _, set := range sz.db {
		if setCheck(set) {
			for _, card := range set.Cards {
				dbCardName := sz.norm.Normalize(card.Name)

				// These sets sometimes have extra stuff that we stripped earlier
				if set.Type == "funny" {
					s := mtgban.SplitVariants(dbCardName)
					dbCardName = s[0]
				}

				// Check card name
				cardCheck := dbCardName == cardName

				// Narrow results with the number callback
				if numberCheck != nil {
					cardCheck = cardCheck && numberCheck(set, card)
				}

				if cardCheck {
					ret := mtgban.Card{
						Id:   card.UUID,
						Name: card.Name,
						Set:  set.Name,
						Foil: c.IsFoil,
					}
					if (card.HasNonFoil || card.IsAlternative) && c.IsFoil {
						ret.Id += "_f"
					}
					return &ret, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("Card not found '%s->%s' in '%s->%s' (foil=%v) %q", c.Name, cardName, c.Edition, setName, c.IsFoil, c)
}
