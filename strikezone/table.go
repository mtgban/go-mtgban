package strikezone

var allVariants = map[string]map[string]string{
	"Arcane Denial": map[string]string{
		"1": "22a",
		"2": "22b",
	},
}

var atqVariants = map[string]map[string]string{
	"Strip Mine": map[string]string{
		"No Sky, No Tower":    "82a",
		"Sky, Even Terraces":  "82b",
		"No Sky wth Tower":    "82c",
		"Sky Uneven Terraces": "82d",
	},
	"Mishra's Factory": map[string]string{
		"Spring":        "80a",
		"Summer Green":  "80b",
		"Autumn Orange": "80c",
		"Winter Snow":   "80d",
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
	"Mountain": map[string]string{
		"Phillippines- Rise Terrace": "3",
	},
}

var chrVariants = map[string]map[string]string{
	"Urza's Mine": map[string]string{
		"Mouth":  "114a",
		"Pulley": "114c",
		"Sphere": "114b",
		"Tower":  "114d",
	},
	"Urza's Power Plant": map[string]string{
		"Pot":     "115a",
		"Columns": "115b",
		"Sphere":  "115c",
		"Bug":     "115d",
	},
	"Urza's Tower": map[string]string{
		"Forest":    "116a",
		"Plains":    "116b",
		"Mountains": "116c",
		"Shore":     "116d",
	},
}

var pelpVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"Broceliande France": "6",
	},
	"Swamp": map[string]string{
		"Ardenees Fagnes Belgium": "5",
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

var ogwVariants = map[string]map[string]string{
	"Wastes": map[string]string{
		"Full Art 183": "183",
		"Full Art 184": "184",
	},
}

var plsVariants = map[string]map[string]string{
	"Ertai, the Corrupted": map[string]string{
		"":              "107",
		"Alternate Art": "107★",
	},
	"Skyship Weatherlight": map[string]string{
		"":              "133",
		"Alternate Art": "133★",
	},
	"Tahngarth, Talruum Hero": map[string]string{
		"": "74",
		"Alternate Planeshift Art": "74★",
	},
}

var setVariants = map[string]map[string]map[string]string{
	"Alliances":                 allVariants,
	"Antiquities":               atqVariants,
	"Asia Pacific Land Program": palpVariants,
	"Chronicles":                chrVariants,
	"European Land Program":     pelpVariants,
	"Magic Premiere Shop 2005":  pmpsVariants,
	"Oath of the Gatewatch":     ogwVariants,
	"Planeshift":                plsVariants,
}
