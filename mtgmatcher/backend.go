package mtgmatcher

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
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
	finish := "nonfoil"
	if co.Etched {
		finish = "etched"
	} else if co.Foil {
		finish = "foil"
	}
	return fmt.Sprintf("%s|%s", co.Card, finish)
}

type alternateProps struct {
	OriginalName string
	IsFlavor     bool
}

var backend struct {
	// Map of set code : mtgjson.Set
	Sets map[string]*mtgjson.Set

	// Map of normalized name : cardinfo
	Cards map[string]cardinfo

	// Map of uuid ; CardObject
	UUIDs map[string]CardObject

	// Map with token names
	Tokens map[string]bool
	// DFC with equal names on both sides
	DFCSameNames map[string]bool

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

	// Slice with every possible uuid
	AllUUIDs []string

	Scryfall  map[string]string
	Tcgplayer map[string]string
}

var logger = log.New(ioutil.Discard, "", log.LstdFlags)

const (
	suffixFoil   = "_f"
	suffixEtched = "_e"
)

// Editions with interesting tokens
var setAllowedForTokens = map[string]bool{
	// League Tokens
	"L12": true,
	"L13": true,
	"L14": true,
	"L15": true,
	"L16": true,
	"L17": true,

	// Magic Player Rewards
	"MPR": true,
	"PR2": true,
	"P03": true,
	"P04": true,

	// FNM
	"F12": true,
	"F17": true,
	"F18": true,

	// FtV: Lore
	"V16": true,

	// Holiday
	"H17": true,

	// Secret lair
	"SLD": true,

	// Guild kits
	"GK1": true,
	"GK2": true,

	// Token sets
	"PHEL": true,
	"PL21": true,
	"PLNY": true,
	"WDMU": true,

	"10E": true,
	"A25": true,
	"AFR": true,
	"ALA": true,
	"ARB": true,
	"BFZ": true,
	"BNG": true,
	"DKA": true,
	"DMU": true,
	"DOM": true,
	"FRF": true,
	"ISD": true,
	"JOU": true,
	"M15": true,
	"MBS": true,
	"NPH": true,
	"NEO": true,
	"RTR": true,
	"SOM": true,
	"SHM": true,
	"WAR": true,
	"ZEN": true,

	// Theros token sets
	"TBTH": true,
	"TDAG": true,
	"TFTH": true,

	// Funny token sets
	"SUNF": true,
	"UGL":  true,
	"UNF":  true,
	"UST":  true,
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

var specialTags = map[string]string{
	"Badlands":            "dual",
	"Bayou":               "dual",
	"Plateau":             "dual",
	"Savannah":            "dual",
	"Scrubland":           "dual",
	"Taiga":               "dual",
	"Tropical Island":     "dual",
	"Tundra":              "dual",
	"Underground Sea":     "dual",
	"Volcanic Island":     "dual",
	"Blackcleave Cliffs":  "fastland",
	"Blooming Marsh":      "fastland",
	"Botanical Sanctum":   "fastland",
	"Concealed Courtyard": "fastland",
	"Copperline Gorge":    "fastland",
	"Darkslick Shores":    "fastland",
	"Inspiring Vantage":   "fastland",
	"Razorverge Thicket":  "fastland",
	"Seachrome Coast":     "fastland",
	"Spirebluff Canal":    "fastland",
	"Arid Mesa":           "fetchland",
	"Bloodstained Mire":   "fetchland",
	"Flooded Strand":      "fetchland",
	"Marsh Flats":         "fetchland",
	"Misty Rainforest":    "fetchland",
	"Polluted Delta":      "fetchland",
	"Scalding Tarn":       "fetchland",
	"Verdant Catacombs":   "fetchland",
	"Windswept Heath":     "fetchland",
	"Wooded Foothills":    "fetchland",
	"Adarkar Wastes":      "painland",
	"Battlefield Forge":   "painland",
	"Brushland":           "painland",
	"Caves of Koilos":     "painland",
	"Karplusan Forest":    "painland",
	"Llanowar Wastes":     "painland",
	"Shivan Reef":         "painland",
	"Sulfurous Springs":   "painland",
	"Underground River":   "painland",
	"Yavimaya Coast":      "painland",
	"Blood Crypt":         "shockland",
	"Breeding Pool":       "shockland",
	"Godless Shrine":      "shockland",
	"Hallowed Fountain":   "shockland",
	"Overgrown Tomb":      "shockland",
	"Sacred Foundry":      "shockland",
	"Steam Vents":         "shockland",
	"Stomping Ground":     "shockland",
	"Temple Garden":       "shockland",
	"Watery Grave":        "shockland",
}

func okForTokens(set *mtgjson.Set) bool {
	return setAllowedForTokens[set.Code] ||
		strings.Contains(set.Name, "Duel Deck")
}

func skipSet(set *mtgjson.Set) bool {
	// Skip unsupported sets
	switch set.Code {
	case "PRED", // a single foreign card
		"PSAL", "PS11", "PHUK", "PHJ", // foreign-only
		"OLGC", "PMIC", "OVNT": // oversize
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
	// In case of incorrect data present in the file
	if len(set.Cards)+len(set.Tokens) == 0 {
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
	tokens := map[string]bool{}
	dfcSameNames := map[string]bool{}
	scryfall := map[string]string{}
	tcgplayer := map[string]string{}
	alternates := map[string]alternateProps{}

	for code, set := range ap.Data {
		if skipSet(set) {
			delete(ap.Data, code)
			continue
		}

		var filteredCards []mtgjson.Card

		allCards := set.Cards

		// Load token names
		for _, token := range set.Tokens {
			tokens[token.Name] = true
		}

		if okForTokens(set) {
			// Append tokens to the list of considered cards
			allCards = append(allCards, set.Tokens...)
		} else {
			// Clean a bit of memory
			set.Tokens = nil
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
			// Skip non-English promo cards
			case "PLG21":
				switch card.Number {
				case "C1", "C2":
					continue
				}
			// Remove frame effects and borders where they don't belong
			case "STA", "PLIST":
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

			// Set any custom tag
			customTag, found := specialTags[card.Name]
			if found {
				card.Identifiers["customTag"] = customTag
			}

			// Override any "double_faced_token" entries and emblems
			if strings.Contains(card.Layout, "token") || card.Layout == "emblem" {
				card.Layout = "token"
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
				// Skip faces of DFCs with same names that aren't reskin version of other cars,
				// so that face names don't pollute the main dictionary with a wrong rename
				if set.Code == "SLD" && card.IsDFCSameName() && card.FlavorName == "" {
					// Save the names so that we don't have to keep a list
					dfcSameNames[Normalize(name)] = true
					continue
				}
				alternates[Normalize(name)] = alternateProps{
					OriginalName: card.Name,
					IsFlavor:     i > 0,
				}
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

			norm := Normalize(name)
			_, found = cards[norm]
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

		for _, product := range set.SealedProduct {
			uuids[product.UUID] = CardObject{
				Card: mtgjson.Card{
					UUID:        product.UUID,
					Name:        product.Name,
					SetCode:     code,
					Identifiers: product.Identifiers,
					Rarity:      "Product",
				},
				Sealed:  true,
				Edition: set.Name,
			}
		}
	}

	duplicate(ap.Data, cards, uuids, "Legends Italian", "LEG", "ITA", "1995-09-01")
	duplicate(ap.Data, cards, uuids, "The Dark Italian", "DRK", "ITA", "1995-08-01")
	duplicate(ap.Data, cards, uuids, "Chronicles Japanese", "CHR", "JPN", "1995-07-01")
	duplicate(ap.Data, cards, uuids, "Alternate Fourth Edition", "4ED", "ALT", "1995-04-01")

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

	// Finally save all the non-sealed uuids generated
	var allUUIDs []string
	for uuid, card := range uuids {
		if card.Sealed {
			continue
		}
		allUUIDs = append(allUUIDs, uuid)
	}

	// Filter out token names with same names as real cards
	for tokenName := range tokens {
		_, found := cards[Normalize(tokenName)]
		if found {
			delete(tokens, tokenName)
		}
	}

	backend.Hashes = hashes
	backend.AllUUIDs = allUUIDs
	backend.AllNames = names
	backend.AllSealed = sealed
	backend.Sets = ap.Data
	backend.Cards = cards
	backend.Tokens = tokens
	backend.DFCSameNames = dfcSameNames
	backend.UUIDs = uuids
	backend.Scryfall = scryfall
	backend.Tcgplayer = tcgplayer
	backend.AlternateProps = alternates
	backend.AlternateNames = altNames
}

var langs = map[string]string{
	"JPN": "Japanese",
	"ITA": "Italian",
	"ALT": "English",
}

func duplicate(sets map[string]*mtgjson.Set, cards map[string]cardinfo, uuids map[string]CardObject, name, code, tag, date string) {
	// Copy base set information
	dup := mtgjson.Set{}
	dup = *sets[code]

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

		// Remove store references from sets that differ by more than language
		if tag == "ALT" {
			altIdentifiers := map[string]string{}
			for k, v := range dup.Cards[i].Identifiers {
				altIdentifiers[k] = v
			}
			delete(altIdentifiers, "tcgplayerProductId")
			dup.Cards[i].Identifiers = altIdentifiers
		}

		// Add the new uuid to the UUID map
		uuids[dup.Cards[i].UUID] = CardObject{
			Card:    dup.Cards[i],
			Edition: name,
		}
	}

	sets[dup.Code] = &dup
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
