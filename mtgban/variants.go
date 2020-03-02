package mtgban

var allVariants = map[string]map[string]string{
	"Aesthir Glider": map[string]string{
		"Clouds":        "116a",
		"Facing Left":   "116a",
		"AesthirGlider": "116a",
		"Facing Right":  "116b",
		"Moon":          "116b",
		"AZ002":         "116b",
	},
	"Agent of Stromgald": map[string]string{
		"Arms Crossed":        "64a",
		"AgentOfStromgald":    "64a",
		"Holding Staff":       "64b",
		"Woman Holding Staff": "64b",
		"AZ003":               "64b",
	},
	"Arcane Denial": map[string]string{
		"1":            "22a",
		"Axe":          "22a",
		"ArcaneDenial": "22a",
		"2":            "22b",
		"Sword":        "22b",
		"AZ005":        "22b",
	},
	"Astrolabe": map[string]string{
		"Globe/Windows":  "118a",
		"Astrolabe":      "118a",
		"Full Room View": "118a",
		"Map Background": "118b",
		"Close-Up View":  "118b",
		"AZ009":          "118b",
	},
	"Awesome Presence": map[string]string{
		"Arms Spread":        "23a",
		"Open Arms":          "23a",
		"AwsomePresence":     "23a",
		"Man Being Chased":   "23b",
		"Man Being Attacked": "23b",
		"AZ011":              "23b",
	},
	"Balduvian War-Makers": map[string]string{
		"Two Clubs, Side View":    "66a",
		"Sky Background":          "66a",
		"Landscape in Background": "66a",
		"BalduvianWar":            "66a",
		"Green Background":        "66b",
		"AZ016":                   "66b",
	},
	"Benthic Explorers": map[string]string{
		"Sitting on Rocks": "24a",
		"Facing Forward":   "24a",
		"BenthicExplorers": "24a",
		"Swimming":         "24b",
		"Facing Right":     "24b",
		"AZ018":            "24b",
	},
	"Bestial Fury": map[string]string{
		"Facing Front": "67a",
		"BestialFury":  "67a",
		"Facing Left":  "67b",
		"AZ020":        "67b",
	},
	"Carrier Pigeons": map[string]string{
		"Hand":            "1a",
		"Man With Pigeon": "1a",
		"CarrierPigeons":  "1a",
		"Trees":           "1b",
		"Three Pigeons":   "1b",
		"AZ024":           "1b",
	},
	"Casting of Bones": map[string]string{
		"Finger Ornament": "44a",
		"Hand":            "44a",
		"CastingOfBones":  "44a",
		"Hooded Figure":   "44b",
		"AZ027":           "44b",
	},
	"Deadly Insect": map[string]string{
		"Bird":         "86a",
		"Hummingbird":  "86a",
		"DeadlyInsect": "86a",
		"Elf":          "86b",
		"Red Robe":     "86b",
		"AZ030":        "86b",
	},
	"Elvish Ranger": map[string]string{
		"Female":       "88a",
		"ElvishRanger": "88a",
		"Male":         "88b",
		"AZ037":        "88b",
	},
	"Enslaved Scout": map[string]string{
		"Crouching":     "71a",
		"EnslavedScout": "71a",
		"Horses":        "71b",
		"AZ041":         "71b",
	},
	"Errand of Duty": map[string]string{
		"Page Holding Sword": "2a",
		"ErrandOfDuty":       "2a",
		"Bright Red Suit":    "2a",
		"Horse":              "2b",
		"With Horse":         "2b",
		"AZ043":              "2b",
	},
	"False Demise": map[string]string{
		"Cave-In":     "27a",
		"Rocks":       "27a",
		"FalseDemise": "27a",
		"Underwater":  "27b",
		"AZ046":       "27b",
	},
	"Feast or Famine": map[string]string{
		"Falling into Pit": "49a",
		"FeastOrFamine":    "49a",
		"Knife":            "49b",
		"AZ049":            "49b",
	},
	"Fevered Strength": map[string]string{
		"Chains":           "50a",
		"Lifting Rock":     "50a",
		"FeveredStrength":  "50a",
		"Foaming Mouth":    "50b",
		"Foaming at Mouth": "50b",
		"AZ052":            "50b",
	},
	"Foresight": map[string]string{
		"Mermaid With Shiny Shell": "29a",
		"Facing Front":             "29a",
		"Foresight":                "29a",
		"Facing Right":             "29b",
		"White Dress":              "29b",
		"AZ056":                    "29b",
	},
	"Fyndhorn Druid": map[string]string{
		"Facing Left":   "90a",
		"FyndhornDriud": "90a",
		"Facing Front":  "90b",
		"Facing Right":  "90b",
		"AZ058":         "90b",
	},
	"Gift of the Woods": map[string]string{
		"Female":         "92a",
		"giftOfTheWoods": "92a",
		"Male":           "92b",
		"AZ061":          "92b",
	},
	"Gorilla Berserkers": map[string]string{
		"Closed Mouth":      "93a",
		"Peaceful":          "93a",
		"GorillaBerserkers": "93a",
		"Attacking":         "93b",
		"Roaring":           "93b",
		"Holding Spear":     "93b",
		"AZ062":             "93b",
	},
	"Gorilla Chieftain": map[string]string{
		"2 Gorillas":       "94a",
		"Two Gorillas":     "94a",
		"GorillaChieftain": "94a",
		"4 Gorillas":       "94b",
		"Four Gorillas":    "94b",
		"AZ065":            "94b",
	},
	"Gorilla Shaman": map[string]string{
		"Holding Baby":  "72a",
		"Facing Left":   "72a",
		"GorillaShaman": "72a",
		"Skulls":        "72b",
		"Facing Right":  "72b",
		"AZ067":         "72b",
	},
	"Gorilla War Cry": map[string]string{
		"Colorful Headdress": "73a",
		"Leaning Left":       "73a",
		"GOrillaWarCry":      "73a",
		"Red Cub":            "73b",
		"Leaning Right":      "73b",
		"AZ069":              "73b",
	},
	"Guerrilla Tactics": map[string]string{
		"Tripwire":         "74a",
		"Kneeling Knight":  "74a",
		"AZ070":            "74a",
		"Cliff":            "74b",
		"Cliffside":        "74b",
		"GuerrillaTactics": "74b",
	},
	"Insidious Bookworms": map[string]string{
		"Bookshelf":          "51a",
		"InsidiousBookworms": "51a",
		"Single Book":        "51b",
		"AZ077":              "51b",
	},
	"Kjeldoran Escort": map[string]string{
		"Green Blanketed Dog": "7a",
		"Facing Front":        "7a",
		"KjeldoranEscort":     "7a",
		"Red Blanketed Dog":   "7b",
		"Facing Left":         "7b",
		"AZ084":               "7b",
	},
	"Kjeldoran Pride": map[string]string{
		"Bear":            "9a",
		"Facing Left":     "9a",
		"Kjeldoran Pride": "9a",
		"Eagle":           "9b",
		"Facing Right":    "9b",
		"AZ087":           "9b",
	},
	"Lat-Nam's Legacy": map[string]string{
		"Scroll":    "30a",
		"Bookshelf": "30b",
		"Open Book": "30b",
		"Book":      "30b",
	},
	"Lim-Dûl's High Guard": map[string]string{
		"LimDulHighGuard": "55a",
		"AZ095":           "55b",
	},
	"Lim-Dul's High Guard": map[string]string{
		"Red Armor":    "55a",
		"Two Swords":   "55b",
		"Yellow Armor": "55b",
	},
	"Martyrdom": map[string]string{
		"Knight standing, holding sword": "10a",
		"Alive":     "10a",
		"Martyrdom": "10a",
		"Dead":      "10b",
		"Knight wounded on ground": "10b",
		"AZ102":                    "10b",
	},
	"Noble Steeds": map[string]string{
		"Trees in Forefront":  "11a",
		"NobleSteeds":         "11a",
		"Trees":               "11a",
		"Two Steeds Close Up": "11b",
		"Close-Up View":       "11b",
		"AZ110":               "11b",
	},
	"Phantasmal Fiend": map[string]string{
		"Close-Up":        "57a",
		"Close-Up View":   "57a",
		"PhantasmalFiend": "57a",
		"Doorway":         "57b",
		"AZ113":           "57b",
	},
	"Phyrexian Boon": map[string]string{
		"Female":        "58a",
		"PhyrexianBoon": "58a",
		"Male":          "58b",
		"AZ118":         "58b",
	},
	"Phyrexian War Beast": map[string]string{
		"Propeller Left":    "127a",
		"PhyrexianWarBeast": "127a",
		"Propeller Right":   "127b",
		"AZ121":             "127b",
	},
	"Reinforcements": map[string]string{
		"Quote Darien of Kjeldor": "12a",
		"Not Fighting Orc":        "12a",
		"Reinforcements":          "12a",
		"Fighting Orc":            "12b",
		"Quote General Varchild":  "12b",
		"AZ127":                   "12b",
	},
	"Reprisal": map[string]string{
		"Red Dragon":    "13a",
		"AZ129":         "13a",
		"Green Shark":   "13b",
		"Green Monster": "13b",
		"Reprisal":      "13b",
	},
	"Royal Herbalist": map[string]string{
		"Female":         "15a",
		"RoyalHerbalist": "15a",
		"Male":           "15b",
		"AZ133":          "15b",
	},
	"Soldevi Adnate": map[string]string{
		"Female":   "60a",
		"Two Eyes": "60a",
		"Male":     "60b",
		"One Eye":  "60b",
	},
	"Soldevi Heretic": map[string]string{
		"Blue Robe":      "33a",
		"SoldeviHeretic": "33a",
		"Red Robe":       "33b",
		"AZ146":          "33b",
	},
	"Soldevi Sage": map[string]string{
		"2 Candles":         "34a",
		"Male":              "34a",
		"SoldeviSage":       "34a",
		"Female":            "34b",
		"Old Woman Sitting": "34b",
		"AZ149":             "34b",
	},
	"Soldevi Sentry": map[string]string{
		"Close Up":               "132a",
		"Close-Up View":          "132a",
		"SoldeviSentry":          "132a",
		"Attacking":              "132b",
		"Silver Sentry Fighting": "132b",
		"AZ150":                  "132b",
	},
	"Soldevi Steam Beast": map[string]string{
		"Beast in Mountains": "133a",
		"Mountain":           "133a",
		"SoldeviSteamBeast":  "133a",
		"Pink Sun":           "133b",
		"Purple Sun":         "133b",
		"AZ152":              "133b",
	},
	"Stench of Decay": map[string]string{
		"Hand Covering Nose": "61a",
		"Covering Face":      "61a",
		"StenchOfDecay":      "61a",
		"Red Flower":         "61b",
		"Flower":             "61b",
		"Holding Flower":     "61b",
		"AZ157":              "61b",
	},
	"Storm Crow": map[string]string{
		"Flying Left":  "36a",
		"StormCrow":    "36a",
		"Flying Right": "36b",
		"AZ161":        "36b",
	},
	"Storm Shaman": map[string]string{
		"Female":      "81a",
		"StormShaman": "81a",
		"Male":        "81b",
		"AZ163":       "81b",
	},
	"Swamp Mosquito": map[string]string{
		"Black Trees":   "63a",
		"Fallen Tree":   "63a",
		"SwampMosquito": "63a",
		"Brown Trees":   "63b",
		"AZ169":         "63b",
	},
	"Taste of Paradise": map[string]string{
		"One Person Holding Fruit": "100a",
		"Woman Alone":              "100a",
		"TasteOfParadise":          "100a",
		"Man and Woman":            "100b",
		"Two People":               "100b",
		"AZ172":                    "100b",
	},
	"Undergrowth": map[string]string{
		"Fox and Badger": "102a",
		"Fox":            "102a",
		"Undergrowth":    "102a",
		"Elf":            "102b",
		"Holding Axe":    "102b",
		"AZ178":          "102b",
	},
	"Varchild's Crusader": map[string]string{
		"Black Horse and Hollow Log": "82a",
		"Forest":                     "82a",
		"VarchildCrusader":           "82a",
		"Castle":                     "82b",
		"Brown Horse and Castle":     "82b",
		"AZ182":                      "82b",
	},
	"Veteran's Voice": map[string]string{
		"Over the Shoulder":     "84a",
		"Men Standing Together": "84a",
		"VeteranVoice":          "84a",
		"Side-by-Side":          "84b",
		"Men Standing Apart":    "84b",
		"AZ185":                 "84b",
	},
	"Viscerid Armor": map[string]string{
		"Crashing Wave":        "41a",
		"Alone":                "41a",
		"VisceridArmor":        "41a",
		"Humans in Foreground": "41b",
		"Attacking Two Men":    "41b",
		"AZ187":                "41b",
	},
	"Whip Vine": map[string]string{
		"3 Vines":       "103a",
		"No Bird":       "103a",
		"WhipVine":      "103a",
		"Bird":          "103b",
		"Ensnared Bird": "103b",
		"AZ192":         "103b",
	},
	"Wild Aesthir": map[string]string{
		"Blue Mountains":    "21a",
		"WildAethir":        "21a",
		"Green Background":  "21b",
		"Wings Thrown Back": "21b",
		"AZ195":             "21b",
	},
	"Yavimaya Ancients": map[string]string{
		"Rearing Horse":    "104a",
		"Horse":            "104a",
		"YavimayaAncients": "104a",
		"No Horse":         "104b",
		"Trees":            "104b",
		"AZ197":            "104b",
	},
}

var athVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"A":        "84",
		"Arena":    "84",
		"B":        "85",
		"Portal 2": "85",
	},
	"Mountain": map[string]string{
		"A":              "82",
		"Arabian Nights": "82",
		"B":              "83",
		"Mirage":         "83",
	},
	"Plains": map[string]string{
		"A":        "78",
		"Mirage":   "78",
		"B":        "79",
		"Portal 1": "79",
	},
	"Swamp": map[string]string{
		"A":       "80",
		"Ice Age": "80",
		"B":       "81",
		"Tempest": "81",
	},
}

var atqVariants = map[string]map[string]string{
	"Mishra's Factory": map[string]string{
		"AQ044":                    "80a",
		"Spring":                   "80a",
		"b Spring":                 "80a",
		"AQ045":                    "80b",
		"Summer Green":             "80b",
		"Summer":                   "80b",
		"c Summer":                 "80b",
		"atq-80c-mishra-s-factory": "80c",
		"Autumn Orange":            "80c",
		"Autumn":                   "80c",
		"Fall":                     "80c",
		"a Fall":                   "80c",
		"AQ046":                    "80d",
		"Winter Snow":              "80d",
		"Winter":                   "80d",
		"d Winter":                 "80d",
	},
	"Strip Mine": map[string]string{
		"No Sky, No Tower":      "82a",
		"No Sky, No Cave/Tower": "82a",
		"No Horizon":            "82a",
		"atq-82a-strip-mine":    "82a",
		"Sky, Even Terraces":    "82b",
		"Cave":                  "82b",
		"Even Horizon":          "82b",
		"atq-82b-strip-mine":    "82b",
		"No Sky wth Tower":      "82c",
		"Tower":                 "82c",
		"Small Tower":           "82c",
		"atq-82c-strip-mine":    "82c",
		"Sky Uneven Terraces":   "82d",
		"Sky, No Cave or Tower": "82d",
		"Uneven Stripe":         "82d",
		"AQ070":                 "82d",
	},
	"Urza's Mine": map[string]string{
		"Pulley":           "83a",
		"AQ084":            "83a",
		"Mine Face":        "83b",
		"Mouth":            "83b",
		"AQ085":            "83b",
		"Submerged Sphere": "83c",
		"Clawed Sphere":    "83c",
		"Sphere":           "83c",
		"AQ083":            "83c",
		"Tower":            "83d",
		"Ladder":           "83d",
		"AQ086":            "83d",
	},
	"Urza's Power Plant": map[string]string{
		"Sphere with Tubes": "84a",
		"Sphere":            "84a",
		"AQ089":             "84a",
		"Red Columns":       "84b",
		"Columns":           "84b",
		"AQ090":             "84b",
		"Bug":               "84c",
		"AQ088":             "84c",
		"Rock In Pan":       "84d",
		"Rock in Pot":       "84d",
		"Pot":               "84d",
		"AQ091":             "84d",
	},
	"Urza's Tower": map[string]string{
		"Red Leaves":            "85a",
		"Forest":                "85a",
		"AQ092":                 "85a",
		"Sunset with Shoreline": "85b",
		"Shore":                 "85b",
		"AQ093":                 "85b",
		"Plains":                "85c",
		"AQ094":                 "85c",
		"Mountains":             "85d",
		"AQ095":                 "85d",
	},
}

var pal01Variants = map[string]map[string]string{
	"Forest": map[string]string{
		"Arena 2001 Ice Age": "1",
		"Pat Morrissey":      "1",
		"Arena 2002 Beta":    "11",
		"Christopher Rush":   "11",
	},
}

var palpVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"APAC a Japan": "1",
		"APAC b China": "6",
		"APAC c Korea": "11",
	},
	"Island": map[string]string{
		"APAC a Hong Kong": "2",
		"APAC b Japan":     "7",
		"APAC c Singapore": "12",
	},
	"Mountain": map[string]string{
		"APAC a Phillippines":        "3",
		"Phillippines- Rise Terrace": "3",
		"APAC b Taiwan":              "8",
		"APAC c Japan":               "13",
	},
	"Plains": map[string]string{
		"APAC a Japan":     "4",
		"APAC b Australia": "9",
		"APAC c China":     "14",
	},
	"Swamp": map[string]string{
		"APAC a New Zealand":    "5",
		"APAC b Taiwan":         "10",
		"Ron Spears, Fireballs": "10",
		"APAC c Australia":      "15",
		"Ron Spears, Zombie":    "15",
	},
}

var brbVariants = map[string]map[string]string{
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
	"Island": map[string]string{
		"A": "112",
		"B": "114",
		"C": "113",
		"D": "111",
		"E": "110",
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
	"Swamp": map[string]string{
		"A": "133",
		"B": "134",
		"C": "136",
		"D": "135",
	},
}

var chkVariants = map[string]map[string]string{
	"Brothers Yamazaki": map[string]string{
		"Facing Left":  "160a",
		"Facing Right": "160b",
	},
}

var oldLandVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"A":                        "300",
		"A Rocks":                  "300",
		"B - Not Tournament Legal": "300",
		"Rocks":                    "300",
		"B":                        "301",
		"B Path":                   "301",
		"A - Not Tournament Legal": "301",
		"Pathway":                  "301",
		"C":                        "302",
		"C Dark Tree":              "302",
		"C - Not Tournament Legal": "302",
		"Eyes in treehole":         "302",
	},
	"Island": map[string]string{
		"A":                           "291",
		"A Purple":                    "291",
		"B - Not Tournament Legal":    "291",
		"Purple island, gold horizon": "291",
		"B":                             "292",
		"B Light Purple":                "292",
		"A - Not Tournament Legal":      "292",
		"Purple island, orange horizon": "293",
		"C": "293",
		"C - Not Tournament Legal": "293",
		"C Dark Purple":            "293",
		"Green island":             "292",
	},
	"Mountain": map[string]string{
		"A":                         "297",
		"A Small Tree":              "297",
		"B - Not Tournament Legal":  "297",
		"Gray mountains":            "297",
		"B":                         "298",
		"B Snowy":                   "298",
		"A - Not Tournament Legal":  "298",
		"Blue mountains":            "298",
		"C":                         "299",
		"C Dark Red":                "299",
		"C - Not Tournament Legal":  "299",
		"Brown mountain, green sky": "299",
	},
	"Plains": map[string]string{
		"A":                        "289",
		"A Light":                  "288",
		"B - Not Tournament Legal": "288",
		"No trees on plain":        "288",
		"B":                        "288",
		"B Little Trees":           "289",
		"A - Not Tournament Legal": "289",
		"Trees on plain":           "289",
		"C":                        "290",
		"C - Not Tournament Legal": "290",
		"C Dark":                   "290",
		"Cliff in background":      "290",
	},
	"Swamp": map[string]string{
		"A":                        "294",
		"5219":                     "294",
		"A Light":                  "294",
		"B - Not Tournament Legal": "294",
		"One branch in foreground": "294",
		"B": "295",
		"A - Not Tournament Legal": "295",
		"Brown, two branches":      "295",
		"UN248":                    "295",
		"B Two Branches":           "295",
		"C":                        "296",
		"Swamp2":                   "296",
		"C Dark":                   "296",
		"C - Not Tournament Legal": "296",
		"Black, two branches":      "296",
	},
}

var chrVariants = map[string]map[string]string{
	"Urza's Mine": map[string]string{
		"Mine Face":        "114a",
		"Mouth":            "114a",
		"CH094":            "114a",
		"Sphere":           "114b",
		"Submerged Sphere": "114b",
		"Clawed Sphere":    "114b",
		"CH095":            "114b",
		"Pulley":           "114c",
		"CH096":            "114c",
		"Tower":            "114d",
		"Ladder":           "114d",
		"CH097":            "114d",
	},
	"Urza's Power Plant": map[string]string{
		"Pot":               "115a",
		"Rock In Pan":       "115a",
		"Rock in Pot":       "115a",
		"CH098":             "115a",
		"Red Columns":       "115b",
		"Columns":           "115b",
		"CH099":             "115b",
		"Sphere with Tubes": "115c",
		"Sphere":            "115c",
		"CH101":             "115c",
		"Bug":               "115d",
		"CH100":             "115d",
	},
	"Urza's Tower": map[string]string{
		"Sunset with Shoreline": "116a",
		"Forest":                "116a",
		"CH105":                 "116a",
		"Plains":                "116b",
		"CH103":                 "116b",
		"Mountains":             "116c",
		"CH104":                 "116c",
		"Red Leaves":            "116d",
		"Shore":                 "116d",
		"CH102":                 "116d",
	},
}

var cn2Variants = map[string]map[string]string{
	"Kaya, Ghost Assassin": map[string]string{
		"CN2075":               "75",
		"kaya_ghost_assassin2": "222",
	},
}

var dkmVariants = map[string]map[string]string{
	"Guerrilla Tactics": map[string]string{
		"13a - Kneeling Knight": "13a",
		"b": "13b",
	},
	"Phyrexian War Beast": map[string]string{
		"37A - Propeller Right": "37a",
		"b": "37b",
	},
	"Storm Shaman": map[string]string{
		"21a - Female": "21a",
		"21b":          "21b",
	},
	"Yavimaya Ancients": map[string]string{
		"Rearing Horse": "31a",
		"Trees":         "31b",
	},
}

var pelpVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"EURO a Germany":         "1",
		"EURO b France":          "6",
		"EURO Land, Broceliande": "6",
		"Broceliande, France":    "6",
		"Broceliande France":     "6",
		"EURO c United Kingdom":  "11",
	},
	"Island": map[string]string{
		"EURO a Scandanavia":    "2",
		"EURO b Italy":          "7",
		"Venice, Italy":         "7",
		"EURO c United Kingdom": "12",
	},
	"Mountain": map[string]string{
		"EURO a Italy":  "3",
		"EURO b Spain":  "8",
		"EURO c France": "13",
	},
	"Plains": map[string]string{
		"EURO a United Kingdom": "4",
		"EURO b Netherlands":    "9",
		"EURO c Russia":         "14",
	},
	"Swamp": map[string]string{
		"EURO a Belgium":          "5",
		"Ardenees Fagnes Belgium": "5",
		"EURO b United Kingdom":   "10",
		"EURO c France":           "15",
	},
}

var femVariants = map[string]map[string]string{
	"Armor Thrull": map[string]string{
		"FE001": "33a",
		"FE002": "33b",
		"FE003": "33c",
		"FE004": "33d",
	},
	"Basal Thrull": map[string]string{
		"FE005": "34a",
		"FE006": "34b",
		"FE007": "34c",
		"FE008": "34d",
	},
	"Brassclaw Orcs": map[string]string{
		"Rob Alexander, Spear":       "49a",
		"FE100":                      "49a",
		"FE101":                      "49b",
		"Rob Alexander, No Spear":    "49c",
		"Rob Alexander, Gloved Fist": "49c",
		"FE102": "49c",
		"FE103": "49d",
	},
	"Combat Medic": map[string]string{
		"FE133": "1a",
		"FE134": "1b",
		"FE135": "1c",
		"FE136": "1d",
	},
	"Dwarven Soldier": map[string]string{
		"FE107": "53a",
		"FE108": "53b",
		"FE109": "53c",
	},
	"Elven Fortress": map[string]string{
		"FE067": "65a",
		"FE068": "65b",
		"FE069": "65c",
		"FE070": "65d",
	},
	"Elvish Scout": map[string]string{
		"FE075": "68a",
		"FE076": "68b",
		"FE077": "68c",
	},
	"Elvish Hunter": map[string]string{
		"FE072": "67a",
		"FE073": "67b",
		"FE074": "67c",
	},
	"Farrel's Zealot": map[string]string{
		"FE139": "3a",
		"FE140": "3b",
		"FE141": "3c",
	},
	"Goblin Chirurgeon": map[string]string{
		"FE110": "54a",
		"FE111": "54b",
		"FE112": "54c",
	},
	"Goblin Grenade": map[string]string{
		"FE114": "56a",
		"FE115": "56b",
		"FE116": "56c",
	},
	"Goblin War Drums": map[string]string{
		"FE119": "58a",
		"FE120": "58b",
		"FE121": "58c",
		"FE122": "58d",
	},
	"High Tide": map[string]string{
		"FE035": "18a",
		"FE036": "18b",
		"FE037": "18c",
	},
	"Homarid": map[string]string{
		"Tedin": "19c",
		"FE038": "19a",
		"FE039": "19b",
		"FE040": "19c",
		"FE041": "19d",
	},
	"Homarid Warrior": map[string]string{
		"FE044": "22a",
		"FE045": "22b",
		"FE046": "22c",
	},
	"Hymn to Tourach": map[string]string{
		"FE012": "38a",
		"FE013": "38b",
		"FE014": "38c",
		"FE015": "38d",
	},
	"Icatian Infantry": map[string]string{
		"FE144": "7a",
		"FE145": "7b",
		"FE146": "7c",
		"FE147": "7d",
	},
	"Icatian Moneychanger": map[string]string{
		"FE152": "10a",
		"FE153": "10b",
		"FE154": "10c",
	},
	"Icatian Scout": map[string]string{
		"FE157": "13a",
		"FE158": "13b",
		"FE159": "13c",
		"FE160": "13d",
	},
	"Initiates of the Ebon Hand": map[string]string{
		"FE016": "39a",
		"FE017": "39b",
		"FE018": "39c",
	},
	"Merseine": map[string]string{
		"FE047": "23a",
		"FE048": "23b",
		"FE049": "23c",
		"FE050": "23d",
	},
	"Mindstab Thrull": map[string]string{
		"FE019": "40a",
		"FE020": "40b",
		"FE021": "40c",
	},
	"Necrite": map[string]string{
		"FE022": "41a",
		"FE023": "41b",
		"FE024": "41c",
	},
	"Night Soil": map[string]string{
		"FE080": "71a",
		"FE081": "71b",
		"FE082": "71c",
	},
	"Orcish Spy": map[string]string{
		"FE124": "61a",
		"FE125": "61b",
		"FE126": "61c",
	},
	"Orcish Veteran": map[string]string{
		"FE127": "62a",
		"FE128": "62b",
		"FE129": "62c",
		"FE130": "62d",
	},
	"Order of the Ebon Hand": map[string]string{
		"FE025": "42a",
		"FE026": "42b",
		"FE027": "42c",
	},
	"Order of Leitbur": map[string]string{
		"Facing Right, Bryon Wackwitz":  "16a",
		"Bryon Wackwitz, Female Knight": "16a",
		"FE163": "16a",
		"Facing Left, Bryon Wackwitz": "16b",
		"Bryon Wackwitz, Male Knight": "16b",
		"FE164": "16b",
		"FE165": "16c",
	},
	"Spore Cloud": map[string]string{
		"Myrfors": "72b",
		"FE083":   "72a",
		"FE084":   "72b",
		"FE085":   "72c",
	},
	"Thallid": map[string]string{
		"FE087": "74a",
		"FE088": "74b",
		"FE089": "74c",
		"FE090": "74d",
	},
	"Thorn Thallid": map[string]string{
		"FE096": "80a",
		"FE097": "80b",
		"FE098": "80c",
		"FE099": "80d",
	},
	"Tidal Flats": map[string]string{
		"Rob Alexander Lush Horizon":       "27a",
		"Rob Alexander, Lake":              "27a",
		"FE054":                            "27a",
		"Rob Alexander, Sun":               "27b",
		"Rob Alexander Sun Through Clouds": "27b",
		"FE055": "27b",
		"FE056": "27c",
	},
	"Vodalian Mage": map[string]string{
		"Poole": "30b",
		"FE059": "30a",
		"FE060": "30b",
		"FE061": "30c",
	},
	"Vodalian Soldiers": map[string]string{
		"FE062": "31a",
		"FE063": "31b",
		"FE064": "31c",
		"FE065": "31d",
	},
}

var ed5Variants = map[string]map[string]string{
	"Forest": map[string]string{
		"A": "446",
		"B": "448",
		"C": "447",
		"D": "449",
	},
	"Island": map[string]string{
		"A": "434",
		"B": "437",
		"C": "436",
		"D": "435",
	},
	"Mountain": map[string]string{
		"A": "442",
		"B": "445",
		"C": "444",
		"D": "443",
	},
	"Plains": map[string]string{
		"A": "430",
		"B": "432",
		"C": "433",
		"D": "431",
	},
	"Swamp": map[string]string{
		"A": "440",
		"B": "439",
		"C": "441",
		"D": "438",
	},
}

var ed4Variants = map[string]map[string]string{
	"Forest": map[string]string{
		"A Rocks":     "376",
		"B Path":      "377",
		"C Dark Tree": "378",
	},
	"Island": map[string]string{
		"A Purple":       "367",
		"B Light Purple": "368",
		"C Dark Purple":  "369",
	},
	"Mountain": map[string]string{
		"A Small Tree": "373",
		"B Snowy":      "374",
		"C Dark Red":   "375",
	},
	"Plains": map[string]string{
		"A Light":        "364",
		"B Little Trees": "365",
		"C Dark":         "366",
	},
	"Swamp": map[string]string{
		"A Light":        "370",
		"B Two Branches": "371",
		"C Dark":         "372",
	},
}

var prwkVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"Ravnica Weekend Golgari":  "A06",
		"Ravnica Weekend Selesnya": "A09",
	},
	"Island": map[string]string{
		"Ravnica Weekend Dimir": "A01",
		"Ravnica Weekend Izzet": "A03",
	},
	"Mountain": map[string]string{
		"Ravnica Weekend Izzet": "A04",
		"Ravnica Weekend Boros": "A07",
	},
	"Plains": map[string]string{
		"Ravnica Weekend Boros":    "A08",
		"Ravnica Weekend Selesnya": "A10",
	},
	"Swamp": map[string]string{
		"Ravnica Weekend Dimir":   "A02",
		"Ravnica Weekend Golgari": "A05",
	},
}

var hmlVariants = map[string]map[string]string{
	"Abbey Matron": map[string]string{
		"OR137":       "2a",
		"AbbeyMatron": "2b",
		"Hood":        "2a",
		"No Hood":     "2b",
	},
	"Aliban's Tower": map[string]string{
		"Glowing Tower":     "61a",
		"Knights on Horses": "61b",
		"OR006":             "61a",
		"AlibansTower":      "61b",
	},
	"Ambush Party": map[string]string{
		"Doorway":     "63a",
		"Mountain":    "63b",
		"OR007":       "63a",
		"AmbushParty": "63b",
	},
	"Anaba Bodyguard": map[string]string{
		"Alone":          "66a",
		"With Human":     "66b",
		"OR012":          "66a",
		"AnabaBodyguard": "66b",
	},
	"Anaba Shaman": map[string]string{
		"Quote Irini Sngir":      "67a",
		"Facing Left":            "67a",
		"AnabaShaman":            "67a",
		"Facing Right":           "67b",
		"Baki, Wizard Attendant": "67b",
		"OR014":                  "67b",
	},
	"Aysen Bureaucrats": map[string]string{
		"One Bureaucrat":   "3a",
		"Two Bureaucrats":  "3b",
		"OR023":            "3a",
		"AysenBureaucrats": "3b",
	},
	"Carapace": map[string]string{
		"Purple Carapace": "84a",
		"Red Carapace":    "84b",
		"OR033":           "84a",
		"Carapace":        "84b",
	},
	"Cemetery Gate": map[string]string{
		"No Zombie":    "44a",
		"Zombie":       "44b",
		"OR035":        "44a",
		"CemeteryGate": "44b",
	},
	"Dark Maze": map[string]string{
		"Man Kneeling":    "25a",
		"Man Laying Down": "25b",
		"DarkMaze":        "25a",
		"OR043":           "25b",
	},
	"Dry Spell": map[string]string{
		"Fish Bones":      "46a",
		"Skull With Helm": "46b",
		"DrySpell":        "46a",
		"OR050":           "46b",
	},
	"Dwarven Trader": map[string]string{
		"Man and Woman":   "72a",
		"Woman and Horse": "72b",
		"OR054":           "72a",
		"DwarvenTrader":   "72b",
	},
	"Feast of the Unicorn": map[string]string{
		"Goblins":      "47a",
		"Horse Head":   "47b",
		"OR059":        "47a",
		"UnicornFeast": "47b",
	},
	"Folk of An-Havva": map[string]string{
		"Dancing":         "87a",
		"Sitting on Wall": "87b",
		"OR062":           "87a",
		"FolkOfAn":        "87b",
	},
	"Giant Albatross": map[string]string{
		"Ship's Mast":  "27a",
		"Over a Ship":  "27a",
		"OR068":        "27a",
		"Clouds":       "27b",
		"Over Ocean":   "27b",
		"GiantAlbatro": "27b",
	},
	"Hungry Mist": map[string]string{
		"Lantern":    "88a",
		"No Lantern": "88b",
		"HungryMist": "88a",
		"OR075":      "88b",
	},
	"Labyrinth Minotaur": map[string]string{
		"Maze Background":   "30a",
		"Black Background":  "30b",
		"OR086":             "30a",
		"LabyrinthMinotaur": "30b",
	},
	"Memory Lapse": map[string]string{
		"Female":                     "32a",
		"Quote Chandler, Female Art": "32a",
		"OR093": "32a",
		"Male":  "32b",
		"Quote Reveka, Wizard Savant, Male Art": "32b",
		"MemoryLapse":                           "32b",
	},
	"Mesa Falcon": map[string]string{
		"Flying":     "10a",
		"Perched":    "10b",
		"OR003":      "10a",
		"MesaFalcon": "10b",
	},
	"Samite Alchemist": map[string]string{
		"Window":          "13a",
		"Green Beaker":    "13b",
		"OR112":           "13a",
		"SamiteAlchemist": "13b",
	},
	"Reef Pirates": map[string]string{
		"Zombies":     "36a",
		"Ships":       "36b",
		"OR104":       "36a",
		"ReefPirates": "36b",
	},
	"Sengir Bats": map[string]string{
		"Perched":    "57a",
		"Perching":   "57a",
		"OR118":      "57a",
		"Flying":     "57b",
		"SengirBats": "57b",
	},
	"Shrink": map[string]string{
		"Man in Shadow": "97a",
		"Giant Woman":   "97b",
		"OR125":         "97a",
		"Shrink":        "97b",
	},
	"Torture": map[string]string{
		"Man with Markings": "59a",
		"Hooded Figure":     "59b",
		"OR086":             "59a",
		"Torture":           "59b",
	},
	"Trade Caravan": map[string]string{
		"Giraffe on Right": "19a",
		"No Moon":          "19a",
		"TradeCaravan":     "19a",
		"Moon":             "19b",
		"Moon Top Left":    "19b",
		"OR132":            "19b",
	},
	"Willow Faerie": map[string]string{
		"Male":         "99a",
		"Female":       "99b",
		"OR137":        "99a",
		"WillowFaerie": "99b",
	},
}

var iceVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"A": "380",
		"B": "381",
		"C": "382",
	},
	"Island": map[string]string{
		"A": "368",
		"B": "369",
		"C": "370",
	},
	"Mountain": map[string]string{
		"A": "376",
		"B": "377",
		"C": "378",
	},
	"Plains": map[string]string{
		"A": "364",
		"B": "365",
		"C": "366",
	},
	"Swamp": map[string]string{
		"A": "373",
		"B": "374",
		"C": "375",
	},
}

var itpVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"A Rocks":     "65",
		"B Path":      "66",
		"C Dark Tree": "67",
	},
	"Island": map[string]string{
		"A Purple":       "56",
		"B Light Purple": "57",
		"C Dark Purple":  "58",
	},
	"Mountain": map[string]string{
		"A Small Tree": "62",
		"B Snowy":      "63",
		"C Dark Red":   "64",
	},
	"Plains": map[string]string{
		"A Light":        "53",
		"B Little Trees": "54",
		"C Dark":         "55",
	},
	"Swamp": map[string]string{
		"A Light":        "59",
		"B Two Branches": "60",
		"C Dark":         "61",
	},
}

var leaVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"A Rocks": "294",
		"B Path":  "295",
	},
	"Island": map[string]string{
		"A Purple":       "288",
		"B Light Purple": "289",
	},
	"Mountain": map[string]string{
		"A Small Tree": "292",
		"B Snowy":      "293",
	},
	"Plains": map[string]string{
		"A Light":        "286",
		"B Little Trees": "287",
	},
	"Swamp": map[string]string{
		"A Light":        "290",
		"B Two Branches": "291",
	},
}

var pmpsVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"The Golgari Swarm":     "303",
		"The Gruul Clans":       "305",
		"The Selesnya Conclave": "304",
		"The Simic Combine":     "306",
	},
	"Island": map[string]string{
		"The Azorius Senate": "291",
		"The House Dimir":    "294",
		"The Izzet League":   "292",
		"The Simic Combine":  "293",
	},
	"Mountain": map[string]string{
		"The Boros Legion":   "302",
		"The Gruul Clans":    "300",
		"The Izzet League":   "301",
		"The Cult of Rakdos": "299",
	},
	"Plains": map[string]string{
		"The Azorius Senate":    "289",
		"The Boros Legion":      "290",
		"The Orzhov Syndicate":  "287",
		"The Selesnya Conclave": "288",
	},
	"Swamp": map[string]string{
		"The House Dimir":      "296",
		"The Golgari Swarm":    "295",
		"The Orzhov Syndicate": "297",
		"The Cult of Rakdos":   "298",
	},
}

var mirVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"A": "347",
		"B": "348",
		"C": "349",
		"D": "350",
	},
	"Island": map[string]string{
		"A": "335",
		"B": "336",
		"C": "337",
		"D": "338",
	},
	"Mountain": map[string]string{
		"A": "343",
		"B": "344",
		"C": "345",
		"D": "346",
	},
	"Plains": map[string]string{
		"A": "331",
		"B": "332",
		"C": "333",
		"D": "334",
	},
	"Swamp": map[string]string{
		"A": "339",
		"B": "340",
		"C": "341",
		"D": "342",
	},
}

var ogwVariants = map[string]map[string]string{
	"Wastes": map[string]string{
		"Full Art 183": "183",
		"OGW183":       "183",
		"183 Intro":    "183a",
		"OGW185":       "183a",
		"Full Art 184": "184",
		"OGW184":       "184",
		"184 Intro":    "184a",
		"OGW186":       "184a",
	},
}

var plsVariants = map[string]map[string]string{
	"Ertai, the Corrupted": map[string]string{
		"":                   "107",
		"Alternate Art":      "107★",
		"Alternate Art Foil": "107★",
	},
	"Skyship Weatherlight": map[string]string{
		"":                   "133",
		"Alternate Art":      "133★",
		"Alternate Art Foil": "133★",
	},
	"Tahngarth, Talruum Hero": map[string]string{
		"":                         "74",
		"Alternate Art":            "74★",
		"Alternate Art Foil":       "74★",
		"Alternate Planeshift Art": "74★",
	},
}

var po2Variants = map[string]map[string]string{
	"Forest": map[string]string{
		"A": "163",
		"B": "164",
		"C": "165",
	},
	"Island": map[string]string{
		"A": "154",
		"B": "155",
		"C": "156",
	},
	"Mountain": map[string]string{
		"A": "160",
		"B": "161",
		"C": "162",
	},
	"Plains": map[string]string{
		"A": "151",
		"B": "152",
		"C": "153",
	},
	"Swamp": map[string]string{
		"A": "157",
		"B": "158",
		"C": "159",
	},
}

var porVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"A": "212",
		"B": "213",
		"C": "214",
		"D": "215",
	},
	"Island": map[string]string{
		"A": "200",
		"B": "201",
		"C": "202",
		"D": "203",
	},
	"Mountain": map[string]string{
		"A": "208",
		"B": "209",
		"C": "210",
		"D": "211",
	},
	"Plains": map[string]string{
		"A": "196",
		"B": "197",
		"C": "198",
		"D": "199",
	},
	"Swamp": map[string]string{
		"A": "204",
		"B": "205",
		"C": "206",
		"D": "207",
	},
}

var ptcVariants = map[string]map[string]string{
	"Circle of Protection: Green": map[string]string{
		"4th Edition - Sideboard - Michael Locanto":  "ml16sb",
		"4th Edition - Sideboard - Bertrand Lestree": "bl16sb",
		"Ice Age - Sideboard - Bertrand Lestree":     "bl14sb",
		"Ice Age - Sideboard - Michael Locanto":      "ml14sb",
	},
	"Circle of Protection: Red": map[string]string{
		"4th Edition - Sideboard - Michael Locanto":  "ml17sb",
		"4th Edition - Sideboard - Bertrand Lestree": "bl17sb",
		"Ice Age - Sideboard - Bertrand Lestree":     "bl15sb",
		"Ice Age - Sideboard - Michael Locanto":      "ml15sb",
	},
	"Forest": map[string]string{
		"4th Edition Path - Bertrand Lestree - 1996":      "bl377",
		"4th Edition Path - Preston Poulter - 1996":       "pp377",
		"4th Edition Rock - Bertrand Lestree - 1996":      "bl376",
		"4th Edition Rock - Preston Poulter - 1996":       "pp376",
		"4th Edition Dark Tree - Preston Poulter - 1996":  "pp378",
		"4th Edition Dark Tree - Bertrand Lestree - 1996": "bl378",
	},
	"Island": map[string]string{
		"4th Edition Purple - Michael Locanto - 1996":            "ml367",
		"4th Edition Light Purple - Michael Locanto - 1996":      "ml368",
		"4th Edition Light Purple - Shawn Hammer Regnier - 1996": "shr368",
		"4th Edition Dark Purple - Shawn Hammer Regnier - 1996":  "shr369",
		"4th Edition Dark Purple - Michael Locanto - 1996":       "ml369",
	},
	"Mountain": map[string]string{
		"4th Edition - Little Trees - Eric Tam":     "et373",
		"4th Edition - Little Trees - Mark Justice": "mj373",
		"4th Edition - Snowy - Eric Tam":            "et374",
		"4th Edition - Snowy - Mark Justice":        "mj374",
		"4th Edition - Dark Red - Eric Tam":         "et375",
		"4th Edition - Dark Red - Mark Justice":     "mj375",
	},
	"Plains": map[string]string{
		"4th Edition Light - Bertrand Lestree - 1996":            "bl364",
		"4th Edition Light - Michael Locanto - 1996":             "ml364",
		"4th Edition Light - Preston Poulter - 1996":             "pp364",
		"4th Edition Light - Shawn Hammer Regnier - 1996":        "shr364",
		"4th Edition Little Trees - Bertrand Lestree - 1996":     "bl365",
		"4th Edition Little Trees - Eric Tam - 1996":             "et365",
		"4th Edition Little Trees - Mark Justice - 1996":         "mj365",
		"4th Edition Little Trees - Michael Locanto - 1996":      "ml365",
		"4th Edition Little Trees - Preston Poulter - 1996":      "pp365",
		"4th Edition Little Trees - Shawn Hammer Regnier - 1996": "shr365",
		"4th Edition Dark - Bertrand Lestree - 1996":             "bl366",
		"4th Edition Dark - Eric Tam - 1996":                     "et366",
		"4th Edition Dark - Mark Justice - 1996":                 "mj366",
		"4th Edition Dark - Michael Locanto - 1996":              "ml366",
		"4th Edition Dark - Preston Poulter - 1996":              "pp366",
		"4th Edition Dark - Shawn Hammer Regnier - 1996":         "shr366",
	},
	"Swamp": map[string]string{
		"4th Edition Light - George Baxter - 1996":        "gb370",
		"4th Edition Light - Leon Lindback - 1996":        "ll370",
		"4th Edition Two Branches - Leon Lindback - 1996": "ll371",
		"4th Edition Two Branches - George Baxter - 1996": "gb371",
		"4th Edition Dark - Leon Lindback - 1996":         "ll372",
		"4th Edition Dark - George Baxter - 1996":         "gb372",
	},
	"Memory Lapse": map[string]string{
		"Statue A - Sideboard - Shawn Hammer Regnier": "shr32bsb",
		"Puzzle B - Sideboard - Shawn Hammer Regnier": "shr32asb",
	},
}

var rinVariants = map[string]map[string]string{
	"Urza's Mine": map[string]string{
		"RI053":     "175",
		"miniera53": "176",
		"miniera56": "177",
		"miniera54": "178",
	},
	"Urza's Power Plant": map[string]string{
		"centrale57": "179",
		"centrale60": "180",
		"centrale58": "181",
		"RI054":      "182",
	},
	"Urza's Tower": map[string]string{
		"RI055":   "183",
		"torre63": "184",
		"torre62": "185",
		"torre64": "186",
	},
}

var prw2Variants = map[string]map[string]string{
	"Forest": map[string]string{
		"Ravnica Weekend Gruul": "B08",
		"Ravnica Weekend Simic": "B09",
	},
	"Island": map[string]string{
		"Ravnica Weekend Azorius": "B02",
		"Ravnica Weekend Simic":   "B10",
	},
	"Mountain": map[string]string{
		"Ravnica Weekend Rakdos": "B06",
		"Ravnica Weekend Gruul":  "B07",
	},
	"Plains": map[string]string{
		"Ravnica Weekend Azorius": "B01",
		"Ravnica Weekend Orzhov":  "B03",
	},
	"Swamp": map[string]string{
		"Ravnica Weekend Orzhov": "B04",
		"Ravnica Weekend Rakdos": "B05",
	},
}

var ed3Variants = map[string]map[string]string{
	"Forest": map[string]string{
		"A Rocks":     "304",
		"A":           "304",
		"B Path":      "305",
		"B":           "305",
		"C Dark Tree": "306",
		"C":           "306",
	},
	"Island": map[string]string{
		"A Purple":       "295",
		"A":              "295",
		"B Light Purple": "296",
		"B":              "296",
		"C Dark Purple":  "297",
		"C":              "297",
	},
	"Mountain": map[string]string{
		"A Small Tree": "301",
		"A":            "301",
		"B Snowy":      "302",
		"B":            "302",
		"C Dark Red":   "303",
		"C":            "303",
	},
	"Plains": map[string]string{
		"A Light":        "292",
		"A":              "292",
		"B Little Trees": "293",
		"B":              "293",
		"C Dark":         "294",
		"C":              "294",
	},
	"Swamp": map[string]string{
		"A Light":        "298",
		"A":              "298",
		"B Two Branches": "299",
		"B":              "299",
		"C Dark":         "300",
		"C":              "300",
	},
}

var sldVariants = map[string]map[string]string{
	"Serum Visions": map[string]string{
		"Secret Lair 29 Collantes": "29",
		"Secret Lair 30 DXTR":      "30",
		"Secret Lair 31 YS":        "31",
		"Secret Lair 32 Zuverza":   "32",
	},
}

var soiVariants = map[string]map[string]string{
	"Tamiyo's Journal": map[string]string{
		"Entry 434": "265",
		"Entry 546": "265†",
		"Entry 653": "265†a",
		"Entry 711": "265†b",
		"Entry 855": "265†c",
		"Entry 922": "265†d",
	},
}

var psoiVariants = map[string]map[string]string{
	"Tamiyo's Journal": map[string]string{
		"Shadows over Innistrad Prerelease": "265s†",
	},
}

var tmpVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"A": "347",
		"B": "348",
		"C": "349",
		"D": "350",
	},
	"Island": map[string]string{
		"A": "335",
		"B": "336",
		"C": "337",
		"D": "338",
	},
	"Mountain": map[string]string{
		"A": "343",
		"B": "344",
		"C": "345",
		"D": "346",
	},
	"Plains": map[string]string{
		"A": "331",
		"B": "332",
		"C": "333",
		"D": "334",
	},
	"Swamp": map[string]string{
		"A": "339",
		"B": "340",
		"C": "341",
		"D": "342",
	},
}

var ustVariants = map[string]map[string]string{
	"Amateur Auteur": map[string]string{
		"Ravnica":   "3a",
		"Innistrad": "3b",
		"Theros":    "3c",
		"Zendikar":  "3d",
	},
	"Beast in Show": map[string]string{
		"Tyrranax":  "103a",
		"Gnarlid":   "103b",
		"Baloth":    "103c",
		"Thragtusk": "103d",
	},
	"Everythingamajig": map[string]string{
		"Move":      "147a",
		"Draw":      "147b",
		"Flip":      "147c",
		"Add":       "147d",
		"Sacrifice": "147e",
		"Scry":      "147f",
	},
	"Extremely Slow Zombie": map[string]string{
		"Summer": "54a",
		"Fall":   "54b",
		"Winter": "54c",
		"Spring": "54d",
	},
	"Garbage Elemental": map[string]string{
		"2/4": "82a",
		"3/1": "82b",
		"3/2": "82c",
		"3/3": "82d",
		"4/3": "82e",
		"6/5": "82f",
	},
	"Ineffable Blessing": map[string]string{
		"Choose flavor":      "113a",
		"Choose artist":      "113b",
		"Choose border":      "113c",
		"Choose rarity":      "113d",
		"Choose odd or even": "113e",
		"Choose a number":    "113f",
	},
	"Knight of the Kitchen Sink": map[string]string{
		"Black borders":          "12a",
		"Even collector numbers": "12b",
		"Loose lips":             "12c",
		"Odd collector numbers":  "12d",
		"Two-word names":         "12e",
		"Watermarks":             "12f",
	},
	"Novellamental": map[string]string{
		"''My grandmother…''": "41a",
		"''This pendant…''":   "41b",
		"''The chain…''":      "41c",
		"''My heart…''":       "41d",
	},
	"Secret Base": map[string]string{
		"Order of the Widget":      "165a",
		"Version 2":                "165b",
		"Agents of S.N.E.A.K.":     "165b",
		"Crossbreed Labs":          "165e",
		"Goblin Explosioneers":     "165d",
		"League of Dastardly Doom": "165c",
	},
	"Sly Spy": map[string]string{
		"Reveal hand":                   "67a",
		"Destroy creature facing left":  "67b",
		"Lose a finger":                 "67c",
		"Destroy creature facing right": "67d",
		"Reveal top of library":         "67e",
		"Roll six-sided die":            "67f",
	},
	"Target Minotaur": map[string]string{
		"Ice":   "98a",
		"Rain":  "98b",
		"Fire":  "98c",
		"Roots": "98d",
	},
	"Very Cryptic Command": map[string]string{
		"Switch) [Alternate Art]": "49a",
		"Untap":                   "49b",
		"Draw":                    "49c",
		"Return":                  "49d",
		"Counter":                 "49e",
		"Scry":                    "49f",
	},

	"Hazmat Suit (Used)": map[string]string{
		"Used": "57",
	},

	"Curious Killbot": map[string]string{
		"": "145a",
	},
	"Delighted Killbot": map[string]string{
		"": "145b",
	},
	"Despondent Killbot": map[string]string{
		"": "145c",
	},
	"Enraged Killbot": map[string]string{
		"": "145d",
	},
}

var wc99Variants = map[string]map[string]string{
	"Mountain": map[string]string{
		"6th Edition 346 - Mark Le Pine - 1999": "mlp346a",
		"Urza's Saga 346 - Mark Le Pine - 1999": "mlp346b",
	},
	"Swamp": map[string]string{
		"Urza's Saga 346 - Jakub Slemr - 1999": "js340",
		"Tempest B - Jakub Slemr - 1999":       "js340a",
		"6th Edition 340 - Jakub Slemr - 1999": "js340b",
	},
	"Forest": map[string]string{
		"6th Edition 347 - Matt Linde - 1999": "ml347b",
	},
}

var wc00Variants = map[string]map[string]string{
	"Phyrexian Colossus": map[string]string{
		"Jon Finkel - 2000": "jf306",
	},
}

var wc01Variants = map[string]map[string]string{
	"Forest": map[string]string{
		"Mercadian Masques 347 - Jan Tomcani - 2001": "jt347a",
		"Mercadian Masques 348 - Jan Tomcani - 2001": "jt348",
		"Invasion 347 - Jan Tomcani - 2001":          "jt347",
		"Invasion 348 - Jan Tomcani - 2001":          "jt348a",
		"Invasion 349 - Jan Tomcani - 2001":          "jt349a",
	},
	"Island": map[string]string{
		"7th Edition 335 - Antoine Ruel - 2001":       "ar335",
		"Invasion 335 - Alex Borteh - 2001":           "ab335a",
		"Invasion 335 - Antoine Ruel - 2001":          "ar335a",
		"Invasion 336 - Alex Borteh - 2001":           "ab336a",
		"Invasion 336 - Antoine Ruel - 2001":          "ar336",
		"Invasion 337 - Alex Borteh - 2001":           "ab337",
		"Invasion 338 - Alex Borteh - 2001":           "ab338a",
		"Mercadian Masques 335 - Alex Borteh - 2001":  "ab335b",
		"Mercadian Masques 335 - Antoine Ruel - 2001": "ar335b",
		"Mercadian Masques 336 - Alex Borteh - 2001":  "ab336",
		"Mercadian Masques 336 - Antoine Ruel - 2001": "ar336a",
		"Mercadian Masques 337 - Alex Borteh - 2001":  "ab337a",
		"Mercadian Masques 338 - Alex Borteh - 2001":  "ab338",
	},
	"Mountain": map[string]string{
		"Mercadian Masques 343 - Jan Tomcani - 2001":     "jt343a",
		"Invasion 343 - Tom van de Logt - 2001":          "tvdl343",
		"Mercadian Masques 343 - Tom van de Logt - 2001": "tvdl343b",
	},
	"Swamp": map[string]string{
		"Invasion 339 - Tom van de Logt - 2001":          "tvdl339",
		"Mercadian Masques 339 - Tom van de Logt - 2001": "tvdl339a",
	},
	"Counterspell": map[string]string{
		"7th Edition Antoine Ruel - 2001": "ar67",
	},
	"Misdirection": map[string]string{
		"7th Edition Antoine Ruel - 2001": "ab87asb",
	},
}

var wc02Variants = map[string]map[string]string{
	"Island": map[string]string{
		"7th Edition 335 - Carlos Romao - 2002": "cr335b",
		"Invasion 335 - Carlos Romao - 2002":    "cr335a",
		"Invasion 336 - Raphael Levy - 2002":    "rl336",
		"Invasion 337 - Carlos Romao - 2002":    "cr337",
		"Invasion 337 - Raphael Levy - 2002":    "rl337a",
		"Odyssey 336 - Raphael Levy - 2002":     "rl336a",
		"Odyssey 336 - Sim Han How - 2002":      "shh336a",
		"Odyssey 337 - Carlos Romao - 2002":     "cr337a",
	},
	"Plains": map[string]string{
		"Odyssey 331 - Brian Kibler - 2002": "bk331a",
	},
}

var VariantsTable = map[string]map[string]map[string]string{
	"Alliances":                     allVariants,
	"Anthologies":                   athVariants,
	"Antiquities":                   atqVariants,
	"Arena League 2001":             pal01Variants,
	"Asia Pacific Land Program":     palpVariants,
	"Battle Royale Box Set":         brbVariants,
	"Champions of Kamigawa":         chkVariants,
	"Chronicles":                    chrVariants,
	"Collectors’ Edition":           oldLandVariants,
	"Conspiracy: Take the Crown":    cn2Variants,
	"Deckmasters":                   dkmVariants,
	"European Land Program":         pelpVariants,
	"Fallen Empires":                femVariants,
	"Fifth Edition":                 ed5Variants,
	"Fourth Edition":                ed4Variants,
	"GRN Ravnica Weekend":           prwkVariants,
	"Homelands":                     hmlVariants,
	"Ice Age":                       iceVariants,
	"Intl. Collectors’ Edition":     oldLandVariants,
	"Introductory Two-Player Set":   itpVariants,
	"Limited Edition Alpha":         leaVariants,
	"Limited Edition Beta":          oldLandVariants,
	"Magic Premiere Shop 2005":      pmpsVariants,
	"Mirage":                        mirVariants,
	"Oath of the Gatewatch":         ogwVariants,
	"Planeshift":                    plsVariants,
	"Portal Second Age":             po2Variants,
	"Portal":                        porVariants,
	"Pro Tour Collector Set":        ptcVariants,
	"Rinascimento":                  rinVariants,
	"RNA Ravnica Weekend":           prw2Variants,
	"Revised Edition":               ed3Variants,
	"Secret Lair Drop":              sldVariants,
	"Shadows over Innistrad":        soiVariants,
	"Shadows over Innistrad Promos": psoiVariants,
	"Tempest":                       tmpVariants,
	"Unlimited Edition":             oldLandVariants,
	"Unstable":                      ustVariants,
	"World Championship Decks 1999": wc99Variants,
	"World Championship Decks 2000": wc00Variants,
	"World Championship Decks 2001": wc01Variants,
	"World Championship Decks 2002": wc02Variants,
}
