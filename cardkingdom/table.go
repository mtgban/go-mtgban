package cardkingdom

var setTable = map[string]string{
	"10th Edition":                   "Tenth Edition",
	"3rd Edition":                    "Revised Edition",
	"4th Edition":                    "Fourth Edition",
	"5th Edition":                    "Fifth Edition",
	"6th Edition":                    "Classic Sixth Edition",
	"7th Edition":                    "Seventh Edition",
	"8th Edition":                    "Eighth Edition",
	"9th Edition":                    "Ninth Edition",
	"Alpha":                          "Limited Edition Alpha",
	"Archenemy - Nicol Bolas":        "Archenemy: Nicol Bolas",
	"Battle Royale":                  "Battle Royale Box Set",
	"Beatdown":                       "Beatdown Box Set",
	"Beta":                           "Limited Edition Beta",
	"Collectors Ed Intl":             "Intl. Collectors’ Edition",
	"Collectors Ed":                  "Collectors’ Edition",
	"Commander Anthology Vol. II":    "Commander Anthology Volume II",
	"Commander":                      "Commander 2011",
	"Conspiracy - Take the Crown":    "Conspiracy: Take the Crown",
	"Deckmaster":                     "Deckmasters",
	"Guilds of Ravnica: Guild Kits":  "GRN Guild Kit",
	"Modern Event Deck":              "Modern Event Deck 2014",
	"Portal 3K":                      "Portal Three Kingdoms",
	"Portal II":                      "Portal Second Age",
	"Ravnica Allegiance: Guild Kits": "RNA Guild Kit",
	"Ravnica":                        "Ravnica: City of Guilds",
	"Shadows Over Innistrad":         "Shadows over Innistrad",
	"Throne of Eldraine Variants":    "Throne of Eldraine",
	"Timeshifted":                    "Time Spiral Timeshifted",
	"Unlimited":                      "Unlimited Edition",

	"Masterpiece Series: Inventions":     "Kaladesh Inventions",
	"Masterpiece Series: Invocations":    "Amonkhet Invocations",
	"Masterpiece Series: Expeditions":    "Zendikar Expeditions",
	"Masterpiece Series: Mythic Edition": "Mythic Edition",

	"Duel Decks: Phyrexia vs. The Coalition":   "Duel Decks: Phyrexia vs. the Coalition",
	"Global Series: Jiang Yanggu & Mu Yanling": "Global Series Jiang Yanggu & Mu Yanling",
	"Premium Deck Series: Fire & Lightning":    "Premium Deck Series: Fire and Lightning",
	"War of the Spark JPN Planeswalkers":       "War of the Spark",

	"Promo Pack":                     "Promotional",
	"Japanese Jace vs. Chandra Foil": "Duel Decks: Jace vs. Chandra",
}

var promosetTable = map[string]string{
	"15th Anniversary Foil":  "15th Anniversary Cards",
	"2017 Gift Pack":         "2017 Gift Pack",
	"2018 Gift Pack":         "M19 Gift Pack",
	"APAC Blue":              "Asia Pacific Land Program",
	"APAC Clear":             "Asia Pacific Land Program",
	"APAC Red":               "Asia Pacific Land Program",
	"Book Promo":             "Magazine Inserts",
	"Book":                   "HarperPrism Book Promos",
	"Booth Foil":             "URL/Convention Promos",
	"Convention Foil":        "URL/Convention Promos",
	"Convention":             "Dragon Con",
	"Dragonfury Promo":       "Tarkir Dragonfury",
	"Euro Set Blue":          "European Land Program",
	"Euro Set Purple":        "European Land Program",
	"Euro Set Red":           "European Land Program",
	"Grand Prix Foil":        "Grand Prix Promos",
	"GP Foil":                "Grand Prix Promos",
	"Guru Land":              "Guru",
	"Hascon Promo Foil":      "HasCon 2017",
	"Hero's Path Foil":       "Journey into Nyx Hero's Path",
	"Holiday Foil":           "Happy Holidays",
	"JSS Foil":               "Junior Super Series",
	"Legend Promo":           "DCI Legend Membership",
	"Legend":                 "DCI Legend Membership",
	"MC Qualifier Foil":      "Pro Tour Promos",
	"Media Insert Foil":      "URL/Convention Promos",
	"Mirrodin Pure Promo":    "New Phyrexia Promos",
	"Nationals":              "Nationals Promos",
	"Ponies: The Galloping":  "Ponies: The Galloping",
	"Pro Tour Foil":          "Pro Tour Promos",
	"RPTQ Promo Foil":        "Pro Tour Promos",
	"Redeem Foil":            "Wizards of the Coast Online Store",
	"Resale Foil":            "Resale Promos",
	"Standard Showdown 2017": "XLN Standard Showdown",
	"Standard Showdown 2018": "M19 Standard Showdown",
	"Store Foil":             "Resale Promos",
	"Store Promo Foil":       "Resale Promos",
	"Summer Foil":            "Summer of Magic",
	"Ugin's Fate":            "Ugin's Fate",
	"WMC Qualifier":          "World Magic Cup Qualifiers",
	"WMCQ Foil":              "World Magic Cup Qualifiers",
	"Welcome 2016":           "Welcome Deck 2016",
	"Welcome 2017":           "Welcome Deck 2017",
}

var cardTable = map[string]string{
	"Ach! Hans, Run!": "\"Ach! Hans, Run!\"",
	"BFM Left":        "B.F.M. (Big Furry Monster)",
	"BFM Right":       "B.F.M. (Big Furry Monster - Right)",

	"Knight of the Kitchen SInk (B - Gravy Boat)":     "Knight of the Kitchen Sink (B - Gravy Boat)",
	"The Ultimate Nightmare of WotC Customer Service": "The Ultimate Nightmare of Wizards of the Coast® Customer Service",
	"Our Market Research":                             "Our Market Research Shows That Players Like Really Long Card Names So We Made this Card to Have the Absolute Longest Card Name Ever Elemental",
}

// This table adjusts the set for a few oddly categorized cards
var card2setTable = map[string]string{
	// Wrong FNM cards
	"Cast Down (FNM Foil)":              "Dominaria Promos",
	"Opt (FNM Foil)":                    "Dominaria Promos",
	"Shanna, Sisay's Legacy (FNM Foil)": "Dominaria Promos",
	"Elvish Rejuvenator (FNM Foil)":     "Core Set 2019 Promos",
	"Murder (FNM Foil)":                 "Core Set 2019 Promos",
	"Militia Bugler (FNM Foil)":         "Core Set 2019 Promos",
	"Conclave Tribunal (FNM Foil)":      "Guilds of Ravnica Promos",
	"Sinister Sabotage (FNM Foil)":      "Guilds of Ravnica Promos",
	"Thought Erasure (FNM Foil)":        "Guilds of Ravnica Promos",
	"Growth Spiral (FNM Foil)":          "Ravnica Allegiance Promos",
	"Light Up the Stage (FNM Foil)":     "Ravnica Allegiance Promos",
	"Mortify (FNM Foil)":                "Ravnica Allegiance Promos",
	"Augur of Bolas (FNM Foil)":         "War of the Spark Promos",
	"Paradise Druid (FNM Foil)":         "War of the Spark Promos",
	"Dovin's Veto (FNM Foil)":           "War of the Spark Promos",

	// Duplicated promos
	"Giant Growth (FNM Foil)":       "Friday Night Magic 2000",
	"Sakura-Tribe Elder (FNM Foil)": "Friday Night Magic 2009",

	// Wrong Prerelease cards
	"Ass Whuppin' (Prerelease Foil)":          "Release Events",
	"Force of Nature (Prerelease Foil)":       "Release Events",
	"Rukh Egg (Prerelease Foil)":              "Release Events",
	"Bloodlord of Vaasgoth (Prerelease Foil)": "Magic 2012 Promos",
	"Mayor of Avabruck (Prerelease Foil)":     "Innistrad Promos",
	"Moonsilver Spear (Prerelease Foil)":      "Avacyn Restored Promos",
	"Plains (Prerelease Foil)":                "Dragon's Maze Promos",

	// Wrong or duplicated cards
	"Balduvian Horde (Judge Foil)": "Worlds",

	// Wrong extended art cards
	"Liliana's Specter (Extended Art)":  "Magic 2011 Promos",
	"Nissa's Chosen (Extended Art)":     "Zendikar Promos",
	"Kalastria Highborn (Extended Art)": "Worldwake Promos",
	"Staggershock (Extended Art)":       "Rise of the Eldrazi Promos",

	// Wrong set under "extended art"
	"Electrolyze (Extended Art)":                   "Champs and States",
	"Niv-Mizzet, the Firemind (Extended Art Foil)": "Champs and States",
	"Rakdos Guildmage (Extended Art)":              "Champs and States",
	"Voidslime (Extended Art Foil)":                "Champs and States",
	"Urza's Factory (Extended Art)":                "Champs and States",
	"Serra Avenger (Extended Art Foil)":            "Champs and States",
	"Blood Knight (Extended Art)":                  "Champs and States",
	"Groundbreaker (Extended Art Foil)":            "Champs and States",
	"Imperious Perfect (Extended Art)":             "Champs and States",
	"Doran, the Siege Tower (Extended Art Foil)":   "Champs and States",
	"Bramblewood Paragon (Extended Art)":           "Champs and States",
	"Mutavault (Extended Art Foil)":                "Champs and States",

	// Standard events
	"Canopy Vista (Alternate Art)":     "BFZ Standard Series",
	"Cinder Glade (Alternate Art)":     "BFZ Standard Series",
	"Prairie Stream (Alternate Art)":   "BFZ Standard Series",
	"Smoldering Marsh (Alternate Art)": "BFZ Standard Series",
	"Sunken Hollow (Alternate Art)":    "BFZ Standard Series",

	// Wrong treasure chest under bab
	"Legion's Landing (Buy-a-Box Foil)":         "XLN Treasure Chest",
	"Search for Azcanta (Buy-a-Box Foil)":       "XLN Treasure Chest",
	"Arguel's Blood Fast (Buy-a-Box Foil)":      "XLN Treasure Chest",
	"Vance's Blasting Cannons (Buy-a-Box Foil)": "XLN Treasure Chest",
	"Growing Rites of Itlimoc (Buy-a-Box Foil)": "XLN Treasure Chest",
	"Conqueror's Galleon (Buy-a-Box Foil)":      "XLN Treasure Chest",
	"Dowsing Dagger (Buy-a-Box Foil)":           "XLN Treasure Chest",
	"Primal Amulet (Buy-a-Box Foil)":            "XLN Treasure Chest",
	"Thaumatic Compass (Buy-a-Box Foil)":        "XLN Treasure Chest",
	"Treasure Map (Buy-a-Box Foil)":             "XLN Treasure Chest",

	// Wrong bab
	"Day of Judgment (Buy-A-Box Foil)": "Zendikar Promos",
	"Flusterstorm (Buy-a-Box)":         "Modern Horizons",

	// Non-promo sets
	"Firesong and Sunspeaker (Buy-A-Box Foil)":        "Dominaria",
	"Nexus of Fate (Buy-a-Box Foil)":                  "Core Set 2019",
	"Impervious Greatwurm (Buy-a-Box Foil)":           "Guilds of Ravnica",
	"The Haunt of Hightower (Buy-a-Box Foil)":         "Ravnica Allegiance",
	"Kenrith, the Returned King (Buy-A-Box Foil)":     "Throne of Eldraine",
	"Kenrith, the Returned King (Buy-A-Box Non-Foil)": "Throne of Eldraine",
	"Tezzeret, Master of the Bridge (Buy-A-Box Foil)": "War of the Spark",
	"Rienne, Angel of Rebirth (Buy-A-Box Foil)":       "Core Set 2020",

	// Unique
	"Sylvan Ranger (WPN - #51)": "Wizards Play Network 2010",
	"Sylvan Ranger (WPN - #70)": "Wizards Play Network 2011",

	// Not convention
	"Deeproot Champion (Convention Foil)":  "Ixalan Promos",
	"Death Baron (Convention Foil)":        "Core Set 2019 Promos",
	"Nightpack Ambusher (Convention Foil)": "Core Set 2020 Promos",

	// Duplicated promos when promos are mixed in the normal cards
	"Ertai, the Corrupted (Alternate Art Foil)":    "Planeshift",
	"Skyship Weatherlight (Alternate Art Foil)":    "Planeshift",
	"Tahngarth, Talruum Hero (Alternate Art Foil)": "Planeshift",

	// Duel deck promos
	"Jace Beleren (Japanese Jace vs. Chandra Foil)":   "Duel Decks: Jace vs. Chandra",
	"Chandra Nalaar (Japanese Jace vs. Chandra Foil)": "Duel Decks: Jace vs. Chandra",

	// Miscellaneous promos
	"Forest (Arena 2002 Foil)":              "Arena League 2001",
	"Crystalline Sliver (DCI Foil)":         "Friday Night Magic 2003",
	"Underworld Dreams (DCI Foil)":          "Two-Headed Giant Tournament",
	"Magister of Worth (Launch Promo Foil)": "Launch Parties",
	"Jace Beleren (Book Promo)":             "Miscellaneous Book Promos",
	"Scent of Cinder (Alternate Art)":       "Magazine Inserts",

	"Necropotence (New York 1996 - Not Tournament Legal)": "Pro Tour Collector Set",
}

var allVariants = map[string]map[string]string{
	"Agent of Stromgald": map[string]string{
		"Staff":   "64a",
		"Doorway": "64b",
	},
	"Arcane Denial": map[string]string{
		"Axe":   "22a",
		"Sword": "22b",
	},
	"Awesome Presence": map[string]string{
		"Open Arms":          "23a",
		"Man Being Attacked": "23b",
	},
	"Benthic Explorers": map[string]string{
		"Facing Front": "24a",
		"Facing Right": "24b",
	},
	"Deadly Insect": map[string]string{
		"Bird": "86a",
		"Elf":  "86b",
	},
	"Elvish Ranger": map[string]string{
		"Facing Left":  "88a",
		"Facing Right": "88b",
	},
	"Feast or Famine": map[string]string{
		"Falling": "49a",
		"Knife":   "49b",
	},
	"Foresight": map[string]string{
		"Facing Front": "29a",
		"Facing Right": "29b",
	},
	"Gift of the Woods": map[string]string{
		"Cat":  "92a",
		"Wolf": "92b",
	},
	"Gorilla Shaman": map[string]string{
		"Facing Left":  "72a",
		"Facing Right": "72b",
	},
	"Guerrilla Tactics": map[string]string{
		"Tripwire": "74a",
		"Cliff":    "74b",
	},
	"Kjeldoran Escort": map[string]string{
		"Green Blanket": "7a",
		"Red Blanket":   "7b",
	},
	"Lat-Nam's Legacy": map[string]string{
		"Scroll": "30a",
		"Book":   "30b",
	},
	"Martyrdom": map[string]string{
		"Alive": "10a",
		"Dead":  "10b",
	},
	"Noble Steeds": map[string]string{
		"Tree":     "11a",
		"Close-Up": "11b",
	},
	"Reinforcements": map[string]string{
		"Orc":    "12a",
		"No Orc": "12b",
	},
	"Royal Herbalist": map[string]string{
		"Pink Jar":    "15a",
		"Green Smoke": "15b",
	},
	"Soldevi Adnate": map[string]string{
		"Two Eyes": "60a",
		"One Eye":  "60b",
	},
	"Soldevi Sage": map[string]string{
		"Reading": "34a",
		"Writing": "34b",
	},
	"Storm Shaman": map[string]string{
		"Facing Front": "81a",
		"Facing Left":  "81b",
	},
	"Swamp Mosquito": map[string]string{
		"Black Trees": "63a",
		"Brown Trees": "63b",
	},
	"Undergrowth": map[string]string{
		"Elf": "102a",
		"Fox": "102b",
	},
	"Whip Vine": map[string]string{
		"No Bird": "103a",
		"Bird":    "103b",
	},
	"Wild Aesthir": map[string]string{
		"Wings Behind Back": "21a",
		"Wings Spread":      "21b",
	},
}

var pal01Variants = map[string]map[string]string{
	"Forest": map[string]string{
		"Arena 2001 Foil": "1",
		"Arena 2002 Foil": "11",
	},
}

var athVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"A": "78",
		"B": "79",
	},
	"Swamp": map[string]string{
		"A": "80",
		"B": "81",
	},
	"Mountain": map[string]string{
		"A": "82",
		"B": "83",
	},
	"Forest": map[string]string{
		"A": "84",
		"B": "85",
	},
}

var atqVariants = map[string]map[string]string{
	"Mishra's Factory": map[string]string{
		"Spring": "80a",
		"Summer": "80b",
		"Autumn": "80c",
		"Winter": "80d",
	},
	"Strip Mine": map[string]string{
		"A": "82a",
		"B": "82b",
		"C": "82c",
		"D": "82d",
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

var palpVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"APAC Red":   "4",
		"APAC Clear": "9",
		"APAC Blue":  "14",
	},
	"Island": map[string]string{
		"APAC Red":   "2",
		"APAC Clear": "7",
		"APAC Blue":  "12",
	},
	"Swamp": map[string]string{
		"APAC Red":   "5",
		"APAC Clear": "10",
		"APAC Blue":  "15",
	},
	"Mountain": map[string]string{
		"APAC Red":   "3",
		"APAC Clear": "8",
		"APAC Blue":  "13",
	},
	"Forest": map[string]string{
		"APAC Red":   "1",
		"APAC Clear": "6",
		"APAC Blue":  "11",
	},
}

var brbVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"A": "127",
		"B": "128",
		"C": "130",
		"D": "131",
		"E": "132",
		"F": "125",
		"G": "124",
		"H": "126",
		"I": "129",
	},
	"Island": map[string]string{
		"A": "112",
		"B": "114",
		"C": "113",
		"D": "111",
		"E": "110",
	},
	"Swamp": map[string]string{
		"A": "133",
		"B": "134",
		"C": "136",
		"D": "135",
	},
	"Mountain": map[string]string{
		"A": "118",
		"B": "119",
		"C": "115",
		"D": "116",
		"E": "117",
		"F": "120",
		"G": "121",
		"H": "122",
		"I": "123",
	},
	"Forest": map[string]string{
		"A": "103",
		"B": "104",
		"C": "109",
		"D": "107",
		"E": "108",
		"F": "101",
		"G": "102",
		"H": "105",
		"I": "106",
	},
}

var chkVariants = map[string]map[string]string{
	"Brothers Yamazaki": map[string]string{
		"160 A": "160a",
		"160 B": "160b",
	},
}

var chrVariants = map[string]map[string]string{
	"Urza's Mine": map[string]string{
		"Mouth":  "114a",
		"Sphere": "114b",
		"Pulley": "114c",
		"Tower":  "114d",
	},
	"Urza's Power Plant": map[string]string{
		"Pot":     "115a",
		"Columns": "115b",
		"Sphere":  "115c",
		"Bug":     "115d",
	},
	"Urza's Tower": map[string]string{
		"Shore":     "116a",
		"Plains":    "116b",
		"Mountains": "116c",
		"Forest":    "116d",
	},
}

var ceVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"A - Not Tournament Legal": "289",
		"B - Not Tournament Legal": "288",
		"C - Not Tournament Legal": "290",
	},
	"Island": map[string]string{
		"A - Not Tournament Legal": "292",
		"B - Not Tournament Legal": "291",
		"C - Not Tournament Legal": "293",
	},
	"Swamp": map[string]string{
		"A - Not Tournament Legal": "295",
		"B - Not Tournament Legal": "294",
		"C - Not Tournament Legal": "296",
	},
	"Mountain": map[string]string{
		"A - Not Tournament Legal": "298",
		"B - Not Tournament Legal": "297",
		"C - Not Tournament Legal": "299",
	},
	"Forest": map[string]string{
		"A - Not Tournament Legal": "301",
		"B - Not Tournament Legal": "300",
		"C - Not Tournament Legal": "302",
	},
}

var dd2Variants = map[string]map[string]string{
	"Jace Beleren": map[string]string{
		"Foil": "1",
		"Japanese Jace vs. Chandra Foil": "1★",
	},
	"Chandra Nalaar": map[string]string{
		"Foil": "34",
		"Japanese Jace vs. Chandra Foil": "34★",
	},
}

var pelpVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"Euro Set Blue":   "4",
		"Euro Set Red":    "9",
		"Euro Set Purple": "14",
	},
	"Island": map[string]string{
		"Euro Set Blue":   "2",
		"Euro Set Red":    "7",
		"Euro Set Purple": "12",
	},
	"Swamp": map[string]string{
		"Euro Set Blue":   "5",
		"Euro Set Red":    "10",
		"Euro Set Purple": "15",
	},
	"Mountain": map[string]string{
		"Euro Set Blue":   "3",
		"Euro Set Red":    "8",
		"Euro Set Purple": "13",
	},
	"Forest": map[string]string{
		"Euro Set Blue":   "1",
		"Euro Set Red":    "6",
		"Euro Set Purple": "11",
	},
}

var femVariants = map[string]map[string]string{
	"Basal Thrull": map[string]string{
		"Kaja Foglio":   "34a",
		"Phil Foglio":   "34b",
		"Kane-Ferguson": "34c",
		"Rush":          "34d",
	},
	"Goblin Chirurgeon": map[string]string{
		"Gelon":   "54a",
		"Foglio":  "54b",
		"Frazier": "54c",
	},
	"Goblin Grenade": map[string]string{
		"Spencer": "56a",
		"Frazier": "56b",
		"Rush":    "56c",
	},
	"Goblin War Drums": map[string]string{
		"Frazier":  "58a",
		"Ferguson": "58b",
		"Hudson":   "58c",
		"Menges":   "58d",
	},
	"High Tide": map[string]string{
		"Tucker":   "18a",
		"Maddocks": "18b",
		"Weber":    "18c",
	},
	"Hymn to Tourach": map[string]string{
		"Van Camp":  "38a",
		"Danforth":  "38b",
		"Hoover":    "38c",
		"Kirschner": "38d",
	},
	"Icatian Infantry": map[string]string{
		"Beard Jr.": "7a",
		"Rush":      "7b",
		"Shuler":    "7c",
		"Tucker":    "7d",
	},
	"Initiates of the Ebon Hand": map[string]string{
		"Hudson":   "39a",
		"Danforth": "39b",
		"Foglio":   "39c",
	},
	"Necrite": map[string]string{
		"Spencer": "41a",
		"Rush":    "41b",
		"Tucker":  "41c",
	},
	"Night Soil": map[string]string{
		"Everingham": "71a",
		"Hudson":     "71b",
		"Tucker":     "71c",
	},
	"Order of Leitbur": map[string]string{
		"Wackwitz - Facing Left":  "16a",
		"Wackwitz - Facing Right": "16b",
	},
	"Order of the Ebon Hand": map[string]string{
		"Benson":  "42a",
		"Rush":    "42b",
		"Spencer": "42c",
	},
	"Spore Cloud": map[string]string{
		"Van Camp": "72a",
		"Myrfors":  "72b",
		"Weber":    "72c",
	},
}

var ed5Variants = map[string]map[string]string{
	"Plains": map[string]string{
		"A": "430",
		"B": "432",
		"C": "433",
		"D": "431",
	},
	"Island": map[string]string{
		"A": "434",
		"B": "437",
		"C": "436",
		"D": "435",
	},
	"Swamp": map[string]string{
		"A": "440",
		"B": "439",
		"C": "441",
		"D": "438",
	},
	"Mountain": map[string]string{
		"A": "442",
		"B": "445",
		"C": "444",
		"D": "443",
	},
	"Forest": map[string]string{
		"A": "446",
		"B": "448",
		"C": "447",
		"D": "449",
	},
}

var ed4Variants = map[string]map[string]string{
	"Plains": map[string]string{
		"A": "365",
		"B": "364",
		"C": "366",
	},
	"Island": map[string]string{
		"A": "368",
		"B": "367",
		"C": "369",
	},
	"Swamp": map[string]string{
		"A": "371",
		"B": "370",
		"C": "372",
	},
	"Mountain": map[string]string{
		"A": "374",
		"B": "373",
		"C": "375",
	},
	"Forest": map[string]string{
		"A": "377",
		"B": "376",
		"C": "378",
	},
}

var hmlVariants = map[string]map[string]string{
	"Ambush Party": map[string]string{
		"Doorway":  "63a",
		"Mountain": "63b",
	},
	"Anaba Bodyguard": map[string]string{
		"With Human": "66a",
		"Alone":      "66b",
	},
	"Carapace": map[string]string{
		"Arrow": "84a",
		"Sword": "84b",
	},
	"Cemetery Gate": map[string]string{
		"Vampire":    "44a",
		"No Vampire": "44b",
	},
	"Memory Lapse": map[string]string{
		"Runes":  "32a",
		"Jigsaw": "32b",
	},
	"Torture": map[string]string{
		"Tools": "59a",
		"Runes": "59b",
	},
}

var iceVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"A": "365",
		"B": "364",
		"C": "366",
	},
	"Island": map[string]string{
		"A": "368",
		"B": "370",
		"C": "369",
	},
	"Swamp": map[string]string{
		"A": "373",
		"B": "374",
		"C": "375",
	},
	"Mountain": map[string]string{
		"A": "378",
		"B": "377",
		"C": "376",
	},
	"Forest": map[string]string{
		"A": "381",
		"B": "380",
		"C": "382",
	},
}

var leaVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"A": "287",
		"B": "286",
	},
	"Island": map[string]string{
		"A": "289",
		"B": "288",
	},
	"Swamp": map[string]string{
		"A": "291",
		"B": "290",
	},
	"Mountain": map[string]string{
		"A": "293",
		"B": "292",
	},
	"Forest": map[string]string{
		"A": "294",
		"B": "295",
	},
}

var lebVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"A": "289",
		"B": "288",
		"C": "290",
	},
	"Island": map[string]string{
		"A": "292",
		"B": "291",
		"C": "293",
	},
	"Swamp": map[string]string{
		"A": "295",
		"B": "294",
		"C": "296",
	},
	"Mountain": map[string]string{
		"A": "298",
		"B": "297",
		"C": "299",
	},
	"Forest": map[string]string{
		"A": "301",
		"B": "300",
		"C": "302",
	},
}

var pmpsVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"MPS 2005 - Azorius":  "289",
		"MPS 2005 - Boros":    "290",
		"MPS 2005 - Orzhov":   "287",
		"MPS 2005 - Selesnya": "288",
	},
	"Island": map[string]string{
		"MPS 2005 - Azorius": "291",
		"MPS 2005 - Dimir":   "294",
		"MPS 2005 - Izzet":   "292",
		"MPS 2005 - Simic":   "293",
	},
	"Swamp": map[string]string{
		"MPS 2005 - Dimir":   "296",
		"MPS 2005 - Golgari": "295",
		"MPS 2005 - Orzhov":  "297",
		"MPS 2005 - Rakdos":  "298",
	},
	"Mountain": map[string]string{
		"MPS 2005 - Boros":  "302",
		"MPS 2005 - Gruul":  "300",
		"MPS 2005 - Izzet":  "301",
		"MPS 2005 - Rakdos": "299",
	},
	"Forest": map[string]string{
		"MPS 2005 - Golgari":  "303",
		"MPS 2005 - Gruul":    "305",
		"MPS 2005 - Selesnya": "304",
		"MPS 2005 - Simic":    "306",
	},
}

var mirVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"A": "334",
		"B": "332",
		"C": "333",
		"D": "331",
	},
	"Island": map[string]string{
		"A": "335",
		"B": "336",
		"C": "337",
		"D": "338",
	},
	"Swamp": map[string]string{
		"A": "341",
		"B": "340",
		"C": "339",
		"D": "342",
	},
	"Mountain": map[string]string{
		"A": "345",
		"B": "346",
		"C": "343",
		"D": "344",
	},
	"Forest": map[string]string{
		"A": "349",
		"B": "350",
		"C": "348",
		"D": "347",
	},
}

var plsVariants = map[string]map[string]string{
	"Ertai, the Corrupted": map[string]string{
		"":                   "107",
		"Alternate Art Foil": "107★",
	},
	"Skyship Weatherlight": map[string]string{
		"":                   "133",
		"Alternate Art Foil": "133★",
	},
	"Tahngarth, Talruum Hero": map[string]string{
		"":                   "74",
		"Alternate Art Foil": "74★",
	},
}

var po2Variants = map[string]map[string]string{
	"Plains": map[string]string{
		"A": "152",
		"B": "153",
		"C": "151",
	},
	"Island": map[string]string{
		"A": "154",
		"B": "156",
		"C": "155",
	},
	"Swamp": map[string]string{
		"A": "157",
		"B": "159",
		"C": "158",
	},
	"Mountain": map[string]string{
		"A": "162",
		"B": "160",
		"C": "161",
	},
	"Forest": map[string]string{
		"A": "163",
		"B": "165",
		"C": "164",
	},
}

var porVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"A": "196",
		"B": "198",
		"C": "197",
		"D": "199",
	},
	"Island": map[string]string{
		"A": "200",
		"B": "201",
		"C": "202",
		"D": "203",
	},
	"Swamp": map[string]string{
		"A": "204",
		"B": "207",
		"C": "205",
		"D": "206",
	},
	"Mountain": map[string]string{
		"A": "208",
		"B": "209",
		"C": "211",
		"D": "210",
	},
	"Forest": map[string]string{
		"A": "212",
		"B": "215",
		"C": "213",
		"D": "214",
	},
}

var ed3Variants = map[string]map[string]string{
	"Plains": map[string]string{
		"A": "293",
		"B": "292",
		"C": "294",
	},
	"Island": map[string]string{
		"A": "295",
		"B": "297",
		"C": "296",
	},
	"Swamp": map[string]string{
		"A": "299",
		"B": "298",
		"C": "300",
	},
	"Mountain": map[string]string{
		"A": "302",
		"B": "301",
		"C": "303",
	},
	"Forest": map[string]string{
		"A": "305",
		"B": "304",
		"C": "306",
	},
}

var tmpVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"A": "333",
		"B": "332",
		"C": "331",
		"D": "334",
	},
	"Island": map[string]string{
		"A": "338",
		"B": "336",
		"C": "337",
		"D": "335",
	},
	"Swamp": map[string]string{
		"A": "340",
		"B": "339",
		"C": "341",
		"D": "342",
	},
	"Mountain": map[string]string{
		"A": "346",
		"B": "344",
		"C": "345",
		"D": "343",
	},
	"Forest": map[string]string{
		"A": "347",
		"B": "350",
		"C": "348",
		"D": "349",
	},
}

var uglVariants = map[string]map[string]string{
	"B.F.M.": map[string]string{
		"Big Furry Monster":         "28",
		"Big Furry Monster - Right": "29",
	},
}

var ustVariants = map[string]map[string]string{
	"Amateur Auteur": map[string]string{
		"A - Innistrad": "3b",
		"B - Ravnica":   "3a",
		"C - Theros":    "3c",
		"D - Zendikar":  "3d",
	},
	"Beast in Show": map[string]string{
		"A - Baloth - \"Something really...\"":   "103c",
		"B - Gnarlid - \"We've not seen...\"":    "103b",
		"C - Raptor - \"A stunning example...\"": "103a",
		"D - Thragtusk - \"Not a surprise...\"":  "103d",
	},
	"Everythingamajig": map[string]string{
		"A - 'Move a Counter'":   "147a",
		"B - 'Draw a Card'":      "147b",
		"C - 'Flip a Coin'":      "147c",
		"D - 'Add one Mana'":     "147d",
		"E - 'Sacrifice a Land'": "147e",
		"F - 'Scry 2'":           "147f",
	},
	"Extremely Slow Zombie": map[string]string{
		"A - Fall":   "54b",
		"B - Spring": "54d",
		"C - Summer": "54a",
		"D - Winter": "54c",
	},
	"Garbage Elemental": map[string]string{
		"A 'Frenzy'":      "82a",
		"B 'Contraption'": "82b",
		"C 'Battle Cry'":  "82c",
		"D 'Cascade'":     "82d",
		"E 'Unleash'":     "82e",
		"F 'Last Strike'": "82f",
	},
	"Ineffable Blessing": map[string]string{
		"A - Flavorful/Bland":            "113a",
		"B - Artist":                     "113b",
		"C - White-border/Silver-border": "113c",
		"D - Rarity":                     "113d",
		"E - Odd or Even":                "113e",
		"F - Number":                     "113f",
	},
	"Knight of the Kitchen Sink": map[string]string{
		"A - Garlic Press": "12a",
		"B - Gravy Boat":   "12b",
		"C - Juicer":       "12c",
		"D - Melon Baller": "12d",
		"E - Olive Forks":  "12e",
		"F - Tea Cozy":     "12f",
	},
	"Novellamental": map[string]string{
		"A \"My grandmother...\"": "41a",
		"B \"This pendant...\"":   "41b",
		"C \"The chain...\"":      "41c",
		"D \"My heart...\"":       "41d",
	},
	"Secret Base": map[string]string{
		"E - S.N.E.A.K.":           "165b",
		"A - Crossbreed Labs":      "165e",
		"C - Goblin Explosioneers": "165d",
		"B - Dastardly Doom":       "165c",
		"D - Order of the Widget":  "165a",
	},
	"Sly Spy": map[string]string{
		"A 'Reveals hand'":               "67a",
		"B 'Left'":                       "67b",
		"C 'Player loses a finger'":      "67c",
		"D 'Right'":                      "67d",
		"E 'Reveal top card of library'": "67e",
		"F 'Roll a six-sided die'":       "67f",
	},
	"Target Minotaur": map[string]string{
		"A - Acid":  "98b",
		"B - Fire":  "98c",
		"C - Ice":   "98a",
		"D - Vines": "98d",
	},
	"Very Cryptic Command": map[string]string{
		"A 'B&W'":                             "49a",
		"B 'Untap two target permanents'":     "49b",
		"C 'Draw card from opponent library'": "49c",
		"D 'Return target permanent'":         "49d",
		"E 'Counter target black-border'":     "49e",
		"F 'Scry 3'":                          "49f",
	},
}

var sldVariants = map[string]map[string]string{
	"Serum Visions": map[string]string{
		"029 - Collantes Foil": "29",
		"030 - DXTR Foil":      "30",
		"031 - YS Foil":        "31",
		"032 - Zuverza Foil":   "32",
	},
}

// This table contains any promo card that cannot be uniquely mapped in card.go
// Mostly due to Promo Pack or duplicated release/launch events
var promoVariants = map[string]map[string]string{
	"Plains": map[string]string{
		"Promo Pack": "1",
	},
	"Island": map[string]string{
		"Promo Pack": "2",
	},
	"Swamp": map[string]string{
		"Promo Pack": "3",
	},
	"Mountain": map[string]string{
		"Promo Pack": "4",
	},
	"Forest": map[string]string{
		"Promo Pack": "5",
	},

	"Banefire": map[string]string{
		"Prerelease Foil": "130s",
		"Promo Pack":      "130p",
	},
	"Beast Whisperer": map[string]string{
		"Resale Foil":     "123*",
		"Prerelease Foil": "123s",
		"Promo Pack":      "123p",
	},
	"Benalish Marshal": map[string]string{
		"Prerelease Foil": "6s",
		"Promo Pack":      "6p",
	},
	"Captivating Crew": map[string]string{
		"Prerelease Foil": "137s",
		"Promo Pack":      "137p",
	},
	"Chandra's Regulator": map[string]string{
		"Bundle Foil":     "131",
		"Prerelease Foil": "131s",
		"Promo Pack":      "131p",
	},
	"Death Baron": map[string]string{
		"Convention Foil": "90",
		"Prerelease Foil": "90s",
		"Promo Pack":      "90p",
	},
	"Deathbringer Regent": map[string]string{
		"Launch Promo Foil": "96",
		"Prerelease Foil":   "96s",
	},
	"Deeproot Champion": map[string]string{
		"Convention Foil": "185",
		"Prerelease Foil": "185s",
		"Promo Pack":      "185p",
	},
	"Endbringer": map[string]string{
		"Launch Foil":     "3",
		"Prerelease Foil": "3s",
	},
	"Etali, Primal Storm": map[string]string{
		"Prerelease Foil": "100s",
		"Promo Pack":      "100p",
	},
	"Experimental Frenzy": map[string]string{
		"Promo Pack": "99p",
	},
	"Ghalta, Primal Hunger": map[string]string{
		"Store Championship Foil": "130",
		"Prerelease Foil":         "130s",
		"Promo Pack":              "130p",
	},
	"Isolated Chapel": map[string]string{
		"Prerelease Foil": "241s",
		"Promo Pack":      "241p",
	},
	"Karn's Bastion": map[string]string{
		"Planeswalker Weekend Foil": "248",
		"Prerelease Foil":           "248s",
		"Promo Pack":                "248p",
	},
	"Leonin Warleader": map[string]string{
		"Prerelease Foil": "23s",
		"Promo Pack":      "23p",
	},
	"Negate": map[string]string{
		"Textless":   "8",
		"Promo Pack": "69",
	},
	"Nicol Bolas, Dragon-God": map[string]string{
		"Prerelease Foil": "207s",
		"Promo Pack":      "207p",
	},
	"Nightpack Ambusher": map[string]string{
		"Convention Foil": "185",
		"Prerelease Foil": "185s",
		"Promo Pack":      "185p",
	},
	"Ramunap Excavator": map[string]string{
		"Draft Weekend Foil": "129",
		"Prerelease Foil":    "129s",
	},
	"Ripjaw Raptor": map[string]string{
		"Prerelease Foil": "203s",
		"Promo Pack":      "203p",
	},
	"River's Rebuke": map[string]string{
		"Prerelease Foil": "71s",
		"Promo Pack":      "71p",
	},
	"Siege-Gang Commander": map[string]string{
		"Prerelease Foil": "143s",
		"Promo Pack":      "143p",
	},
	"Sorcerous Spyglass": map[string]string{
		"Promo Pack - XLN": "248p",
		"Promo Pack - ELD": "233p",
	},
	"Steel Leaf Champion": map[string]string{
		"Store Championship Foil": "182",
		"Prerelease Foil":         "182s",
		"Promo Pack":              "182p",
	},
	"Temple of Mystery": map[string]string{
		"Magic 2015 Clash Pack": "6",
		"Prerelease Foil":       "255s",
		"Promo Pack":            "255p",
	},
	"Time Wipe": map[string]string{
		"Planeswalker Weekend Foil": "223",
		"Prerelease Foil":           "223s",
		"Promo Pack":                "223p",
	},
	"Vantress Gargoyle": map[string]string{
		"Extended Art":    "349",
		"Prerelease Foil": "71s",
		"Promo Pack":      "71p",
	},
}

var setVariants = map[string]map[string]map[string]string{
	"Alliances":                    allVariants,
	"Arena League 2001":            pal01Variants,
	"Anthologies":                  athVariants,
	"Antiquities":                  atqVariants,
	"Asia Pacific Land Program":    palpVariants,
	"Battle Royale Box Set":        brbVariants,
	"Champions of Kamigawa":        chkVariants,
	"Chronicles":                   chrVariants,
	"Collectors’ Edition":          ceVariants,
	"Duel Decks: Jace vs. Chandra": dd2Variants,
	"European Land Program":        pelpVariants,
	"Fallen Empires":               femVariants,
	"Fifth Edition":                ed5Variants,
	"Fourth Edition":               ed4Variants,
	"Homelands":                    hmlVariants,
	"Ice Age":                      iceVariants,
	"Intl. Collectors’ Edition":    ceVariants,
	"Limited Edition Alpha":        leaVariants,
	"Limited Edition Beta":         lebVariants,
	"Magic Premiere Shop 2005":     pmpsVariants,
	"Mirage":                       mirVariants,
	"Planeshift":                   plsVariants,
	"Portal Second Age":            po2Variants,
	"Portal":                       porVariants,
	"Revised Edition":              ed3Variants,
	"Secret Lair Drop Series":      sldVariants,
	"Tempest":                      tmpVariants,
	"Unlimited Edition":            lebVariants,
	"Unglued":                      uglVariants,
	"Unstable":                     ustVariants,

	// Custom sets
	"Promotional": promoVariants,
}
