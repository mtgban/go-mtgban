package strikezone

// Tags that should be wrapped in a variant () but they aren't
var tagsTable = []string{
	"2011 Holiday",
	"2015 Judge Promo",
	"BIBB",
	"BIBB Promo",
	"Bundle Promo",
	"Buy a Box",
	"Convention Foil M19",
	"Draft Weekend",
	"FNM",
	"Full Art Game Day Promo",
	"Full Box Promo",
	"Godzilla",
	"GP Promo",
	"Grand Prix 2018",
	"Holiday Promo",
	"Judge 2020",
	"Judge Promo",
	"League Promo",
	"LGS Promo",
	"M19 Prerelease",
	"MCQ Promo",
	"MagicFest 2019",
	"MagicFest 2020",
	"Media Promo",
	"Players Tour Promo",
	"Players Tour Qualifier PTQ Promo",
	"Prerelease Ixalan",
	"Prerelease Promo",
	"Prerelease",
	"SDCC 2015",
	"SDCC 2017",
	"Shooting Star Promo",
	"Stained Glass",
	"Standard Showdown 2017",
	"Standard Showdown 2018",
	"Store Champ",
	"Store Championship",
	"X Box Promo 2013",
	"Walmart Promo",
}

// Table for typos and errors in variants and card names
var cardTable = map[string]string{
	// Various promos
	"Flusterstorm Judge Promo (DCI)":         "Flusterstorm (Judge Promo)",
	"Island Ravnica Weekend A003":            "Island Ravnica Weekend A03",
	"Naughty Nice Holiday Promo":             "Naughty (Happy Holidays)",
	"Sol Ring GP MagicFest Promo (Foil)":     "Sol Ring (MagicFest 2019)",
	"Sol Ring GP MagicFest Promo (Non Foil)": "Sol Ring (MagicFest 2019)",
	"Spellseeker Judge":                      "Spellseeker (Judge)",
	"Underworld Dreams (DCI)":                "Underworld Dreams (2HG)",
	"Yixlid Jailer (DCI)":                    "Yixlid Jailer (WPN)",

	// SDCC
	"Ajani Caller of the Pride 2013 SDCC Comicon Promo":  "Ajani Caller of the Pride (2013 SDCC)",
	"Ajani Steadfast (2104 SDCC)":                        "Ajani Steadfast (2014 SDCC)",
	"Chandra Fire of Kaladesh SDCC 2015":                 "Chandra Fire of Kaladesh (2015 SDCC)",
	"Chandra Pyromaster SDCC Comicon Promo (2013)":       "Chandra Pyromaster (2013 SDCC)",
	"Chandra Torch of Defiance SDCC 2017":                "Chandra Torch of Defiance (2017 SDCC)",
	"Chandra Torch of Defiance SDCC 2018":                "Chandra Torch of Defiance (2018 SDCC)",
	"Garruk Apex Predator SDCC 2014":                     "Garruk Apex Predator (2014 SDCC)",
	"Garruk Caller of Beast 2013 SDCC Comicon Promo":     "Garruk Caller of Beasts (2013 SDCC)",
	"Jace Memory Adept 2013 SDCC Comicon Promo":          "Jace Memory Adept (2013 SDCC)",
	"Jace Vryns Prodigy SDCC 2015":                       "Jace Vryns Prodigy (2015 SDCC)",
	"Liliana Heretical Healer SDCC 2015":                 "Liliana Heretical Healer (2015 SDCC)",
	"Liliana of the Dark Realms 2013 SDCC Comicon Promo": "Liliana of the Dark Realms (2013 SDCC)",
	"Nissa Vastwood Seer SDCC 2015":                      "Nissa Vastwood Seer (2015 SDCC)",
	"Nissa Vital Force 2018 SDCC":                        "Nissa Vital Force (2018 SDCC)",

	// Real typos
	"Goldari Thug":               "Golgari Thug",
	"Ink-Eyes Servan of Oni":     "Ink-Eyes Servant of Oni",
	"Laughing Hyenas":            "Laughing Hyena",
	"Teshar Acnestor s Aspostle": "Teshar, Ancestor's Apostle",
	"Rin and Seri Inseperable":   "Rin and Seri, Inseparable",

	// Typos AND edition
	"Faithless Lootoing (IDW Promo)":                "Faithless Looting (IDW)",
	"Geist of Sain Traft (WMCQ)":                    "Geist of Saint Traft (WMCQ)",
	"Grafdiffer s Cage (Prerelease)":                "Grafdigger's Cage (Prerelease)",
	"Honor the Pure (M10 Release Event Promo)":      "Honor of the Pure (M10 Release Event Promo)",
	"Jace, Weilder of Mysteries Stained Glass":      "Jace, Wielder of Mysteries (Stained Glass)",
	"Karador Ghost Chieftan (Judge Promo)":          "Karador Ghost Chieftain (Judge Promo)",
	"Polukranos Unchanied (Prerelease)":             "Polukranos Unchained (Prerelease)",
	"Selfless Spirt (Eldritch Moon Prerelease)":     "Selfless Spirit (Prerelease)",
	"Showcase Aanax, Hardened in the Forge":         "Anax, Hardened in the Forge (Showcase)",
	"ShowcaseTeferi's Protege":                      "Teferi's Protege (Showcase)",
	"Some Disassembly Require (2017 Holiday Promo)": "Some Disassembly Required (Happy Holidays)",
	"SwampMagic Fest 2019":                          "Swamp (MagicFest 2019)",
	"Sengir, the Dark Baron (BIBB Alt Art Promo)":   "Sengir, the Dark Baron (Prerelease)",

	"Growing Rites of Ittlimoc (Itlimoc Cradle of the Sun BIBB Alt Art)": "Growing Rites of Itlimoc (BIBB Alt Art)",
	"Legion s Landing Adanto the First Fort (Prerelease)":                "Legion's Landing (Prerelease)",

	// Split cards
	"Discovery Dispersal": "Discovery",
	"Expansion Explosion": "Expansion",
	"Find Finality":       "Find",
	"Fire/Ice (Fire)":     "Fire",
	"Revival Revenge":     "Revival",
	"Thrash Threat":       "Thrash",
	"Connive Concoct":     "Connive",
	"Smelt / Herd / Saw":  "Smelt // Herd // Saw",

	"Needleverge Pathway / Pillarverge Pathway(Borderless": "Needleverge Pathway // Pillarverge Pathway",
	"Sejiri Shelter Sejiri Glacier":                        "Sejiri Shelter // Sejiri Glacier",
	"Silundi Vision Silundi Isle":                          "Silundi Vision // Silundi Isle",
	"Turntimber Symbyosis / Turntimber, Serpentine Wood":   "Turntimber Symbiosis // Turntimber, Serpentine Wood",

	// Funny cards
	"Who What When Where Why": "Who",
	"(Untitled Card":          "_____",
}

// Map a card/edition set to a correct edition
var card2setTable = map[string]string{
	"Counterspell (Arena Non Foil)":        "DCI Legend Membership",
	"Nicol Bolas Planeswalker (Archenemy)": "Archenemy: Nicol Bolas",
	"Serra Angel (Beta Art)":               "Wizards of the Coast Online Store",

	"Glistener Elf (WPN)":             "Friday Night Magic 2012",
	"Lingering Souls (WPN)":           "Friday Night Magic 2012",
	"Lightning Bolt (Beta Art Promo)": "Judge Gift Cards 1998",
	"Wood Elves (Promo)":              "Gateway 2006",
	"Elvish Lyrist (FNM)":             "Junior Super Series",

	"Tempered Steel (Full Art Textless)": "Scars of Mirrodin Promos",
}

// These cards don't have any variant, we know they are Promotional Cards,
// but we can't rely on the automatic matching
var promo2setTable = map[string]string{
	"Show and Tell":             "Judge Gift Cards 2013",
	"Overwhelming Forces":       "Judge Gift Cards 2013",
	"Nekusar the Mindrazer":     "Judge Gift Cards 2014",
	"Hanna Ship s Navigator":    "Judge Gift Cards 2014",
	"Riku of Two Reflections":   "Judge Gift Cards 2014",
	"Feldon of the Third Path":  "Judge Gift Cards 2015",
	"Yuriko the Tiger s Shadow": "Judge Gift Cards 2019",
	"Teetering Peaks":           "Friday Night Magic 2011",
	"Dismember":                 "Friday Night Magic 2012",
	"Ancient Grudge":            "Friday Night Magic 2012",
}
