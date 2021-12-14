package mtgmatcher

import (
	"strconv"
	"strings"
	"time"

	"github.com/kodabb/go-mtgban/mtgmatcher/mtgjson"

	"github.com/google/uuid"
)

func Match(inCard *Card) (cardId string, err error) {
	if backend.Sets == nil {
		return "", ErrDatastoreEmpty
	}

	// Look up by uuid (validate it first, remove any extras after the underscore)
	if inCard.Id != "" {
		id := strings.Split(inCard.Id, "_")[0]
		_, err := uuid.Parse(id)
		if err != nil {
			// Another try to see if it's a plain number for tcg
			_, err := strconv.Atoi(id)
			if err != nil {
				inCard.Id = ""
			}
		}
	}
	if inCard.Id != "" {
		outId := ""
		co, found := backend.UUIDs[inCard.Id]
		if !found {
			// Second chance, lookup by scryfall id
			co, found = backend.UUIDs[backend.Scryfall[inCard.Id]]
			if !found {
				// Last chance, lookup by tcg id
				co, found = backend.UUIDs[backend.Tcgplayer[inCard.Id]]
			}
		}

		if found {
			outId = output(co.Card, inCard.Foil, inCard.isEtched())

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
						maybeId := output(altCo.Card, inCard.Foil, inCard.isEtched())
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

	// In case id lookup failed, an no more data is present
	if inCard.Name == "" {
		return "", ErrCardDoesNotExist
	}

	// Binderpos weird syntax, with the edition embedded in the name
	if strings.Contains(inCard.Name, "[") {
		vars := strings.Split(inCard.Name, "[")
		if len(vars) > 1 {
			maybeEdition := strings.TrimSuffix(strings.TrimSpace(vars[1]), "]")
			set, err := GetSetByName(maybeEdition)
			if err == nil {
				inCard.Name = vars[0]
				inCard.Edition = set.Name
			}
		}
	}
	// TCG collection weird syntax, with foil in the name itself
	if strings.HasSuffix(inCard.Name, "[Foil]") {
		inCard.Foil = true
		inCard.Name = strings.TrimSuffix(inCard.Name, " [Foil]")
		inCard.Name = strings.TrimSuffix(inCard.Name, "-")
		inCard.Name = strings.TrimSuffix(inCard.Name, " ")
	}

	// Simple case in which there is a variant embedded in the name
	if strings.Contains(inCard.Name, "(") {
		vars := SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.addToVariant(strings.Join(vars[1:], " "))
			if Contains(inCard.Variation, "foil") {
				inCard.Foil = true
			}
		}
	}

	// Skip unsupported sets
	if inCard.isUnsupported() {
		return "", ErrUnsupported
	}

	// Get the card basic info to retrieve the Printings array
	entry, found := backend.Cards[Normalize(inCard.Name)]
	if !found {
		ogName := inCard.Name
		// Fixup up the name and try again
		adjustName(inCard)
		if ogName != inCard.Name {
			logger.Printf("Adjusted name from '%s' to '%s'", ogName, inCard.Name)
		}

		entry, found = backend.Cards[Normalize(inCard.Name)]
		if !found {
			// Return a safe error if it's a token
			if IsToken(ogName) || Contains(inCard.Variation, "Oversize") {
				return "", ErrUnsupported
			}
			return "", ErrCardDoesNotExist
		}
	}

	// Restore the card to the canonical MTGJSON name
	ogName := inCard.Name
	inCard.Name = entry.Name
	if entry.Flavor != "" {
		inCard.addToVariant("Reskin")
	}

	// Fix up edition
	ogEdition := inCard.Edition
	adjustEdition(inCard)
	if ogName != inCard.Name {
		logger.Printf("Adjusted name from '%s' to '%s'", ogName, inCard.Name)
	}
	if ogEdition != inCard.Edition {
		logger.Printf("Adjusted edition from '%s' to '%s'", ogEdition, inCard.Edition)
	}

	// Extra check, after any possible edition adjustment has been done
	switch {
	// For any custom token set that may have leaked here
	// Note we cannot use Contains because "token" is filtered away
	case strings.Contains(strings.ToLower(inCard.Edition), "token") ||
		strings.Contains(strings.ToLower(inCard.Variation), "token"):
		return "", ErrUnsupported
	// For any unsupported set that wasn't processed previously
	case inCard.Contains("Oversize"):
		return "", ErrUnsupported
	}

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
			// Return a safe error if it's a token
			if isToken(ogName) || Contains(inCard.Variation, "Oversize") {
				return "", ErrUnsupported
			}
			return "", ErrCardNotInEdition
		}
	}

	// This map will contain the setCode and an array of possible matches for
	// each edition.
	cardSet := map[string][]mtgjson.Card{}

	// Only one printing, it *has* to be it
	if len(printings) == 1 {
		cardSet[printings[0]] = MatchInSet(inCard.Name, printings[0])
	} else if !inCard.Promo {
		// If multiple printing, try filtering to the closest name
		// described by the inCard.Edition.
		// This is skipped if we're in the wildcard Promo mode, as we
		// need as many editions as possible.
		logger.Println("Several printings found, iterating over edition name")

		// First loop, search for a perfect match
		for _, setCode := range printings {
			// Perfect match, the card *has* to be present in the set
			if Equals(backend.Sets[setCode].Name, inCard.Edition) {
				logger.Println("Found a perfect match with", inCard.Edition, setCode)
				cardSet[setCode] = MatchInSet(inCard.Name, setCode)

				set := backend.Sets[setCode]
				setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
				if err != nil {
					continue
				}

				// In case it's a well known promo, consider the promo sets (or vice
				// versa for promo sets) in order to let filtering take care of them
				if inCard.isPrerelease() || inCard.isPromoPack() ||
					(inCard.isBundle() && setDate.After(PromosForEverybodyYay)) ||
					(inCard.isBaB() && setDate.After(BuyABoxInExpansionSetsDate)) {
					setName := backend.Sets[setCode].Name
					if !strings.HasSuffix(setName, "Promos") {
						setCode = "P" + setCode
						set, found := backend.Sets[setCode]
						if found {
							logger.Println("Detected possible promo, adding edition", set.Name, setCode)
							cardSet[setCode] = MatchInSet(inCard.Name, setCode)
						}
					} else {
						setCode = strings.TrimPrefix(setCode, "P")
						set, found := backend.Sets[setCode]
						if found {
							logger.Println("Detected possible non-promo, adding edition", set.Name, setCode)
							cardSet[setCode] = MatchInSet(inCard.Name, setCode)
						}
					}
				}
			}
		}

		// Second loop, hope that a portion of the edition is in the set Name
		// This may result in false positives under certain circumstances.
		if len(cardSet) == 0 {
			logger.Println("No perfect match found, trying with heuristics")
			for _, setCode := range printings {
				set := backend.Sets[setCode]
				setDate, err := time.Parse("2006-01-02", set.ReleaseDate)
				if err != nil {
					continue
				}
				if Contains(set.Name, inCard.Edition) ||
					// If a card is promotional, only consider promotional sets
					(inCard.isGenericPromo() && strings.HasSuffix(set.Name, "Promos")) ||
					// If it is Bundle or BaB, also consider base sets if recent enough
					(inCard.isBundle() && !strings.HasSuffix(set.Name, "Promos") && setDate.After(PromosForEverybodyYay)) ||
					(inCard.isBaB() && !strings.HasSuffix(set.Name, "Promos") && setDate.After(BuyABoxInExpansionSetsDate)) {
					logger.Println("Found a possible match with", inCard.Edition, setCode)
					cardSet[setCode] = MatchInSet(inCard.Name, setCode)
				}
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
		if inCard.Variation == "" {
			err = ErrCardMissingVariant
		}
	// Victory
	case 1:
		logger.Println("Found it!")
		cardId = output(outCards[0], inCard.Foil, inCard.isEtched())
	// FOR SHAME
	default:
		logger.Println("Aliasing...")
		alias := newAliasingError()
		for i := range outCards {
			alias.dupes = append(alias.dupes, output(outCards[i], inCard.Foil, inCard.isEtched()))
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

	// Rename cards that were translated differently
	if strings.Contains(inCard.Edition, "Ikoria") {
		if strings.Contains(inCard.Name, "Mothra's") && strings.Contains(inCard.Name, "Cocoon") {
			inCard.Name = "Mothra's Great Cocoon"
			return
		} else if strings.Contains(inCard.Name, "Battra") {
			inCard.Name = "Battra, Dark Destroyer"
			return
		} else if strings.Contains(inCard.Name, "Mechagodzilla") {
			inCard.Name = "Mechagodzilla, the Weapon"
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
	// Also valid when MaybePrefix preference is set.
	// Attempt first to check cards in the same edition if possible
	for _, set := range backend.Sets {
		if Equals(set.Name, inCard.Edition) {
			for _, card := range set.Cards {
				if (card.Layout != "normal" || inCard.MaybePrefix) && HasPrefix(card.Name, inCard.Name) {
					inCard.Name = card.Name
					return
				}
			}
		}
	}
	for cardName, props := range backend.Cards {
		if (props.Layout != "normal" || inCard.MaybePrefix) && HasPrefix(cardName, inCard.Name) {
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

	set, found := backend.Sets[strings.ToUpper(edition)]
	if found {
		edition = set.Name
	}
	ed, found := EditionTable[edition]
	if found {
		edition = ed
	}
	set, found = backend.Sets[strings.ToUpper(variation)]
	if found && inCard.isJudge() {
		edition = set.Name
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
	case Contains(edition, "Double Masters"):
		if Contains(edition, "Box Toppers") ||
			Contains(edition, "Extras") ||
			Contains(edition, "Variants") {
			edition = "Double Masters"
			if !inCard.isBasicLand() {
				variation = "Borderless"
			}
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
	case inCard.Contains("Timeshifted") && !inCard.Contains("Modern"):
		if len(MatchInSet(inCard.Name, "TSB")) != 0 {
			edition = backend.Sets["TSB"].Name
		} else if len(MatchInSet(inCard.Name, "TSR")) != 0 {
			edition = backend.Sets["TSR"].Name
		}
	default:
		for _, tag := range []string{
			"(Collector Edition)",
			"Collectors",
			"Collector Booster",
			"Extras",
			"Variants",
			"Foil Etched",
			"Etched",
		} {
			// Strip away any extra tags
			if strings.HasSuffix(edition, tag) {
				edition = strings.TrimSuffix(edition, tag)
				edition = strings.TrimSpace(edition)
				edition = strings.TrimSuffix(edition, ":")
				edition = strings.TrimSuffix(edition, "-")
				edition = strings.TrimSpace(edition)
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
		ed := ParseCommanderEdition(edition, variation)
		if ed != "" {
			edition = ed
		}
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
	}
	inCard.Edition = edition
	inCard.Variation = variation

	// Special handling since so many providers get this wrong
	switch {
	// Do nothing here, prevent tags from being mixed up
	case inCard.isMysteryList():

	// XLN Treasure Chest
	case inCard.isBaB() && len(MatchInSet(inCard.Name, "PXTC")) != 0:
		edition = backend.Sets["PXTC"].Name

	// BFZ Standard Series
	case inCard.isGenericAltArt() && len(MatchInSet(inCard.Name, "PSS1")) != 0:
		edition = backend.Sets["PSS1"].Name

	// Champs and States
	case inCard.isGenericExtendedArt() && len(MatchInSet(inCard.Name, "PCMP")) != 0:
		edition = backend.Sets["PCMP"].Name

	// Portal Demo Game
	case inCard.isPortalAlt() && len(MatchInSet(inCard.Name, "PPOD")) != 0:
		edition = backend.Sets["PPOD"].Name

	// Secret Lair {Ultimate,Drop}
	case inCard.Contains("Secret") || Contains(inCard.Variation, "Lair"):
		if len(MatchInSet(inCard.Name, "SLU")) != 0 {
			edition = backend.Sets["SLU"].Name
		} else if len(MatchInSet(inCard.Name, "SLD")) != 0 {
			edition = backend.Sets["SLD"].Name
		}

	// Summer of Magic
	case (inCard.isWPNGateway() || strings.Contains(inCard.Variation, "Summer")) &&
		len(MatchInSet(inCard.Name, "PSUM")) != 0:
		edition = backend.Sets["PSUM"].Name

	// Untagged Planeshift Alternate Art - these could be solved with the
	// Promo handling, but they are not set as such in mtgjson/scryfall
	case inCard.isGenericPromo() && len(MatchInSet(inCard.Name, "PLS")) == 2:
		edition = "PLS"
		variation = "Alternate Art"
		inCard.Promo = false

	// Rename the official name to the the more commonly used name
	case inCard.Edition == "Commander Legends" && inCard.isShowcase():
		variation = "Etched"

	// Planechase deduplication
	case inCard.Equals("Planechase") && len(MatchInSet(inCard.Name, "OHOP")) != 0:
		edition = backend.Sets["OHOP"].Name
	case inCard.Equals("Planechase 2012") && len(MatchInSet(inCard.Name, "OPC2")) != 0:
		edition = backend.Sets["OPC2"].Name
	case inCard.Equals("Planechase Anthology") && len(MatchInSet(inCard.Name, "OPCA")) != 0:
		edition = backend.Sets["OPCA"].Name

	// The first Gift Pack often get folded in the main Core Set 2019 or in the
	// related Promos set, so use a lax way to dected the original expansion
	case ((Contains(inCard.Edition, "Core") && Contains(inCard.Edition, "2019")) || inCard.isGenericPromo()) && len(MatchInSet(inCard.Name, "G18")) == 1:
		edition = backend.Sets["G18"].Name

	// Adjust edition for non-English sets
	case (inCard.Edition == "Legends" || inCard.Edition == "The Dark") && Contains(inCard.Variation, "Italian"):
		edition += " Italian"
	case inCard.Edition == "Renaissance" && Contains(inCard.Variation, "Italian"):
		edition = "Rinascimento"
		// This set has lots of variants, strip away any excess data
		variation = strings.ToLower(inCard.Variation)
		variation = strings.Replace(inCard.Variation, "italian", "", 1)
		variation = strings.TrimSpace(inCard.Variation)
	case inCard.Edition == "Chronicles" && Contains(inCard.Variation, "Japanese"):
		edition += " Japanese"
		// This set has lots of variants, strip away any excess data
		variation = strings.ToLower(inCard.Variation)
		variation = strings.Replace(inCard.Variation, "japanese", "", 1)
		variation = strings.TrimSpace(inCard.Variation)
	case inCard.Edition == "Fourth Edition" && Contains(inCard.Variation, "Japanese"):
		edition = "Fourth Edition Foreign Black Border"

	// Separate timeshifted cards
	case inCard.Contains("Modern Horizons") &&
		(inCard.Contains("Retro Frame") || inCard.Contains("Timeshift")) &&
		len(MatchInSet(inCard.Name, "H1R")) != 0:
		edition = backend.Sets["H1R"].Name

	// Challenger decks promos
	case inCard.Contains("Challenger Decks") && len(MatchInSet(inCard.Name, "Q06")) != 0:
		edition = backend.Sets["Q06"].Name

	// Single card mismatches
	default:
		switch inCard.Name {
		case "Rhox":
			if inCard.isGenericAltArt() || inCard.isGenericPromo() {
				edition = "Starter 2000"
			}
		case "Balduvian Horde":
			if inCard.isJudge() || inCard.isGenericPromo() || inCard.isDCIPromo() {
				edition = "World Championship Promos"
			}
		case "Disenchant":
			if inCard.isArena() && inCard.Foil {
				edition = "Friday Night Magic 2003"
			}
		case "Nalathni Dragon":
			edition = "Dragon Con"
			variation = ""
		case "Ass Whuppin'",
			"Rukh Egg",
			"Scholar of the Lost Trove":
			if inCard.isPrerelease() {
				variation = "Release"
				edition = "Release Events"
			}
		case "Ajani Vengeant":
			if inCard.isRelease() {
				variation = "Prerelease"
			}
		case "Tamiyo's Journal":
			if inCard.Variation == "" && inCard.Foil {
				variation = "Foil"
			}
		case "Underworld Dreams":
			if inCard.isDCIPromo() || inCard.isArena() {
				edition = "Two-Headed Giant Tournament"
			}
		case "Jace Beleren":
			if inCard.isDCIPromo() {
				edition = "Miscellaneous Book Promos"
			}
		case "Serra Angel":
			if inCard.isDCIPromo() || inCard.isBaB() {
				edition = "Wizards of the Coast Online Store"
			}
		case "Incinerate", "Counterspell":
			if inCard.isDCIPromo() {
				edition = "DCI Legend Membership"
			}
		case "Kamahl, Pit Fighter", "Char":
			if inCard.isDCIPromo() {
				edition = "15th Anniversary Cards"
			}
		case "Fling", "Sylvan Ranger":
			if ExtractNumber(inCard.Variation) == "" {
				if inCard.isDCIPromo() {
					edition = "Wizards Play Network 2010"
				} else if inCard.isWPNGateway() {
					edition = "Wizards Play Network 2011"
				}
			}
		case "Hall of Triumph":
			if inCard.isGenericPromo() {
				edition = "Journey into Nyx Promos"
			}
		case "Teferi, Master of Time":
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
		case "Vorinclex, Monstrous Raider":
			// Missing the proper FrameEffect property
			if inCard.isShowcase() || Contains(inCard.Variation, "Phyrexian") {
				num := ExtractNumber(inCard.Variation)
				if num == "" {
					if Contains(inCard.Variation, "Phyrexian") {
						variation = "333"
					} else if inCard.isShowcase() {
						variation = "320"
					}
				}
			}
		case "Mind Stone":
			if inCard.isWPNGateway() {
				if Contains(inCard.Variation, "Gateway") {
					edition = "Gateway 2007"
				} else {
					edition = "Wizards Play Network 2021"
				}
			}
		case "Runo Stromkirk", "Runo Stromkirk // Krothuss, Lord of the Deep":
			if inCard.isShowcase() || Contains(inCard.Variation, "Eternal") {
				num := ExtractNumber(inCard.Variation)
				if num == "" {
					if Contains(inCard.Variation, "Eternal") {
						variation = "327"
					} else if inCard.isShowcase() {
						variation = "316"
					}
				}
			}
		}
	}
	inCard.Edition = edition
	inCard.Variation = variation

	// Restore the original name for specific editions
	switch inCard.Edition {
	case "Secret Lair Drop":
		switch inCard.Name {
		case "Krark's Thumb",
			"Okaun, Eye of Chaos",
			"Propaganda",
			"Stitch in Time",
			"Zndrsplt, Eye of Wisdom":
			inCard.Name = inCard.Name + " // " + inCard.Name
		}
	}
}
