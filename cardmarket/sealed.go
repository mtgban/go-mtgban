package cardmarket

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgmatcher"
)

type CardMarketSealed struct {
	LogCallback    mtgban.LogCallbackFunc
	MaxConcurrency int
	Affiliate      string

	inventoryDate time.Time
	exchangeRate  float64

	// Used to skip unrelated products
	productList MKMProductList

	// Debug aid, set it to print a full product mapping
	filterEdition string
	skipPrices    bool

	inventory mtgban.InventoryRecord

	client *MKMClient
}

func (mkm *CardMarketSealed) printf(format string, a ...interface{}) {
	if mkm.LogCallback != nil {
		mkm.LogCallback("[MKMSealed] "+format, a...)
	}
}

func NewScraperSealed(appToken, appSecret string) (*CardMarketSealed, error) {
	mkm := CardMarketSealed{}
	mkm.inventory = mtgban.InventoryRecord{}
	mkm.client = NewMKMClient(appToken, appSecret)
	mkm.MaxConcurrency = defaultConcurrency
	rate, err := mtgban.GetExchangeRate("EUR")
	if err != nil {
		return nil, err
	}
	mkm.exchangeRate = rate
	mkm.filterEdition = ""
	mkm.skipPrices = false
	return &mkm, nil
}

var skipSealedEditions = map[string]string{
	"Amonkhet Invocations":           "",
	"Anthologies":                    "",
	"Armada Comics":                  "",
	"Chronicles: Japanese":           "",
	"Coldsnap Theme Decks":           "",
	"Duel Decks: Anthology":          "",
	"Duel Decks: Mind vs. Might":     "",
	"Foreign Black Bordered":         "",
	"Fourth Edition: Black Bordered": "",
	"Guru Lands":                     "",
	"Kaladesh Inventions":            "",
	"Legends Italian":                "",
	"MKM Series 2016":                "",
	"MKM Series 2017":                "",
	"MKM Series 2018":                "",
	"MKM Series":                     "",
	"Magic Game Night":               "",
	"Magic the Gathering Products":   "",
	"Misprints":                      "",
	"Multiverse Gift Box":            "",
	"Mystical Archive":               "",
	"Rinascimento":                   "",
	"Starter 2000":                   "",
	"Summer Magic":                   "",
	"The Dark Italian":               "",
	"The List":                       "",
	"TokyoMTG Products":              "",
	"Ultimate Box Toppers":           "",
	"Welcome Deck 2016":              "",
	"Zendikar Expeditions":           "",
	"Zendikar Rising Expeditions":    "",
	"Introductory Two-Player Set":    "",
	"War of the Spark: Japanese Alternate-Art Planeswalkers": "",
}

var skipSealedProducts = []string{
	"Common Set",
	"Uncommon Set",
	"Rare Set",
	"Mythic Set",
	"Basic Land Set",
	"Dual Land Set",
	"Dual Lands Set",
	"P9 Set",
	"Fetchland Set",
	"Oversized Set",
	"Phenomena Set",
	"Scheme Set",
	"Plane Set",
	"Planes Set",
	"Contraption Set",
	"Timeshifted Set",
	"Four Seasons Set",
	"Masterpiece Set",

	"Player's Guide",
	"Strategy Guide",
	"Official Guide",
	"Comic",
	"Bloodlines",
	"Assasin's Blade",
	"Emperor's Fist",
	"Jedit",
	"Hazezon",
	"Johan",
	"Zendikar: In the Teeth of Akoum",
	"Scars of Mirrodin: The Quest for Karn",
	"Outlaw: Champions of Kamigawa",
	"Heretic: Betrayers of Kamigawa",
	"Guardian: Saviors of Kamigawa",
	"The Shattered Alliance",
	"Champion's Trial",
	"The Eternal Ice",
	"The Fifth Dawn",
	"The Moons of Mirrodin",
	"Ravnica: War of the Spark",
	"Alara Unbroken",
	"The Darksteel Eye",
	"The Thran",
	" #1",
	" #2",
	" #3",
	" #4",
	"Sample Chapter",
	"Game Guide",
	"Rulebook",
	"Storybook",
	"Instructions Booklet",

	"Empty",
	"Venser Deck",
	"Koth Deck",
	"Tibalt Deck",
	"Sorin Deck",

	"Card Box",
	"Storage Box",
	"Flip Box",
	"DeckBox",
	"Deck Vault",
	"Combo Pack",
	"Land Pack",

	"Magic Online Code",
	"Arena Code Card",

	"Collector's Album",
	"Binder",
	"Portfolio",
	"Button",
	"Deckbox",

	"D4 ",
	"D6 ",
	"D8 ",
	"D20 ",
	"Planar Die",
	"Loyalty Dice",
	"Sticker",
	"Divider",
	"Playmat",
	"Pendant",
	"Sample Deck",
	"Collectors Coin",
	"Flashlight Keychain",
	"Counter",
	"Sheet",
	"Token",
	"Acorn Stash",
	"Giant",
	" Pin",
	"T-Shirt",
	"Lifepad",
	"Art Print",
	"Six Card",
	"Poster",
	"Sleeves",
	"Warstorm Surge",
}

func (mkm *CardMarketSealed) processEdition(channel chan<- responseChan, pair *MKMExpansionIdPair) error {
	productsInEdition, err := mkm.client.MKMProductsInExpansion(pair.IdExpansion)
	if err != nil {
		return err
	}

	sealedProduct := []int{}
	sealedNames := []string{}
	for _, productItem := range mkm.productList {
		if productItem.ExpansionId != pair.IdExpansion {
			continue
		}
		skip := false

		// Skip all singles (this API returns only singles, while we have all products,
		// so we drop whatever is not a sealed product)
		for _, product := range productsInEdition {
			if product.IdProduct == productItem.IdProduct {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Skip generic random terms found in sealed products
		for _, tag := range skipSealedProducts {
			if strings.Contains(strings.ToLower(productItem.Name), strings.ToLower(tag)) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		// Skip even more generic terms that might conflict with actual terms
		switch {
		// Skip Books
		case productItem.Name == pair.Name &&
			!(strings.Contains(productItem.Name, "Mythic Edition") ||
				strings.Contains(productItem.Name, "Game Night") ||
				productItem.Name == "Unsanctioned" ||
				productItem.Name == "Modern Event Deck 2014"):
			continue
		// Skip plastic deckboxes (but not "starter deck box")
		case strings.Contains(productItem.Name, "Deck Box") &&
			!(strings.Contains(productItem.Name, "Starter") ||
				strings.Contains(productItem.Name, "Theme")):
			continue
		// Skip "Complete Set" except for some sets where "full set" means "sealed"
		case strings.Contains(productItem.Name, "Full Set") &&
			!(strings.Contains(productItem.Name, "Sealed") ||
				strings.Contains(productItem.Name, "From the Vault") ||
				strings.Contains(productItem.Name, "Commander's Arsenal") ||
				strings.Contains(productItem.Name, "Commander Anthology") ||
				strings.Contains(productItem.Name, "Ponies: The Galloping") ||
				strings.Contains(productItem.Name, "Signature Spellbook") ||
				strings.Contains(productItem.Name, "Global Series") ||
				strings.Contains(productItem.Name, "Premium Deck Series") ||
				strings.Contains(productItem.Name, "Duel Decks")):
			continue
		case strings.Contains(productItem.Name, "Duel Decks") &&
			strings.Contains(productItem.Name, "Display (6 Decks)"):
			continue
		case strings.HasSuffix(productItem.Name, "Emblem"),
			strings.HasSuffix(productItem.Name, "Statue"):
			continue
		}

		sealedProduct = append(sealedProduct, productItem.IdProduct)
		sealedNames = append(sealedNames, productItem.Name)
	}

	if len(sealedProduct) == 0 {
		mkm.printf("No sealed products in %s", pair.Name)
	}
	for i, productId := range sealedProduct {
		uuid, err := mkm.processUUID(sealedNames[i], pair.Name)
		if err != nil {
			mkm.printf("%s", err.Error())
			continue
		} else if uuid == "" {
			// Silent skip
			continue
		}

		// Debug aid
		if mkm.skipPrices {
			channel <- responseChan{
				cardId: uuid,
			}
			continue
		}

		err = mkm.processProduct(channel, productId, uuid)
		if err != nil {
			mkm.printf("%s", err.Error())
			continue
		}
	}
	return nil
}

var skips = map[string]string{
	// Some FTVs are marked as (Sealed) and some as (Full Set),
	// but some (Full Set) refer to the actual cards
	"From the Vault: Angels: Full Set":       "",
	"From the Vault: Dragons: Full Set":      "",
	"From the Vault: Exiled: Full Set":       "",
	"From the Vault: Annihilation: Full Set": "",
	"From the Vault: Legends: Full Set":      "",
	"From the Vault: Realms: Full Set":       "",
	"From the Vault: Relics: Full Set":       "",
	"From the Vault: Twenty: Full Set":       "",

	// Empty
	"Commander Anthology Box":    "",
	"Commander Anthology II Box": "",

	// Missing upstream
	"Renaissance Booster Pack":                                    "",
	"Renaissance Booster Box":                                     "",
	"Eighth Edition Box Set":                                      "",
	"Eighth Edition: Demo Game Booster":                           "",
	"Eighth Edition Advanced Booster":                             "",
	"Ninth Edition: Fast Track 2 Player Starter Set":              "",
	"Ninth Edition Advanced Booster":                              "",
	"Ninth Edition Sampler Booster":                               "",
	"Tenth Edition: 2 Player Starter Set (Blue)":                  "",
	"Tenth Edition: 2 Player Starter Set (Black)":                 "",
	"Tenth Edition: 2 Player Starter Set (Green)":                 "",
	"Tenth Edition: 2 Player Starter Set (Red)":                   "",
	"Tenth Edition: 2 Player Starter Set (White)":                 "",
	"Magic 2012: Grab for Power Intro Pack":                       "",
	"Magic 2012 Intro Pack Complete of 5":                         "",
	"Magic 2012: Sacred Assault Intro Pack":                       "",
	"Magic 2012: Entangling Webs Intro Pack":                      "",
	"Magic 2012: Mystical Might Intro Pack":                       "",
	"Magic 2012: Blood and Fire Intro Pack":                       "",
	"Core 2020: Spellslinger Starter Kit":                         "",
	"Grixis Shambling Army Intro Pack":                            "",
	"Naya Domain Intro Pack":                                      "",
	"Esper Air Assault Intro Pack":                                "",
	"Bant on the March Intro Pack":                                "",
	"Global Series Jiang Yanggu & Mu Yanling: Full Set (Chinese)": "",
	"Jund Appetite for War Intro Pack":                            "",
	"Seventh Edition: 2 Player Starter Set":                       "",
	"Seventh Edition Advanced Booster":                            "",
	"Morningtide: Warrior's Code Theme Deck":                      "",
	"Morningtide: Shamanism Theme Deck":                           "",
	"Morningtide: Battalion Theme Deck":                           "",
	"Morningtide: Going Rogue Theme Deck":                         "",
	"Portal Three Kingdoms: Wei Kingdom Theme Deck":               "",
	"Portal Three Kingdoms: Shu Kingdom Theme Deck":               "",
	"Portal Three Kingdoms: Wu Kingdom Theme Deck":                "",
	"Kaladesh Holiday Buy a Box Booster":                          "",
	"Torment: Insanity Theme Deck":                                "",
	"Grixis Undead Intro Pack":                                    "",
	"Esper Artifice Intro Pack":                                   "",
	"Naya Behemoths Intro Pack":                                   "",
	"Bant Exalted Intro Pack":                                     "",
	"Primordial Jund Intro Pack":                                  "",
	"Visions: Legion of Glory Theme Deck":                         "",
	"Visions: Wild-Eyed Frenzy Theme Deck":                        "",
	"Visions: Savage Stompdown Theme Deck":                        "",
	"Visions: Unnatural Forces Theme Deck":                        "",
	"Mirrodin Besieged Event Deck Set":                            "",
	"Zendikar Rising Welcome Booster":                             "",
	"Ravnica Allegiance Collector Booster Box":                    "",
	"Archenemy: All 4 Decks":                                      "",
	"Signature Spellbook: Gideon: Full Set (Booster)":             "",
	"Signature Spellbook: Jace: Full Set (Booster)":               "",
	"Signature Spellbook: Chandra: Full Set (Booster)":            "",
	"Unlimited: Starter Deck Box":                                 "",
	"Ravnica Allegiance: Mythic Edition":                          "",
	"Gatecrash: Event Deck Set":                                   "",
	"Avacyn Restored Intro Pack Set of 5":                         "",
	"Innistrad Intro Pack Set of 5":                               "",
	"War of the Spark: Mythic Edition":                            "",
	"Archenemy: Nicol Bolas Box":                                  "",
}

var renames = map[string]string{
	// Typo
	"Battle for Zendikar:\"Eldrazi Assault\" Intro Pack (Red)": "Battle for Zendikar: \"Eldrazi Assault\" Intro Pack (Red)",

	"Eighth Edition: 2 Player Starter Set":     "Box Set Core Game",
	"Ninth Edition: Core 2 Player Starter Set": "9th Edition Boxed Set Core Game",
	"Portal Second Age: 2 Player Starter Set":  "Portal Second Age 2 Player Start",

	"Mystery Booster Box":                 "Mystery Booster Booster Box Retail Exclusive",
	"Mystery Booster: Convention Edition": "Mystery Booster Pack Convention Edition",
	"Mystery Booster":                     "Mystery Booster Pack Retail Exclusive",

	// Intro Packs due to "All $x" instead of "Set of $x"
	"Magic 2012 Intro Pack Set of 5":        "Magic 2012 M12 All 5 Intro Packs",
	"Magic 2013 Intro Pack Set of 5":        "Magic 2013 M13 All 5 Intro Packs",
	"Mirrodin Besieged Intro Pack Set of 4": "Mirrodin Besieged All 4 Intro Packs",
	"New Phyrexia Intro Pack Set of 5":      "New Phyrexia All 5 Intro Packs",
	"Theros Intro Pack Set":                 "Theros All 5 Intro Packs",
	"Dark Ascension Intro Pack Set of 5":    "Dark Ascension All 5 Intro Packs",

	"Duel Decks: Jace vs. Chandra Full Set":          "Duel Decks Jace vs Chandra Box Set",
	"Dragon's Maze: Strength of Selesnya Event Deck": "Dragon's Maze Event Deck",

	// Commander renames
	"Commander: Ikoria Deck Set":          "Commander 2020 Set of 5",
	"Commander: Zendikar Rising Deck Set": "Zendikar Rising Commander Deck Set of 2",
	"Commander Legends Deck Set":          "Commander Legends Commander Deck Set of 2",
	"Commander: Kaldheim Deck Set":        "Kaldheim Commander Deck Set of 2",
	"Commander: Strixhaven Deck Set":      "Commander 2021 Set of 5",

	"Guilds of Ravnica: Guild Kit Set":   "Guilds of Ravnica Guild Kits Set of 5",
	"Ravnica Allegiance: Guild Kits Set": "Ravnica Allegiance Guild Kit Set of 5",

	"Zendikar Rising Holiday Gift Fat Pack Bundle": "Zendikar Rising Bundle Gift Edition",
	"Throne of Eldraine Holiday Gift Bundle":       "Throne of Eldraine Bundle Gift Edition",

	// Duels of the Planeswalkers
	"Ears of the Elves Deck":     "DotP Ears of the Elves Nissa Revane Deck",
	"Thoughts of the Wind Deck":  "DotP Thoughts of the Wind Jace Beleren Deck",
	"Hands of Flame Deck":        "DotP Hands of Flame Chandra Nalaar Deck",
	"Eyes of Shadow Deck":        "DotP Eyes of Shadow Liliana Vess Deck",
	"Teeth of the Predator Deck": "DotP Teeth of the Predator Garruk Wildspeaker Deck",

	"Commander Collection: Green: Premium Edition": "Commander Collection Green Premium",

	// SLD stuff
	"Secret Lair: Ultimate Edition 2 (Gray Box)":        "Secret Lair Ultimate Edition 2 Box Grey",
	"Secret Lair: Ultimate Edition 2 (Black Box)":       "Secret Lair Ultimate Edition 2 Box Black",
	"Secret Lair Drop Series Bundle":                    "Secret Lair Drop Series Full Bundle",
	"Secret Lair Drop Series: Theros Stargazing Bundle": "Secret Lair Drop Theros Stargazing Bundle Vol I V",
	"Secret Lair Drop Series: Summer Superdrop Bundle":  "Secret Lair Drop Summer Superdrop Bundle",
	"Secret Lair Drop Series: Secretversary Superdrop":  "Secret Lair Drop Secretversary Superdrop The Bundle Bundle",

	"Secret Lair Drop Series: Theros Stargazing: Vol. I":  "Secret Lair Drop Theros Stargazing VolI Heliod",
	"Secret Lair Drop Series: Theros Stargazing: Vol. II": "Secret Lair Drop Theros Stargazing VolII Thassa",

	"Secret Lair Drop Series: Showcase: Kaldheim – Part 1": "Secret Lair Drop Showcase Kaldheim Part 1 Non Foil",
	"Secret Lair Drop Series: Showcase: Kaldheim – Part 2": "Secret Lair Drop Showcase Kaldheim Part 2 Non Foil",

	"Secret Lair Drop Series: Wizards of the Coast Presents: After Great Deliberation, We Have Compiled and Remastered the Greatest Magic: The Gathering Cards of All Time, Ever": "Secret Lair Drop April Fools",

	"Secret Lair Drop Series: The Unfathomable Crushing Brutality of Basic Lands": "Secret Lair Drop The Unfathomable Crushing Brutality of Basic Lands Non Foil",
}

var deckVariants = map[string]string{
	// Typos
	"Artful Destruction": "Artufl Destruction",
	"Devastation":        "Devestation",
	"Rumbler":            "Rambler",
	// THR
	"Favors from Nyx":            "Celestial Archon",
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
	// JOU (below due to too many decks using wubrg)
	// FRF
	"Unflinching Assault":      "Mardu",
	"Cunning Plan":             "Jeskai",
	"Grave Advantage":          "Sultai",
	"Stampeding Hordes":        "Abzan",
	"Surprise Attack":          "Temur",
	"Battle with Speed":        "Mardu",
	"Battle with Cunning":      "Jeskai",
	"Battle with Ruthlessness": "Sultai",
	"Battle with Endurance":    "Abzan",
	"Battle with Savagery":     "Temur",
	// DTK (below due to conflict with FRF)
	// ORI
	"Demonic Deals":    "Black",
	"Take to the Sky":  "Blue",
	"Hunting Pack":     "Green",
	"Assemble Victory": "Red",
	"Brave the Battle": "White",
	// BFZ
	"Ultimate Sacrifice": "Event Deck",
	"Eldrazi Assault":    "Red",
	"Swarming Instinct":  "Blue",
	"Call of Blood":      "Black",
	"Zendikar's Rage":    "Green",
	"Rallying Cry":       "White",
	// M15
	"Infernal Intervention":  "Black",
	"Hit The Ground Running": "Blue",
	"Will Of The Masses":     "Green",
	"Flames of the Dragon":   "Red",
	"Price of Glory":         "White",
}

func (mkm *CardMarketSealed) processUUID(name, edition string) (string, error) {
	ogname := name
	_, found := skips[name]
	if found {
		return "", nil
	}

	fixup, found := renames[name]
	if found {
		name = fixup
	} else {
		if !strings.Contains(name, "Set of") {
			name = strings.Replace(name, "Prerelease Pack Set", "Prerelease Pack Complete", 1)
			name = strings.Replace(name, "Intro Pack Set", "Intro Pack Complete", 1)
			name = strings.Replace(name, "Event Deck Set", "Event Deck Complete", 1)
			// A Box is 2x Sets
			if strings.HasSuffix(name, "Intro Pack Box") {
				return "", nil
			}
		}
		if strings.Contains(name, "Commander") {
			name = strings.Replace(name, "Deck Set", "complete", 1)
			name = mtgmatcher.SplitVariants(name)[0]
			name = strings.TrimSuffix(name, " Deck")
		}
		name = strings.Replace(name, "2-Player Clash Pack", "Clash Pack", 1)
		name = strings.Replace(name, "Two-Player Clash Pack", "Clash Pack", 1)
		name = strings.Replace(name, "Fat Pack Bundle", "Bundle", 1)
		if strings.HasSuffix(name, "Booster") {
			name += " Pack"
		}
		name = strings.Replace(name, "/", "", -1)
	}

	switch edition {
	case "Alpha",
		"Beta",
		"Unlimited",
		"Revised":
		name = strings.Replace(name, edition, edition+" Edition", 1)
		edition += " Edition"
	case "Sixth Edition":
		edition = "Classic Sixth Edition"
	case "Duels of the Planeswalkers Decks":
		edition = "Duels of the Planeswalkers"
	case "Mirrodin Besieged":
		name = strings.Replace(name, "Faction Pack", "Faction Booster Pack", 1)
	case "Return to Ravnica":
		if strings.Contains(name, "Guild Booster") {
			return "", nil
		}
		name = strings.Replace(name, "Guild Pack", "Prerelease Pack", 1)
	case "Dragon's Maze",
		"Gatecrash":
		name = strings.Replace(name, "Guild Pack", "Prerelease Pack", 1)
	case "Core 2019",
		"Core 2020",
		"Core 2021":
		edition = strings.Replace(edition, "Core", "Core Set", 1)
		name = strings.Replace(name, "Core", "Core Set", 1)
		if strings.Contains(name, "Welcome") {
			return "", nil
		}
	case "Commander Anthology",
		"Commander Anthology II":
		if !strings.HasSuffix(name, "Full Set") {
			return "", nil
		}
	case "Mystery Booster":
		edition = "Mystery Booster Retail Edition Foils"
		if strings.Contains(name, "Convention") {
			edition = "Mystery Booster Playtest Cards"
		}
	case "Secret Lair Drop Series":
		edition = "Secret Lair Drop"
		name = strings.Replace(name, " Series", "", 1)
		// Skip the full set of singles
		if strings.HasSuffix(name, " Set") {
			return "", nil
		}
	case "Planechase Anthology",
		"Explorers of Ixalan",
		"Welcome Deck 2017":
		if !strings.Contains(name, "Full Set") {
			return "", nil
		}
	case "Unsanctioned":
		if name != "Unsactioned" {
			return "", nil
		}
	case "Starter 1999":
		if !strings.Contains(name, "Booster Pack") {
			return "", nil
		}
	case "Magic 2013":
		if name == "Deck Builder's Toolkit (Magic 2012)" {
			edition = "Magic 2012"
		}
	default:
		if strings.Contains(name, "Duel Decks") && !strings.Contains(name, "Full Set") {
			return "", nil
		}
	}

	if strings.HasPrefix(name, "Deck Builder's Toolkit") {
		eds := mtgmatcher.SplitVariants(name)
		if len(eds) > 1 {
			name = eds[1] + " " + "Deck Builders Toolkit"
		}
	}

	set, err := mtgmatcher.GetSetByName(edition)
	if err != nil {
		return "", fmt.Errorf("edition %s not found - skipping %s", edition, name)
	}

	var uuid string
	if uuid == "" {
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

	// Most Deck Boxes are missing upstream
	if strings.HasSuffix(name, "Deck Box") {
		return "", nil
	}

	if uuid == "" {
		// Make sure edition is present from now on, and separated with a colon
		if !strings.Contains(name, edition) {
			name = edition + ": " + name
			// Make sure the colon separates the edition
		} else if strings.Contains(name, edition) && !strings.Contains(name, ": ") {
			name = strings.Replace(name, edition, edition+": ", 1)
		}
	}

	if uuid == "" {
		var deckType string
		// Keep the name of planewalker only
		for _, deckType = range []string{
			"Planeswalker Deck",
			"Intro Pack",
			"Prerelease Pack",
			"Challenge Deck",
			"Event Deck",
			"Brawl Deck",
			"Theme Deck",
		} {
			if strings.Contains(name, deckType) && !strings.HasSuffix(name, "Box") {
				break
			}
		}

		for _, tag := range []string{": ", " \"", " ("} {
			if strings.Contains(name, tag) {
				probe := ""
				subtypes := strings.Split(name, tag)
				if len(subtypes) == 1 {
					continue
				}

				// Always pick the last element available
				probe = subtypes[len(subtypes)-1]
				// Skip elements that could be confused with the edition
				// example "Commander: Zendikar Rising: \"Land's Wrath\""
				if strings.Contains(edition, probe) {
					continue
				}

				probe = strings.Replace(probe, deckType, "", 1)

				// Keep the name of planewalker only
				if strings.Contains(name, "Planeswalker Deck") {
					if strings.Contains(name, ",") {
						probe = strings.Split(probe, ",")[0]
					} else {
						probe = strings.Fields(probe)[0]
					}
				}

				probe = strings.Replace(probe, "\"", "", -1)
				probe = strings.TrimSpace(probe)
				probe = mtgmatcher.SplitVariants(probe)[0]
				probe = strings.TrimSpace(probe)

				switch set.Code {
				case "JOU":
					fixup, found := map[string]string{
						// Green is fine
						"Black": "Pantheons Power",
						"Red":   "Voracious Rage",
						"Blue":  "Fates Foreseen",
						"White": "Mortals of Myth",
					}[probe]
					if found {
						probe = fixup
					}
				case "DTK":
					fixup, found := map[string]string{
						"GW":                       "Dromoka",
						"WU":                       "Ojutai",
						"UB":                       "Silumgar",
						"BR":                       "Kolaghan",
						"RG":                       "Atarka",
						"Battle with Endurance":    "Dromoka",
						"Battle with Cunning":      "Ojutai",
						"Battle with Ruthlessness": "Silumgar",
						"Battle with Speed":        "Kolaghan",
						"Battle with Savagery":     "Atarka",
					}[probe]
					if found {
						probe = fixup
					}
				default:
					fixup, found := deckVariants[probe]
					if found {
						probe = fixup
					}
				}

				if probe != "" {
					for _, product := range set.SealedProduct {
						if mtgmatcher.SealedContains(product.Name, probe) {
							// One additional check to avoid aliasing sets with recurring themes
							// such as RTR and DGM using the guids names for both deck types
							if (strings.Contains(product.Name, "Intro") && deckType != "Intro Pack") ||
								(strings.Contains(product.Name, "Theme") && deckType != "Theme Deck") ||
								(strings.Contains(product.Name, "Prerelease") && deckType != "Prerelease Pack") {
								continue
							}
							uuid = product.UUID
							break
						}
					}
				}

				if uuid != "" {
					break
				}
			}
		}
	}

	if uuid == "" {
		// Don't return errors for known missing products
		if strings.HasSuffix(name, "Complete") ||
			strings.HasSuffix(name, "Pack Box") ||
			strings.HasSuffix(name, "Theme Deck Box") ||
			strings.HasSuffix(name, "Gift Pack") ||
			strings.HasSuffix(name, "Set of 5") ||
			strings.HasSuffix(name, "Gift Box") {
			return "", nil
		}
		return "", fmt.Errorf("%s was found (%s) // %s was not found", edition, set.Code, ogname)
	}

	if mkm.filterEdition != "" {
		co, _ := mtgmatcher.GetUUID(uuid)
		mkm.printf("%s -> %s", ogname, co.Name)
	}
	return uuid, nil
}

func (mkm *CardMarketSealed) processProduct(channel chan<- responseChan, idProduct int, uuid string) error {
	anyLang := false
	articles, err := mkm.client.MKMArticles(idProduct, anyLang)
	if err != nil {
		return err
	}

	if len(articles) == 0 {
		return nil
	}

	u, err := url.Parse("https://www.cardmarket.com/en/Magic/Products/Search")
	if err != nil {
		return err
	}

	for _, article := range articles {
		if article.Price == 0 {
			return nil
		}

		// Filter by language (the search option in the API seems to have no effect)
		if article.Language.LanguageName != "English" {
			continue
		}

		// Skip all the silly non-really-sealed listings
		if mtgmatcher.Contains(article.Comments, "empty") ||
			mtgmatcher.Contains(article.Comments, "only the deck") ||
			mtgmatcher.Contains(article.Comments, "only 60 cards") ||
			mtgmatcher.Contains(article.Comments, "deck only") ||
			mtgmatcher.Contains(article.Comments, "cards only") ||
			mtgmatcher.Contains(article.Comments, "only cards") ||
			mtgmatcher.Contains(article.Comments, "all cards sleeved") ||
			mtgmatcher.Contains(article.Comments, "just the") ||
			mtgmatcher.Contains(article.Comments, "not sealed") ||
			mtgmatcher.Contains(article.Comments, "open") ||
			mtgmatcher.Contains(article.Comments, "used") ||
			mtgmatcher.Contains(article.Comments, "sampler") ||
			mtgmatcher.Contains(article.Comments, "ouvert") ||
			mtgmatcher.Contains(article.Comments, "abierto") ||
			mtgmatcher.Contains(article.Comments, "without") ||
			mtgmatcher.Contains(article.Comments, "missing") ||
			mtgmatcher.Contains(article.Comments, "just") ||
			mtgmatcher.Contains(article.Comments, "damaged") ||
			mtgmatcher.Contains(article.Comments, "no box") {
			continue
		}

		v := url.Values{}
		v.Set("searchString", article.Product.Name)
		if mkm.Affiliate != "" {
			v.Set("utm_source", mkm.Affiliate)
			v.Set("utm_medium", "text")
			v.Set("utm_campaign", "card_prices")
		}
		v.Set("language", "1")
		u.RawQuery = v.Encode()

		out := responseChan{
			cardId: uuid,
			entry: mtgban.InventoryEntry{
				Conditions: "NM",
				Price:      article.Price * mkm.exchangeRate,
				Quantity:   article.Count,
				SellerName: article.Seller.Username,
				URL:        u.String(),
				OriginalId: fmt.Sprint(article.IdProduct),
				InstanceId: fmt.Sprint(article.IdArticle),
			},
		}

		channel <- out

		// Only keep the first price found
		break
	}

	return nil
}

func (mkm *CardMarketSealed) scrapeAll() error {
	editionList, err := mkm.client.ListExpansionIds()
	if err != nil {
		return err
	}
	mkm.printf("Parsing %d editions", len(editionList))

	expansions := make(chan MKMExpansionIdPair)
	channel := make(chan responseChan)
	var wg sync.WaitGroup

	for i := 0; i < mkm.MaxConcurrency; i++ {
		wg.Add(1)
		go func() {
			for expansion := range expansions {
				err := mkm.processEdition(channel, &expansion)
				if err != nil {
					mkm.printf("expansion id %d returned %s", expansion.IdExpansion, err)
				}
			}
			wg.Done()
		}()
	}

	go func() {
		lenEds := len(editionList)
		for i, pair := range editionList {
			if mkm.filterEdition != "" && !mtgmatcher.Contains(pair.Name, mkm.filterEdition) {
				continue
			}
			// Skip editions that do not have sealed product associated to them
			_, found := skipSealedEditions[pair.Name]
			if found ||
				strings.HasSuffix(pair.Name, "Promos") ||
				strings.HasSuffix(pair.Name, "Extras") ||
				strings.HasPrefix(pair.Name, "Pro Tour") ||
				strings.HasPrefix(pair.Name, "Euro Lands") ||
				strings.HasPrefix(pair.Name, "APAC Lands") ||
				strings.HasPrefix(pair.Name, "WCD") {
				continue
			}
			mkm.printf("Processing id %d - %s (%d/%d)", pair.IdExpansion, pair.Name, i+1, lenEds)
			expansions <- pair
		}
		close(expansions)

		wg.Wait()
		close(channel)
	}()

	for result := range channel {
		res, found := mkm.inventory[result.cardId]
		if found {
			co, _ := mtgmatcher.GetUUID(result.cardId)
			mkm.printf("%s already present\n-old: %s\n-new: %s", co, res[0], result.entry)
			continue
		}
		err := mkm.inventory.Add(result.cardId, &result.entry)
		if err != nil {
			_, cerr := mtgmatcher.GetUUID(result.cardId)
			if cerr != nil {
				mkm.printf("%s - %s: %s", result.entry.OriginalId, cerr.Error(), result.cardId)
				continue
			}
			mkm.printf("%s - %s", result.ogId, err.Error())
			continue
		}
	}

	mkm.printf("Total number of requests: %d", mkm.client.RequestNo())
	mkm.printf("Total number of products found: %d", len(mkm.inventory))
	mkm.inventoryDate = time.Now()
	return nil
}

func (mkm *CardMarketSealed) scrape() error {
	productList, err := mkm.client.MKMProductList()
	if err != nil {
		return err
	}
	mkm.productList = productList
	mkm.printf("Loading %d products", len(productList))

	mkm.printf("Retrieving every single sealed item")
	return mkm.scrapeAll()
}

func (mkm *CardMarketSealed) Inventory() (mtgban.InventoryRecord, error) {
	if len(mkm.inventory) > 0 {
		return mkm.inventory, nil
	}

	err := mkm.scrape()
	if err != nil {
		return nil, err
	}

	return mkm.inventory, nil
}

func (mkm *CardMarketSealed) Info() (info mtgban.ScraperInfo) {
	info.Name = "Cardmarket"
	info.Shorthand = "MKMSealed"
	info.CountryFlag = "EU"
	info.InventoryTimestamp = &mkm.inventoryDate
	info.SealedMode = true
	return
}
