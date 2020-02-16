package miniaturemarket

var allVariants = map[string]map[string]string{
	"Arcane Denial": map[string]string{
		"Axe":   "22a",
		"Sword": "22b",
	},
	"Soldevi Adnate": map[string]string{
		"Female": "60a",
		"Male":   "60b",
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
	"Antiquities":           atqVariants,
	"Chronicles":            chrVariants,
	"European Land Program": pelpVariants,
	"Planeshift":            plsVariants,
	"Unlimited Edition":     ed2Variants,
	"Unstable":              ustVariants,
}
