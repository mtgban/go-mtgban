package mtgmatcher

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
	"golang.org/x/exp/slices"
)

type DataStore interface {
	Load() cardBackend
}

type cardinfo struct {
	Name      string
	Printings []string
	Layout    string
}

// CardObject is an extension of mtgjson.Card, containing fields that cannot
// be easily represented in the original object.
type CardObject struct {
	Card
	Edition string
	Foil    bool
	Etched  bool
	Sealed  bool
}

// Card implements the Stringer interface
func (co CardObject) String() string {
	if co.Sealed {
		return co.Card.String()
	}
	finish := "nonfoil"
	if co.Etched {
		finish = "etched"
	} else if co.Foil {
		finish = "foil"
	}
	return fmt.Sprintf("%s|%s", co.Card, finish)
}

type alternateProps struct {
	OriginalName   string
	OriginalNumber string
	IsFlavor       bool
}

var backend cardBackend

type cardBackend struct {
	// Slice of all set codes loaded
	AllSets []string

	// Map of set code : Set
	Sets map[string]*Set

	// Map of normalized name : cardinfo
	// Only the main canonical name is stored here
	CardInfo map[string]cardinfo

	// Map of uuid : CardObject
	UUIDs map[string]CardObject

	// Slice with token names (not normalized and without any "Token" tags)
	Tokens []string

	// Slice with every uniquely normalized name
	AllNames []string
	// Slice with every unique name, as it would appear on a card
	AllCanonicalNames []string
	// Slice with every unique name, lower case
	AllLowerNames []string

	// Slice with every uniquely normalized product name
	AllSealed []string
	// Slice with every unique product name, as defined by mtgjson
	AllCanonicalSealed []string
	// Slice with every unique product name, lower case
	AllLowerSealed []string

	// Map of all normalized names to slice of uuids
	Hashes map[string][]string

	// Map of face/flavor names to set of canonical properties, such as original
	// name, and number, as well as a way to determine FlavorNames
	// Neither key nor values are normalized
	AlternateProps map[string]alternateProps

	// Slice with every uniquely normalized alternative name
	AlternateNames []string

	// Slice with every possible non-sealed uuid
	AllUUIDs []string
	// Slice with every possible sealed uuid
	AllSealedUUIDs []string

	// Scryfall UUID to MTGJSON UUID
	Scryfall map[string]string
	// TCG player Product ID to MTGJSON UUID
	Tcgplayer map[string]string

	// A list of keywords mapped to the full Commander set name
	CommanderKeywordMap map[string]string

	// A list of promo types as exported by mtgjson
	AllPromoTypes []string

	// A list of deck names of Secret Lair Commander cards
	SLDDeckNames []string
}

var logger = log.New(io.Discard, "", log.LstdFlags)

const (
	suffixFoil   = "_f"
	suffixEtched = "_e"
)

var languageCode2LanguageTag = map[string]string{
	"en":    "",
	"fr":    "French",
	"de":    "German",
	"it":    "Italian",
	"ja":    "Japanese",
	"ko":    "Korean",
	"ru":    "Russian",
	"es":    "Spanish",
	"pt":    "Portuguese",
	"pt-bz": "Portuguese",
	"zs":    "Chinese Simplified",
	"zt":    "Chinese Traditional",
	"zhs":   "Chinese Simplified",
	"zht":   "Chinese Traditional",
}

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
	"1110", "1111", "1112", "1113", "1114", "1115", "1116", "1117",
}

// List of numbers that need to have their number/uuid revisioned due
// to having foil and nonfoil merged in the same card object
var foilDupes = map[string][]string{
	"SLD": {
		"1614", "1615", "1616", "1617",
		"1618", "1619", "1620", "1621", "1622",
		"1623", "1624", "1625", "1626",
		"1703", "1704", "1705", "1706", "1707",
		"9990", "9991", "9992", "9993",
	},
	"M3C": {
		"9",
		"10", "11", "12", "13", "14", "15", "16", "32", "33", "34", "35", "36", "37", "38", "39", "40",
		"41", "42", "43", "44", "45", "46", "47", "48", "49", "50", "51", "52", "53", "54", "55", "56",
		"57", "58", "59", "60", "61", "62", "63", "64", "65", "66", "67", "68", "69", "70", "71", "72",
		"73", "74", "75", "76", "77", "78", "79", "80", "81", "82", "83", "92",
		"127", "134", "152", "154", "155", "156", "157", "158", "159", "160", "161", "162", "163", "164",
		"165", "166", "167", "168", "169", "170", "171", "172", "173", "174", "175", "176", "177", "178",
		"179", "180", "181", "182", "183", "184", "185", "186", "187", "188", "189", "190", "191", "192",
		"193", "194", "195", "196", "197", "198", "199", "200", "201", "202", "203", "204", "205", "206",
		"207", "208", "209", "210", "211", "212", "213", "214", "215", "216", "217", "218", "219", "220",
		"221", "222", "223", "224", "225", "226", "227", "228", "229", "230", "231", "232", "233", "234",
		"235", "236", "237", "238", "239", "240", "241", "242", "243", "244", "245", "246", "247", "248",
		"249", "250", "251", "252", "253", "254", "255", "256", "257", "258", "259", "260", "261", "262",
		"263", "264", "265", "266", "267", "268", "269", "270", "271", "272", "273", "274", "275", "276",
		"277", "278", "279", "280", "281", "282", "283", "284", "285", "286", "287", "288", "289", "290",
		"291", "292", "293", "294", "295", "296", "297", "298", "299", "300", "301", "302", "303", "304",
		"305", "306", "307", "308", "309", "310", "311", "312", "313", "314", "315", "316", "317", "318",
		"319", "320", "321", "322", "323", "324", "325", "326", "327", "328", "329", "330", "331", "332",
		"333", "334", "335", "336", "337", "338", "339", "340", "341", "342", "343", "344", "345", "346",
		"347", "348", "349", "350", "351", "352", "353", "354", "355", "356", "357", "358", "359", "360",
		"361", "362", "363", "364", "365", "366", "367", "368", "369", "370", "371", "372", "373", "374",
		"375", "376", "377", "378", "379", "380", "381", "382", "383", "384", "385", "386", "387", "388",
		"389", "390", "391", "392", "393", "394", "395", "396", "397", "398", "399", "400", "401", "402",
		"403", "404", "405", "406", "407", "408", "409", "410", "411",
	},
}

func okForTokens(set *Set) bool {
	return slices.Contains(setAllowedForTokens, set.Code) ||
		strings.Contains(set.Name, "Duel Deck")
}

func skipSet(set *Set) bool {
	// Skip unsupported sets
	switch set.Code {
	case "PRED", // a single foreign card
		"PSAL", "PS11", "PHUK", // salvat05, salvat11, hachette
		"OLGC", "OLEP", "OVNT", "O90P": // oversize
		return true
	}
	// Skip online sets, and any token-based sets
	if set.IsOnlineOnly ||
		(set.Type == "token" && !okForTokens(set)) ||
		strings.HasSuffix(set.Name, "Art Series") ||
		strings.HasSuffix(set.Name, "Minigames") ||
		strings.HasSuffix(set.Name, "Front Cards") ||
		strings.Contains(set.Name, "Heroes of the Realm") {
		return true
	}
	// In case there is nothing interesting in the set
	if len(set.Cards)+len(set.Tokens)+len(set.SealedProduct) == 0 {
		return true
	}
	return false
}

func sortPrintings(ap AllPrintings, printings []string) {
	sort.Slice(printings, func(i, j int) bool {
		setDateI, errI := time.Parse("2006-01-02", ap.Data[printings[i]].ReleaseDate)
		setDateJ, errJ := time.Parse("2006-01-02", ap.Data[printings[j]].ReleaseDate)
		if errI != nil || errJ != nil {
			return false
		}

		if setDateI.Equal(setDateJ) {
			return ap.Data[printings[i]].Name < ap.Data[printings[j]].Name
		}

		return setDateI.After(setDateJ)
	})
}

func scryfallImageURL(card Card, version string) string {
	number := card.Number

	// Retrieve the original number if present
	dupe, found := card.Identifiers["originalScryfallNumber"]
	if found {
		number = dupe
	}

	// Support BAN's custom sets
	code := strings.ToLower(card.SetCode)
	if strings.HasSuffix(code, "ita") {
		code = strings.TrimSuffix(code, "ita")
		number += "/it"
	} else if strings.HasSuffix(code, "jpn") {
		code = strings.TrimSuffix(code, "jpn")
		number += "/ja"
	}
	code = strings.TrimSuffix(code, "alt")

	return fmt.Sprintf("https://api.scryfall.com/cards/%s/%s?format=image&version=%s", code, number, version)
}

func sealedImageURL(card Card) string {
	tcgId, found := card.Identifiers["tcgplayerProductId"]
	if !found {
		return ""
	}
	return "https://product-images.tcgplayer.com/" + tcgId + ".jpg"
}

func (ap AllPrintings) Load() cardBackend {
	uuids := map[string]CardObject{}
	cardInfo := map[string]cardinfo{}
	scryfall := map[string]string{}
	tcgplayer := map[string]string{}
	alternates := map[string]alternateProps{}
	commanderKeywordMap := map[string]string{}
	var promoTypes []string
	var allCardNames []string
	var tokens []string
	var allSets []string

	for code, set := range ap.Data {
		// Filer out unneeded data
		if skipSet(set) {
			delete(ap.Data, code)
			continue
		}

		// Load all possible card names
		for _, card := range set.Cards {
			if !slices.Contains(allCardNames, card.Name) {
				allCardNames = append(allCardNames, card.Name)
			}
		}

		// Load token names (that don't have the same name of a real card)
		for _, token := range set.Tokens {
			if !slices.Contains(tokens, token.Name) && !slices.Contains(allCardNames, token.Name) {
				tokens = append(tokens, token.Name)
			}
		}
	}

	for code, set := range ap.Data {
		var filteredCards []Card

		allSets = append(allSets, code)

		allCards := set.Cards

		if okForTokens(set) {
			// Append tokens to the list of considered cards
			// if they are not named in the same way of a real card
			for _, token := range set.Tokens {
				if !slices.Contains(allCardNames, token.Name) {
					allCards = append(allCards, token)
				}
			}
		} else {
			// Clean a bit of memory
			set.Tokens = nil
		}

		switch set.Code {
		// Remove reference to an online-only set
		case "PMIC":
			set.ParentCode = ""
		}

		for _, card := range allCards {
			// Skip anything non-paper
			if card.IsOnlineOnly {
				continue
			}

			card.Images = map[string]string{}
			card.Images["full"] = scryfallImageURL(card, "normal")
			card.Images["thumbnail"] = scryfallImageURL(card, "small")
			card.Images["crop"] = scryfallImageURL(card, "art_crop")

			// Custom modifications or skips
			switch set.Code {
			// Override non-English Language
			case "FBB":
				card.Language = "Italian"
			case "4BB":
				card.Language = "Japanese"
			// Missing variant tags
			case "PALP":
				card.FlavorText = missingPALPtags[card.Number]
			case "PELP":
				card.FlavorText = missingPELPtags[card.Number]
			// Remove frame effects and borders where they don't belong
			case "STA", "PLST":
				card.FrameEffects = nil
				card.BorderColor = "black"
			case "SLD":
				switch card.Number {
				// One of the tokens is a DFC but burns a card number, skip it
				case "28":
					continue
				// Source is "technically correct" but it gets too messy to track
				case "589":
					card.Finishes = []string{"nonfoil", "etched"}
				default:
					num, _ := strconv.Atoi(card.Number)
					// Override the frame type for the Braindead drops
					if num == 821 || num == 824 || (num >= 1652 && num <= 1666) {
						card.FrameVersion = "2015"
					}
				}
			// Only keep dungeons, and fix their layout to make sure they are tokens
			case "AFR":
				if card.SetCode == "TAFR" {
					switch card.Number {
					case "20", "21", "22":
						card.Layout = "token"
					default:
						continue
					}
				}
			// Override all to tokens so that duplicates get named differently
			case "TFTH", "TBTH", "TDAG":
				card.Layout = "token"
			}

			// Override any "double_faced_token" entries and emblems
			if strings.Contains(card.Layout, "token") || card.Layout == "emblem" {
				card.Layout = "token"
			}

			// Make sure this property is correctly initialized
			if strings.HasSuffix(card.Number, "p") && !slices.Contains(card.PromoTypes, mtgjson.PromoTypePromoPack) {
				card.PromoTypes = append(card.PromoTypes, mtgjson.PromoTypePromoPack)
			}

			// Rename DFCs into a single name
			dfcSameName := card.IsDFCSameName()
			if dfcSameName {
				card.Name = strings.Split(card.Name, " // ")[0]
			}

			for i, name := range []string{card.FaceName, card.FlavorName, card.FaceFlavorName} {
				// Skip empty entries
				if name == "" {
					continue
				}
				// Only keep the main face
				if card.Layout == "token" {
					continue
				}
				// Skip FaceName entries that could be aliased
				// ie 'Start' could be Start//Finish and Start//Fire
				switch name {
				case "Bind",
					"Fire",
					"Smelt",
					"Start":
					continue
				}
				// Skip faces of DFCs with same names that aren't reskin version of other cars
				if dfcSameName && card.FlavorName == "" {
					continue
				}
				// Rename the sub-name of a DFC card
				if dfcSameName {
					name = strings.Split(name, " // ")[0]
				}

				// If the name is unique, keep track of the numbers so that they
				// can be decoupled later for reprints of the main card.
				// If the name is not unique, we might overwrite data and lose
				// track of the main version
				props := alternateProps{
					OriginalName:   card.Name,
					OriginalNumber: card.Number,
					IsFlavor:       i > 0,
				}
				_, found := alternates[name]
				if found {
					props.OriginalNumber = ""
				}
				alternates[name] = props
			}

			// MTGJSON v5 contains duplicated card info for each face, and we do
			// not need that level of detail, so just skip any extra side.
			if card.Side != "" && card.Side != "a" {
				continue
			}

			// Filter out unneeded printings
			var printings []string
			for i := range card.Printings {
				subset, found := ap.Data[card.Printings[i]]
				// If not found it means the set was already deleted above
				if !found || skipSet(subset) {
					continue
				}
				printings = append(printings, card.Printings[i])
			}
			// Sort printings by most recent sets first
			sortPrintings(ap, printings)

			card.Printings = printings

			// Tokens do not come with a printing array, add it
			// It'll be updated later with the sets discovered so far
			if card.Layout == "token" {
				card.Printings = []string{set.Code}
			}

			// Now assign the card to the list of cards to be saved
			filteredCards = append(filteredCards, card)

			// Quick dictionary of valid card names and their printings
			name := card.Name

			// Due to several cards having the same name of a token we hardcode
			// this value to tell them apart in the future -- checks and names
			// are still using the official Scryfall name (without the extra Token)
			if card.Layout == "token" {
				name += " Token"
			}

			// Deduplicate clashing names
			switch name {
			case "Pick Your Poison",
				"Red Herring":
				if strings.Contains(set.Name, "Playtest") {
					name += " Playtest"
				}
			}

			norm := Normalize(name)
			_, found := cardInfo[norm]
			if !found {
				cardInfo[norm] = cardinfo{
					Name:      card.Name,
					Printings: card.Printings,
					Layout:    card.Layout,
				}
			} else if card.Layout == "token" {
				// If already present, check if this set is already contained
				// in the current array, otherwise add it
				// Note the setCode will be from the parent
				if !slices.Contains(cardInfo[norm].Printings, code) {
					printings := append(cardInfo[norm].Printings, set.Code)
					sortPrintings(ap, printings)

					ci := cardinfo{
						Name:      card.Name,
						Printings: printings,
						Layout:    card.Layout,
					}
					cardInfo[norm] = ci
				}
			}

			// Custom properties for tokens
			if card.Layout == "token" {
				card.Printings = cardInfo[Normalize(card.Name+" Token")].Printings
				card.Rarity = "token"
			}
			if card.IsOversized {
				card.Rarity = "oversize"
			}

			// Initialize custom lookup tables
			scryfallId, found := card.Identifiers["scryfallId"]
			if found {
				scryfall[scryfallId] = card.UUID
			}
			for _, tag := range []string{"tcgplayerProductId", "tcgplayerEtchedProductId"} {
				tcgplayerId, found := card.Identifiers[tag]
				if found {
					tcgplayer[tcgplayerId] = card.UUID
				}
			}

			// Shared card object
			co := CardObject{
				Card:    card,
				Edition: set.Name,
			}

			// Save the original uuid
			co.Identifiers["mtgjsonId"] = card.UUID

			// Append "_f" and "_e" to uuids, unless etched is the only printing.
			// If it's not etched, append "_f", unless foil is the only printing.
			// Leave uuids unchanged, if there is a single printing of any kind.
			if card.HasFinish(mtgjson.FinishEtched) {
				uuid := card.UUID

				// Etched + Nonfoil [+ Foil]
				if card.HasFinish(mtgjson.FinishNonfoil) {
					// Save the card object
					uuids[uuid] = co
				}

				// Etched + Foil
				if card.HasFinish(mtgjson.FinishFoil) {
					// Set the main property
					co.Foil = true
					// Make sure "_f" is appended if a different version exists
					if card.HasFinish(mtgjson.FinishNonfoil) {
						uuid = card.UUID + suffixFoil
						co.UUID = uuid
					}
					// Save the card object
					uuids[uuid] = co
				}

				// Etched
				// Set the main properties
				co.Foil = false
				co.Etched = true
				// If there are alternative finishes, always append the suffix
				if card.HasFinish(mtgjson.FinishNonfoil) || card.HasFinish(mtgjson.FinishFoil) {
					uuid = card.UUID + suffixEtched
					co.UUID = uuid
				}
				// Save the card object
				uuids[uuid] = co
			} else if card.HasFinish(mtgjson.FinishFoil) {
				uuid := card.UUID

				// Foil [+ Nonfoil]
				if card.HasFinish(mtgjson.FinishNonfoil) {
					// Save the card object
					uuids[uuid] = co

					// Update the uuid for the *next* finish type
					uuid = card.UUID + suffixFoil
					co.UUID = uuid
				}

				// Foil
				co.Foil = true
				// Save the card object
				uuids[uuid] = co
			} else {
				// Single printing, use as-is
				uuids[card.UUID] = co
			}

			// Add to the ever growing list of promo types
			for _, promoType := range card.PromoTypes {
				if !slices.Contains(promoTypes, promoType) {
					promoTypes = append(promoTypes, promoType)
				}
			}
		}

		// Replace the original array with the filtered one
		set.Cards = filteredCards

		// Adjust the setBaseSize to take into account the cards with
		// the same name in the same set (also make sure that it is
		// correctly initialized)
		setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
		if err != nil {
			continue
		}
		if setDate.After(PromosForEverybodyYay) {
			for _, card := range set.Cards {
				if card.HasPromoType(mtgjson.PromoTypeBoosterfun) {
					// Usually boosterfun cards have real numbers
					cn, err := strconv.Atoi(card.Number)
					if err == nil {
						set.BaseSetSize = cn - 1
					}
					break
				}
			}
		}

		// Retrieve the best describing word for a commander set and save it for later reuse
		if strings.HasSuffix(set.Name, "Commander") && !strings.Contains(set.Name, "Display") {
			keyword := longestWordInEditionName(strings.TrimSuffix(set.Name, "Commander"))
			commanderKeywordMap[keyword] = set.Name
		}

		for _, product := range set.SealedProduct {
			card := Card{
				UUID:        product.UUID,
				Name:        product.Name,
				SetCode:     code,
				Identifiers: product.Identifiers,
				Rarity:      "Product",
				Layout:      product.Category,
				Side:        product.Subtype,
				// Will be filled later
				SourceProducts: map[string][]string{},
				Images:         map[string]string{},
			}

			card.Images["full"] = sealedImageURL(card)
			card.Images["thumbnail"] = sealedImageURL(card)
			card.Images["crop"] = sealedImageURL(card)

			uuids[product.UUID] = CardObject{
				Card:    card,
				Sealed:  true,
				Edition: set.Name,
			}
		}
	}

	duplicate(ap.Data, cardInfo, uuids, "Legends Italian", "LEG", "ITA", "1995-09-01")
	duplicate(ap.Data, cardInfo, uuids, "The Dark Italian", "DRK", "ITA", "1995-08-01")
	duplicate(ap.Data, cardInfo, uuids, "Alternate Fourth Edition", "4ED", "ALT", "1995-04-01")
	allSets = append(allSets, "LEGITA", "DRKITA", "4EDALT")

	duplicateCards(ap.Data, uuids, "SLD", "JPN", sldJPNLangDupes)
	duplicateCards(ap.Data, uuids, "PURL", "JPN", []string{"1"})

	for setCode, numbers := range foilDupes {
		spinoffFoils(ap.Data, uuids, setCode, numbers)
	}

	// Add all names and associated uuids to the global names and hashes arrays
	hashes := map[string][]string{}
	var names, fullNames, lowerNames []string
	var sealed, fullSealed, lowerSealed []string
	for uuid, card := range uuids {
		norm := Normalize(card.Name)
		_, found := hashes[norm]
		if !found {
			if card.Sealed {
				sealed = append(sealed, norm)
				fullSealed = append(fullSealed, card.Name)
				lowerSealed = append(lowerSealed, strings.ToLower(card.Name))
			} else {
				names = append(names, norm)
				fullNames = append(fullNames, card.Name)
				lowerNames = append(lowerNames, strings.ToLower(card.Name))
			}
		}
		hashes[norm] = append(hashes[norm], uuid)
	}
	// Add all alternative names too
	var altNames []string
	for altName, altProps := range alternates {
		altNorm := Normalize(altName)
		altNames = append(altNames, altNorm)
		fullNames = append(fullNames, altName)
		lowerNames = append(lowerNames, strings.ToLower(altName))
		if altProps.IsFlavor {
			// Retrieve all the uuids with a FlavorName attached
			allAltUUIDs := hashes[Normalize(altProps.OriginalName)]
			for _, uuid := range allAltUUIDs {
				if uuids[uuid].FlavorName != "" {
					hashes[altNorm] = append(hashes[altNorm], uuid)
				}
			}
		} else {
			// Copy the original uuids
			hashes[altNorm] = append(hashes[altNorm], hashes[Normalize(altProps.OriginalName)]...)
		}
	}

	// Finally save all the  uuids generated
	var allUUIDs []string
	var allSealedUUIDs []string
	for uuid, co := range uuids {
		if co.Sealed {
			allSealedUUIDs = append(allSealedUUIDs, uuid)
			continue
		}
		allUUIDs = append(allUUIDs, uuid)
	}

	// Remove promo tags that apply to a single finish only
	for uuid, card := range uuids {
		if !card.Foil && !card.Etched {
			for _, promoType := range []string{
				mtgjson.PromoTypeSilverFoil,
				mtgjson.PromoTypeRainbowFoil,
				mtgjson.PromoTypeRippleFoil,
			} {
				if card.HasPromoType(promoType) {
					var filtered []string
					for _, pt := range card.PromoTypes {
						if pt != promoType {
							filtered = append(filtered, pt)
						}
					}
					card.PromoTypes = filtered
					uuids[uuid] = card
				}
			}
		}
	}

	sort.Strings(promoTypes)
	sort.Strings(allSets)

	sort.Strings(names)
	sort.Strings(fullNames)
	sort.Strings(lowerNames)
	sort.Strings(sealed)
	sort.Strings(fullSealed)
	sort.Strings(lowerSealed)

	fillinSealedContents(ap.Data, uuids)

	var backend cardBackend

	backend.Hashes = hashes
	backend.AllSets = allSets
	backend.AllUUIDs = allUUIDs
	backend.AllSealedUUIDs = allSealedUUIDs

	backend.AllNames = names
	backend.AllCanonicalNames = fullNames
	backend.AllLowerNames = lowerNames

	backend.AllSealed = sealed
	backend.AllCanonicalSealed = fullSealed
	backend.AllLowerSealed = lowerSealed

	backend.Sets = ap.Data
	backend.CardInfo = cardInfo
	backend.Tokens = tokens
	backend.UUIDs = uuids
	backend.Scryfall = scryfall
	backend.Tcgplayer = tcgplayer
	backend.AlternateProps = alternates
	backend.AlternateNames = altNames
	backend.AllPromoTypes = promoTypes

	backend.CommanderKeywordMap = commanderKeywordMap
	backend.SLDDeckNames = fillinSLDdecks(ap.Data["SLD"])

	return backend
}

func fillinSLDdecks(set *Set) []string {
	var output []string
	for _, product := range set.SealedProduct {
		if strings.HasPrefix(product.Name, "Secret Lair Commander") {
			name := strings.TrimPrefix(product.Name, "Secret Lair Commander Deck ")
			if !slices.Contains(output, name) {
				output = append(output, name)
			}
		}
	}
	return output
}

// Add a map of which kind of products sealed contains
func fillinSealedContents(sets map[string]*Set, uuids map[string]CardObject) {
	result := map[string][]string{}
	tmp := map[string][]string{}

	for _, set := range sets {
		for _, product := range set.SealedProduct {
			dedup := map[string]int{}
			list := SealedWithinSealed(set.Code, product.UUID)
			for _, item := range list {
				dedup[item]++
			}
			for uuid := range dedup {
				tmp[product.UUID] = append(tmp[product.UUID], uuid)
			}
		}
	}

	// Reverse to be compatible with SourceProducts model
	for _, list := range tmp {
		for _, item := range list {
			for key, sublist := range tmp {
				// Add if item is in the sublist, and the key was not already added
				if slices.Contains(sublist, item) && !slices.Contains(result[item], key) {
					result[item] = append(result[item], key)
				}
			}
		}
	}

	for uuid, co := range uuids {
		if !co.Sealed {
			continue
		}

		res, found := result[uuid]
		if !found {
			continue
		}

		uuids[uuid].SourceProducts["sealed"] = res
	}
}

// Match the name of the deck with the product UUID(s)
func findDeck(setCode, deckName string) []string {
	var list []string

	set, found := backend.Sets[setCode]
	if !found {
		return nil
	}

	for _, deck := range set.Decks {
		if deck.Name != deckName {
			continue
		}
		list = append(list, deck.SealedProductUUIDs...)
	}

	return list
}

// Return a list of sealed products contained by the input product
// Decks and Packs and Card cannot contain other sealed product, so they are ignored here
func SealedWithinSealed(setCode, sealedUUID string) []string {
	var list []string

	set, found := backend.Sets[setCode]
	if !found {
		return nil
	}

	for _, product := range set.SealedProduct {
		if sealedUUID != product.UUID {
			continue
		}

		for key, contents := range product.Contents {
			for _, content := range contents {
				switch key {
				case "sealed":
					list = append(list, content.UUID)

				case "variable":
					for _, config := range content.Configs {
						for _, sealed := range config["sealed"] {
							list = append(list, sealed.UUID)
						}
						for _, deck := range config["deck"] {
							decklist := findDeck(deck.Set, deck.Name)
							list = append(list, decklist...)
						}
					}
				}
			}
		}
	}

	return list
}

var langs = map[string]string{
	"JPN": "Japanese",
	"ITA": "Italian",
	"ALT": "English",
}

func duplicate(sets map[string]*Set, cardInfo map[string]cardinfo, uuids map[string]CardObject, name, code, tag, date string) {
	// Copy base set information
	dup := *sets[code]

	// Update with new info
	dup.Name = name
	dup.Code = code + tag
	dup.ParentCode = code
	dup.ReleaseDate = date

	// Copy card information
	dup.Cards = make([]Card, len(sets[code].Cards))
	for i := range sets[code].Cards {
		// Skip misprints from main sets
		if strings.HasSuffix(sets[code].Cards[i].Number, mtgjson.SuffixVariant) {
			continue
		}

		// Update printings for the original set
		printings := append(sets[code].Cards[i].Printings, dup.Code)
		sets[code].Cards[i].Printings = printings

		// Loop through all other sets mentioned
		for _, setCode := range printings {
			// Skip the set being added, there might be cards containing
			// the set code being processed due to variants
			if setCode == dup.Code {
				continue
			}
			_, found := sets[setCode]
			if !found {
				continue
			}
			if skipSet(sets[setCode]) {
				continue
			}

			for j := range sets[setCode].Cards {
				// Name match, can't break after the first because there could
				// be other variants
				if sets[setCode].Cards[j].Name == sets[code].Cards[i].Name {
					sets[setCode].Cards[j].Printings = printings
				}
			}
		}

		// Update with new info
		dup.Cards[i] = sets[code].Cards[i]
		dup.Cards[i].UUID += "_" + strings.ToLower(tag)
		dup.Cards[i].SetCode = dup.Code
		dup.Cards[i].Language = langs[tag]

		// Update images
		dup.Cards[i].Images = map[string]string{}
		dup.Cards[i].Images["full"] = scryfallImageURL(dup.Cards[i], "normal")
		dup.Cards[i].Images["thumbnail"] = scryfallImageURL(dup.Cards[i], "small")
		dup.Cards[i].Images["crop"] = scryfallImageURL(dup.Cards[i], "art_crop")

		// Update printings for the CardInfo map
		ci := cardInfo[Normalize(dup.Cards[i].Name)]
		ci.Printings = printings
		cardInfo[Normalize(dup.Cards[i].Name)] = ci

		// Remove store references to avoid duplicates
		altIdentifiers := map[string]string{}
		for k, v := range dup.Cards[i].Identifiers {
			altIdentifiers[k] = v
		}
		delete(altIdentifiers, "tcgplayerProductId")
		dup.Cards[i].Identifiers = altIdentifiers

		// Add the new uuid to the UUID map
		uuids[dup.Cards[i].UUID] = CardObject{
			Card:    dup.Cards[i],
			Edition: name,
		}
	}

	sets[dup.Code] = &dup
}

func duplicateCards(sets map[string]*Set, uuids map[string]CardObject, code, tag string, numbers []string) {
	var duplicates []Card

	for i := range sets[code].Cards {
		// Skip unneeded
		if !slices.Contains(numbers, sets[code].Cards[i].Number) {
			continue
		}

		mainUUID := sets[code].Cards[i].UUID

		// Update with new info
		dupeCard := sets[code].Cards[i]
		dupeCard.UUID = mainUUID + "_" + strings.ToLower(tag)
		dupeCard.Language = langs[tag]

		// Update images
		dupeCard.Images = map[string]string{}
		dupeCard.Images["full"] = scryfallImageURL(dupeCard, "normal")
		dupeCard.Images["thumbnail"] = scryfallImageURL(dupeCard, "small")
		dupeCard.Images["crop"] = scryfallImageURL(dupeCard, "art_crop")

		duplicates = append(duplicates, dupeCard)

		// Add the new uuid to the UUID map
		for _, suffixTag := range []string{suffixEtched, suffixFoil, ""} {
			uuid := mainUUID + suffixTag
			co, found := uuids[uuid]
			if !found {
				continue
			}

			dupeCard.UUID = mainUUID + "_" + strings.ToLower(tag) + suffixTag
			uuids[dupeCard.UUID] = CardObject{
				Card:    dupeCard,
				Edition: sets[code].Name,
				Etched:  co.Etched,
				Foil:    co.Foil,
			}
		}
	}

	sets[code].Cards = append(sets[code].Cards, duplicates...)
}

func spinoffFoils(sets map[string]*Set, uuids map[string]CardObject, code string, numbers []string) {
	var newCardsArray []Card

	for i := range sets[code].Cards {
		dupeCard := sets[code].Cards[i]

		// Skip unneeded (just preserve the card as-is)
		if !slices.Contains(numbers, sets[code].Cards[i].Number) {
			newCardsArray = append(newCardsArray, dupeCard)
			continue
		}

		// Retrieve the main card object
		co, found := uuids[dupeCard.UUID]
		if !found {
			continue
		}

		// Change properties
		dupeCard.Finishes = []string{"nonfoil"}

		// Propagate changes across the board
		co.Card = dupeCard
		uuids[dupeCard.UUID] = co
		newCardsArray = append(newCardsArray, dupeCard)

		// Move to the foil version
		co, found = uuids[dupeCard.UUID+suffixFoil]
		if !found {
			continue
		}

		// Change properties
		delete(uuids, dupeCard.UUID+suffixFoil)
		dupeCard.UUID = strings.Split(dupeCard.UUID, "_")[0] + "+foil"
		dupeCard.Identifiers["originalScryfallNumber"] = dupeCard.Number
		dupeCard.Number += mtgjson.SuffixSpecial
		dupeCard.Finishes = []string{"foil"}

		// Update images
		dupeCard.Images = map[string]string{}
		dupeCard.Images["full"] = scryfallImageURL(dupeCard, "normal")
		dupeCard.Images["thumbnail"] = scryfallImageURL(dupeCard, "small")
		dupeCard.Images["crop"] = scryfallImageURL(dupeCard, "art_crop")

		// Update or create the new card object, add the new card to the list
		co.Card = dupeCard
		uuids[dupeCard.UUID] = co
		newCardsArray = append(newCardsArray, dupeCard)
	}

	sets[code].Cards = newCardsArray
}

func SetGlobalDatastore(datastore cardBackend) {
	backend = datastore
}

func LoadDatastore(reader io.Reader) error {
	var buf bytes.Buffer
	tee := io.TeeReader(reader, &buf)

	datastore, err := LoadAllPrintings(tee)
	if err != nil {
		datastore, err = LoadLorcana(&buf)
		if err != nil {
			return err
		}
	}

	backend = datastore.Load()
	return nil
}

func LoadDatastoreFile(filename string) error {
	reader, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer reader.Close()
	return LoadDatastore(reader)
}

func SetGlobalLogger(userLogger *log.Logger) {
	logger = userLogger
}
