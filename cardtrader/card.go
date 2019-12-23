package cardtrader

import (
	"fmt"
	"strings"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

type CTCard struct {
	Name string `json:"name"`
	Set  string `json:"set"`
	Foil bool   `json:"foil"`

	Vendor    string  `json:"vendor"`
	Pricing   float64 `json:"price"`
	Condition string  `json:"conditions"`
	Qty       int     `json:"quantity"`

	scryfallId string
	number     string
	setCode    string
}

// These cards are under the wrong set in CT db
var card2setTable = map[string]string{
	"Archfiend of Ifnir":       "Amonkhet Promos",
	"Birds of Paradise":        "Magic 2011 Promos",
	"Burning Sun's Avatar":     "Ixalan Promos",
	"Captain's Hook":           "Rivals of Ixalan Promos",
	"Cathedral of War":         "Magic 2013 Promos",
	"Celestial Colonnade":      "Worldwake Promos",
	"Chandra's Phoenix":        "Magic 2012 Promos",
	"Day of Judgment":          "Zendikar Promos",
	"Death Baron":              "Core Set 2019 Promos",
	"Devil's Play":             "Shadows over Innistrad Promos",
	"Fated Conflagration":      "Born of the Gods Promos",
	"Goblin Dark-Dwellers":     "Oath of the Gatewatch Promos",
	"Goblin Rabblemaster":      "Magic 2015 Promos",
	"Gravecrawler":             "Dark Ascension Promos",
	"Guul Draz Assassin":       "Rise of the Eldrazi Promos",
	"Impervious Greatwurm":     "Guilds of Ravnica",
	"Magister of Worth":        "Launch Parties",
	"Memoricide":               "Scars of Mirrodin Promos",
	"Mirran Crusader":          "Mirrodin Besieged Promos",
	"Nexus of Fate":            "Core Set 2019",
	"Nightveil Specter":        "Gatecrash Promos",
	"Ojutai's Command":         "Dragons of Tarkir Promos",
	"Ratchet Bomb":             "Magic 2014 Promos",
	"Rattleclaw Mystic":        "Khans of Tarkir Promos",
	"Render Silent":            "Dragon's Maze Promos",
	"Rienne, Angel of Rebirth": "Core Set 2020",
	"Ruinous Path":             "Battle for Zendikar Promos",
	"Scrap Trawler":            "Aether Revolt Promos",
	"Shamanic Revelation":      "Fate Reforged Promos",
	"Skyship Stalker":          "Kaladesh Promos",
	"Surgical Extraction":      "New Phyrexia Promos",
	"Thalia, Heretic Cathar":   "Eldritch Moon Promos",
	"Wildfire Eternal":         "Hour of Devastation Promos",

	"Elusive Tormentor // Insidious Mist": "Shadows over Innistrad Promos",

	"Arguel's Blood Fast // Temple of Aclazotz":              "XLN Treasure Chest",
	"Conqueror's Galleon // Conqueror's Foothold":            "XLN Treasure Chest",
	"Dowsing Dagger // Lost Vale":                            "XLN Treasure Chest",
	"Growing Rites of Itlimoc // Itlimoc, Cradle of the Sun": "XLN Treasure Chest",
	"Legion's Landing // Adanto, thFirst Fort":               "XLN Treasure Chest",
	"Primal Amulet // Primal Wellspring":                     "XLN Treasure Chest",
	"Search for Azcanta // Azcanta, the Sunken Ruin":         "XLN Treasure Chest",
	"Thaumatic Compass // Spires of Orazca":                  "XLN Treasure Chest",
	"Treasure Map // Treasure Cove":                          "XLN Treasure Chest",
	"Vance's Blasting Cannons // Spitfire Bastion":           "XLN Treasure Chest",
}

func (c *CTCard) CanonicalCard(db mtgjson.MTGDB) (*mtgban.Card, error) {
	// Search by set code
	set, found := db[c.setCode]
	if found {
		for _, card := range set.Cards {
			// Use collector number to identify, fallback to card name if not available
			check := c.number == card.Number
			if c.number == "" {
				check = c.Name == card.Name
			}
			if check {
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

	// Search by any of the scryfall ids (if available)
	if c.scryfallId != "" {
		for _, set := range db {
			for _, card := range set.Cards {
				if c.scryfallId == card.ScryfallId ||
					c.scryfallId == card.ScryfallIllustrationId ||
					c.scryfallId == card.ScryfallOracleId {
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

	// Search by (possibly modified) set name
	setName := c.Set
	switch setName {
	case "Buy a Box ", "URL/Convention Promos":
		setName = card2setTable[c.Name]
	default:
		if strings.HasPrefix(setName, "WCD") {
			year := 0
			n, _ := fmt.Sscanf(setName, "WCD %d: ", &year)
			if n == 1 {
				setName = fmt.Sprintf("World Championship Decks %d", year)
			}
		}
	}

	for _, set := range db {
		if set.Name == c.Set {
			for _, card := range set.Cards {
				if c.number == card.Number {
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

	return nil, fmt.Errorf("Card not found '%s' in '%s' (%v)", c.Name, c.Set, c.Foil)
}

func (c *CTCard) Conditions() string {
	conditions := "NM"

	switch c.Condition {
	case "Near Mint", "Mint":
		conditions = "NM"
	case "Slightly Played":
		conditions = "SP"
	case "Moderately Played":
		conditions = "MP"
	case "Played", "Heavily Played":
		conditions = "HP"
	case "Poor":
		conditions = "PO"
	}

	return conditions
}
func (c *CTCard) Market() string {
	return c.Vendor
}
func (c *CTCard) Price() float64 {
	return c.Pricing
}
func (c *CTCard) TradePrice() float64 {
	return 0
}
func (c *CTCard) Quantity() int {
	return c.Qty
}
