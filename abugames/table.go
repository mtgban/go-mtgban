package abugames

var athVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"A": "84",
		"B": "85",
	},
	"Mountain": map[string]string{
		"A": "83",
		"B": "82",
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

var atqVariants = map[string]map[string]string{
	"Mishra's Factory": map[string]string{
		"b Spring": "80a",
		"c Summer": "80b",
		"a Fall":   "80c",
		"d Winter": "80d",
	},
	/*
		"Strip Mine": map[string]string{
			"atq-82a-strip-mine": "82a",
			"atq-82b-strip-mine": "82b",
			"atq-82c-strip-mine": "82c",
			"AQ070":              "82d",
		},
		"Urza's Mine": map[string]string{
			"AQ084": "83a",
			"AQ085": "83b",
			"AQ083": "83c",
			"AQ086": "83d",
		},
		"Urza's Power Plant": map[string]string{
			"AQ089": "84a",
			"AQ090": "84b",
			"AQ088": "84c",
			"AQ091": "84d",
		},
		"Urza's Tower": map[string]string{
			"AQ092": "85a",
			"AQ093": "85b",
			"AQ094": "85c",
			"AQ095": "85d",
		},
	*/
}

var pal01Variants = map[string]map[string]string{
	"Forest": map[string]string{
		"Arena 2001 Ice Age": "1",
		"Arena 2002 Beta":    "11",
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
		"APAC a Phillippines": "3",
		"APAC b Taiwan":       "8",
		"APAC c Japan":        "13",
	},
	"Plains": map[string]string{
		"APAC a Japan":     "4",
		"APAC b Australia": "9",
		"APAC c China":     "14",
	},
	"Swamp": map[string]string{
		"APAC a New Zealand": "5",
		"APAC b Taiwan":      "10",
		"APAC c Australia":   "15",
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

var oldLandVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"A Rocks":     "300",
		"B Path":      "301",
		"C Dark Tree": "302",
	},
	"Island": map[string]string{
		"A Purple":       "291",
		"B Light Purple": "292",
		"C Dark Purple":  "293",
	},
	"Mountain": map[string]string{
		"A Small Tree": "297",
		"B Snowy":      "298",
		"C Dark Red":   "299",
	},
	"Plains": map[string]string{
		"A Light":        "288",
		"B Little Trees": "289",
		"C Dark":         "290",
	},
	"Swamp": map[string]string{
		"A Light":        "294",
		"B Two Branches": "295",
		"C Dark":         "296",
	},
}

var pelpVariants = map[string]map[string]string{
	"Forest": map[string]string{
		"EURO a Germany":        "1",
		"EURO b France":         "6",
		"EURO c United Kingdom": "11",
	},
	"Island": map[string]string{
		"EURO a Scandanavia":    "2",
		"EURO b Italy":          "7",
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
		"EURO a Belgium":        "5",
		"EURO b United Kingdom": "10",
		"EURO c France":         "15",
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
		"183 Intro": "183a",
		"184 Intro": "184a",
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
		"":              "74",
		"Alternate Art": "74★",
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
		"B Path":      "305",
		"C Dark Tree": "306",
	},
	"Island": map[string]string{
		"A Purple":       "295",
		"B Light Purple": "296",
		"C Dark Purple":  "297",
	},
	"Mountain": map[string]string{
		"A Small Tree": "301",
		"B Snowy":      "302",
		"C Dark Red":   "303",
	},
	"Plains": map[string]string{
		"A Light":        "292",
		"B Little Trees": "293",
		"C Dark":         "294",
	},
	"Swamp": map[string]string{
		"A Light":        "298",
		"B Two Branches": "299",
		"C Dark":         "300",
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
	"Garbage Elemental": map[string]string{
		"3/1": "82b",
	},

	// haxx
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

var setVariants = map[string]map[string]map[string]string{
	"Anthologies":                   athVariants,
	"Antiquities":                   atqVariants,
	"Arena League 2001":             pal01Variants,
	"Asia Pacific Land Program":     palpVariants,
	"Battle Royale Box Set":         brbVariants,
	"Collectors’ Edition":           oldLandVariants,
	"European Land Program":         pelpVariants,
	"Fifth Edition":                 ed5Variants,
	"Fourth Edition":                ed4Variants,
	"GRN Ravnica Weekend":           prwkVariants,
	"Ice Age":                       iceVariants,
	"Intl. Collectors’ Edition":     oldLandVariants,
	"Introductory Two-Player Set":   itpVariants,
	"Limited Edition Alpha":         leaVariants,
	"Limited Edition Beta":          oldLandVariants,
	"Mirage":                        mirVariants,
	"Oath of the Gatewatch":         ogwVariants,
	"Planeshift":                    plsVariants,
	"Portal Second Age":             po2Variants,
	"Portal":                        porVariants,
	"Pro Tour Collector Set":        ptcVariants,
	"RNA Ravnica Weekend":           prw2Variants,
	"Revised Edition":               ed3Variants,
	"Secret Lair Drop":              sldVariants,
	"Tempest":                       tmpVariants,
	"Unlimited Edition":             oldLandVariants,
	"Unstable":                      ustVariants,
	"World Championship Decks 1999": wc99Variants,
	"World Championship Decks 2000": wc00Variants,
	"World Championship Decks 2001": wc01Variants,
	"World Championship Decks 2002": wc02Variants,
}
