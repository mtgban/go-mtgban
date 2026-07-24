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
