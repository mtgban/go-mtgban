package mtgmatcher

import (
	"strconv"
	"strings"

	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"
)

func Match(inCard *Card) (cardId string, err error) {
	if backend.Sets == nil {
		return "", ErrDatastoreEmpty
	}

	// Look up by uuid
	if inCard.Id != "" {
		co, found := backend.UUIDs[inCard.Id]
		if found {
			outId := output(co.Card, inCard.Foil)

			// Validate that what we found is correct
			co = backend.UUIDs[outId]
			// If the input card was requested as foil, we should double check
			// if the original card has a foil under a separate id
			if co.Foil != inCard.Foil {
				// So we iterate over the Variations array and try outputing ids
				// until we find a perfect match in foiling status
				for _, variation := range co.Variations {
					altCo := backend.UUIDs[variation]
					// We assume that the collector number between the two version
					// stays the same, with a different suffix
					if strings.HasPrefix(co.Number, altCo.Number) ||
						strings.HasPrefix(altCo.Number, co.Number) {
						maybeId := output(altCo.Card, inCard.Foil)
						altCo = backend.UUIDs[maybeId]
						if altCo.Foil == inCard.Foil {
							outId = maybeId
							break
						}
					}
				}
			}
			return outId, nil
		}
	}

	// Get the card basic info to retrieve the Printings array
	entry, found := backend.Cards[Normalize(inCard.Name)]
	if !found {
		// Fixup up the name and try again
		adjustName(inCard)

		entry, found = backend.Cards[Normalize(inCard.Name)]
		if !found {
			return "", ErrCardDoesNotExist
		}
	}

	// Restore the card to the canonical MTGJSON name
	inCard.Name = entry.Name

	// Fix up edition
	adjustEdition(inCard)

	logger.Println("Processing", inCard, entry.Printings)

	// If there are multiple printings of the card, filter out to the
	// minimum common elements, using the rules defined.
	printings := entry.Printings
	if len(printings) > 1 {
		printings = filterPrintings(inCard, printings)
		logger.Println("Filtered printings:", printings)

		// Filtering was too aggressive or wrong data fed,
		// in either case, nothing else to be done here.
		if len(printings) == 0 {
			return "", ErrCardNotInEdition
		}
	}

	// This map will contain the setCode and an array of possible matches for
	// each edition.
	cardSet := map[string][]mtgjson.Card{}

	// Only one printing, it *has* to be it
	if len(printings) == 1 {
		cardSet[printings[0]] = MatchInSet(inCard.Name, printings[0])
	} else {
		// If multiple printing, try filtering to the closest name
		// described by the inCard.Edition.
		logger.Println("Several printings found, iterating over edition name")

		// First loop, search for a perfect match
		for _, setCode := range printings {
			// Perfect match, the card *has* to be present in the set
			if Equals(backend.Sets[setCode].Name, inCard.Edition) {
				logger.Println("Found a perfect match with", inCard.Edition, setCode)
				cardSet[setCode] = MatchInSet(inCard.Name, setCode)
			}
		}

		// Second loop, hope that a portion of the edition is in the set Name
		// This may result in false positives under certain circumstances.
		if len(cardSet) == 0 {
			logger.Println("No perfect match found, trying with heuristics")
			for _, setCode := range printings {
				set := backend.Sets[setCode]
				if Contains(set.Name, inCard.Edition) ||
					(inCard.isGenericPromo() && strings.HasSuffix(set.Name, "Promos")) {
					logger.Println("Found a possible match with", inCard.Edition, setCode)
					cardSet[setCode] = MatchInSet(inCard.Name, setCode)
				}
			}
		}

		// Third loop, YOLO
		// Let's consider every edition and hope the second pass will filter
		// duplicates out. This may result in false positives of course.
		if len(cardSet) == 0 {
			logger.Println("No loose match found, trying all")
			for _, setCode := range printings {
				cardSet[setCode] = MatchInSet(inCard.Name, setCode)
			}
		}
	}

	// Determine if any deduplication needs to be performed
	logger.Println("Found these possible matches")
	single := len(cardSet) == 1
	for _, dupCards := range cardSet {
		single = single && len(dupCards) == 1
		for _, card := range dupCards {
			logger.Println(card.SetCode, card.Name, card.Number)
		}
	}

	// Use the result as-is if it comes from a single card in a single set
	var outCards []mtgjson.Card
	if single {
		logger.Println("Single printing, using it right away")
		for _, outCards = range cardSet {
		}
	} else {
		// Otherwise do a second pass filter, using all inCard details
		logger.Println("Now filtering...")
		outCards = filterCards(inCard, cardSet)

		for _, card := range outCards {
			logger.Println(card.SetCode, card.Name, card.Number)
		}
	}

	// Just keep the first card found for gold-bordered sets
	if len(outCards) > 1 {
		if inCard.isWorldChamp() {
			logger.Println("Dropping a few extra entries...")
			logger.Println(outCards[1:])
			outCards = []mtgjson.Card{outCards[0]}
		}
	}

	// Finish line
	switch len(outCards) {
	// Not found, rip
	case 0:
		logger.Println("No matches...")
		err = ErrCardWrongVariant
	// Victory
	case 1:
		logger.Println("Found it!")
		cardId = output(outCards[0], inCard.Foil)
	// FOR SHAME
	default:
		logger.Println("Aliasing...")
		alias := newAliasingError()
		for i := range outCards {
			alias.dupes = append(alias.dupes, output(outCards[i], inCard.Foil))
		}
		err = alias
	}

	return
}

// Return an array of mtgjson.Card containing all the cards with the exact
// same name as the input name in the Set identified by setCode.
func MatchInSet(cardName string, setCode string) (outCards []mtgjson.Card) {
	set := backend.Sets[setCode]
	for _, card := range set.Cards {
		if cardName == card.Name {
			outCards = append(outCards, card)
		}
	}
	return
}

// Try to fixup the name of the card or move extra varitions to the
// variant attribute. This should only be used in case the card name
// was not found.
func adjustName(inCard *Card) {
	// Move the card number from name to variation
	num := ExtractNumber(inCard.Name)
	if num != "" {
		fields := strings.Fields(inCard.Name)
		for i, field := range fields {
			if strings.Contains(field, num) {
				fields = append(fields[:i], fields[i+1:]...)
				break
			}
		}
		inCard.Name = strings.Join(fields, " ")
		inCard.addToVariant(num)
		return
	}

	// Move any single letter variation from name to beginning variation
	if inCard.IsBasicLand() {
		fields := strings.Fields(inCard.Name)
		if len(fields) > 1 {
			_, err := strconv.Atoi(strings.TrimPrefix(fields[1], "0"))
			isNum := err == nil
			isLetter := len(fields[1]) == 1

			if isNum || isLetter {
				oldVariation := inCard.Variation
				cuts := Cut(inCard.Name, " "+fields[1])

				inCard.Name = cuts[0]
				inCard.Variation = cuts[1]
				if oldVariation != "" {
					inCard.Variation += " " + oldVariation
				}
				return
			}
		}
	}

	// Check if the input name is the reskinned one
	// Currently appearing in IKO and some promo sets (PLGS and IKO BaB)
	if strings.Contains(inCard.Edition, "Ikoria") ||
		Contains(inCard.Edition, "Promos") {
		for _, code := range []string{"IKO", "PLGS"} {
			for _, card := range backend.Sets[code].Cards {
				if Equals(inCard.Name, card.FlavorName) {
					inCard.Name = card.Name
					inCard.Edition = code
					inCard.addToVariant("Godzilla")
					return
				}
			}
		}
		// In case both names are used for the promo
		switch {
		case Contains(inCard.Name, "Mechagodzilla, Battle Fortress"):
			inCard.Name = "Hangarback Walker"
			inCard.Edition = "PLGS"
			inCard.addToVariant("Godzilla")
		case Contains(inCard.Name, "Mothra's Giant Cocoon"):
			inCard.Name = "Mysterious Egg"
			inCard.Edition = "IKO"
			inCard.addToVariant("Godzilla")
		case Contains(inCard.Name, "Terror of the City"):
			inCard.Name = "Dirge Bat"
			inCard.Edition = "IKO"
			inCard.addToVariant("Godzilla")
		case Contains(inCard.Name, "Mechagodzilla"):
			inCard.Name = "Crystalline Giant"
			inCard.Edition = "IKO"
			inCard.addToVariant("Godzilla")
		}
		// Found!
		if inCard.isReskin() {
			return
		}
	}

	// Special case for Un-sets that sometimes drop the parenthesis
	if strings.Contains(inCard.Edition, "Unglued") ||
		strings.Contains(inCard.Edition, "Unhinged") ||
		strings.Contains(inCard.Edition, "Unstable") ||
		strings.Contains(inCard.Edition, "Unsanctioned") {
		if HasPrefix(inCard.Name, "B.F.M.") {
			cardName := inCard.Name
			inCard.Name = "B.F.M. (Big Furry Monster)"
			if Contains(cardName, "Left") || Contains(inCard.Variation, "Left") {
				inCard.Variation = "28"
			} else if Contains(cardName, "Right") || Contains(inCard.Variation, "Right") {
				inCard.Variation = "29"
			}
			return
		}
		if HasPrefix(inCard.Name, "Our Market Research") {
			inCard.Name = LongestCardEver
			return
		}
		if HasPrefix(inCard.Name, "The Ultimate Nightmare") {
			inCard.Name = NightmareCard
			return
		}
		if Contains(inCard.Name, "Surgeon") && Contains(inCard.Name, "Commander") {
			inCard.Name = "Surgeon ~General~ Commander"
			return
		}

		for cardName, props := range backend.Cards {
			if HasPrefix(cardName, inCard.Name) {
				inCard.Name = props.Name
				return
			}
		}
	}

	// Altenatively try checking across any prefix, as long as it's a double
	// sided card, for some particular cases, like meld cards, or Treasure Chest
	// Attempt first to check cards in the same edition if possible
	for _, set := range backend.Sets {
		if Equals(set.Name, inCard.Edition) {
			for _, card := range set.Cards {
				if card.Layout != "normal" && HasPrefix(card.Name, inCard.Name) {
					inCard.Name = card.Name
					return
				}
			}
		}
	}
	for cardName, props := range backend.Cards {
		if props.Layout != "normal" && HasPrefix(cardName, inCard.Name) {
			inCard.Name = props.Name
			return
		}
	}
}

// Try to fixup the edition and variant of the card, using well-known variantions,
// or use edition/variant attributes to determine the correct edition/variant combo,
// or look up known cards in small sets.
func adjustEdition(inCard *Card) {
	edition := inCard.Edition
	variation := inCard.Variation

	// Need to decouple The List and Mystery booster first or it will confuse
	// later matching. For an uptodate list of aliased cards visit this link:
	// https://scryfall.com/search?q=in%3Aplist+%28in%3Amb1+or+in%3Afmb1%29+%28e%3Amb1+or+e%3Aplist+or+e%3Afmb1%29&unique=prints&as=grid&order=name
	// Skip only if the edition or variation are explictly set as The List
	if edition != "The List" && variation != "The List" &&
		(inCard.Contains("Mystery Booster") || inCard.Contains("The List")) {
		if (inCard.Foil || inCard.Contains("Foil") && !inCard.Contains("Non")) && len(MatchInSet(inCard.Name, "FMB1")) != 0 {
			edition = "FMB1"
		} else if inCard.Contains("Test") && len(MatchInSet(inCard.Name, "CMB1")) != 0 {
			edition = "CMB1"
		} else {
			// Adjust property, can only be non-foil from here
			inCard.Foil = false

			// Check if card is is only one of these two sets
			mb1s := MatchInSet(inCard.Name, "MB1")
			plists := MatchInSet(inCard.Name, "PLIST")
			if len(mb1s) == 1 && len(plists) == 0 {
				edition = "MB1"
			} else if len(mb1s) == 0 && len(plists) == 1 {
				edition = "PLIST"
			} else if len(mb1s) == 1 && len(plists) == 1 {
				switch variation {
				// If it has one of these special treatments it's PLIST definitely
				case "Player Rewards",
					"MagicFest",
					"Commander",
					"Extended Art",
					"Signature Spellbook: Jace",
					"The List Textless",
					"Player Rewards Promo",
					"Player Rewards Textless",
					"RNA MagicFest Promo",
					"Commander: 2011 Edition",
					"Commander: 2015 Edition",
					"Champs Full Art",
					"State Champs Promo":
					edition = "PLIST"
				default:
					// Otherwise it's probably MB1, including the indistinguishable
					// ones, unless variation has additional information
					edition = "MB1"

					// Adjust variation to get a correct edition name
					ed, found := EditionTable[variation]
					if found {
						variation = ed
					}

					// Check if the card name has the appropriate variation that
					// lets us determine it's from PLIST
					if AliasedPLISTTable[inCard.Name][variation] {
						edition = "PLIST"
					}
				}
			}
		}
		// Ignore this, we have all we need now
		variation = ""
	}

	set, found := backend.Sets[strings.ToUpper(edition)]
	if found {
		edition = set.Name
	}
	ed, found := EditionTable[edition]
	if found {
		edition = ed
	}
	ed, found = EditionTable[variation]
	// This set has one land with a variant named as an expansion,
	// so what is found should not overwrite the edition in this case
	if found && edition != "Anthologies" {
		edition = ed
	}
	inCard.Edition = edition
	inCard.Variation = variation

	// Adjust box set
	switch {
	case Equals(edition, "Double Masters Box Toppers"),
		Equals(edition, "Double Masters: Extras"),
		Equals(edition, "Double Masters: Variants"):
		edition = "Double Masters"
		if !inCard.isBasicLand() {
			variation = "Borderless"
		}
	case strings.Contains(edition, "Mythic Edition"),
		strings.Contains(inCard.Variation, "Mythic Edition"):
		edition = "Mythic Edition"
	case strings.Contains(edition, "Invocations") ||
		((edition == "Hour of Devastation" || edition == "Amonkhet") &&
			strings.Contains(inCard.Variation, "Invocation")):
		edition = "Amonkhet Invocations"
	case strings.Contains(edition, "Inventions"):
		edition = "Kaladesh Inventions"
	case strings.Contains(edition, "Expeditions") && !strings.Contains(edition, "Rising"):
		edition = "Zendikar Expeditions"
	case strings.Contains(edition, "Expeditions") && strings.Contains(edition, "Rising"):
		edition = "Zendikar Rising Expeditions"
	default:
		for _, tag := range []string{
			"(Collector Edition)", "Collectors", "Extras", "Variants",
		} {
			// Strip away any extra tags
			if strings.HasSuffix(edition, tag) {
				edition = strings.TrimSuffix(edition, tag)
				edition = strings.TrimSpace(edition)
				edition = strings.TrimSuffix(edition, ":")
				// If no other variation, set this flag to do a best effort search
				if variation == "" {
					inCard.beyondBaseSet = true
				}
				break
			}
		}
	}

	switch {
	case strings.Contains(edition, "Commander"):
		edition = ParseCommanderEdition(edition)
	case strings.Contains(variation, "Ravnica Weekend") ||
		(strings.Contains(edition, "Weekend") && !Contains(edition, "Planeswalker")):
		edition, variation = inCard.ravnicaWeekend()
	case inCard.Contains("Guild Kit"):
		edition = inCard.ravnicaGuidKit()
	case strings.Contains(variation, "APAC Set") || strings.Contains(variation, "Euro Set"):
		num := ExtractNumber(variation)
		if num != "" {
			variation = strings.Replace(variation, num+" ", "", 1)
		}
	case strings.HasPrefix(variation, "Junior") && strings.Contains(variation, "APAC"),
		strings.HasPrefix(variation, "Junior APAC Series U"):
		edition = "Junior APAC Series"
	case strings.HasPrefix(variation, "Junior Super Series"),
		strings.HasPrefix(variation, "MSS Foil"),
		strings.HasPrefix(variation, "MSS #J"),
		strings.HasPrefix(variation, "MSS Promo J"),
		strings.HasPrefix(variation, "JSS #J"),
		strings.Contains(variation, "JSS Foil") && !Contains(variation, "euro"):
		edition = "Junior Super Series"
	case strings.HasPrefix(variation, "Junior Series Europe"),
		strings.HasPrefix(variation, "Junior Series E"),
		strings.HasPrefix(variation, "Junior Series #E"),
		strings.HasPrefix(variation, "Junior Series Promo E"),
		strings.HasPrefix(variation, "Junior Series Promo Foil E"),
		strings.HasPrefix(variation, "ESS Foil E"),
		strings.HasPrefix(variation, "European JrS E"),
		strings.HasPrefix(variation, "European JSS Foil E"):
		edition = "Junior Series Europe"
	case Equals(inCard.Name, "Teferi, Master of Time"):
		num := ExtractNumber(variation)
		_, err := strconv.Atoi(num)
		if err == nil {
			if inCard.isPrerelease() {
				variation = num + "s"
			} else if inCard.isPromoPack() {
				variation = num + "p"
			}
		}
		if num == "" {
			if inCard.isPrerelease() {
				variation = "75s"
			} else if inCard.isPromoPack() {
				variation = "75p"
			} else if inCard.isBorderless() {
				variation = "281"
			} else if inCard.isShowcase() {
				variation = "290"
			} else {
				variation = "75"
			}
		}
	}
	inCard.Edition = edition
	inCard.Variation = variation

	// Special handling since so many providers get this wrong
	switch {
	// XLN Treasure Chest
	case inCard.isBaB() && len(MatchInSet(inCard.Name, "PXTC")) != 0:
		inCard.Edition = backend.Sets["PXTC"].Name
	// BFZ Standard Series
	case inCard.isGenericAltArt() && len(MatchInSet(inCard.Name, "PSS1")) != 0:
		inCard.Edition = backend.Sets["PSS1"].Name
	// Champs and States
	case inCard.isGenericExtendedArt() && len(MatchInSet(inCard.Name, "PCMP")) != 0:
		inCard.Edition = backend.Sets["PCMP"].Name
	// Portal Demo Game
	case inCard.isPortalAlt() && len(MatchInSet(inCard.Name, "PPOD")) != 0:
		inCard.Edition = backend.Sets["PPOD"].Name
	// Secret Lair {Ultimate,Drop}
	case inCard.Contains("Secret") || Contains(inCard.Variation, "Lair"):
		if len(MatchInSet(inCard.Name, "SLU")) != 0 {
			inCard.Edition = backend.Sets["SLU"].Name
		} else if len(MatchInSet(inCard.Name, "SLD")) != 0 {
			inCard.Edition = backend.Sets["SLD"].Name
		}
	// Summer of Magic
	case (inCard.isWPNGateway() || strings.Contains(inCard.Variation, "Summer")) &&
		len(MatchInSet(inCard.Name, "PSUM")) != 0:
		inCard.Edition = backend.Sets["PSUM"].Name

	// Single card mismatches
	case Equals(inCard.Name, "Rhox") && inCard.isGenericAltArt():
		inCard.Edition = "Starter 2000"
	case Equals(inCard.Name, "Balduvian Horde") && (strings.Contains(inCard.Variation, "Judge") || strings.Contains(inCard.Edition, "Promo") || inCard.isDCIPromo()):
		inCard.Edition = "World Championship Promos"
	case Equals(inCard.Name, "Nalathni Dragon"):
		inCard.Variation = ""
		inCard.Edition = "Dragon Con"
	case Equals(inCard.Name, "Ass Whuppin'") && inCard.isPrerelease():
		inCard.Edition = "Release Events"
	case Equals(inCard.Name, "Rukh Egg") && inCard.isPrerelease():
		inCard.Edition = "Release Events"
	case Equals(inCard.Name, "Ajani Vengeant") && inCard.isRelease():
		inCard.Variation = "Prerelease"
	case Equals(inCard.Name, "Tamiyo's Journal") && inCard.Variation == "" && inCard.Foil:
		inCard.Variation = "Foil"
	case Equals(inCard.Name, "Underworld Dreams") && (inCard.isDCIPromo() || inCard.isArena()):
		inCard.Edition = "Two-Headed Giant Tournament"
	case Equals(inCard.Name, "Jace Beleren") && inCard.isDCIPromo():
		inCard.Edition = "Miscellaneous Book Promos"
	case Equals(inCard.Name, "Serra Angel") && (inCard.isDCIPromo() || inCard.isBaB()):
		inCard.Edition = "Wizards of the Coast Online Store"

	case Equals(inCard.Name, "Incinerate") && inCard.isDCIPromo():
		inCard.Edition = "DCI Legend Membership"
	case Equals(inCard.Name, "Counterspell") && inCard.isDCIPromo():
		inCard.Edition = "DCI Legend Membership"

	case Equals(inCard.Name, "Kamahl, Pit Fighter") && inCard.isDCIPromo():
		inCard.Edition = "15th Anniversary Cards"
	case Equals(inCard.Name, "Char") && inCard.isDCIPromo():
		inCard.Edition = "15th Anniversary Cards"

	case (Equals(inCard.Name, "Fling") || Equals(inCard.Name, "Sylvan Ranger")) && inCard.isDCIPromo() && ExtractNumber(inCard.Variation) == "":
		inCard.Edition = "Wizards Play Network 2010"
	case (Equals(inCard.Name, "Fling") || Equals(inCard.Name, "Sylvan Ranger")) && inCard.isWPNGateway() && ExtractNumber(inCard.Variation) == "":
		inCard.Edition = "Wizards Play Network 2011"

	case inCard.Edition == "Commander Legends" && inCard.isShowcase():
		inCard.Variation = "Foil Etched"

	// Planechase deduplication
	case inCard.Equals("Planechase") && len(MatchInSet(inCard.Name, "OHOP")) != 0:
		inCard.Edition = backend.Sets["OHOP"].Name
	case inCard.Equals("Planechase Anthology") && len(MatchInSet(inCard.Name, "OPCA")) != 0:
		inCard.Edition = backend.Sets["OPCA"].Name

	// Missing the proper FrameEffect property
	case Equals(inCard.Name, "Vorinclex, Monstrous Raider") && (inCard.isShowcase() || Contains(inCard.Variation, "Phyrexian")):
		num := ExtractNumber(inCard.Variation)
		if num == "" {
			if Contains(inCard.Variation, "Phyrexian") {
				inCard.Variation = "333"
			} else if inCard.isShowcase() {
				inCard.Variation = "320"
			}
		}
	}
}
