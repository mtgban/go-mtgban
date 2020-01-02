package miniaturemarket

var setTable = map[string]string{
	"10th Edition":                      "Tenth Edition",
	"4th Edition":                       "Fourth Edition",
	"5th Edition":                       "Fifth Edition",
	"6th Edition":                       "Classic Sixth Edition",
	"7th Edition":                       "Seventh Edition",
	"8th Edition":                       "Eighth Edition",
	"9th Edition":                       "Ninth Edition",
	"Battle Royale":                     "Battle Royale Box Set",
	"Beatdown":                          "Beatdown Box Set",
	"Collector's Edition":               "Collectors’ Edition",
	"Commander Anthology Volume 2":      "Commander Anthology Volume II",
	"International Collector's Edition": "Intl. Collectors’ Edition",
	"Journey Into Nyx":                  "Journey into Nyx",
	"Masterpiece: Mythic Edition":       "Mythic Edition",
	"Modern Event Deck":                 "Modern Event Deck 2014",
	"Modern Masters 2013":               "Modern Masters",
	"Planechase 2009":                   "Planechase",
	"Revised":                           "Revised Edition",
	"Shadows Over Innistrad":            "Shadows over Innistrad",
	"Time Spiral (Timeshifted)":         "Time Spiral Timeshifted",
	"Unlimited":                         "Unlimited Edition",

	"Premium Deck Series: Fire & Lightning":    "Premium Deck Series: Fire and Lightning",
	"Global Series: Jiang Yanggu & Mu Yanling": "Global Series Jiang Yanggu & Mu Yanling",
	"Throne of Eldraine (Collector Edition)":   "Throne of Eldraine",
}

var cardTable = map[string]string{
	"Asylum Visitior":                         "Asylum Visitor",
	"B.F.M. (Big Furry Monster) (Left Side)":  "B.F.M. (Big Furry Monster)",
	"B.F.M. (Big Furry Monster) (Right Side)": "B.F.M. (Big Furry Monster) (b)",
	"Fiesty Stegosaurus":                      "Feisty Stegosaurus",
	"Who / What / When / Where / Why":         "Who",
}

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
	"Alliances":   allVariants,
	"Antiquities": atqVariants,
	"Chronicles":  chrVariants,
	"Unstable":    ustVariants,
}

var unlVariants = map[string]string{
	"Forest A [Pathway]":                       "301",
	"Forest B [Rocks]":                         "300",
	"Forest C [Eyes in treehole]":              "302",
	"Island A [Green island]":                  "292",
	"Island B [Purple island, gold horizon]":   "291",
	"Island C [Purple island, orange horizon]": "293",
	"Mountain A [Blue mountains]":              "298",
	"Mountain B [Gray mountains]":              "297",
	"Mountain C [Brown mountain, green sky]":   "299",
	"Plains A [Trees on plain]":                "289",
	"Plains B [No trees on plain]":             "288",
	"Plains C [Cliff in background]":           "290",
	"Swamp A [Brown, two branches]":            "295",
	"Swamp B [One branch in foreground]":       "294",
	"Swamp C [Black, two branches]":            "296",
}
