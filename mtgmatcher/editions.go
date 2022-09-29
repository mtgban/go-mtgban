package mtgmatcher

var EditionTable = map[string]string{
	// Main expansions
	"10th Edition":         "Tenth Edition",
	"3rd Edition":          "Revised Edition",
	"3rd Edition/Revised":  "Revised Edition",
	"4th Edition":          "Fourth Edition",
	"5th Edition":          "Fifth Edition",
	"6th Edition":          "Classic Sixth Edition",
	"6th Edition Classic":  "Classic Sixth Edition",
	"Sixth Edition":        "Classic Sixth Edition",
	"7th Edition":          "Seventh Edition",
	"8th Edition":          "Eighth Edition",
	"8th Starter":          "Eighth Edition",
	"9th Edition":          "Ninth Edition",
	"Alpha":                "Limited Edition Alpha",
	"Alpha Edition":        "Limited Edition Alpha",
	"Beta":                 "Limited Edition Beta",
	"Beta Edition":         "Limited Edition Beta",
	"Classic 6th Edition":  "Classic Sixth Edition",
	"Origins":              "Magic Origins",
	"Ravnica":              "Ravnica: City of Guilds",
	"Revised":              "Revised Edition",
	"Revised 3rd Edition":  "Revised Edition",
	"Unlimited":            "Unlimited Edition",
	"Summer Magic (Edgar)": "Summer Magic / Edgar",
	"Summer Magic":         "Summer Magic / Edgar",

	"Universes Beyond: Warhammer 40,000": "Warhammer 40,000",

	// PL22
	"APAC Year of the Tiger":     "Year of the Tiger 2022",
	"Textless Year of the Tiger": "Year of the Tiger 2022",
	"Year of the Tiger":          "Year of the Tiger 2022",

	// Double Feature
	"Innistrad: Double Feature - Crimson Vow":   "Innistrad: Double Feature",
	"Innistrad: Double Feature - Midnight Hunt": "Innistrad: Double Feature",

	// Adventures in the Forgotten Realms Ampersand
	"AFR Ampersand Promos": "Adventures in the Forgotten Realms Promos",
	"Ampersand Foil":       "Adventures in the Forgotten Realms Promos",
	"Ampersand PROMOS":     "Adventures in the Forgotten Realms Promos",
	"Ampersand":            "Adventures in the Forgotten Realms Promos",
	"D&D: Adventures in the Forgotten Realms":           "Adventures in the Forgotten Realms",
	"D&D: Adventures in the Forgotten Realms: Variants": "Adventures in the Forgotten Realms: Variants",

	// Strixhaven Mystical Archive alts
	"Mystical Archive":                          "Strixhaven Mystical Archive",
	"Mystical Archive: Japanese alternate-art":  "Strixhaven Mystical Archive",
	"Strixhaven Mystical Archive - Foil Etched": "Strixhaven Mystical Archive",
	"Strixhaven Mystical Archive JPN":           "Strixhaven Mystical Archive",
	"Strixhaven Mystical Archives":              "Strixhaven Mystical Archive",
	"Strixhaven: Mystical Archives":             "Strixhaven Mystical Archive",
	"Strixhaven: School of Mages Etched":        "Strixhaven Mystical Archive",
	"Strixhaven: School of Mages Japanese":      "Strixhaven Mystical Archive",

	// JPN planeswalkers and similar
	"War of the Spark JPN Planeswalkers":                          "War of the Spark",
	"War of the Spark Japanese Alt Art":                           "War of the Spark",
	"War of the Spark Japanese Alternate Art":                     "War of the Spark",
	"War of the Spark: Japanese alternate-art Planeswalker Promo": "War of the Spark Promos",
	"War of the Spark: Japanese alternate-art Planeswalker":       "War of the Spark",

	// Gift pack
	"2017 Gift Pack":       "2017 Gift Pack",
	"2018 Gift Pack":       "M19 Gift Pack",
	"Gift Box 2017":        "2017 Gift Pack",
	"Gift Pack 2017":       "2017 Gift Pack",
	"Gift Pack 2018":       "M19 Gift Pack",
	"Gift Pack 2017 Promo": "2017 Gift Pack",
	"Gift Pack 2018 Promo": "M19 Gift Pack",
	"Shooting Star Promo":  "2017 Gift Pack",
	"Mark Poole Art Promo": "2017 Gift Pack",
	"Poole 2017 Gift Pack": "2017 Gift Pack",

	// Treasure Chest
	"Ixalan Treasure Chest": "XLN Treasure Chest",
	"Treasure Chest Promo":  "XLN Treasure Chest",
	"Treasure Map":          "XLN Treasure Chest",

	// Game Night
	"Game Night 2018":               "Game Night",
	"Game Night 2019":               "Game Night: 2019",
	"Magic Game Night":              "Game Night",
	"Magic Game Night 2019":         "Game Night 2019",
	"Game Night: 2018":              "Game Night",
	"Game Night: 2019":              "Game Night: 2019",
	"Magic Game Night 2018 Box Set": "Game Night",
	"Magic Game Night 2019 Box Set": "Game Night: 2019",

	// Old school lands
	"Apac Lands":       "Asia Pacific Land Program",
	"Apac Land Promos": "Asia Pacific Land Program",
	"APAC Land":        "Asia Pacific Land Program",
	"APAC Lands":       "Asia Pacific Land Program",
	"GURU":             "Guru",
	"Guru Land":        "Guru",
	"Guru Lands":       "Guru",
	"Guru":             "Guru",
	"Euro Land Promos": "European Land Program",
	"Euro Lands":       "European Land Program",
	"European Lands":   "European Land Program",

	// Secret Lair extra cards
	"SLD Stained Glass Promo": "Secret Lair Drop",
	"Stained Glass":           "Secret Lair Drop",
	"Stained Glass Art":       "Secret Lair Drop",
	"Stained Glass Promo":     "Secret Lair Drop",

	// Ponies
	"Ponies: The Galloping": "Ponies: The Galloping",

	// Champs
	"Champs / States Promo":  "Champs and States",
	"Champs":                 "Champs and States",
	"Champs Promos":          "Champs and States",
	"Champs & States Promos": "Champs and States",

	// Welcome decks
	"Shadows over Innistrad Welcome Deck": "Welcome Deck 2016",
	"Amonkhet Welcome Deck":               "Welcome Deck 2017",
	"Magic 2016":                          "Welcome Deck 2016",
	"Magic 2017":                          "Welcome Deck 2017",
	"Welcome Deck 2016":                   "Welcome Deck 2016",
	"Welcome Deck 2017":                   "Welcome Deck 2017",

	// Holiday cards
	"Happy Holidays":     "Happy Holidays",
	"Holiday Foil":       "Happy Holidays",
	"Holiday Promo":      "Happy Holidays",
	"WOTC Employee Card": "Happy Holidays",

	// Standard Series
	"Standard Series":                          "BFZ Standard Series",
	"Standard Series Promo":                    "BFZ Standard Series",
	"Standard Series Promos":                   "BFZ Standard Series",
	"2017 Standard Showdown":                   "XLN Standard Showdown",
	"2018 Standard Showdown":                   "M19 Standard Showdown",
	"2017 Standard Showdown Guay":              "XLN Standard Showdown",
	"2018 Standard Showdown Danner":            "M19 Standard Showdown",
	"Rebecca Guay Standard Showdown 2017":      "XLN Standard Showdown",
	"Alayna Danner Standard Showdown 2018":     "M19 Standard Showdown",
	"Rebecca Guay Standard Showdown":           "XLN Standard Showdown",
	"Alayna Danner Standard Showdown":          "M19 Standard Showdown",
	"Standard Showdown 2017":                   "XLN Standard Showdown",
	"Standard Showdown 2018":                   "M19 Standard Showdown",
	"Standard Showdown Rebecca Guay":           "XLN Standard Showdown",
	"Standard Showdown Alayna Danner":          "M19 Standard Showdown",
	"Alayna Danner Art":                        "M19 Standard Showdown",
	"Rebecca Guay Art Standard Showdown Promo": "XLN Standard Showdown",
	"Ixalan Standard Showdown":                 "XLN Standard Showdown",
	"Illus.Alayna Danner":                      "M19 Standard Showdown",
	"Illus.Rebecca Guay":                       "XLN Standard Showdown",

	// Modern Masters
	"Modern Masters 2013":            "Modern Masters",
	"Modern Masters 2013 Edition":    "Modern Masters",
	"Modern Masters 2015 Edition":    "Modern Masters 2015",
	"Modern Masters 2017 Edition":    "Modern Masters 2017",
	"Modern Masters: 2013 Edition":   "Modern Masters",
	"Modern Masters: 2015 Edition":   "Modern Masters 2015",
	"Modern Masters: 2017 Edition":   "Modern Masters 2017",
	"Ultimate Box Toppers":           "Ultimate Box Topper",
	"Ultimate Masters - Box Toppers": "Ultimate Box Topper",
	"Ultimate Masters - Variants":    "Ultimate Box Topper",
	"Ultimate Masters Box Topper":    "Ultimate Box Topper",
	"Ultimate Masters Box Toppers":   "Ultimate Box Topper",
	"Ultimate Masters: Box Topper":   "Ultimate Box Topper",
	"Ultimate Masters: Box Toppers":  "Ultimate Box Topper",
	"Modern Horizons - Retro Frames": "Modern Horizons 1 Timeshifts",
	"Modern Horizons II":             "Modern Horizons 2",

	// CE and IE editions
	"Collector's Edition - International": "Intl. Collectors' Edition",
	"Collectors Ed Intl":                  "Intl. Collectors' Edition",
	"Collectors' Edition - International": "Intl. Collectors' Edition",
	"International Collector's Edition":   "Intl. Collectors' Edition",
	"International Collectors Edition":    "Intl. Collectors' Edition",
	"International Collectors’ Edition":   "Intl. Collectors' Edition",
	"International Edition":               "Intl. Collectors' Edition",
	"Collector's Edition (Domestic)":      "Collectors' Edition",
	"Collector's Edition - Domestic":      "Collectors' Edition",
	"Collector's Edition":                 "Collectors' Edition",
	"Collectors Ed":                       "Collectors' Edition",
	"Collectors' Edition":                 "Collectors' Edition",
	"Collectors’ Edition":                 "Collectors' Edition",
	"Collectors Edition":                  "Collectors' Edition",

	// Portal
	"Portal 1":          "Portal",
	"Portal Demo Game":  "Portal",
	"Portal II":         "Portal Second Age",
	"Portal 2nd Age":    "Portal Second Age",
	"Portal 3K":         "Portal Three Kingdoms",
	"Portal 3 Kingdoms": "Portal Three Kingdoms",

	// Duel Decks
	"Japanese Jace vs. Chandra Foil": "Duel Decks: Jace vs. Chandra",
	"Duel Deck Heros VS Monsters":    "Duel Decks: Heroes vs. Monsters",
	"Duel Decks: Heros vs. Monsters": "Duel Decks: Heroes vs. Monsters",
	"Duel Decks Kiora vs. Elspeth":   "Duel Decks: Elspeth vs. Kiora",
	"Duel Decks: Kiora vs. Elspeth":  "Duel Decks: Elspeth vs. Kiora",
	"Duel Decks: Kiora vs Elspeth":   "Duel Decks: Elspeth vs. Kiora",
	"DD: Anthology":                  "Duel Decks Anthology",

	// Global Series
	"Global Series - Planeswalker Decks - Jiang Yanggu & Mu Yanling": "Global Series Jiang Yanggu & Mu Yanling",

	"Global Series Jiang Yanggu And Mu Yanling":  "Global Series Jiang Yanggu & Mu Yanling",
	"Global Series: Jiangg Yanggu & Mu Yanling":  "Global Series Jiang Yanggu & Mu Yanling",
	"Global Series: Jiang Yanggu & Mu Yanling":   "Global Series Jiang Yanggu & Mu Yanling",
	"Global Series: Jiang Yanggu and Mu Yanling": "Global Series Jiang Yanggu & Mu Yanling",

	// Premium Deck Series
	"Fire & Lightning":                      "Premium Deck Series: Fire and Lightning",
	"PDS: Fire & Lightning":                 "Premium Deck Series: Fire and Lightning",
	"Premium Deck Fire and Lightning":       "Premium Deck Series: Fire and Lightning",
	"Premium Deck: Fire and Lightning":      "Premium Deck Series: Fire and Lightning",
	"Premium Deck Series Fire & Lightning":  "Premium Deck Series: Fire and Lightning",
	"Premium Deck Series: Fire & Lightning": "Premium Deck Series: Fire and Lightning",
	"Graveborn":                             "Premium Deck Series: Graveborn",
	"PDS: Graveborn":                        "Premium Deck Series: Graveborn",
	"Premium Deck Graveborn":                "Premium Deck Series: Graveborn",
	"Premium Deck: Graveborn":               "Premium Deck Series: Graveborn",
	"Slivers":                               "Premium Deck Series: Slivers",
	"PDS: Slivers":                          "Premium Deck Series: Slivers",
	"Premium Deck Slivers":                  "Premium Deck Series: Slivers",
	"Premium Deck: Slivers":                 "Premium Deck Series: Slivers",

	// Planechase
	"Planechase":                        "Planechase",
	"Planechase 2009":                   "Planechase",
	"Planechase 2012":                   "Planechase 2012",
	"Planechase (2009 Edition)":         "Planechase",
	"Planechase (2012 Edition)":         "Planechase 2012",
	"Planechase 2009 Edition":           "Planechase",
	"Planechase 2012 Edition":           "Planechase 2012",
	"Planechase Planes: 2009 Edition":   "Planechase Planes",
	"Planechase Planes: 2012 Edition":   "Planechase 2012 Planes",
	"Planechase: 2009 Edition":          "Planechase",
	"Planechase: 2012 Edition":          "Planechase 2012",
	"Planechase: 2009 Edition - Planes": "Planechase Planes",
	"Planechase: 2012 Edition - Planes": "Planechase 2012 Planes",

	// Deckmasters
	"Deckmaster Promo":               "Deckmasters",
	"Deckmaster":                     "Deckmasters",
	"Deckmasters":                    "Deckmasters",
	"Deckmasters Garfield vs Finkel": "Deckmasters",
	"Deckmasters Garfield Vs Finkel": "Deckmasters",

	// Junior Super/Europe/APAC Series
	"European Junior Series":          "Junior Series Europe",
	"Junior Series Promo":             "Junior Series Europe",
	"Junior Series Promos":            "Junior Series Europe",
	"Euro JSS Promo":                  "Junior Series Europe",
	"Junior Series":                   "Junior Series Europe",
	"APAC Series":                     "Junior APAC Series",
	"Japan JSS":                       "Junior APAC Series",
	"Junior APAC Series":              "Junior APAC Series",
	"Junior APAC Series Promos":       "Junior APAC Series",
	"Junior Super Series":             "Junior Super Series",
	"Junior Super Series Promos":      "Junior Super Series",
	"Magic Scholarship":               "Junior Super Series",
	"Magic Scholarship Series":        "Junior Super Series",
	"Magic Scholarship Series Promo":  "Junior Super Series",
	"Magic Scholarship Series Promos": "Junior Super Series",
	"Scholarship Series":              "Junior Super Series",
	"Scholarship Series Promo":        "Junior Super Series",
	"MSS":                             "Junior Super Series",
	"MSS Promo":                       "Junior Super Series",
	"JSS":                             "Junior Super Series",
	"JSS DCI PROMO":                   "Junior Super Series",
	"JSS Foil":                        "Junior Super Series",
	"JSS Promo":                       "Junior Super Series",
	"JSS/MSS Promos":                  "Junior Super Series",
	"Japan Junior Tournament":         "Japan Junior Tournament",
	"Japan Junior Tournament Promo":   "Japan Junior Tournament",
	"Japan Junior Tournament Promos":  "Japan Junior Tournament",
	"Japanese JSS Promo":              "Japan Junior Tournament",

	// GP Promos
	"2010 Grand Prix Promo":              "Grand Prix Promos",
	"2018 Grand Prix Promo":              "Grand Prix Promos",
	"GP Promo":                           "Grand Prix Promos",
	"GP Promos":                          "Grand Prix Promos",
	"Gran Prix Promo":                    "Grand Prix Promos",
	"Grand Prix":                         "Grand Prix Promos",
	"Grand Prix Foil":                    "Grand Prix Promos",
	"Grand Prix Promo":                   "Grand Prix Promos",
	"Grand Prix 2018":                    "MagicFest 2019",
	"MagicFest 2019":                     "MagicFest 2019",
	"MagicFest 2020":                     "MagicFest 2020",
	"MagicFest Foil - 2020":              "MagicFest 2020",
	"FOIL 2019 MF MagicFest GP Promo":    "MagicFest 2019",
	"NONFOIL 2019 MF MagicFest GP Promo": "MagicFest 2019",
	"FOIL 2020 MF MagicFest GP Promo":    "MagicFest 2020",
	"NONFOIL 2020 MF MagicFest GP Promo": "MagicFest 2020",

	// Nationals
	"2018 Nationals Promo": "Nationals Promos",
	"Nationals":            "Nationals Promos",

	// Pro Tour Promos
	"2011 Pro Tour Promo":                 "Pro Tour Promos",
	"MC Qualifier":                        "Pro Tour Promos",
	"MCQ Promo":                           "Pro Tour Promos",
	"MCQ":                                 "Pro Tour Promos",
	"Mythic Championship Qualifier Promo": "Pro Tour Promos",
	"Mythic Championship":                 "Pro Tour Promos",
	"Players Tour Promo":                  "Pro Tour Promos",
	"Players Tour Qualifier PTQ Promo":    "Pro Tour Promos",
	"Players Tour Qualifier":              "Pro Tour Promos",
	"Premier Play":                        "Pro Tour Promos",
	"Pro Tour Foil":                       "Pro Tour Promos",
	"Pro Tour Promo":                      "Pro Tour Promos",
	"Pro Tour Promos":                     "Pro Tour Promos",
	"Pro Tour":                            "Pro Tour Promos",
	"RPTQ Promo Foil":                     "Pro Tour Promos",
	"RPTQ Promo":                          "Pro Tour Promos",
	"RPTQ":                                "Pro Tour Promos",
	"Qualifier":                           "Pro Tour Promos",
	"Regional PTQ Promo Foil":             "Pro Tour Promos",
	"Regional PTQ Promo":                  "Pro Tour Promos",
	"Regional PTQ":                        "Pro Tour Promos",

	// Worlds
	"Worlds":                  "World Championship Promos",
	"World Championship Foil": "World Championship Promos",

	// WMCQ
	"2015 World Magic Cup Qualifier":      "World Magic Cup Qualifiers",
	"2016 WMCQ Promo":                     "World Magic Cup Qualifiers",
	"2017 WMCQ Promo":                     "World Magic Cup Qualifiers",
	"DCI Promo World Magic Cup Qualifier": "World Magic Cup Qualifiers",
	"Promos: WMCQ":                        "World Magic Cup Qualifiers",
	"WCQ":                                 "World Magic Cup Qualifiers",
	"WMC Promo":                           "World Magic Cup Qualifiers",
	"WMC Qualifier":                       "World Magic Cup Qualifiers",
	"WMC":                                 "World Magic Cup Qualifiers",
	"WMCQ Foil":                           "World Magic Cup Qualifiers",
	"WMCQ Promo":                          "World Magic Cup Qualifiers",
	"WMCQ Promos":                         "World Magic Cup Qualifiers",
	"WMCQ Promo 2016":                     "World Magic Cup Qualifiers",
	"WMCQ Promo 2017":                     "World Magic Cup Qualifiers",
	"WMCQ":                                "World Magic Cup Qualifiers",
	"WMCQ Promo Cards":                    "World Magic Cup Qualifiers",
	"World Magic Cup":                     "World Magic Cup Qualifiers",
	"World Magic Cup Promo":               "World Magic Cup Qualifiers",
	"World Magic Cup Qualifier 2016":      "World Magic Cup Qualifiers",
	"World Magic Cup Qualifier Promo":     "World Magic Cup Qualifiers",
	"World Cup Qualifier Promo":           "World Magic Cup Qualifiers",

	// Tarkir extra sets
	"Dragonfury":                              "Tarkir Dragonfury",
	"Dragonfury Promo":                        "Tarkir Dragonfury",
	"Dragons of Tarkir Dragonfury Game":       "Tarkir Dragonfury",
	"Dragons of Tarkir Dragonfury Game Promo": "Tarkir Dragonfury",
	"Tarkir Dragonfury":                       "Tarkir Dragonfury",
	"Tarkir Dragonfury Promo":                 "Tarkir Dragonfury",
	"Tarkir Dragonfury Promos":                "Tarkir Dragonfury",
	"Ugin's Fate Promo":                       "Ugin's Fate",
	"Ugin's Fate":                             "Ugin's Fate",
	"Ugins Fate":                              "Ugin's Fate",
	"Ugins Fate Promo":                        "Ugin's Fate",
	"Ugins Fate Promos":                       "Ugin's Fate",
	"Ugin's Fate Alternate Art Promo":         "Ugin's Fate",
	"Ugin’s Fate Promo":                       "Ugin's Fate",
	"Ugin's Fate Promos":                      "Ugin's Fate",
	"Ugin's Fate promos":                      "Ugin's Fate",

	// Resale
	"Media Promo":     "Resale Promos",
	"Repack Insert":   "Resale Promos",
	"Resale Foil":     "Resale Promos",
	"Resale Promo":    "Resale Promos",
	"Resale Walmart ": "Resale Promos",
	"Resale Walmart":  "Resale Promos",
	"Resale":          "Resale Promos",
	"Store Foil":      "Resale Promos",
	"Walmart Resale":  "Resale Promos",

	// Convention
	"Dragon'Con 1994":       "Dragon Con",
	"HASCON":                "HasCon 2017",
	"HASCON 2017":           "HasCon 2017",
	"Hascon Promo Foil":     "HasCon 2017",
	"Hascon Promo":          "HasCon 2017",
	"Hascon 2017 Promo":     "HasCon 2017",
	"PAX Prime Promo":       "URL/Convention Promos",
	"2012 Convention Promo": "URL/Convention Promos",
	"URL Convention Promo":  "URL/Convention Promos",

	// Wotc Store
	"Foil Beta Picture":       "Wizards of the Coast Online Store",
	"Online Store Card":       "Wizards of the Coast Online Store",
	"Original Artwork":        "Wizards of the Coast Online Store",
	"Redemption Original Art": "Wizards of the Coast Online Store",

	// Archenemy
	"Archenemy":                        "Archenemy",
	"Archenemy: 2010 Edition":          "Archenemy",
	"Archenemy - Nicol Bolas":          "Archenemy: Nicol Bolas",
	"Archenemy Nicol Bolas":            "Archenemy: Nicol Bolas",
	"Archenemy: Nicol Bolas":           "Archenemy: Nicol Bolas",
	"Archenemy Schemes (2010 Edition)": "Archenemy Schemes",
	"Archenemy 'Schemes'":              "Archenemy Schemes",

	// Various
	"Battle Royale":            "Battle Royale Box Set",
	"Battle the Horde":         "Battle the Horde",
	"Conspiracy: 2014 Edition": "Conspiracy",
	"Introductory 4th Edition": "Introductory Two-Player Set",
	"PS3 Promo":                "Duels of the Planeswalkers 2012 Promos",
	"Starter":                  "Starter 1999",
	"Starter 2000":             "Starter 2000",
	"Vanguard":                 "Vanguard Series",

	// MB1/PLIST need to be explicitly set to override the edition
	"The List":         "The List",
	"Mystery Booster":  "Mystery Booster",
	"SLX Cards":        "Universes Within",
	"Universes Within": "Universes Within",
	"Secret Lair Commander: Heads I Win, Tales You Lose": "Heads I Win, Tails You Lose",

	// Time Spiral Remastered retro frame
	"Time Spiral Remastered: Extras": "Time Spiral Remastered",

	// Beatdown
	"Beatdown":      "Beatdown Box Set",
	"Beatdown Foil": "Beatdown Box Set",

	// Coldsnap Theme Decks
	"Coldsnap Reprints":            "Coldsnap Theme Decks",
	"Coldsnap Theme Deck Reprints": "Coldsnap Theme Decks",
	"Coldsnap Theme-Deck Reprints": "Coldsnap Theme Decks",

	// Modern Event Deck
	"Magic Modern Event Deck":                     "Modern Event Deck 2014",
	"Modern Event Deck":                           "Modern Event Deck 2014",
	"Modern Event Deck - March of the Multitudes": "Modern Event Deck 2014",

	// Custom set codes
	"MED2": "Mythic Edition",
	"MED3": "Mythic Edition",
	"RMB1": "Mystery Booster Retail Edition Foils",

	// Alt Fourth
	"4th Edition - Alternate":    "Alternate Fourth Edition",
	"Alternate 4th Edition":      "Alternate Fourth Edition",
	"Alternate 4th":              "Alternate Fourth Edition",
	"Fourth (Alternate Edition)": "Alternate Fourth Edition",
	"Fourth Edition (Alt)":       "Alternate Fourth Edition",
	"Fourth Edition: Alternate":  "Alternate Fourth Edition",

	// Foreign-only
	"3rd Edition (Foreign Black Border)":     "Foreign Black Border",
	"3rd Edition BB":                         "Foreign Black Border",
	"Black Bordered (foreign)":               "Foreign Black Border",
	"Foreign BB":                             "Foreign Black Border",
	"Foreign Black Bordered":                 "Foreign Black Border",
	"Foreign Limited - FBB":                  "Foreign Black Border",
	"Italian Revised FBB":                    "Foreign Black Border",
	"Revised Black Border Italian":           "Foreign Black Border",
	"Revised Edition - FBB":                  "Foreign Black Border",
	"Revised Edition (Foreign Black Border)": "Foreign Black Border",
	"Revised Edition Foreign Black Border":   "Foreign Black Border",

	"4th Edition BB":                 "Fourth Edition Foreign Black Border",
	"Fourth Edition Black Bordered":  "Fourth Edition Foreign Black Border",
	"Fourth Edition: Black Bordered": "Fourth Edition Foreign Black Border",

	"Italian Renaissance": "Rinascimento",
	"Renaissance Italian": "Rinascimento",
	"Italian Legends":     "Legends Italian",
}
