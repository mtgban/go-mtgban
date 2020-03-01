package miniaturemarket

import (
	"strconv"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgban"
	"github.com/kodabb/go-mtgban/mtgjson"
)

var promosetTable = map[string]string{
	"APAC Land":                "Asia Pacific Land Program",
	"Convention":               "URL/Convention Promos",
	"DCI Legend Membership":    "DCI Legend Membership",
	"Grand Prix":               "Grand Prix Promos",
	"Guru Land":                "Guru",
	"HASCON":                   "HasCon 2017",
	"Happy Holidays":           "Happy Holidays",
	"Junior Super Series":      "Junior Super Series",
	"MTG 15th Anniversary":     "15th Anniversary Cards",
	"Magic Scholarship Series": "Junior Super Series",
	"Mythic Championship":      "Pro Tour Promos",
	"Nationals":                "Nationals Promos",
	"Pones: The Galloping":     "Ponies: The Galloping",
	"Pro Tour":                 "Pro Tour Promos",
	"Repack Insert":            "Resale Promos",
	"Regional PTQ":             "Pro Tour Promos",
	"Standard Series":          "BFZ Standard Series",
	"Tarkir Dragonfury":        "Tarkir Dragonfury",
	"Two-Headed Giant":         "Two-Headed Giant Tournament",
	"Ugin's Fate":              "Ugin's Fate",
	"World Championship":       "World Championship Promos",
	"World Magic Cup":          "World Magic Cup Qualifiers",
}

var card2setTable = map[string]string{
	"Fling (WPN) (#50)":    "Wizards Play Network 2010",
	"Fling (WPN) (#69)":    "Wizards Play Network 2011",
	"Serra Angel (DCI)":    "Wizards of the Coast Online Store",
	"Sol Ring (Commander)": "MagicFest 2019",

	"Ajani Vengeant (Launch Party)":        "Prerelease Events",
	"Dauntless Dourbark (Champs / States)": "Gateway 2007",
	"Stocking Tiger (Repack Insert)":       "Happy Holidays",
	"Atarka, World Render (Repack Insert)": "Resale Promos",
	"Rukh Egg (MTG 10th Anniversary)":      "Release Events",
	"Ass Whuppin' (Pre-Release)":           "Release Events",
	"Faerie Conclave (WPN)":                "Summer of Magic",
	"Treetop Village (WPN)":                "Summer of Magic",

	"Celestine Reef (Pre-Release)":             "Promotional Planes",
	"Horizon Boughs (WPN)":                     "Promotional Planes",
	"Mirrored Depths (WPN)":                    "Promotional Planes",
	"Tember City (WPN)":                        "Promotional Planes",
	"Stairs to Infinity (Launch Party)":        "Promotional Planes",
	"Tazeem (Launch Party)":                    "Promotional Planes",
	"Drench the Soil in Their Blood (WPN)":     "Promotional Schemes",
	"Imprison This Insolent Wretch (WPN)":      "Promotional Schemes",
	"Perhaps You've Met My Cohort (WPN)":       "Promotional Schemes",
	"Plots That Span Centuries (Launch Party)": "Promotional Schemes",
	"Your Inescapable Doom (WPN)":              "Promotional Schemes",

	"Arguel's Blood Fast / Temple of Aclazotz (Buy-a-Box)":              "XLN Treasure Chest",
	"Conqueror's Galleon / Conqueror's Foothold (Buy-a-Box)":            "XLN Treasure Chest",
	"Dowsing Dagger / Lost Vale (Buy-a-Box)":                            "XLN Treasure Chest",
	"Growing Rites of Itlimoc / Itlimoc, Cradle of the Sun (Buy-a-Box)": "XLN Treasure Chest",
	"Legion's Landing / Adanto, the First Fort (Buy-a-Box)":             "XLN Treasure Chest",
	"Primal Amulet / Primal Wellspring (Buy-a-Box)":                     "XLN Treasure Chest",
	"Search for Azcanta / Azcanta, the Sunken Ruin (Buy-a-Box)":         "XLN Treasure Chest",
	"Thaumatic Compass / Spires of Orazca (Buy-a-Box)":                  "XLN Treasure Chest",
	"Treasure Map / Treasure Cove (Buy-a-Box)":                          "XLN Treasure Chest",
	"Vance's Blasting Cannons / Spitfire Bastion (Buy-a-Box)":           "XLN Treasure Chest",

	"Blood Knight (Champs / States)":             "Champs and States",
	"Bramblewood Paragon (Champs / States)":      "Champs and States",
	"Doran, the Siege Tower (Champs / States)":   "Champs and States",
	"Electrolyze (Champs / States)":              "Champs and States",
	"Groundbreaker (Champs / States)":            "Champs and States",
	"Imperious Perfect (Champs / States)":        "Champs and States",
	"Mutavault (Champs / States)":                "Champs and States",
	"Niv-Mizzet, the Firemind (Champs / States)": "Champs and States",
	"Rakdos Guildmage (Champs / States)":         "Champs and States",
	"Serra Avenger (Champs / States)":            "Champs and States",
	"Urza's Factory (Champs / States)":           "Champs and States",
	"Voidslime (Champs / States)":                "Champs and States",

	"Sultai Charm (Gift Box)":                         "Khans of Tarkir Promos",
	"Scythe Leopard (Gift Box)":                       "Battle for Zendikar Promos",
	"Ravenous Bloodseeker (Gift Box)":                 "Shadows over Innistrad Promos",
	"Dreg Mangler (Gift Box)":                         "Return to Ravnica Promos",
	"Chief of the Foundry (Gift Box)":                 "Kaladesh Promos",
	"Deeproot Champion (Convention)":                  "Ixalan Promos",
	"Karametra's Acolyte (Gift Box)":                  "Theros Promos",
	"Sorcerous Spyglass (M-660-012-1NM)":              "Ixalan Promos",
	"Sorcerous Spyglass (M-660-012-3F)":               "Ixalan Promos",
	"Sorcerous Spyglass (Pre-Release) (M-650-124-3F)": "Ixalan Promos",
	"Death Baron (Convention)":                        "Core Set 2019 Promos",
	"Astral Drift (Pre-Release)":                      "Modern Horizons Promos",
	"Nightpack Ambusher (Convention)":                 "Core Set 2020 Promos",
	"Sorcerous Spyglass (M-660-016-1NM)":              "Throne of Eldraine Promos",
	"Sorcerous Spyglass (M-660-016-3F)":               "Throne of Eldraine Promos",
	"Sorcerous Spyglass (Pre-Release) (M-650-176-3F)": "Throne of Eldraine Promos",

	"Firesong and Sunspeaker (Buy-a-Box)":        "Dominaria",
	"Nexus of Fate (Buy-a-Box)":                  "Core Set 2019",
	"Flusterstorm (Buy-a-Box)":                   "Modern Horizons",
	"Impervious Greatwurm (Buy-a-Box)":           "Guilds of Ravnica",
	"The Haunt of Hightower (Buy-a-Box)":         "Ravnica Allegiance",
	"Tezzeret, Master of the Bridge (Buy-a-Box)": "War of the Spark",
	"Rienne, Angel of Rebirth (Buy-a-Box)":       "Core Set 2020",
	"Kenrith, the Returned King (Buy-a-Box)":     "Throne of Eldraine",
	"Athreos, Shroud-Veiled (Buy-a-Box)":         "Theros Beyond Death",

	"Rhox (Alternate Art Foil)":                    "Starter 2000",
	"Ertai, the Corrupted (Alternate Art Foil)":    "Planeshift",
	"Skyship Weatherlight (Alternate Art Foil)":    "Planeshift",
	"Tahngarth, Talruum Hero (Alternate Art Foil)": "Planeshift",

	"Demonic Tutor (Judge Rewards) (Anna Steinbauer)": "Judge Gift Cards 2020",
	"Demonic Tutor (Judge Rewards) (Daarken)":         "Judge Gift Cards 2008",
	"Vampiric Tutor (Judge Rewards) (Gary Leach)":     "Judge Gift Cards 2000",
	"Vampiric Tutor (Judge Rewards) (Lucas Graciano)": "Judge Gift Cards 2018",
	"Vindicate (Judge Rewards) (Karla Ortiz)":         "Judge Gift Cards 2013",
	"Vindicate (Judge Rewards) (Mark Zug)":            "Judge Gift Cards 2007",
	"Wasteland (Judge Rewards) (Carl Critchlow)":      "Judge Gift Cards 2010",
	"Wasteland (Judge Rewards) (Steve Belledin)":      "Judge Gift Cards 2015",
}

var card2numberTable = map[string]string{
	"B.F.M. (Big Furry Monster) (Left Side)":  "28",
	"B.F.M. (Big Furry Monster) (Right Side)": "29",

	"Stocking Tiger (Repack Insert)": "13†",
}

func (mm *Miniaturemarket) parseSet(c *mmCard) (setName string, setCheck mtgban.SetCheckFunc) {
	// Function to determine whether we're parsing the correct set
	setCheck = func(set mtgjson.Set) bool {
		return set.Name == setName
	}

	setName = c.Edition

	variants := mtgban.SplitVariants(c.Name)
	specifier := ""
	if len(variants) > 1 {
		specifier = variants[1]
	}
	cardName := variants[0]

	ed, found := mtgban.EditionTable[setName]
	if found {
		setName = ed
		return
	}

	ed, found = card2setTable[c.Name]
	if found {
		setName = ed
		return
	}

	ed, found = promosetTable[specifier]
	if found {
		setName = ed
		return
	}

	switch setName {
	case "Guild Kit":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasSuffix(set.Name, "Guild Kit")
		}
	case "Duel Decks: Anthology":
		if specifier != "" {
			setName = "Duel Decks Anthology: " + specifier
			setName = strings.Replace(setName, " vs ", " vs. ", 1)
			return
		}
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, "Duel Decks Anthology")
		}
	case "Promo Pack":
		switch cardName {
		case "Plains", "Island", "Swamp", "Mountain", "Forest":
			setName = "M20 Promo Packs"
		default:
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasSuffix(set.Name, "Promos") || set.Type == "expansion"
			}
		}
	case "Archenemy", "Archenemy: Nicol Bolas":
		setCheck = func(set mtgjson.Set) bool {
			return set.Name == setName || set.Name == setName+" Schemes"
		}
	case "Mystery Booster", "Secret Lair", "Planechase 2012", "Planechase Anthology", "Time Spiral":
		setCheck = func(set mtgjson.Set) bool {
			return strings.HasPrefix(set.Name, setName)
		}
	case "Portal":
		if specifier == "No Flavor Text" || specifier == "Reminder Text" {
			switch cardName {
			case "Armored Pegasus", "Bull Hippo", "Cloud Pirates",
				"Feral Shadow", "Snapping Drake", "Storm Crow":
				setName = "Portal Demo Game"
			}
		}
	default:
		switch specifier {
		case "Duels of the Planeswalkers":
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Duels of the Planeswalkers")
			}
		case "Player Rewards":
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Magic Player Rewards")
			}
		case "Clash Pack":
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasSuffix(set.Name, "Clash Pack")
			}
		case "Gateway", "WPN":
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Wizards Play Network") ||
					strings.HasPrefix(set.Name, "Gateway")
			}
		case "Gift Pack", "Gift Box":
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasSuffix(set.Name, "Gift Pack")
			}
		case "Standard Showdown":
			if len(variants) > 2 {
				switch variants[2] {
				case "Rebecca Guay":
					setName = "XLN Standard Showdown"
				case "Alayna Danner":
					setName = "M19 Standard Showdown"
				}
			}
		case "Media Insert":
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "IDW Comics") ||
					set.Name == "Dragon Con" ||
					set.Name == "Magazine Inserts" ||
					set.Name == "HarperPrism Book Promos" ||
					set.Name == "Miscellaneous Book Promos"
			}
		case "Game Day", "Launch Party":
			setCheck = func(set mtgjson.Set) bool {
				return set.Name == "Release Events" ||
					set.Name == "Launch Parties" ||
					strings.HasSuffix(set.Name, "Promos")
			}
		case "MagicFest":
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "MagicFest")
			}
		case "Judge Rewards":
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Judge Gift Cards")
			}
		case "Friday Night Magic":
			if len(variants) > 2 {
				setName = "Friday Night Magic " + variants[2]
				return
			}
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Friday Night Magic") ||
					strings.HasSuffix(set.Name, "Promos")
			}
		case "Ravnica Weekend":
			if len(variants) > 2 {
				if strings.HasPrefix(variants[2], "A") {
					setName = "GRN Ravnica Weekend"
				} else if strings.HasPrefix(variants[2], "B") {
					setName = "RNA Ravnica Weekend"
				} else {
					setName = "UNKNOWN " + variants[2]
				}
			}
		case "Arena League":
			if len(variants) > 2 {
				switch variants[2] {
				case "Tony Roberts":
					setName = "Arena League 1996"
				case "Anthony S. Waters", "Donato Giancola",
					"Rob Alexander, Urza's Saga", "John Avon, Urza's Saga":
					setName = "Arena League 1999"
				case "Mercadian Masques":
					setName = "Arena League 2000"
				case "Pat Morrissey", "Anson Maddocks", "Tom Wanerstrand",
					"Christopher Rush", "Douglas Shuler":
					setName = "Arena League 2001"
				case "Mark Poole":
					setName = "Arena League 2002"
				case "Rob Alexander", "Rob Alexander 2003":
					setName = "Arena League 2003"
				case "John Avon 2004":
					setName = "Arena League 2004"
				case "Don Thompson":
					setName = "Arena League 2005"
				case "John Avon 2006":
					setName = "Arena League 2006"
				}
				return
			}
			setCheck = func(set mtgjson.Set) bool {
				return strings.HasPrefix(set.Name, "Arena League")
			}
		case "Pre-Release":
			setCheck = func(set mtgjson.Set) bool {
				return set.Name == "Prerelease Events" ||
					strings.HasSuffix(set.Name, "Promos") ||
					strings.HasSuffix(set.Name, "Hero's Path")
			}
		default:
			switch {
			case strings.HasPrefix(specifier, "EURO"):
				setName = "European Land Program"
			case strings.HasPrefix(specifier, "San Diego Comic Con"):
				last := strings.Fields(specifier)
				setName = "San Diego Comic-Con " + last[len(last)-1]
			default:
				if setName == "Promo" {
					setCheck = func(set mtgjson.Set) bool {
						return strings.HasSuffix(set.Name, "Promos")
					}
				}
			}
		}
	}

	return
}

func (mm *Miniaturemarket) parseNumber(c *mmCard, setName string) (cardName string, numberCheck mtgban.NumberCheckFunc) {
	cardName = c.Name
	variants := mtgban.SplitVariants(c.Name)
	specifier := ""
	extra := ""
	if len(variants) > 1 {
		specifier = variants[1]
	}
	if len(variants) > 2 {
		extra = variants[2]
	}
	cardName = variants[0]

	number := ""

	defer func() {
		// If we set number but no special numberCheck, use a default one
		if number != "" && numberCheck == nil {
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				return card.Number == number
			}
		}

		variants = mtgban.SplitVariants(cardName)
		cardName = variants[0]

		// Only keep one of the split cards
		if strings.Contains(cardName, " / ") {
			s := strings.Split(cardName, " / ")
			cardName = s[0]
		}
		if strings.Contains(cardName, " // ") {
			s := strings.Split(cardName, " // ")
			cardName = s[0]
		}

	}()

	// Look up card number from the full name only
	no, found := card2numberTable[c.Name]
	if found {
		number = no
		return
	}

	// Look up card number from every detail
	no, found = mtgban.VariantsTable[setName][cardName][specifier]
	if found {
		number = no
		return
	}

	// Override card number for basic lands and a few other cards
	switch cardName {
	case "Plains", "Island", "Swamp", "Mountain", "Forest", "Wastes":
		num := strings.TrimLeft(specifier, "0")
		_, err := strconv.Atoi(num)
		if err == nil {
			number = num
			return
		}
	}
	if strings.HasPrefix(specifier, "#") {
		number = strings.TrimLeft(specifier[1:], "0")
		return
	}

	switch specifier {
	case "Ravnica Weekend":
		if extra != "" {
			number = extra
		}
	case "Showcase Art", "Alternate Art", "Extended Art":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			num, _ := strconv.Atoi(card.Number)
			return card.HasFrameEffect(mtgjson.FrameEffectShowcase) ||
				card.HasFrameEffect(mtgjson.FrameEffectExtendedArt) ||
				card.BorderColor == mtgjson.BorderColorBorderless ||
				num > set.BaseSetSize
		}
	case "Japanese Alternate Art":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			return strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
		}
	case "Pre-Release":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			if set.Name == "Prerelease Events" {
				return true
			}
			if strings.HasSuffix(set.Name, "Hero's Path") {
				return true
			}
			// These cards are tagged as Prerelease even though they are Release,
			// just passthrough them, or they don't have the 's' suffix
			switch card.Name {
			case "Bloodlord of Vaasgoth", //m12
				"Xathrid Gorgon",    //m13
				"Mayor of Avabruck", //inn
				"Moonsilver Spear",  //avr
				"Astral Drift",      //mh1
				"Reya Dawnbringer",  //10e
				"Celestine Reef",    //hop
				"Rukh Egg":          //8ed
				return true
			}

			if strings.HasSuffix(set.Name, "Promos") {
				setDate, _ := time.Parse("2006-01-02", set.ReleaseDate)
				if setDate.After(mtgban.NewPrereleaseDate) {
					return strings.Contains(card.Number, "s")
				}
			}

			return card.IsDateStamped
		}
	case "Friday Night Magic":
		numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
			if strings.HasSuffix(set.Name, "Promos") {
				return card.HasFrameEffect(mtgjson.FrameEffectInverted)
			}
			if number != "" {
				return card.Number == number
			}
			return strings.HasPrefix(set.Name, "Friday")
		}
	case "MagicFest":
		if extra != "" {
			artist := extra
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				return strings.HasPrefix(artist, card.Artist)
			}
		}
	default:
		switch setName {
		case "War of the Spark":
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				return !strings.HasSuffix(card.Number, mtgjson.SuffixSpecial)
			}
		case "Conspiracy: Take the Crown":
			if cardName == "Kaya, Ghost Assassin" {
				number = "75"
				if c.Foil {
					number = "222"
				}
			}
		case "Portal":
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				return !strings.HasSuffix(card.Number, mtgjson.SuffixLightMana)
			}
			if specifier == "No Flavor Text" {
				numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
					return strings.HasSuffix(card.Number, mtgjson.SuffixLightMana)
				}
			}
		case "Promo Pack":
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				if !card.HasFrameEffect(mtgjson.FrameEffectInverted) && strings.HasSuffix(set.Name, "Promos") {
					return strings.Contains(card.Number, "p")
				}
				return card.HasFrameEffect(mtgjson.FrameEffectInverted)
			}
		case "Fallen Empires", "Commander Anthology Volume II":
			if specifier != "" {
				fields := strings.Fields(specifier)
				artist := fields[0]
				numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
					return strings.HasPrefix(card.Artist, artist)
				}
			}
		case "Asia Pacific Land Program":
			if extra != "" {
				artist := extra
				fields := strings.Split(artist, ", ")
				artist = fields[0]
				fields = strings.Fields(artist)
				artist = fields[0]
				numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
					return strings.HasPrefix(card.Artist, artist)
				}
			}
		case "European Land Program":
			fields := strings.Split(specifier, ", ")
			if len(fields) > 1 {
				location := fields[1]
				numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
					return strings.HasPrefix(card.FlavorText, location)
				}
			}

		// Distinguish the light/dark mana symbols
		case "Arabian Nights":
			if specifier != "" {
				numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
					check := false
					if specifier == "Dark" {
						check = !strings.HasSuffix(card.Number, mtgjson.SuffixLightMana)
					} else if specifier == "Light" {
						check = strings.HasSuffix(card.Number, mtgjson.SuffixLightMana)
					}
					return check
				}
			}
		// Skip the various variants from the normal set
		case "Throne of Eldraine":
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				num, _ := strconv.Atoi(card.Number)
				return num <= set.BaseSetSize || (num >= 303 && num <= 333)
			}
		case "Theros Beyond Death":
			numberCheck = func(set mtgjson.Set, card mtgjson.Card) bool {
				num, _ := strconv.Atoi(card.Number)
				return num <= set.BaseSetSize || (num >= 269 && num <= 297)
			}
		}
	}

	return
}
