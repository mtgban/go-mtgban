package mtgmatcher

import (
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

type cardinfo struct {
	Name      string
	Printings []string
	Layout    string
}

// CardObject is an extension of mtgjson.Card, containing fields that cannot
// be easily represented in the original object.
type CardObject struct {
	mtgjson.Card
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

var backend struct {
	// Slice of all set codes loaded
	AllSets []string

	// Map of set code : mtgjson.Set
	Sets map[string]*mtgjson.Set

	// Map of normalized name : cardinfo
	Cards map[string]cardinfo

	// Map of uuid ; CardObject
	UUIDs map[string]CardObject

	// Slice with token names (not normalized and without any "Token" tags)
	Tokens []string

	// Slice with every uniquely normalized name
	AllNames []string
	// Slice with every uniquely normalized product name
	AllSealed []string
	// Map of normalized names to slice of uuids
	Hashes map[string][]string

	// Map of normalized face/flavor names to canonical (non-normalized) names
	// with an extra property to determine FlavorNames
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
}

func okForTokens(set *mtgjson.Set) bool {
	return slices.Contains(setAllowedForTokens, set.Code) ||
		strings.Contains(set.Name, "Duel Deck")
}

func skipSet(set *mtgjson.Set) bool {
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

func sortPrintings(ap mtgjson.AllPrintings, printings []string) {
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

func NewDatastore(ap mtgjson.AllPrintings) {
	uuids := map[string]CardObject{}
	cards := map[string]cardinfo{}
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
		var filteredCards []mtgjson.Card

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
				_, found := alternates[Normalize(name)]
				if found {
					props.OriginalNumber = ""
				}
				alternates[Normalize(name)] = props
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
			_, found := cards[norm]
			if !found {
				cards[norm] = cardinfo{
					Name:      card.Name,
					Printings: card.Printings,
					Layout:    card.Layout,
				}
			} else if card.Layout == "token" {
				// If already present, check if this set is already contained
				// in the current array, otherwise add it
				shouldAddPrinting := true
				for _, printing := range cards[norm].Printings {
					if printing == code {
						shouldAddPrinting = false
						break
					}
				}
				// Note the setCode will be from the parent
				if shouldAddPrinting {
					printings := append(cards[norm].Printings, set.Code)
					sortPrintings(ap, printings)

					ci := cardinfo{
						Name:      card.Name,
						Printings: printings,
						Layout:    card.Layout,
					}
					cards[norm] = ci
				}
			}

			// Custom properties for tokens
			if card.Layout == "token" {
				card.Printings = cards[Normalize(card.Name+" Token")].Printings
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
			uuids[product.UUID] = CardObject{
				Card: mtgjson.Card{
					UUID:        product.UUID,
					Name:        product.Name,
					SetCode:     code,
					Identifiers: product.Identifiers,
					Rarity:      "Product",
					Layout:      product.Category,
					Side:        product.Subtype,
					// Will be filled later
					SourceProducts: map[string][]string{},
				},
				Sealed:  true,
				Edition: set.Name,
			}
		}
	}

	duplicate(ap.Data, cards, uuids, "Legends Italian", "LEG", "ITA", "1995-09-01")
	duplicate(ap.Data, cards, uuids, "The Dark Italian", "DRK", "ITA", "1995-08-01")
	duplicate(ap.Data, cards, uuids, "Alternate Fourth Edition", "4ED", "ALT", "1995-04-01")

	duplicateCards(ap.Data, cards, uuids, "SLD", "JPN", sldJPNLangDupes)
	duplicateCards(ap.Data, cards, uuids, "PURL", "JPN", []string{"1"})

	for setCode, numbers := range foilDupes {
		spinoffFoils(ap.Data, cards, uuids, setCode, numbers)
	}

	// Add all names and associated uuids to the global names and hashes arrays
	hashes := map[string][]string{}
	var names []string
	var sealed []string
	for uuid, card := range uuids {
		norm := Normalize(card.Name)
		_, found := hashes[norm]
		if !found {
			if card.Sealed {
				sealed = append(sealed, norm)
			} else {
				names = append(names, norm)
			}
		}
		hashes[norm] = append(hashes[norm], uuid)
	}
	// Add all alternative names too
	var altNames []string
	for altNorm, altProps := range alternates {
		altNames = append(altNames, altNorm)
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

	backend.Hashes = hashes
	backend.AllSets = allSets
	backend.AllUUIDs = allUUIDs
	backend.AllSealedUUIDs = allSealedUUIDs
	backend.AllNames = names
	backend.AllSealed = sealed
	backend.Sets = ap.Data
	backend.Cards = cards
	backend.Tokens = tokens
	backend.UUIDs = uuids
	backend.Scryfall = scryfall
	backend.Tcgplayer = tcgplayer
	backend.AlternateProps = alternates
	backend.AlternateNames = altNames
	backend.AllPromoTypes = promoTypes

	backend.CommanderKeywordMap = commanderKeywordMap

	fillinSealedContents()
	fillinSLDdecks()
}

func fillinSLDdecks() {
	for _, product := range backend.Sets["SLD"].SealedProduct {
		if strings.HasPrefix(product.Name, "Secret Lair Commander") {
			name := strings.TrimPrefix(product.Name, "Secret Lair Commander Deck ")
			if !slices.Contains(backend.SLDDeckNames, name) {
				backend.SLDDeckNames = append(backend.SLDDeckNames, name)
			}
		}
	}
}

// Add a map of which kind of products sealed contains
func fillinSealedContents() {
	result := map[string][]string{}
	tmp := map[string][]string{}

	for _, set := range backend.Sets {
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

	for uuid, co := range backend.UUIDs {
		if !co.Sealed {
			continue
		}

		res, found := result[uuid]
		if !found {
			continue
		}

		backend.UUIDs[uuid].SourceProducts["sealed"] = res
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

func duplicate(sets map[string]*mtgjson.Set, cards map[string]cardinfo, uuids map[string]CardObject, name, code, tag, date string) {
	// Copy base set information
	dup := *sets[code]

	// Update with new info
	dup.Name = name
	dup.Code = code + tag
	dup.ParentCode = code
	dup.ReleaseDate = date

	// Copy card information
	dup.Cards = make([]mtgjson.Card, len(sets[code].Cards))
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

		// Update printings for the CardInfo map
		ci := cards[Normalize(dup.Cards[i].Name)]
		ci.Printings = printings
		cards[Normalize(dup.Cards[i].Name)] = ci

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

func duplicateCards(sets map[string]*mtgjson.Set, cards map[string]cardinfo, uuids map[string]CardObject, code, tag string, numbers []string) {
	var duplicates []mtgjson.Card

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

func spinoffFoils(sets map[string]*mtgjson.Set, cards map[string]cardinfo, uuids map[string]CardObject, code string, numbers []string) {
	var newCardsArray []mtgjson.Card

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
		dupeCard.Number += mtgjson.SuffixSpecial
		dupeCard.Finishes = []string{"foil"}

		// Update or create the new card object, add the new card to the list
		co.Card = dupeCard
		uuids[dupeCard.UUID] = co
		newCardsArray = append(newCardsArray, dupeCard)
	}

	sets[code].Cards = newCardsArray
}

func LoadDatastore(reader io.Reader) error {
	allprints, err := mtgjson.LoadAllPrintings(reader)
	if err != nil {
		return err
	}

	NewDatastore(allprints)
	return nil
}

func LoadDatastoreFile(filename string) error {
	reader, err := os.Open(filename)
	if err != nil {
		return err
	}
	return LoadDatastore(reader)
}

func SetGlobalLogger(userLogger *log.Logger) {
	logger = userLogger
}
