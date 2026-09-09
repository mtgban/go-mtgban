package yugioh

import (
	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// promoTypeLabels are the words behind a token. Decoration is this side's
// job - the datastore publishes a slug and nothing else, and a slug cannot
// give back the boundaries it dropped, so "otsstamp" reads as "Otsstamp" to
// a title-caser and has to be written down instead.
//
// Only the tokens a title-caser gets wrong are here. A single word it gets
// right on its own - the colours, "emblazoned", "oversized" - so listing
// those would only be a second place to keep them in step with the first.
// The spellings are the catalog's own: "HERO Art", "ScR", "Blu-Ray DVD
// Promo", down to the hyphen in "Blue - DL18".
//
// A printing's mark is not here and is not a promo type: the ink it was made
// in, the version it is and the letter its artwork is filed under are
// published on their own and read into the watermark.
//
// The whole of a promotion's name is the variant beside it, which the builder
// publishes untouched: these are the words of the token, not of the printing.
var promoTypeLabels = map[string]string{
	"1steditionartwork":      "1st Edition Artwork",
	"200thycs":               "200th YCS",
	"25thanniversaryedition": "25th Anniversary Edition",
	"3rdart":                 "3rd Art",
	"4thart":                 "4th Art",
	"6thart":                 "6th Art",
	"7thart":                 "7th Art",
	"8thart":                 "8th Art",
	"9thart":                 "9th Art",
	"adidasexclusive":        "Adidas Exclusive",
	"ageofoverlord":          "Age of Overlord",
	"allianceinsight":        "Alliance Insight",
	"alternateart":           "Alternate Art",
	"backtoduel":             "Back to Duel",
	"battleofchaos":          "Battle of Chaos",
	"battlesoflegend":        "Battles of Legend",
	"blazingvortex":          "Blazing Vortex",
	"bluraydvdpromo":         "Blu-Ray DVD Promo",
	"burstofdestiny":         "Burst of Destiny",
	"burstprotocol":          "Burst Protocol",
	"chaosneosmisprint":      "Chaos Neos Misprint",
	"culinaryconfrontation":  "Culinary Confrontation",
	"datereprint":            "Date Reprint",
	"dawnofmajesty":          "Dawn of Majesty",
	"dimensionforce":         "Dimension Force",
	"doomofdimension":        "Doom of Dimension",
	"dueldevastator":         "Duel Devastator",
	"duelistadvanced":        "Duelist Advanced",
	"duelistnexus":           "Duelist Nexus",
	"duelterminal":           "Duel Terminal",
	"emblazonedalternateart": "Emblazoned Alternate Art",
	"emblazonedsecretrare":   "Emblazoned Secret Rare",
	"eu":                     "EU",
	"extendedart":            "Extended Art",
	"fishrecipe":             "Fish Recipe",
	"heroart":                "HERO Art",
	"infiniteforbidden":      "Infinite Forbidden",
	"japaneseart":            "Japanese Art",
	"japaneseexclusive":      "Japanese Exclusive",
	"judgestamp":             "Judge Stamp",
	"kidswb":                 "Kids WB",
	"legacyofdestruction":    "Legacy of Destruction",
	"lightningoverdrive":     "Lightning Overdrive",
	"mazeofmeurtos":          "Maze of Meurtos",
	"meatrecipe":             "Meat Recipe",
	"neuronengage":           "Neuron Engage!",
	"newart":                 "New Art",
	"originalart":            "Original Art",
	"originalartwork":        "Original Artwork",
	"otsstamp":               "OTS Stamp",
	"phantomrage":            "Phantom Rage",
	"phantomrevenge":         "Phantom Revenge",
	"photonhypernova":        "Photon Hypernova",
	"premieredition":         "Premier Edition",
	"preregistration":        "Pre-registration",
	"pur":                    "PUR",
	"rageoftheabyss":         "Rage of the Abyss",
	"regionalqualifierstamp": "Regional Qualifier Stamp",
	"remoteduelycs":          "Remote Duel YCS",
	"reprintartwork":         "Reprint Artwork",
	"samplepromo":            "Sample Promo",
	"scr":                    "ScR",
	"se":                     "SE",
	"skillcard":              "Skill Card",
	"staffrecipe":            "Staff Recipe",
	"starlightrare":          "Starlight Rare",
	"supremedarkness":        "Supreme Darkness",
	"todaysmenu":             "Today's Menu",
	"top8":                   "Top 8",
	"udsqualifier":           "UDS Qualifier",
	"unlimitedmisprint":      "Unlimited Misprint",
	"videogame":              "Video Game",
	"worldchampionship":      "World Championship",
	"yugiohday":              "Yu-Gi-Oh! Day",
}

// promoTypeLabel spells a token the way a reader should see it: the words
// above where a title-caser cannot work them out, and the title-cased token
// otherwise.
func promoTypeLabel(promoType string) string {
	if label, found := promoTypeLabels[promoType]; found {
		return label
	}
	return mtgmatcher.Title(promoType)
}
