package cardkingdom

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgban"
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

type CardkingdomSealed struct {
	LogCallback mtgban.LogCallbackFunc
	Partner     string

	inventoryDate time.Time
	buylistDate   time.Time

	inventory mtgban.InventoryRecord
	buylist   mtgban.BuylistRecord
}

func NewScraperSealed() *CardkingdomSealed {
	ck := CardkingdomSealed{}
	ck.inventory = mtgban.InventoryRecord{}
	ck.buylist = mtgban.BuylistRecord{}
	return &ck
}

func (ck *CardkingdomSealed) printf(format string, a ...interface{}) {
	if ck.LogCallback != nil {
		ck.LogCallback("[CKSealed] "+format, a...)
	}
}

var promo2uuid = map[string]string{
	"HasCon 2017 Magic: The Gathering Box Set": "60e154f0-91f2-5145-ae10-67d7c1c051f1",
	"Ponies: The Galloping Box Set":            "d615348a-1928-5568-87d8-02498257dd5a",

	"SDCC 2013 Box-Set (Set of Five Planeswalkers)":                  "cac0d5ed-f8ae-50dd-b299-5e4496783808",
	"SDCC 2014 Box-Set (Set of Six Planeswalkers) - No Axe":          "b3d764cc-6073-5636-9c9a-c538d375a602",
	"SDCC 2015 Box-Set (Set of Five Planeswalkers) - Book Included":  "36804c43-56da-54e2-9e4d-6db49c41f05b",
	"SDCC 2015 Box-Set (Set of Five Planeswalkers) - No Book":        "",
	"SDCC 2016 Box-Set (Set of Five Planeswalkers)":                  "0f9327be-6aa4-5591-b402-84fb74311fa8",
	"SDCC 2017 Box-Set (Set of Six Planeswalkers) - No Poster":       "",
	"SDCC 2017 Box-Set (Set of Six Planeswalkers) - Poster Included": "e97f0fb0-37bc-560a-ba05-a64879be6358",
	"SDCC 2018 Box-Set (Set of Five Planeswalkers)":                  "b22605e6-bb94-5fc3-b231-338f92776aa7",
	"SDCC 2019 Box-Set": "67ac0039-82db-58f0-9f8b-a7a6bcc979fb",
}

var renames = map[string]string{
	"8th Edition 2-Player Starter":                      "Box Set Core Game",
	"9th Edition 2-Player Starter":                      "9th Edition Boxed Set Core Game",
	"Alara Reborn Intro Pack: Rumbler":                  "Alara Reborn Intro Pack: Rambler",
	"Battle Royale Multi-Player Box Set":                "Battle Royale Box Set Battle Royale Multi Player Box set",
	"Clash Pack - 2015 Core Set  ":                      "Magic 2015 M15 Clash Pack",
	"Commander Anthology Vol. II Box Set":               "Commander Anthology Volume II Box Set",
	"Commander Collection: Green (Foil)":                "Commander Collection: Green - Premium",
	"Commander Collection: Green (Non-Foil)":            "Commander Collection: Green",
	"Deck Builder's Toolkit - 2010 Core Set":            "Magic Deck Builder's Toolkit",
	"Deck Builder's Toolkit - 2011 Core Set":            "Magic 2011 M11 Deck Builders Toolkit",
	"Double Masters VIP Edition Booster Pack":           "Double Masters VIP Edition Pack",
	"Game Night 2019 Boxset":                            "Magic Game Night Set",
	"Legends Booster Pack (English)":                    "Legends Booster Pack",
	"Mystery Booster Booster Box (Convention Edition)":  "Mystery Booster Booster Box Convention Exclusive",
	"Mystery Booster Booster Box (Retail Edition)":      "Mystery Booster Booster Box Retail Exclusive",
	"Mystery Booster Booster Pack (Convention Edition)": "Mystery Booster Pack Convention Edition",
	"Mystery Booster Booster Pack (Retail Edition)":     "Mystery Booster Pack Retail Exclusive",
	"Portal Second Age 2-Player Game":                   "Portal Second Age 2 Player Start",
	"Vanguard Box Set":                                  "Vanguard Gift Box",
}

// Missing product upstream
var skips = map[string]string{
	"2011 Core Set 6-Card Booster Pack":             "",
	"2012 Core Set Booster Battle Pack Display Box": "",
	"2013 Core Set - Deck Builder's Toolkit":        "",
	"7th Edition 2-Player Starter":                  "",
	"Duel Decks: Anthology Box Set":                 "",
	"Mirrodin Besieged Faction Booster Box":         "",
	"Spellslinger Starter Kit (2019)":               "",
	"Torment Theme Deck: Insanity":                  "",
	"Vanguard Box Set":                              "",
	"Zendikar Rising Draft Booster Variety Pack":    "",
}

var deckVariants = map[string]string{
	// Typos
	"Artful Destruction":    "Artufl Destruction",
	"Devastation":           "Devestation",
	"Nissa Vs. Ob Nixillis": "Nissa Vs. Ob Nixilis",
	// Event decks names
	"Strength of Selesnya": "Dragons Maze Event Deck",
	"Underworld Herald":    "Born of the Gods Event Deck",
	"Rush of the Wild":     "Magic 2014 M14 Event Deck",
	"Wrath of the Mortals": "Journey into Nyx Event Deck",
	"Conquering Hordes":    "Khans of Tarkir Event Deck",
	"Ultimate Sacrifice":   "Battle for Zendikar Event Deck",
	"Inspiring Heroics":    "Theros Event Deck",

	// Intro pack renames
	// THR
	"Favors From Nyx":            "Celestial Archon",
	"Manipulative Monstrosities": "Shipbreaker Kraken",
	"Blazing Beasts of Myth":     "Ember Swallower",
	"Devotion to Darkness":       "Abhorrent Overlord",
	"Anthousa's Army":            "Anthousa Setessan Hero",
	// BNG
	"Death's Beginning":  "Black",
	"Inspiration-Struck": "Blue",
	"Insatiable Hunger":  "Green",
	"Forged in Battle":   "Red",
	"Gifts of the Gods":  "White",
	// JOU
	"The Wilds and the Deep": "Green",
	// M15
	"Infernal Intervention":  "Black",
	"Hit the Ground Running": "Blue",
	"Will of the Masses":     "Green",
	"Flames of the Dragon":   "Red",
	"Price of Glory":         "White",
	// KTK
	"Abzan Siege":     "Abzan",
	"Jeskai Monks":    "Jeskai",
	"Sultai Schemers": "Sultai",
	"Mardu Raiders":   "Mardu",
	"Temur Avalanche": "Temur",
	// FRF
	"Unflinching Assault": "Mardu",
	"Cunning Plan":        "Jeskai",
	"Grave Advantage":     "Sultai",
	"Stampeding Hordes":   "Abzan",
	"Surprise Attack":     "Temur",
	// DTK
	"Massed Ranks":        "Dromoka",
	"Enlightened Mastery": "Ojutai",
	"Cruel Plots":         "Silumgar",
	"Relentless Rush":     "Kolaghan",
	"Furious Forces":      "Atarka",
	// ORI
	"Demonic Deals":    "Black",
	"Take to the Sky":  "Blue",
	"Hunting Pack":     "Green",
	"Assemble Victory": "Red",
	"Brave the Battle": "White",
	// BFZ
	"Eldrazi Assault":   "Red",
	"Swarming Instinct": "Blue",
	"Call of Blood":     "Black",
	"Zendikar's Rage":   "Green",
	"Rallying Cry":      "White",
}

func (ck *CardkingdomSealed) scrape() error {
	ckClient := NewCKClient()
	pricelist, err := ckClient.GetSealedList()
	if err != nil {
		return err
	}

	foundProduct := 0

	for _, sealed := range pricelist {
		if strings.Contains(sealed.Name, "Complete") ||
			strings.Contains(sealed.Name, "Bulk") ||
			strings.Contains(sealed.Name, "Counters") ||
			strings.Contains(sealed.Name, "Token") ||
			strings.Contains(sealed.Name, "Common Set") ||
			strings.Contains(sealed.Name, "Non-English") {
			continue
		}

		var uuid string
		edition := sealed.Edition
		name := sealed.Name

		// Missing upstream atm
		if strings.Contains(edition, "Challenger Decks") {
			continue
		}
		_, found := skips[name]
		if found {
			continue
		}

		switch edition {
		case "Promotional":
			fixup, found := promo2uuid[sealed.Name]
			if found {
				uuid = fixup
			}
			// There are a couple of variants that are known unsupported
			if uuid == "" {
				continue
			}
		case "Alpha":
			name = strings.Replace(name, "Alpha", "Alpha Edition", 1)
		case "Beta":
			name = strings.Replace(name, "Beta", "Beta Edition", 1)
		case "Unlimited":
			name = strings.Replace(name, "Unlimited", "Unlimited Edition", 1)
		case "3rd Edition":
			name = strings.Replace(name, "3rd Edition Revised", "Revised Edition", 1)

		case "Commander 2015", "Zendikar Rising":
			name = mtgmatcher.SplitVariants(name)[0]
		case "Mystery Booster/The List":
			if strings.Contains(name, "Retail Edition") {
				edition = "FMB1"
			} else if strings.Contains(name, "Convention Edition") {
				edition = "CMB1"
			}
		case "Secret Lair":
			edition = "Secret Lair Drop"
			if strings.Contains(name, "Ultimate") {
				edition = "Secret Lair: Ultimate Edition"
			}

		// Only one product tracked for these editions for now
		case "Portal 3K":
			if strings.Contains(name, "Theme Deck") {
				continue
			}
		case "Starter 1999":
			if name != "Starter 1999 Booster Pack" {
				continue
			}

		// Missing upstream
		case "Shards of Alara", "Conflux":
			if strings.Contains(name, "Intro") {
				continue
			}
		case "Morningtide":
			if strings.Contains(name, "Theme Deck") {
				continue
			}
		case "World Championships", "Duels of the Planeswalkers":
			continue

		default:
			// All gift packs are in a single edition
			if edition == "Vanguard" || strings.Contains(name, "Gift Box") || strings.Contains(name, "Gift Pack") {
				edition = "2017 Gift Pack"
				// Currently unsupported upstream
				continue
			}
			if strings.Contains(edition, "Core Set") {
				if edition == "2012 Core Set" && strings.Contains(name, "Intro") {
					continue
				}
				edition = mtgmatcher.SealedNormalize(edition)
			}
		}

		if uuid == "" {
			set, err := mtgmatcher.GetSetByName(edition)
			if err != nil {
				ck.printf("edition %s not found - skipping %s", edition, sealed.Name)
				continue
			}

			fixup, found := renames[name]
			if found {
				name = fixup
			}

			// Short circuit small sets (often containing complex names)
			if len(set.SealedProduct) == 1 {
				uuid = set.SealedProduct[0].UUID
			}
			// Loop through the available product and find the closest name match
			for _, product := range set.SealedProduct {
				if mtgmatcher.SealedEquals(product.Name, name) {
					uuid = product.UUID
					break
				}
			}
		}
		// If still not found, try again, looking for well known dividers
		if uuid == "" {
			set, _ := mtgmatcher.GetSetByName(edition)
			for _, tag := range []string{" - ", ": "} {
				if strings.Contains(name, tag) {
					probe := ""
					subtypes := strings.Split(name, tag)
					if len(subtypes) > 1 {
						probe = subtypes[1]
						// Keep the name of planewalker only
						if strings.Contains(name, "Planeswalker Deck") {
							probe = strings.Split(probe, ",")[0]
							// Work around a special case
							if strings.HasPrefix(probe, "Planeswalker Deck") {
								tmp := strings.Fields(probe)
								if len(tmp) > 2 {
									probe = tmp[2]
								}
							}
						}
						probe = mtgmatcher.SplitVariants(probe)[0]
						fixup, found := deckVariants[probe]
						if found {
							probe = fixup
						}
					}

					if probe != "" {
						for _, product := range set.SealedProduct {
							if mtgmatcher.SealedContains(product.Name, probe) {
								uuid = product.UUID
								break
							}
						}
					}
				}
			}
		}

		if uuid == "" {
			// Do not report error for untracked products
			if strings.HasSuffix(name, "Display") {
				continue
			}
			ck.printf("edition %s was found, but %s was not", edition, sealed.Name)
			continue
		}

		foundProduct++

		u, _ := url.Parse("https://www.cardkingdom.com/")
		sellPrice, err := strconv.ParseFloat(sealed.SellPrice, 64)
		if err != nil {
			ck.printf("%v", err)
		}
		if sealed.SellQuantity > 0 && sellPrice > 0 {
			u.Path = sealed.URL
			if ck.Partner != "" {
				q := u.Query()
				q.Set("partner", ck.Partner)
				q.Set("utm_source", ck.Partner)
				q.Set("utm_medium", "affiliate")
				q.Set("utm_campaign", ck.Partner)
				u.RawQuery = q.Encode()
			}

			out := &mtgban.InventoryEntry{
				Conditions: "NM",
				Price:      sellPrice,
				Quantity:   sealed.SellQuantity,
				URL:        u.String(),
			}
			err = ck.inventory.Add(uuid, out)
			if err != nil {
				ck.printf("%v", err)
			}
		}

		u, _ = url.Parse("https://www.cardkingdom.com/purchasing/mtg_sealed")
		buyPrice, err := strconv.ParseFloat(sealed.BuyPrice, 64)
		if err != nil {
			ck.printf("%v", err)
		}
		if sealed.BuyQuantity > 0 && buyPrice > 0 {
			var priceRatio float64

			if sellPrice > 0 {
				priceRatio = buyPrice / sellPrice * 100
			}

			q := u.Query()
			if ck.Partner != "" {
				q.Set("partner", ck.Partner)
				q.Set("utm_source", ck.Partner)
				q.Set("utm_medium", "affiliate")
				q.Set("utm_campaign", ck.Partner)
			}
			u.RawQuery = q.Encode()

			out := &mtgban.BuylistEntry{
				BuyPrice:   buyPrice,
				TradePrice: buyPrice * 1.3,
				Quantity:   sealed.BuyQuantity,
				PriceRatio: priceRatio,
				URL:        u.String(),
			}
			err = ck.buylist.Add(uuid, out)
			if err != nil {
				ck.printf("%v", err)
			}
		}
	}

	perc := float64(foundProduct) * 100 / float64(len(pricelist))
	ck.printf("Found %d products over %d items (%.02f%%)", foundProduct, len(pricelist), perc)

	ck.inventoryDate = time.Now()
	ck.buylistDate = time.Now()

	return nil
}

func (ck *CardkingdomSealed) Inventory() (mtgban.InventoryRecord, error) {
	if len(ck.inventory) > 0 {
		return ck.inventory, nil
	}

	err := ck.scrape()
	if err != nil {
		return nil, err
	}

	return ck.inventory, nil

}

func (ck *CardkingdomSealed) Buylist() (mtgban.BuylistRecord, error) {
	if len(ck.buylist) > 0 {
		return ck.buylist, nil
	}

	err := ck.scrape()
	if err != nil {
		return nil, err
	}

	return ck.buylist, nil
}

func (ck *CardkingdomSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Card Kingdom"
	info.Shorthand = "CKSealed"
	info.InventoryTimestamp = &ck.inventoryDate
	info.BuylistTimestamp = &ck.buylistDate
	info.SealedMode = true
	return
}
