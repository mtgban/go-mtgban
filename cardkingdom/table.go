package cardkingdom

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

var femVariants = map[string]map[string]string{
	"Homarid": map[string]string{
		"Tedin": "19c",
	},
	"Spore Cloud": map[string]string{
		"Myrfors": "72b",
	},
	"Vodalian Mage": map[string]string{
		"Poole": "30b",
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

var unhVariants = map[string]map[string]string{
	"Look at Me, I'm R&D": map[string]string{
		"": "17",
	},
	"Blast from the Past": map[string]string{
		"": "72",
	},
	"Old Fogey": map[string]string{
		"": "90",
	},
}

var setVariants = map[string]map[string]map[string]string{
	"Collectors’ Edition":       ceVariants,
	"Intl. Collectors’ Edition": ceVariants,
	"Fallen Empires":            femVariants,
	"Planeshift":                plsVariants,
}
