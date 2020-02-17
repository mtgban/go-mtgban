package miniaturemarket

var athVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"A": "84",
		"B": "85",
	},
	"Mountain": map[string]string{
		"A": "82",
		"B": "83",
	},
	"Plains": map[string]string{
		"A": "78",
		"B": "79",
	},
	"Swamp": map[string]string{
		"A": "80",
		"B": "81",
	},
}

var allVariants = map[string]map[string]string{
	"Aesthir Glider": map[string]string{
		"Facing Left":  "116a",
		"Facing Right": "116b",
	},
	"Agent of Stromgald": map[string]string{
		"Arms Crossed":        "64a",
		"Woman Holding Staff": "64b",
	},
	"Arcane Denial": map[string]string{
		"Axe":   "22a",
		"Sword": "22b",
	},
	"Astrolabe": map[string]string{
		"Globe/Windows":  "118a",
		"Map Background": "118b",
	},
	"Awesome Presence": map[string]string{
		"Arms Spread":      "23a",
		"Man Being Chased": "23b",
	},
	"Balduvian War-Makers": map[string]string{
		"Green Background":     "66a",
		"Two Clubs, Side View": "66b",
	},
	"Benthic Explorers": map[string]string{
		"Sitting on Rocks": "24a",
		"Swimming":         "24b",
	},
	"Bestial Fury": map[string]string{
		"Facing Front": "67a",
		"Facing Left":  "67b",
	},
	"Carrier Pigeons": map[string]string{
		"Hand":  "1a",
		"Trees": "1b",
	},
	"Casting of Bones": map[string]string{
		"Finger Ornament": "44a",
		"Hooded Figure":   "44b",
	},
	"Deadly Insect": map[string]string{
		"Bird":     "86a",
		"Red Robe": "86b",
	},
	"Elvish Ranger": map[string]string{
		"Female": "88a",
		"Male":   "88b",
	},
	"Enslaved Scout": map[string]string{
		"Crouching": "71a",
		"Horses":    "71b",
	},
	"Errand of Duty": map[string]string{
		"Horse":              "2b",
		"Page Holding Sword": "2a",
	},
	"False Demise": map[string]string{
		"Cave-In":    "27a",
		"Underwater": "27b",
	},
	"Feast or Famine": map[string]string{
		"Falling into Pit": "49a",
		"Knife":            "49b",
	},
	"Fevered Strength": map[string]string{
		"Chains":           "50a",
		"Foaming at Mouth": "50b",
	},
	"Foresight": map[string]string{
		"Mermaid With Shiny Shell": "29a",
		"White Dress":              "29b",
	},
	"Fyndhorn Druid": map[string]string{
		"Facing Left":  "90a",
		"Facing Right": "90b",
	},
	"Gift of the Woods": map[string]string{
		"Female": "92a",
		"Male":   "92b",
	},
	"Gorilla Berserkers": map[string]string{
		"Closed Mouth": "93a",
		"Roaring":      "93b",
	},
	"Gorilla Chieftain": map[string]string{
		"2 Gorillas": "94a",
		"4 Gorillas": "94b",
	},
	"Gorilla Shaman": map[string]string{
		"Holding Baby": "72a",
		"Skulls":       "72b",
	},
	"Gorilla War Cry": map[string]string{
		"Colorful Headdress": "73a",
		"Red Cub":            "73b",
	},
	"Guerrilla Tactics": map[string]string{
		"Cliff":           "74a",
		"Kneeling Knight": "74b",
	},
	"Insidious Bookworms": map[string]string{
		"Bookshelf":   "51a",
		"Single Book": "51b",
	},
	"Kjeldoran Escort": map[string]string{
		"Green Blanketed Dog": "7a",
		"Red Blanketed Dog":   "7b",
	},
	"Kjeldoran Pride": map[string]string{
		"Bear":  "9a",
		"Eagle": "9b",
	},
	"Lat-Nam's Legacy": map[string]string{
		"Scroll":    "30a",
		"Bookshelf": "30b",
	},
	"Lim-Dul's High Guard": map[string]string{
		"Red Armor":  "55a",
		"Two Swords": "55b",
	},
	"Martyrdom": map[string]string{
		"Knight standing, holding sword": "10a",
		"Knight wounded on ground":       "10b",
	},
	"Noble Steeds": map[string]string{
		"Trees in Forefront":  "11a",
		"Two Steeds Close Up": "11b",
	},
	"Phantasmal Fiend": map[string]string{
		"Close-Up": "57a",
		"Doorway":  "57b",
	},
	"Phyrexian Boon": map[string]string{
		"Female": "58a",
		"Male":   "58b",
	},
	"Phyrexian War Beast": map[string]string{
		"Propeller Left":  "127a",
		"Propeller Right": "127b",
	},
	"Reinforcements": map[string]string{
		"Quote Darien of Kjeldor": "12a",
		"Quote General Varchild":  "12b",
	},
	"Reprisal": map[string]string{
		"Green Monster": "13a",
		"Red Dragon":    "13b",
	},
	"Royal Herbalist": map[string]string{
		"Female": "15a",
		"Male":   "15b",
	},
	"Soldevi Adnate": map[string]string{
		"Female": "60a",
		"Male":   "60b",
	},
	"Soldevi Heretic": map[string]string{
		"Blue Robe": "33a",
		"Red Robe":  "33b",
	},
	"Soldevi Sage": map[string]string{
		"2 Candles":         "34a",
		"Old Woman Sitting": "34b",
	},
	"Soldevi Sentry": map[string]string{
		"Close Up":               "132a",
		"Silver Sentry Fighting": "132b",
	},
	"Soldevi Steam Beast": map[string]string{
		"Beast in Mountains": "133a",
		"Purple Sun":         "133b",
	},
	"Stench of Decay": map[string]string{
		"Hand Covering Nose": "61a",
		"Red Flower":         "61b",
	},
	"Storm Crow": map[string]string{
		"Flying Left":  "36a",
		"Flying Right": "36b",
	},
	"Storm Shaman": map[string]string{
		"Female": "81a",
		"Male":   "81b",
	},
	"Swamp Mosquito": map[string]string{
		"Brown Trees": "63a",
		"Fallen Tree": "63b",
	},
	"Taste of Paradise": map[string]string{
		"One Person Holding Fruit": "100a",
		"Two People":               "100b",
	},
	"Undergrowth": map[string]string{
		"Fox and Badger": "102a",
		"Holding Axe":    "102b",
	},
	"Varchild's Crusader": map[string]string{
		"Black Horse and Hollow Log": "82a",
		"Brown Horse and Castle":     "82b",
	},
	"Veteran's Voice": map[string]string{
		"Over the Shoulder": "84a",
		"Side-by-Side":      "84b",
	},
	"Viscerid Armor": map[string]string{
		"Crashing Wave":        "41a",
		"Humans in Foreground": "41b",
	},
	"Whip Vine": map[string]string{
		"3 Vines":       "103a",
		"Ensnared Bird": "103b",
	},
	"Wild Aesthir": map[string]string{
		"Blue Mountains":    "21a",
		"Wings Thrown Back": "21b",
	},
	"Yavimaya Ancients": map[string]string{
		"Rearing Horse": "104a",
		"Trees":         "104b",
	},
}

var atqVariants = map[string]map[string]string{
	"Strip Mine": map[string]string{
		"No Sky, No Cave/Tower": "82a",
		"Cave":                  "82b",
		"Tower":                 "82c",
		"Sky, No Cave or Tower": "82d",
	},
	"Mishra's Factory": map[string]string{
		"Spring": "80a",
		"Summer": "80b",
		"Fall":   "80c",
		"Winter": "80d",
	},
	"Urza's Mine": map[string]string{
		"Pulley":           "83a",
		"Mine Face":        "83b",
		"Submerged Sphere": "83c",
		"Tower":            "83d",
	},
	"Urza's Power Plant": map[string]string{
		"Sphere with Tubes": "84a",
		"Red Columns":       "84b",
		"Bug":               "84c",
		"Rock In Pan":       "84d",
	},
	"Urza's Tower": map[string]string{
		"Red Leaves":            "85a",
		"Sunset with Shoreline": "85b",
		"Plains":                "85c",
		"Mountains":             "85d",
	},
}

var chrVariants = map[string]map[string]string{
	"Urza's Mine": map[string]string{
		"Mine Face":        "114a",
		"Submerged Sphere": "114b",
		"Pulley":           "114c",
		"Tower":            "114d",
	},
	"Urza's Power Plant": map[string]string{
		"Rock In Pan":       "115a",
		"Red Columns":       "115b",
		"Sphere with Tubes": "115c",
		"Bug":               "115d",
	},
	"Urza's Tower": map[string]string{
		"Sunset with Shoreline": "116a",
		"Plains":                "116b",
		"Mountains":             "116c",
		"Red Leaves":            "116d",
	},
}

var pelpVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"EURO Land, Broceliande": "6",
	},
}

var ed2Variants = map[string]map[string]string{
	"Forest": map[string]string{
		"Rocks":            "300",
		"Pathway":          "301",
		"Eyes in treehole": "302",
	},
	"Island": map[string]string{
		"Purple island, gold horizon":   "291",
		"Green island":                  "292",
		"Purple island, orange horizon": "293",
	},
	"Mountain": map[string]string{
		"Gray mountains":            "297",
		"Blue mountains":            "298",
		"Brown mountain, green sky": "299",
	},
	"Plains": map[string]string{
		"No trees on plain":   "288",
		"Trees on plain":      "289",
		"Cliff in background": "290",
	},
	"Swamp": map[string]string{
		"One branch in foreground": "294",
		"Brown, two branches":      "295",
		"Black, two branches":      "296",
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

var ustVariants = map[string]map[string]string{
	"Amateur Auteur": map[string]string{
		"Innistrad": "3b",
		"Ravnica":   "3a",
		"Theros":    "3c",
		"Zendikar":  "3d",
	},
	"Beast in Show": map[string]string{
		"Baloth":    "103c",
		"Gnarlid":   "103b",
		"Thragtusk": "103d",
		"Tyrranax":  "103a",
	},
	"Everythingamajig": map[string]string{
		"Add":       "147d",
		"Draw":      "147b",
		"Flip":      "147c",
		"Move":      "147a",
		"Sacrifice": "147e",
		"Scry":      "147f",
	},
	"Extremely Slow Zombie": map[string]string{
		"Fall":   "54b",
		"Spring": "54d",
		"Summer": "54a",
		"Winter": "54c",
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
		"Choose a number":    "113f",
		"Choose artist":      "113b",
		"Choose border":      "113c",
		"Choose flavor":      "113a",
		"Choose odd or even": "113e",
		"Choose rarity":      "113d",
	},
	"Knight of the Kitchen Sink": map[string]string{
		"Black borders":          "12a",
		"Even collector numbers": "12b",
		"Loose lips":             "12c",
		"Odd collector numbers":  "12d",
		"Two-word names":         "12e",
		"Watermarks":             "12f",
	},
	"Secret Base": map[string]string{
		"Agents of S.N.E.A.K.":     "165b",
		"Crossbreed Labs":          "165e",
		"Goblin Explosioneers":     "165d",
		"League of Dastardly Doom": "165c",
		"Order of the Widget":      "165a",
	},
	"Sly Spy": map[string]string{
		"Destroy creature facing left":  "67b",
		"Destroy creature facing right": "67d",
		"Lose a finger":                 "67c",
		"Reveal hand":                   "67a",
		"Reveal top of library":         "67e",
		"Roll six-sided die":            "67f",
	},
	"Very Cryptic Command": map[string]string{
		"Counter":                 "49e",
		"Draw":                    "49c",
		"Return":                  "49d",
		"Scry":                    "49f",
		"Untap":                   "49b",
		"Switch) [Alternate Art]": "49a",
	},
}

var setVariants = map[string]map[string]map[string]string{
	"Alliances":             allVariants,
	"Anthologies":           athVariants,
	"Antiquities":           atqVariants,
	"Chronicles":            chrVariants,
	"European Land Program": pelpVariants,
	"Planeshift":            plsVariants,
	"Unlimited Edition":     ed2Variants,
	"Unstable":              ustVariants,
}
