package mtgmatcher

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

type DataStore interface {
	Load() cardBackend
}

type cardinfo struct {
	Name      string
	Printings []string
	Layout    string
}

// CardObject is an extension of Card, containing fields that cannot
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

func okForTokens(set *Set) bool {
	return slices.Contains(setAllowedForTokens, set.Code) ||
		strings.Contains(set.Name, "Duel Deck")
}

func skipSet(set *Set) bool {
	// Skip unsupported sets
	switch set.Code {
	case "PRED", // a single foreign card
		"PSAL", "PS11", "PHUK", // salvat05, salvat11, hachette
		"OAFR", "OCLB", // oversized dungeons
		"UNK", "PUNK", // not on sale anywhere
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

func sortPrintings(sets map[string]*Set, printings []string) {
	sort.Slice(printings, func(i, j int) bool {
		setDateI, errI := time.Parse("2006-01-02", sets[printings[i]].ReleaseDate)
		setDateJ, errJ := time.Parse("2006-01-02", sets[printings[j]].ReleaseDate)
		if errI != nil || errJ != nil {
			return false
		}

		if setDateI.Equal(setDateJ) {
			return sets[printings[i]].Name < sets[printings[j]].Name
		}

		return setDateI.After(setDateJ)
	})
}

func generateImageURL(card Card, version string) string {
	_, found := card.Identifiers["scryfallId"]
	if !found {
		tcgId, found := card.Identifiers["tcgplayerProductId"]
		if !found {
			return ""
		}
		if version == "small" {
			// This size is the default "small" format
			tcgId = "fit-in/146x204/" + tcgId
		}
		return "https://product-images.tcgplayer.com/" + tcgId + ".jpg"
	}

	number := card.Number

	// Retrieve the original number if present
	dupe, found := card.Identifiers["originalScryfallNumber"]
	if found {
		number = dupe
	}

	// Support BAN's custom sets
	code := strings.ToLower(card.SetCode)
	code = strings.TrimSuffix(code, "ita")
	code = strings.TrimSuffix(code, "jpn")
	code = strings.TrimSuffix(code, "alt")

	// Pick the right language for duplicated cards
	if strings.Contains(card.Number, "ita") || strings.Contains(card.Number, "jpn") {
		number += "/" + LanguageTag2LanguageCode[card.Language]
	}

	return fmt.Sprintf("https://api.scryfall.com/cards/%s/%s?format=image&version=%s", code, number, version)
}

// Make sure Printings array is filled, and make token properties uniform
func adjustTokens(sets map[string]*Set) {
	printings := make(map[string][]string)

	// Adjust input data, filtering out unneeded sets, and making sure layout is set
	for code, set := range sets {
		// Remove undesired tokens
		if !okForTokens(set) {
			sets[code].Tokens = nil
			continue
		}

		for _, cardSet := range [][]Card{set.Cards, set.Tokens} {
			for i := range cardSet {
				switch set.Code {
				// Only keep dungeons, and fix their layout to make sure they are tokens
				case "AFR":
					if slices.Contains(cardSet[i].Types, "Dungeon") {
						cardSet[i].Layout = "token"
					}

				// Override all cards to tokens so that duplicates get named differently
				case "TFTH", "TBTH", "TDAG":
					cardSet[i].Layout = "token"
				}

				// Make sure any odd token type stays token
				switch cardSet[i].Layout {
				case "token", "double_faced_token", "emblem":
					cardSet[i].Layout = "token"
					cardSet[i].Rarity = "token"
				}
			}
		}
	}

	// Load up all the printings found among tokens
	for _, set := range sets {
		for _, cardSet := range [][]Card{set.Cards, set.Tokens} {
			for i := range cardSet {
				if cardSet[i].Layout != "token" {
					continue
				}
				if slices.Contains(printings[cardSet[i].Name], set.Code) {
					continue
				}
				printings[cardSet[i].Name] = append(printings[cardSet[i].Name], set.Code)
			}
		}
	}

	// Assign printings to tokens
	// Sorting will happen later
	for _, set := range sets {
		for _, cardSet := range [][]Card{set.Cards, set.Tokens} {
			for i := range cardSet {
				if cardSet[i].Layout != "token" {
					continue
				}
				cardSet[i].Printings = printings[cardSet[i].Name]
			}
		}
	}
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
		// Filter out unneeded data
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

	adjustTokens(ap.Data)

	for code, set := range ap.Data {
		var filteredCards []Card
		var rarities, colors []string

		allSets = append(allSets, code)

		allCards := set.Cards

		// Append tokens to the list of considered cards
		// if they are not named in the same way of a real card
		for _, token := range set.Tokens {
			if !slices.Contains(allCardNames, token.Name) {
				allCards = append(allCards, token)
			}
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
			card.Images["full"] = generateImageURL(card, "normal")
			card.Images["thumbnail"] = generateImageURL(card, "small")
			card.Images["crop"] = generateImageURL(card, "art_crop")

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
				card.PromoTypes = nil
				card.FrameEffects = nil
				card.BorderColor = "black"

			// Promo-only sets
			case "PPC1", "PMIC":
				card.IsPromo = true

			// Missing promo type for this series
			case "DFT":
				num, _ := strconv.Atoi(card.Number)
				if num >= 333 && num <= 346 || num >= 532 && num <= 545 {
					card.PromoTypes = append(card.PromoTypes, "rudedrivers")
				}

			// Upstream cannot properly represent foil cards
			case "SLC":
				if card.SourceProducts == nil {
					card.SourceProducts = map[string][]string{}
				}

				num, _ := strconv.Atoi(card.Number)
				if (num >= 1993 && num <= 2023) || (num >= 1 && num <= 26) {
					card.SourceProducts["foil"] = card.SourceProducts["nonfoil"]
				} else if num == 27 || num >= 28 && num <= 53 {
					card.SourceProducts["foil"] = allCards[0].SourceProducts["nonfoil"]
				}

			case "SLD":
				switch card.Number {
				// One of the tokens is a DFC but burns a card number, skip it
				case "28":
					continue
				// Source is "technically correct" but it gets too messy to track
				case "589":
					card.Finishes = []string{"nonfoil", "etched"}

				// A series of bonus cards that are not tagged as such
				case "59":
					card.IsPromo = true
				case "721":
					card.PromoTypes = append(card.PromoTypes, "convention")
				case "797":
					card.PromoTypes = append(card.PromoTypes, "convention")
				case "8001":
					card.PromoTypes = append(card.PromoTypes, "tourney")
					card.IsPromo = true

				default:
					num, _ := strconv.Atoi(card.Number)
					// Override the frame type for the Braindead drops
					if (num >= 821 && num <= 824) || (num >= 1652 && num <= 1666) {
						card.FrameVersion = "2015"
					}
				}

			case "SLX":
				num, _ := strconv.Atoi(card.Number)
				// These cards have been distributed by stores and not found in products
				if num >= 24 && num <= 30 {
					card.PromoTypes = append(card.PromoTypes, "wizardsplaynetwork")
				}
			}

			// Make sure this property is correctly initialized
			if strings.HasSuffix(card.Number, "p") && !slices.Contains(card.PromoTypes, PromoTypePromoPack) {
				card.PromoTypes = append(card.PromoTypes, PromoTypePromoPack)
			}

			// Rename DFCs into a single name
			// All names need to be redacted
			dfcSameName := IsDFCSameName(card.Name)
			if dfcSameName {
				card.Name = strings.Split(card.Name, " // ")[0]
				card.FlavorName = strings.Split(card.FlavorName, " // ")[0]
				card.FaceName = strings.Split(card.FaceName, " // ")[0]
				card.FaceFlavorName = strings.Split(card.FaceFlavorName, " // ")[0]
				card.PrintedName = strings.Split(card.PrintedName, " // ")[0]
				card.FacePrintedName = strings.Split(card.FacePrintedName, " // ")[0]
				card.Identifiers["isDFCSameName"] = "true"
			}

			for i, name := range []string{
				card.FaceName, card.FlavorName, card.FaceFlavorName, card.PrintedName, card.FacePrintedName,
			} {
				// Skip empty entries
				if name == "" {
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
			sortPrintings(ap.Data, printings)

			card.Printings = printings

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
			if slices.Contains(duplicatedCardNames, name) && strings.Contains(set.Name, "Playtest") {
				name += " Playtest"
			}

			norm := Normalize(name)
			_, found := cardInfo[norm]
			if !found {
				cardInfo[norm] = cardinfo{
					Name:      card.Name,
					Printings: card.Printings,
					Layout:    card.Layout,
				}
			}

			// Custom properties for tokens
			if card.IsOversized {
				card.Rarity = "oversize"
			}

			// Initialize custom lookup tables
			scryfallId, found := card.Identifiers["scryfallId"]
			if found {
				scryfall[scryfallId] = card.UUID
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
			if card.HasFinish(FinishEtched) {
				uuid := card.UUID

				// Etched + Nonfoil [+ Foil]
				if card.HasFinish(FinishNonfoil) {
					// Save the card object
					uuids[uuid] = co
				}

				// Etched + Foil
				if card.HasFinish(FinishFoil) {
					// Set the main property
					co.Foil = true
					// Make sure "_f" is appended if a different version exists
					if card.HasFinish(FinishNonfoil) {
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
				if card.HasFinish(FinishNonfoil) || card.HasFinish(FinishFoil) {
					uuid = card.UUID + suffixEtched
					co.UUID = uuid
				}
				// Save the card object
				uuids[uuid] = co
			} else if card.HasFinish(FinishFoil) {
				uuid := card.UUID

				// Foil [+ Nonfoil]
				if card.HasFinish(FinishNonfoil) {
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

			// Add possible rarities and colors
			if !slices.Contains(rarities, card.Rarity) {
				rarities = append(rarities, card.Rarity)
			}
			for _, color := range card.Colors {
				if !slices.Contains(colors, mtgColorNameMap[color]) {
					colors = append(colors, mtgColorNameMap[color])
				}
			}
			if len(card.Colors) == 0 && !slices.Contains(colors, "colorless") {
				colors = append(colors, "colorless")
			}
			if len(card.Colors) > 1 && !slices.Contains(colors, "multicolor") {
				colors = append(colors, "multicolor")
			}

		}

		// Replace the original array with the filtered one
		set.Cards = filteredCards

		// Assign the rarities and colors present in the set
		sort.Slice(rarities, func(i, j int) bool {
			return mtgRarityMap[rarities[i]] > mtgRarityMap[rarities[j]]
		})
		set.Rarities = rarities
		sort.Slice(colors, func(i, j int) bool {
			return mtgColorMap[colors[i]] > mtgColorMap[colors[j]]
		})
		set.Colors = colors

		// Adjust the setBaseSize to take into account the cards with
		// the same name in the same set (also make sure that it is
		// correctly initialized)
		setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
		if err != nil {
			continue
		}
		if setDate.After(PromosForEverybodyYay) {
			for _, card := range set.Cards {
				if card.HasPromoType(PromoTypeBoosterfun) {
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
			if product.Identifiers == nil {
				product.Identifiers = map[string]string{}
			}
			product.Identifiers["mtgjsonId"] = product.UUID

			card := Card{
				UUID:        product.UUID,
				Name:        product.Name,
				SetCode:     code,
				Identifiers: product.Identifiers,
				Rarity:      "product",
				Layout:      product.Category,
				Side:        product.Subtype,
				// Will be filled later
				SourceProducts: map[string][]string{},
				Images:         map[string]string{},
			}

			// Preserve ReleaseDate information only for SLD, the other sets
			// will derive it from the set date itself
			if code == "SLD" {
				card.OriginalReleaseDate = product.ReleaseDate
			}

			card.Images["full"] = generateImageURL(card, "normal")
			card.Images["thumbnail"] = generateImageURL(card, "small")
			card.Images["crop"] = generateImageURL(card, "normal")

			isEtched := strings.Contains(product.Name, "Etched")
			isFoil := !isEtched
			switch {
			case strings.Contains(product.Name, "Foil") && !strings.Contains(product.Name, "Non"):
			case strings.Contains(product.Name, "Premium"):
			case strings.Contains(product.Name, "VIP Edition"):
			case strings.Contains(product.Name, "Commander Deck") && strings.Contains(product.Name, "Collector Edition"):
			case slices.Contains(productsWithOnlyFoils, product.Name):
			default:
				isFoil = false
			}

			uuids[product.UUID] = CardObject{
				Card:    card,
				Sealed:  true,
				Edition: set.Name,
				Foil:    isFoil,
				Etched:  isEtched,
			}
		}
	}

	duplicate(ap.Data, cardInfo, uuids, "Legends Italian", "LEG", "ITA", "1995-09-01")
	duplicate(ap.Data, cardInfo, uuids, "The Dark Italian", "DRK", "ITA", "1995-08-01")
	duplicate(ap.Data, cardInfo, uuids, "Alternate Fourth Edition", "4ED", "ALT", "1995-04-01")
	allSets = append(allSets, "LEGITA", "DRKITA", "4EDALT")

	sldDupes := duplicateCards(ap.Data, uuids, "SLD", "JPN", sldJPNLangDupes)
	ap.Data["SLD"].Cards = append(ap.Data["SLD"].Cards, sldDupes...)

	purlDupes := duplicateCards(ap.Data, uuids, "PURL", "JPN", []string{"1"})
	ap.Data["PURL"].Cards = append(ap.Data["PURL"].Cards, purlDupes...)

	for setCode := range foilDupes {
		spinoffFoils(ap.Data, uuids, setCode)
	}

	// Add all names and associated uuids to the global names and hashes arrays
	hashes := map[string][]string{}
	var names, fullNames, lowerNames []string
	var sealed, fullSealed, lowerSealed []string
	for uuid, card := range uuids {
		// Load up the ids, both the ones in mtgjson and the ones loaded manually
		for _, tag := range []string{"tcgplayerProductId", "tcgplayerEtchedProductId"} {
			tcgplayerId, found := card.Identifiers[tag]
			if found {
				tcgplayer[tcgplayerId] = card.Identifiers["mtgjsonId"]
			}
		}

		namesToAdd := []string{card.Name}
		if card.Identifiers["isDFCSameName"] == "true" {
			namesToAdd = append(namesToAdd, card.Name+" // "+card.Name)
			if card.FlavorName != "" && !slices.Contains(namesToAdd, card.FlavorName+" // "+card.FlavorName) {
				namesToAdd = append(namesToAdd, card.FlavorName+" // "+card.FlavorName)
			}
			if card.PrintedName != "" && !slices.Contains(namesToAdd, card.PrintedName+" // "+card.PrintedName) {
				namesToAdd = append(namesToAdd, card.PrintedName+" // "+card.PrintedName)
			}
		}

		for _, nameToAdd := range namesToAdd {
			norm := Normalize(nameToAdd)
			_, found := hashes[norm]
			if !found {
				if card.Sealed {
					sealed = append(sealed, norm)
					fullSealed = append(fullSealed, card.Name)
					lowerSealed = append(lowerSealed, strings.ToLower(card.Name))
				} else {
					names = append(names, norm)
					fullNames = append(fullNames, nameToAdd)
					lowerNames = append(lowerNames, strings.ToLower(nameToAdd))
				}
			}
			hashes[norm] = append(hashes[norm], uuid)
		}
	}
	// Add all alternative names too
	var altNames []string
	for altName, altProps := range alternates {
		altNorm := Normalize(altName)
		_, found := hashes[altNorm]
		if !found {
			altNames = append(altNames, altNorm)
			fullNames = append(fullNames, altName)
			lowerNames = append(lowerNames, strings.ToLower(altName))
		}
		if altProps.IsFlavor {
			// Retrieve all the uuids with a FlavorName attached
			allAltUUIDs := hashes[Normalize(altProps.OriginalName)]
			for _, uuid := range allAltUUIDs {
				for _, name := range []string{
					uuids[uuid].FlavorName, uuids[uuid].PrintedName, uuids[uuid].FacePrintedName,
				} {
					if name == "" || slices.Contains(hashes[altNorm], uuid) {
						continue
					}
					if Normalize(name) == altNorm || Normalize(strings.Split(name, " // ")[0]) == altNorm {
						hashes[altNorm] = append(hashes[altNorm], uuid)
					}
				}
			}
		} else {
			// Copy the original uuids, avoiding duplicates for the cards already added
			for _, hash := range hashes[Normalize(altProps.OriginalName)] {
				if slices.Contains(hashes[altNorm], hash) {
					continue
				}
				hashes[altNorm] = append(hashes[altNorm], hash)
			}
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
		if !card.Foil && !card.Etched && !card.Sealed {
			for _, promoType := range []string{
				PromoTypeDoubleExposure,
				PromoTypeGalaxyFoil,
				PromoTypeSilverFoil,
				PromoTypeRainbowFoil,
				PromoTypeRippleFoil,
				PromoTypeSurgeFoil,
			} {
				if card.HasPromoType(promoType) {
					// Filter
					var filtered []string
					for _, pt := range card.PromoTypes {
						if pt != promoType {
							filtered = append(filtered, pt)
						}
					}

					// Update UUID map
					card.PromoTypes = filtered
					uuids[uuid] = card

					// Also update data in the original slice
					for i, c := range ap.Data[card.SetCode].Cards {
						if c.UUID != uuid {
							continue
						}
						ap.Data[card.SetCode].Cards[i].PromoTypes = filtered
					}
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

var mtgRarityMap = map[string]int{
	"token":    1,
	"common":   2,
	"uncommon": 3,
	"rare":     4,
	"mythic":   5,
	"special":  6,
	"oversize": 7,
}

var mtgColorNameMap = map[string]string{
	"W": "white",
	"U": "blue",
	"B": "black",
	"R": "red",
	"G": "green",
}

var mtgColorMap = map[string]int{
	"white":      7,
	"blue":       6,
	"black":      5,
	"red":        4,
	"green":      3,
	"colorless":  2,
	"multicolor": 1,
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

// Duplicate an entire set of cards, using a custom code and a different language
func duplicate(sets map[string]*Set, cardInfo map[string]cardinfo, uuids map[string]CardObject, name, code, tag, date string) {
	// Copy base set information
	dup := *sets[code]

	// Update with new info
	dup.Name = name
	dup.Code = code + tag
	dup.ParentCode = code
	dup.ReleaseDate = date

	// Target slice for later use
	var numbers []string

	// Rework printings information and update cardInfo
	for i := range sets[code].Cards {
		// Skip misprints from main sets
		if strings.HasSuffix(sets[code].Cards[i].Number, SuffixVariant) {
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

		// Update printings for the CardInfo map
		ci := cardInfo[Normalize(sets[code].Cards[i].Name)]
		ci.Printings = printings
		cardInfo[Normalize(sets[code].Cards[i].Name)] = ci

		numbers = append(numbers, sets[code].Cards[i].Number)
	}

	// Add duplicated set (with no cards) to the root
	sets[dup.Code] = &dup

	// Duplicate cards
	dup.Cards = duplicateCards(sets, uuids, code, tag, numbers)

	// Remove store references to avoid duplicates
	for i := range dup.Cards {
		altIdentifiers := map[string]string{}
		for k, v := range dup.Cards[i].Identifiers {
			switch k {
			case "tcgplayerProductId", "tcgplayerEtchedProductId", "mcmId", "mcmEtchedId":
				continue
			}
			altIdentifiers[k] = v
		}
		dup.Cards[i].Identifiers = altIdentifiers
	}
}

// Duplicate certain cards within the same set according to the language tag
func duplicateCards(sets map[string]*Set, uuids map[string]CardObject, code, tag string, numbers []string) []Card {
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
		dupeCard.Identifiers["originalScryfallNumber"] = dupeCard.Number
		dupeCard.Number += strings.ToLower(tag)

		// Set a new code and edition name if we're duplicating a whole set
		_, found := sets[code+tag]
		edition := sets[code].Name
		if found {
			dupeCard.SetCode = code + tag
			edition = sets[dupeCard.SetCode].Name
		}

		// Update images
		dupeCard.Images = map[string]string{}
		dupeCard.Images["full"] = generateImageURL(dupeCard, "normal")
		dupeCard.Images["thumbnail"] = generateImageURL(dupeCard, "small")
		dupeCard.Images["crop"] = generateImageURL(dupeCard, "art_crop")

		// Retrieve Printed data if available
		for _, foreignData := range sets[code].Cards[i].ForeignData {
			if foreignData.Language != dupeCard.Language {
				continue
			}
			dupeCard.PrintedName = foreignData.Name
			dupeCard.PrintedType = foreignData.Type
		}

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
				Edition: edition,
				Etched:  co.Etched,
				Foil:    co.Foil,
			}
		}
	}

	return duplicates
}

// Duplicate certain cards by creating foil versions of them
func spinoffFoils(sets map[string]*Set, uuids map[string]CardObject, code string) {
	var newCardsArray []Card

	var dupes = foilDupes[code]

	for i := range sets[code].Cards {
		dupeCard := sets[code].Cards[i]
		ogUUID := dupeCard.UUID

		// Load the original number if available, or use the main one
		ogNum, found := dupeCard.Identifiers["originalScryfallNumber"]
		if !found {
			ogNum = dupeCard.Number
		}

		// Load the new id
		tcgId, found := dupes[ogNum]

		// Skip unneeded (just preserve the card as-is)
		if !found {
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
		dupeCard.Variations = []string{ogUUID + suffixFoil}

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
		dupeCard.UUID += suffixFoil

		// Clone the map and replace it, overriding the id
		newIdentifiers := map[string]string{}
		for k, v := range dupeCard.Identifiers {
			newIdentifiers[k] = v
		}

		dupeCard.Identifiers = newIdentifiers
		dupeCard.Identifiers["tcgplayerProductId"] = tcgId
		// Signal that the TCG SKUs from MTGJSON are not reliable
		dupeCard.Identifiers["needsNewTCGSKUs"] = "true"

		// In case we are duplicating a card that was *already* duplicated
		_, found = dupeCard.Identifiers["originalScryfallNumber"]
		if !found {
			dupeCard.Identifiers["originalScryfallNumber"] = dupeCard.Number
		}

		dupeCard.Number += SuffixSpecial
		dupeCard.Finishes = []string{"foil"}
		dupeCard.Variations = []string{ogUUID}

		// Update images
		dupeCard.Images = map[string]string{}
		dupeCard.Images["full"] = generateImageURL(dupeCard, "normal")
		dupeCard.Images["thumbnail"] = generateImageURL(dupeCard, "small")
		dupeCard.Images["crop"] = generateImageURL(dupeCard, "art_crop")

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
