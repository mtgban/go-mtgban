package mtgmatcher

import (
	"log"
	"strings"

	"github.com/kodabb/go-mtgmatcher/mtgjson"
)

func Match(inCard *Card) (outCard *Card, err error) {
	if sets == nil {
		return nil, ErrDatastoreEmpty
	}

	// Look up by uuid
	if inCard.Id != "" {
		id := inCard.Id
		if strings.HasSuffix(id, "_f") {
			id = strings.TrimSuffix(id, "_f")
		}
		for _, set := range sets {
			for _, card := range set.Cards {
				if id == card.UUID || id == card.Identifiers["scryfallId"] {
					return inCard.output(card, set), nil
				}
			}
		}
	}

	// Get the card basic info to retrieve the Printings array
	entry, found := cards[Normalize(inCard.Name)]
	if !found {
		// Fixup up the name and try again
		adjustName(inCard)

		entry, found = cards[Normalize(inCard.Name)]
		if !found {
			return nil, ErrCardDoesNotExist
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
			return nil, ErrCardNotInEdition
		}
	}

	// This map will contain the setCode and an array of possible matches for
	// each edition.
	cardSet := map[string][]mtgjson.Card{}

	// Only one printing, it *has* to be it
	if len(printings) == 1 {
		cardSet[printings[0]] = matchInSet(inCard, sets[printings[0]])
	} else {
		// If multiple printing, try filtering to the closest name
		// described by the inCard.Edition.
		logger.Println("Several printings found, iterating over edition name")

		// First loop, search for a perfect match
		for _, setCode := range printings {
			// Perfect match, the card *has* to be present in the set
			if Equals(sets[setCode].Name, inCard.Edition) {
				logger.Println("Found a perfect match with", inCard.Edition, setCode)
				cardSet[setCode] = matchInSet(inCard, sets[setCode])
			}
		}

		// Second loop, hope that a portion of the edition is in the set Name
		// This may result in false positives under certain circumstances.
		if len(cardSet) == 0 {
			logger.Println("No perfect match found, trying with heuristics")
			for _, setCode := range printings {
				set := sets[setCode]
				if Contains(set.Name, inCard.Edition) ||
					(inCard.isGenericPromo() && strings.HasSuffix(set.Name, "Promos")) {
					logger.Println("Found a possible match with", inCard.Edition, setCode)
					cardSet[setCode] = matchInSet(inCard, set)
				}
			}
		}

		// Third loop, YOLO
		// Let's consider every edition and hope the second pass will filter
		// duplicates out. This may result in false positives of course.
		if len(cardSet) == 0 {
			logger.Println("No loose match found, trying all")
			for _, setCode := range printings {
				cardSet[setCode] = matchInSet(inCard, sets[setCode])
			}
		}
	}

	// Determine if any deduplication needs to be performed
	logger.Println("Found these possible matches")
	var foundCode []string
	single := len(cardSet) == 1
	for setCode, dupCards := range cardSet {
		foundCode = []string{setCode}
		single = single && len(dupCards) == 1
		for _, card := range dupCards {
			logger.Println(setCode, card.Name, card.Number)
		}
	}

	// Use the result as-is if it comes from a single card in a single set
	var outCards []mtgjson.Card
	if single {
		logger.Println("Single printing, using it right away")
		outCards = []mtgjson.Card{cardSet[foundCode[0]][0]}
	} else {
		// Otherwise do a second pass filter, using all inCard details
		logger.Println("Now filtering...")
		outCards, foundCode = filterCards(inCard, cardSet)

		for i, card := range outCards {
			logger.Println(foundCode[i], card.Name, card.Number)
		}
	}

	// Just keep the first card found for gold-bordered sets
	if inCard.isWorldChamp() && len(outCards) > 1 {
		logger.Println("Dropping a few WCD entries...")
		logger.Println(outCards[1:])
		outCards = []mtgjson.Card{outCards[0]}
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
		outCard = inCard.output(outCards[0], sets[foundCode[0]])
	// FOR SHAME
	default:
		logger.Println("Aliasing...")
		alias := newAliasingError()
		for i := range outCards {
			alias.dupes = append(alias.dupes, *inCard.output(outCards[i], sets[foundCode[i]]))
		}
		err = alias
	}

	return
}

// Return an array of mtgjson.Card containing all the cards with the exact
// same name as the input inCard in the given mtgjson.Set.
func matchInSet(inCard *Card, set mtgjson.Set) (outCards []mtgjson.Card) {
	for _, card := range set.Cards {
		if inCard.Name == card.Name {
			// MTGJSON v5 contains duplicated card info for each face, and we do
			// not need that level of detail, so just skip any extra side.
			if card.Side != "" && card.Side != "a" {
				continue
			}
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
		log.Println(inCard)
		inCard.Name = strings.Join(fields, " ")
		inCard.addToVariant(num)
		return
	}

	// Move any single letter variation from name to beginning variation
	if inCard.IsBasicLand() {
		fields := strings.Fields(inCard.Name)
		if len(fields) > 1 && len(fields[1]) == 1 {
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

	// Check if the input name is the reskinned one
	if strings.Contains(inCard.Edition, "Ikoria") {
		for _, card := range sets["IKO"].Cards {
			if Equals(inCard.Name, card.FlavorName) {
				inCard.Name = card.Name
				inCard.addToVariant("Godzilla")
				return
			}
		}
	}

	// Many provide the full name with a variety of characters
	for _, sep := range []string{" | ", " || ", " / ", " - ", " and ", " to "} {
		if strings.Contains(inCard.Name, sep) {
			name := strings.Replace(inCard.Name, sep, " // ", 1)

			// Check if the fixed name exists
			_, found := cards[Normalize(name)]
			if found {
				inCard.Name = name
				return
			}

			// If not, try and keep one side only
			name = strings.Split(inCard.Name, " // ")[0]
			_, found = cards[Normalize(name)]
			if found {
				inCard.Name = name
				return
			}
			// Else keep going, maybe it will be found later
		}
	}

	// Special case for Un-sets that sometimes drop the parenthesis
	if strings.Contains(inCard.Edition, "Unglued") ||
		strings.Contains(inCard.Edition, "Unhinged") ||
		strings.Contains(inCard.Edition, "Unstable") ||
		strings.Contains(inCard.Edition, "Unsanctioned") {
		for cardName, props := range cards {
			if HasPrefix(cardName, inCard.Name) {
				inCard.Name = props.Name
				return
			}
		}
	}

	// Altenatively try checking across any prefix, as long as it's a double
	// sided card, for some particular cases, like meld cards, or Treasure Chest
	for cardName, props := range cards {
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
	set, found := sets[edition]
	if found {
		edition = set.Name
	}
	ed, found := EditionTable[edition]
	if found {
		edition = ed
	}
	ed, found = EditionTable[inCard.Variation]
	if found {
		edition = ed
	}

	// Adjust box set
	switch {
	case strings.HasSuffix(edition, "(Collector Edition)"):
		edition = strings.Replace(edition, " (Collector Edition)", "", 1)
	case strings.HasSuffix(edition, "Collectors"):
		edition = strings.TrimSuffix(edition, " Collectors")
	case strings.HasSuffix(edition, "Extras"):
		edition = strings.Replace(edition, " Extras", "", 1)
		edition = strings.Replace(edition, ":", "", 1)
	case strings.HasSuffix(edition, "Variants"):
		edition = strings.Replace(edition, " Variants", "", 1)
		edition = strings.Replace(edition, ":", "", 1)
	case strings.Contains(edition, "Mythic Edition"),
		strings.Contains(inCard.Variation, "Mythic Edition"):
		edition = "Mythic Edition"
	case strings.Contains(edition, "Invocations") ||
		((edition == "Hour of Devastation" || edition == "Amonkhet") &&
			strings.Contains(inCard.Variation, "Invocation")):
		edition = "Amonkhet Invocations"
	case strings.Contains(edition, "Inventions"):
		edition = "Kaladesh Inventions"
	case strings.Contains(edition, "Expeditions"):
		edition = "Zendikar Expeditions"
	}

	variation := inCard.Variation
	switch {
	case strings.Contains(variation, "Ravnica Weekend"):
		num := ExtractNumber(variation)
		if strings.HasPrefix(num, "A") {
			edition = "GRN Ravnica Weekend"
		} else if strings.HasPrefix(num, "B") {
			edition = "RNA Ravnica Weekend"
		}
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
	}
	inCard.Edition = edition
	inCard.Variation = variation

	// Special handling since so many providers get this wrong
	switch {
	// XLN Treasure Chest
	case inCard.isBaB() && len(matchInSet(inCard, sets["PXTC"])) != 0:
		inCard.Edition = sets["PXTC"].Name
	// BFZ Standard Series
	case inCard.isGenericAltArt() && len(matchInSet(inCard, sets["PSS1"])) != 0:
		inCard.Edition = sets["PSS1"].Name
	// Champs and States
	case inCard.isGenericExtendedArt() && len(matchInSet(inCard, sets["PCMP"])) != 0:
		inCard.Edition = sets["PCMP"].Name
	// Portal Demo Game
	case ((Contains(inCard.Variation, "Reminder Text") &&
		!strings.Contains(inCard.Variation, "No")) ||
		Contains(inCard.Variation, "No Flavor Text")) &&
		len(matchInSet(inCard, sets["PPOD"])) != 0:
		inCard.Edition = sets["PPOD"].Name
	// Secret Lair Ultimate
	case strings.Contains(inCard.Edition, "Secret Lair") &&
		len(matchInSet(inCard, sets["SLU"])) != 0:
		inCard.Edition = sets["SLU"].Name
	// Summer of Magic
	case (inCard.isWPNGateway() || strings.Contains(inCard.Variation, "Summer")) &&
		len(matchInSet(inCard, sets["PSUM"])) != 0:
		inCard.Edition = sets["PSUM"].Name

	// Single card mismatches
	case Equals(inCard.Name, "Rhox") && inCard.isGenericAltArt():
		inCard.Edition = "Starter 2000"
	case Equals(inCard.Name, "Balduvian Horde") && (strings.Contains(inCard.Variation, "Judge") || strings.Contains(inCard.Edition, "Promo")):
		inCard.Edition = "World Championship Promos"
	case Equals(inCard.Name, "Nalathni Dragon") && inCard.isIDWMagazineBook():
		inCard.Edition = "Dragon Con"
	case Equals(inCard.Name, "Ass Whuppin'") && inCard.isPrerelease():
		inCard.Variation = "Release Events"
	case Equals(inCard.Name, "Ajani Vengeant") && inCard.isRelease():
		inCard.Variation = "Prerelease"
	case Equals(inCard.Name, "Tamiyo's Journal") && inCard.Variation == "" && inCard.Foil:
		inCard.Variation = "Foil"
	}
}
