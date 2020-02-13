package channelfireball

var athVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"Arena":    "84",
		"Portal 2": "85",
	},
	"Mountain": map[string]string{
		"Arabian Nights": "82",
		"Mirage":         "83",
	},
	"Plains": map[string]string{
		"Mirage":   "78",
		"Portal 1": "79",
	},
	"Swamp": map[string]string{
		"Ice Age": "80",
		"Tempest": "81",
	},
}

var allVariants = map[string]map[string]string{
	"Aesthir Glider": map[string]string{
		"Clouds": "116a",
		"Moon":   "116b",
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
		"No Horizon":    "82a",
		"Even Horizon":  "82b",
		"Small Tower":   "82c",
		"Uneven Stripe": "82d",
	},
	"Mishra's Factory": map[string]string{
		"Spring": "80a",
		"Summer": "80b",
		"Autumn": "80c",
		"Winter": "80d",
	},
	"Urza's Mine": map[string]string{
		"Pulley": "83a",
		"Mouth":  "83b",
		"Sphere": "83c",
		"Ladder": "83d",
	},
	"Urza's Power Plant": map[string]string{
		"Sphere":      "84a",
		"Columns":     "84b",
		"Bug":         "84c",
		"Rock in Pot": "84d",
	},
	"Urza's Tower": map[string]string{
		"Forest":    "85a",
		"Shore":     "85b",
		"Plains":    "85c",
		"Mountains": "85d",
	},
}

var chkVariants = map[string]map[string]string{
	"Brothers Yamazaki": map[string]string{
		"Facing Left":  "160a",
		"Facing Right": "160b",
	},
}

var chrVariants = map[string]map[string]string{
	"Urza's Mine": map[string]string{
		"Mouth":         "114a",
		"Pulley":        "114c",
		"Clawed Sphere": "114b",
		"Ladder":        "114d",
	},
	"Urza's Power Plant": map[string]string{
		"Rock in Pot": "115a",
		"Columns":     "115b",
		"Sphere":      "115c",
		"Bug":         "115d",
	},
	"Urza's Tower": map[string]string{
		"Forest":    "116a",
		"Plains":    "116b",
		"Mountains": "116c",
		"Shore":     "116d",
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
		"Broceliande, France": "6",
	},
	"Island": map[string]string{
		"Venice, Italy": "7",
	},
}

var femVariants = map[string]map[string]string{
	"Tidal Flats": map[string]string{
		"Rob Alexander Lush Horizon":       "27a",
		"Rob Alexander Sun Through Clouds": "27b",
	},
	"Brassclaw Orcs": map[string]string{
		"Rob Alexander, Spear":       "49a",
		"Rob Alexander, Gloved Fist": "49c",
	},
	"Order of Leitbur": map[string]string{
		"Facing Right, Bryon Wackwitz": "16a",
		"Facing Left, Bryon Wackwitz":  "16b",
	},
}

var hmlVariants = map[string]map[string]string{
	"Anaba Shaman": map[string]string{
		"Quote Irini Sngir":      "67a",
		"Baki, Wizard Attendant": "67b",
	},
	"Giant Albatross": map[string]string{
		"Ship's Mast": "27a",
		"Clouds":      "27b",
	},
	"Memory Lapse": map[string]string{
		"Quote Chandler, Female Art":            "32a",
		"Quote Reveka, Wizard Savant, Male Art": "32b",
	},
	"Trade Caravan": map[string]string{
		"Giraffe on Right": "19a",
		"Moon Top Left":    "19b",
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

var setVariants = map[string]map[string]map[string]string{
	"Alliances":                     allVariants,
	"Anthologies":                   athVariants,
	"Antiquities":                   atqVariants,
	"Champions of Kamigawa":         chkVariants,
	"Chronicles":                    chrVariants,
	"Deckmasters":                   dkmVariants,
	"European Land Program":         pelpVariants,
	"Fallen Empires":                femVariants,
	"Homelands":                     hmlVariants,
	"Planeshift":                    plsVariants,
	"Shadows over Innistrad":        soiVariants,
	"Shadows over Innistrad Promos": psoiVariants,
}
