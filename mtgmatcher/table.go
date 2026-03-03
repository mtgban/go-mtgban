package mtgmatcher

var LanguageCode2LanguageTag = map[string]string{
	"en":    "",
	"fr":    "French",
	"de":    "German",
	"it":    "Italian",
	"ja":    "Japanese",
	"jp":    "Japanese",
	"ko":    "Korean",
	"ru":    "Russian",
	"es":    "Spanish",
	"pt":    "Portuguese",
	"pt-bz": "Portuguese",
	"cs":    "Chinese Simplified",
	"ct":    "Chinese Traditional",
	"zs":    "Chinese Simplified",
	"zt":    "Chinese Traditional",
	"zhs":   "Chinese Simplified",
	"zht":   "Chinese Traditional",
	"phi":   "Phyrexian",
	"qya":   "Quenya",
}

var LanguageTag2LanguageCode = map[string]string{
	"":                    "en",
	"English":             "en",
	"French":              "fr",
	"German":              "de",
	"Italian":             "it",
	"Japanese":            "ja",
	"Korean":              "ko",
	"Russian":             "ru",
	"Spanish":             "es",
	"Portuguese":          "pt",
	"Chinese Simplified":  "zhs",
	"Chinese Traditional": "zht",
	"Phyrexian":           "phi",
	"Quenya":              "qya",
}

// This set skips any fictional language
var allLanguageTags = []string{
	"French",
	"German",
	"Italian",
	"Japanese",
	"Korean",
	"Russian",
	"Spanish",

	// Not languages but unique tags found in the language field
	"Brazil",
	"Simplified",
	"Traditional",

	// Languages affected by the tags above
	"Chinese",
	"Portuguese",
}

// Editions with interesting tokens
var setAllowedForTokens = []string{
	// League Tokens
	"L12",
	"L13",
	"L14",
	"L15",
	"L16",
	"L17",

	// Magic Player Rewards
	"MPR",
	"PR2",
	"P03",
	"P04",

	// FNM
	"F12",
	"F17",
	"F18",

	// FtV: Lore
	"V16",

	// Holiday
	"H17",

	// Secret lair
	"SLD",

	// Guild kits
	"GK1",
	"GK2",

	// Global series
	"GS1",

	// Token sets
	"PHEL",
	"PL21",
	"PLNY",
	"WDMU",

	"10E",
	"30A",
	"A25",
	"AFR",
	"ALA",
	"ARB",
	"BFZ",
	"BNG",
	"CLB",
	"DKA",
	"DMU",
	"DOM",
	"FRF",
	"ISD",
	"JOU",
	"M15",
	"MBS",
	"NPH",
	"NEO",
	"NEC",
	"RTR",
	"SOM",
	"SHM",
	"WAR",
	"ZEN",

	// Theros token sets
	"TBTH",
	"TDAG",
	"TFTH",

	// Funny token sets
	"SUNF",
	"UGL",
	"UNF",
	"UST",
}

var missingPELPtags = map[string]string{
	"1":  "Schwarzwald, Germany",
	"2":  "Danish Island, Scandinavia",
	"3":  "Vesuvio, Italy",
	"4":  "Scottish Highlands, United Kingdom, U.K.",
	"5":  "Ardennes Fagnes, Belgium",
	"6":  "Brocéliande, France",
	"7":  "Venezia, Italy",
	"8":  "Pyrenees, Spain",
	"9":  "Lowlands, Netherlands",
	"10": "Lake District National Park, United Kingdom, U.K.",
	"11": "Nottingham Forest, United Kingdom, U.K.",
	"12": "White Cliffs of Dover, United Kingdom, U.K.",
	"13": "Mont Blanc, France",
	"14": "Steppe Tundra, Russia",
	"15": "Camargue, France",
}

var missingPALPtags = map[string]string{
	"1":  "Japan",
	"2":  "Hong Kong",
	"3":  "Banaue Rice Terraces, Philippines",
	"4":  "Japan",
	"5":  "New Zealand",
	"6":  "China",
	"7":  "Meoto Iwa, Japan",
	"8":  "Taiwan",
	"9":  "Uluru, Australia",
	"10": "Japan",
	"11": "Korea",
	"12": "Singapore",
	"13": "Mount Fuji, Japan",
	"14": "Great Wall of China",
	"15": "Indonesia",
}

// List of numbers in SLD that need to be decoupled
var sldJPNLangDupes = []string{
	// Special Guests Yoji Shinkawa
	"1110", "1111", "1112", "1113",
	// Special Guests Junji Ito
	"1114", "1115", "1116", "1117",
	// Miku Sakura Superstar
	"1587", "1594", "1596", "1597", "805", "808",
	"1587★", "1594★", "1596★", "1597★",
	// Miku Digital Sensation
	"1592", "1595", "1599", "1603", "1604", "1607", "806",
	// Miku Electric Entourage
	"1585", "1590", "1593", "1598", "1600", "807",
	// Miku Winter Diva
	"1586", "1588", "1589", "1591", "1601", "1606", "804",
	// Final Fantasy Game Over
	"1858", "1859", "1860", "1861", "1862",
	// Final Fantasy Weapons
	"1863", "1864", "1865", "1866", "1867",
	// Final Fantasy Grimoire
	"1868", "1869", "1870", "1871", "1872",
	// Summer Superdrop 2025 promo
	"909",
}

var productsWithOnlyFoils = []string{
	"Secret Lair Drop Seeing Visions",
	"Secret Lair Drop OMG KITTIES",
	"Secret Lair Drop Kaleidoscope Killers",
	"Secret Lair Drop Eldraine Wonderland",
	"Secret Lair Drop Year of the Rat",
	"Secret Lair Drop Theros Stargazing Vol I Heliod",
	"Secret Lair Drop Theros Stargazing Vol II Thassa",
	"Secret Lair Drop Theros Stargazing Vol III Erebos",
	"Secret Lair Drop Theros Stargazing Vol IV Purphoros",
	"Secret Lair Drop Theros Stargazing Vol V Nylea",
	"Secret Lair Bundle Theros Stargazing Vol I-V",
	"Secret Lair Drop International Womens Day 2020",
	"Secret Lair Drop Thalia Beyond the Helvault",
	"Secret Lair Drop April Fools",
	"Secret Lair Drop The Godzilla Lands",
	"Secret Lair Drop Can You Feel with a Heart of Steel",
	"Secret Lair Drop Mountain Go",
	"Secret Lair Drop The Path Not Traveled",
	"Secret Lair Drop Extra Life 2020",
	"Secret Lair Drop We Hope You Like Squirrels",
	"Secret Lair Drop Here Be Dragons",
	"Secret Lair Drop LOOK AT THE KITTIES",
	"Secret Lair Drop Animar and Friends",
	"Secret Lair Drop MagicCon The Gathering",
	"Secret Lair Drop Calling All Hydra Heads",
}
