package mtgmatcher

import (
	"strconv"
	"strings"
	"time"

	"github.com/mtgban/go-mtgban/mtgmatcher/mtgjson"
	"golang.org/x/exp/slices"

	"github.com/google/uuid"
)

func MatchId(inputId string, finishes ...bool) (string, error) {
	// Remove any extras after the underscore
	id := strings.Split(inputId, "_")[0]

	// Validate it's an actual uuid or a plain number for tcg id
	_, err := uuid.Parse(id)
	if err != nil {
		_, err := strconv.Atoi(id)
		if err != nil {
			return "", ErrCardUnknownId
		}
	}

	// Look up in one of the possible maps
	co, found := backend.UUIDs[inputId]
	if !found {
		// Second chance, lookup by scryfall id
		co, found = backend.UUIDs[backend.Scryfall[inputId]]
		if !found {
			// Last chance, lookup by tcg id
			co, found = backend.UUIDs[backend.Tcgplayer[inputId]]
		}
	}
	if !found {
		return "", ErrCardUnknownId
	}

	isEtched := len(finishes) > 1 && finishes[1]
	isFoil := len(finishes) > 0 && finishes[0] && !isEtched
	outId := output(co.Card, finishes...)

	// Validate that what we found is correct
	co, found = backend.UUIDs[outId]
	if !found {
		return "", ErrCardUnknownId
	}

	// If the input card was requested as foil, we should double check
	// if the original card has a foil under a separate id
	// Skip this for SLD because not really needed and the amount of numbers
	// favors clashes (ie Swamp 486 and 48)
	if co.Foil != isFoil && co.SetCode != "SLD" {
		// So we iterate over the Variations array and try outputing ids
		// until we find a perfect match in foiling status
		for _, variation := range co.Variations {
			altCo := backend.UUIDs[variation]
			// We assume that the collector number between the two version
			// stays the same, with a different suffix
			if strings.HasPrefix(co.Number, altCo.Number) ||
				strings.HasPrefix(altCo.Number, co.Number) {
				maybeId := output(altCo.Card, isFoil, isEtched)
				altCo = backend.UUIDs[maybeId]
				if altCo.Foil == isFoil {
					outId = maybeId
					break
				}
			}
		}
	}
	return outId, nil
}

func Match(inCard *Card) (cardId string, err error) {
	if backend.Sets == nil {
		return "", ErrDatastoreEmpty
	}

	// Adjust flag as needed
	if inCard.isFoil() {
		inCard.Foil = true
	}

	// Set up language
	if inCard.Language != "" {
		lang, found := languageCode2LanguageTag[strings.ToLower(inCard.Language)]
		if found {
			inCard.Language = lang
		} else {
			for _, field := range strings.Fields(inCard.Language) {
				field = Title(field)
				if slices.Contains(allLanguageTags, field) {
					inCard.Language = field
					break
				}
			}
		}
	}
	// Override if needed
	for _, tag := range allLanguageTags {
		if inCard.Contains(tag) {
			inCard.Language = tag
			break
		}
	}

	// Look up by uuid
	if inCard.Id != "" {
		logger.Printf("Perforing id lookup")
		outId, err := MatchId(inCard.Id, inCard.Foil, inCard.isEtched())
		if err == nil {
			co, _ := backend.UUIDs[outId]
			if inCard.Language == "" || strings.Contains(co.Language, inCard.Language) {
				logger.Printf("Id found")
				return outId, nil
			}
			// Tokens are unsupported for broken ids in different languages
			if co.Layout == "token" {
				return "", ErrUnsupported
			}
			logger.Printf("Language validation failed, resetting card")
			inCard.Name = co.Name
			inCard.Edition = co.Edition
			inCard.Variation = co.Number
			inCard.Foil = co.Foil
			if co.Etched {
				inCard.addToVariant("etched")
			}
		}
		logger.Printf("Id lookup failed, attempting full match")
	}

	// In case id lookup failed, an no more data is present
	if inCard.Name == "" {
		return "", ErrCardDoesNotExist
	}
	ogName := inCard.Name

	// Binderpos weird syntax, with the edition embedded in the name
	if strings.Contains(inCard.Name, "[") {
		vars := strings.Split(inCard.Name, "[")
		inCard.Name = strings.TrimSpace(vars[0])
		if len(vars) > 1 {
			maybeEdition := strings.Join(vars[1:], " ")
			maybeEdition = strings.Replace(maybeEdition, "]", "", -1)
			maybeEdition = strings.TrimSpace(maybeEdition)

			set, err := GetSetByName(maybeEdition)
			if err != nil {
				inCard.Variation = maybeEdition
				// TCG Promo Pack prepends a second P to the edition
				if strings.HasPrefix(maybeEdition, "PP") {
					inCard.Variation = "Promo Pack"
				}
			} else {
				inCard.Edition = set.Name
			}
		}
	}
	// Simple case in which there is a variant embedded in the name
	if strings.Contains(inCard.Name, "(") {
		vars := SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.addToVariant(strings.Join(vars[1:], " "))
		}
	}
	if strings.Contains(inCard.Name, " - ") {
		vars := strings.Split(inCard.Name, " - ")
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.addToVariant(strings.Join(vars[1:], " "))
		}
	}
	if ogName != inCard.Name {
		logger.Printf("Pre-adjusted name from '%s' to '%s' '%s'", ogName, inCard.Name, inCard.Variation)
	}

	// Repeat the check in case the card was renamed above
	if inCard.isFoil() {
		inCard.Foil = true
	}

	// Skip unsupported sets
	if inCard.isUnsupported() {
		return "", ErrUnsupported
	}

	switch inCard.Name {
	case "Red Herring",
		"Pick Your Poison":
		if inCard.isMysteryList() {
			inCard.Name += " Playtest"
		}
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
	ogName = inCard.Name
	inCard.Name = entry.Name

	// Fix up edition
	ogEdition := inCard.Edition
	adjustEdition(inCard)
	if ogName != inCard.Name {
		logger.Printf("Re-adjusted name from '%s' to '%s'", ogName, inCard.Name)
		// If renamed, reload metadata in case of duplicate names
		switch inCard.Name {
		case "Unquenchable Fury Token",
			"Red Herring Playtest",
			"Pick Your Poison Playtest":
			entry = backend.Cards[Normalize(inCard.Name)]
			inCard.Name = entry.Name
			logger.Printf("Clashing name adjusted to '%s'", inCard.Name)
		}
	}
	if ogEdition != inCard.Edition {
		logger.Printf("Adjusted edition from '%s' to '%s'", ogEdition, inCard.Edition)
	}

	// Extra check, after any possible edition adjustment has been done
	switch {
	// For any custom token set that may have leaked here
	// Note we cannot use Contains because "token" is filtered away
	case (strings.Contains(strings.ToLower(inCard.Edition), "token") ||
		strings.Contains(strings.ToLower(inCard.Variation), "token")) &&
		!inCard.Contains("League"):
		return "", ErrUnsupported
	// For any unsupported set that wasn't processed previously
	case inCard.Contains("Oversize") &&
		!(inCard.Contains("Commander") || inCard.Contains("Vanguard") ||
			inCard.Contains("Planechase") || inCard.Contains("Archenemy") ||
			inCard.Contains("Player Rewards")):
		return "", ErrUnsupported
	// For any specific missing card
	case inCard.isSpecificUnsupported():
		return "", ErrUnsupported
	}

	logger.Println("Processing", inCard, entry.Printings)

	// If there are multiple printings of the card, filter out to the
	// minimum common elements, using the rules defined.
	// Given that many tokens are not supported, make sure to filter
	// out unrelated editions.
	printings := entry.Printings
	if len(printings) > 1 || backend.Cards[Normalize(ogName)].Layout == "token" {
		printings = filterPrintings(inCard, printings)
		logger.Println("Filtered printings:", printings)

		// Filtering was too aggressive or wrong data fed,
		// in either case, nothing else to be done here.
		if len(printings) == 0 {
			// Return a safe error if it's a token
			if IsToken(ogName) || Contains(inCard.Variation, "Oversize") {
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
	} else if !inCard.promoWildcard {
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
				// JPN cards are skipped because they are well set usually
				if !inCard.isJPN() && (inCard.isPrerelease() || inCard.isPromoPack() ||
					(inCard.isBundle() && setDate.After(PromosForEverybodyYay)) ||
					(inCard.isBaB() && setDate.After(BuyABoxInExpansionSetsDate))) {
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

		logger.Println("Post filtering status...")
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

	// Language check - out of filterCards to catch single cases too
	if inCard.Language != "" || len(outCards) > 1 {
		var filteredOutCards []mtgjson.Card
		for _, card := range outCards {
			if (inCard.Language == "" && card.Language != "English") ||
				!strings.Contains(card.Language, inCard.Language) {
				logger.Println("Dropping different language prints...")
				logger.Println(card.SetCode, card.Name, card.Number, card.Language)
				continue
			}
			filteredOutCards = append(filteredOutCards, card)
		}
		outCards = filteredOutCards
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
		if inCard.Language != "" {
			err = ErrUnsupported
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
	set, found := backend.Sets[setCode]
	if !found {
		return
	}
	for _, card := range set.Cards {
		if cardName == card.Name {
			outCards = append(outCards, card)
		}
	}
	return
}

// Return an array of mtgjson.Card containing all the cards with the exact
// same name as the input name in the Set identified by setCode with the
// specified collector number.
func MatchInSetNumber(cardName, setCode, number string) (outCards []mtgjson.Card) {
	set, found := backend.Sets[setCode]
	if !found {
		return
	}
	for _, card := range set.Cards {
		if cardName == card.Name && card.Number == number {
			outCards = append(outCards, card)
		}
	}
	return
}

// Try to fixup the name of the card or move extra varitions to the
// variant attribute. This should only be used in case the card name
// was not found.
func adjustName(inCard *Card) {
	// Sticker sheet adjustments
	if strings.Contains(inCard.Name, "Sticker") {
		inCard.Name = strings.Replace(inCard.Name, "Sticker", "", 1)
		inCard.Name = strings.Replace(inCard.Name, "Sheet", "", 1)
		inCard.Name = strings.TrimSpace(inCard.Name)
	}

	// Skip for tokens, we need them to be exact or the prefix search interferes
	if strings.Contains(strings.ToLower(inCard.Name), "token") {
		return
	}
	_, found := backend.Cards[Normalize(inCard.Name+" Token")]
	if found {
		inCard.Name += " Token"
		return
	}
	if IsToken(inCard.Name) {
		return
	}

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
		} else if strings.Contains(inCard.Name, "Battra") {
			inCard.Name = "Battra, Dark Destroyer"
		} else if strings.Contains(inCard.Name, "Mechagodzilla") {
			inCard.Name = "Mechagodzilla, the Weapon"
		}
	}
	// Rename reskinned dual faced cards, only keep one side and keep the
	// flavor name, to make the following lookup in AlternateProps work
	if strings.Contains(inCard.Edition, "Secret Lair") {
		if strings.Contains(inCard.Name, "Hawkins National") {
			inCard.Name = "Hawkins National Laboratory"
		} else if strings.Contains(inCard.Name, "Plains") && strings.Contains(inCard.Name, "Battlefield Forge") {
			inCard.Name = "Plains"
			inCard.Variation = "670"
		}
	}
	// Check if this card may be known as something else
	altProps, found := backend.AlternateProps[Normalize(inCard.Name)]
	if found {
		// Stash the current name for later decoupling if needed
		inCard.addToVariant(inCard.Name)
		// Same for number if available
		if altProps.OriginalNumber != "" {
			inCard.addToVariant(altProps.OriginalNumber)
		}

		inCard.Name = altProps.OriginalName
		if altProps.IsFlavor {
			inCard.addToVariant("Reskin")
		}
		return
	}

	// Special case for Un-sets that sometimes drop the parenthesis
	if strings.Contains(inCard.Edition, "The List") ||
		strings.Contains(inCard.Edition, "Unglued") ||
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
		if Contains(inCard.Name, "Who") && Contains(inCard.Name, "What") &&
			Contains(inCard.Name, "When") && Contains(inCard.Name, "Where") &&
			Contains(inCard.Name, "Why") {
			inCard.Name = "Who // What // When // Where // Why"
			return
		}

		for cardName, props := range backend.Cards {
			if HasPrefix(cardName, inCard.Name) {
				inCard.Name = props.Name
				return
			}
		}
	}

	// Rename a DFC with same name
	splits := strings.Split(inCard.Name, "//")
	if len(splits) == 2 && strings.TrimSpace(splits[0]) == strings.TrimSpace(splits[1]) {
		inCard.Name = strings.TrimSpace(splits[0])
		return
	}

	// Altenatively try checking across any prefix, as long as it's a double
	// sided card, for some particular cases, like meld cards, or Treasure Chest
	// Attempt first to check cards in the same edition if possible
	// Skip for tokens
	for _, set := range backend.Sets {
		if Equals(set.Name, inCard.Edition) {
			for _, card := range set.Cards {
				if card.Layout != "normal" && card.Layout != "token" && HasPrefix(card.Name, inCard.Name) {
					inCard.Name = card.Name
					return
				}
			}
		}
	}
	for cardName, props := range backend.Cards {
		if props.Layout != "normal" && props.Layout != "token" && HasPrefix(cardName, inCard.Name) {
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
	if found && (inCard.isJudge() || inCard.isDuelDecks() || inCard.isDuelDecksAnthology()) {
		edition = set.Name
	}
	ed, found = EditionTable[variation]
	// The Anthologies set has one land with a variant named as an expansion,
	// so what is found should not overwrite the edition in this case
	// As for The List, ignore any further variation
	if found && edition != "Anthologies" && !inCard.isMysteryList() {
		edition = ed

		// If edition was found through the variation tag, drop it
		variation = ""
		// Only keep this information if needed
		if inCard.isEtched() {
			variation = "Etched"
		}
	}
	inCard.Edition = edition
	inCard.Variation = variation

	// Adjust box set
	switch {
	case inCard.Contains("Mythic Edition"):
		edition = "Mythic Edition"
	case strings.Contains(edition, "Invocation") ||
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
		// Cut the edition at the first dash, but avoid Prerelease and PromoPack and MB1/List cards
		// since they are often separated with a dash, but are processed elsewhere
		// Test for "- " and " -" to avoid catching dashes in the name of the edition
		if !inCard.isPrerelease() && !inCard.isPromoPack() && !inCard.isMysteryList() &&
			(strings.Contains(edition, "- ") || strings.Contains(edition, " -")) {
			edition = strings.Split(edition, "-")[0]
			edition = strings.TrimSpace(edition)

			// Check if the edition name needs further processing
			ed, found = EditionTable[edition]
			if found {
				edition = ed
			}

			if variation == "" {
				inCard.beyondBaseSet = true
			}
		}
		// Loop through known editions tags
		for _, tag := range []string{
			"Box Toppers",
			"(Collector Edition)",
			"Collectors",
			"Collector Booster",
			"Extras",
			"Variants",
			"Etched",
			"Serialized",
			"Surge Foil",
			"Holiday Release",
			"Alternate Foil",
			"Retro Frame",
		} {
			// Strip away any extra tags
			if strings.HasSuffix(edition, tag) {
				edition = strings.TrimSuffix(edition, tag)
				edition = strings.TrimSpace(edition)
				edition = strings.TrimSuffix(edition, ":")
				edition = strings.TrimSuffix(edition, "-")
				edition = strings.TrimSpace(edition)

				// Check if the edition name needs further processing
				ed, found = EditionTable[edition]
				if found {
					edition = ed
				}

				// If no other variation, set this flag to do a best effort search
				if variation == "" {
					inCard.beyondBaseSet = true
				}
			}
		}
	}

	switch {
	case strings.Contains(edition, "Commander") &&
		(!inCard.Contains("Oversize") || inCard.Contains("Plane") || inCard.Contains("Phenomenon")) &&
		!inCard.Contains("Party"):
		ed := ParseCommanderEdition(edition, variation)
		if ed != "" {
			edition = ed
		}
	case inCard.Contains("Ravnica Weekend"):
		edition, variation = inCard.ravnicaWeekend()
	case inCard.Contains("Guild Kit"):
		edition = inCard.ravnicaGuidKit()
	case strings.Contains(variation, "APAC Set") || strings.Contains(variation, "Euro Set"):
		num := ExtractNumber(variation)
		if num != "" {
			variation = strings.Replace(variation, num+" ", "", 1)
		}
	case strings.HasPrefix(variation, "Junior") && strings.Contains(variation, "APAC"),
		strings.HasPrefix(variation, "Junior APAC Series") && strings.Contains(variation, "U"):
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
	case Contains(variation, "Boosterfun"):
		inCard.beyondBaseSet = true
	case strings.HasPrefix(edition, "Universes Beyond"):
		edition = strings.TrimPrefix(edition, "Universes Beyond")
		edition = strings.TrimLeft(edition, ":- ")
	}
	inCard.Edition = edition
	inCard.Variation = variation

	// Special handling since so many providers get this wrong
	switch {
	// Prevent tags from being mixed up, only take care of edition changes
	case inCard.isMysteryList():
		switch inCard.Name {
		case "Rafiq of the Many":
			edition = "Shards of Alara"
			variation = "250"
		default:
			if !inCard.isReskin() && len(MatchInSet(inCard.Name, "SLX")) != 0 {
				edition = backend.Sets["SLX"].Name
			}
		}

	// XLN Treasure Chest
	case inCard.isBaB() && len(MatchInSet(inCard.Name, "PXTC")) != 0:
		edition = backend.Sets["PXTC"].Name

	// BFZ Standard Series
	case inCard.isGenericAltArt() && len(MatchInSet(inCard.Name, "PSS1")) != 0:
		edition = backend.Sets["PSS1"].Name

	// Champs and States
	case inCard.isGenericExtendedArt() && len(MatchInSet(inCard.Name, "PCMP")) != 0:
		edition = backend.Sets["PCMP"].Name

	// Secret Lair {Ultimate,Drop}
	case inCard.Contains("Secret Lair"):
		// Check if there are also FlavorNames associated to this card
		// It might happen that a non-FlavorName is requested, so check number too
		altProps, found := backend.AlternateProps[Normalize(inCard.Name)]
		if found && len(MatchInSet(altProps.OriginalName, "SLD")) != 0 {
			var shouldRename bool
			cards := MatchInSet(altProps.OriginalName, "SLD")
			num := ExtractNumber(inCard.Variation)
			for _, card := range cards {
				if card.Number == num || (card.FaceFlavorName != "" && Contains(inCard.Variation, card.FaceFlavorName)) {
					shouldRename = true
					break
				}
			}

			if shouldRename {
				inCard.Name = altProps.OriginalName
			}
		}

		switch {
		case inCard.Contains("Plains"), inCard.Contains("Battlefield Forge"):
			if inCard.Contains("Unpeeled") {
				inCard.Name = "Battlefield Forge"
				inCard.Variation = "669"
			} else if inCard.Contains("Peeled") {
				inCard.Name = "Plains"
				inCard.Variation = "670"
			}
		}

		switch {
		case len(MatchInSet(inCard.Name, "SLX")) != 0 &&
			!inCard.isReskin():
			edition = backend.Sets["SLX"].Name
		case len(MatchInSet(inCard.Name, "SLC")) != 0 &&
			(len(MatchInSet(inCard.Name, "SLD")) == 0 ||
				inCard.Contains("30th") ||
				inCard.Contains("Countdown") ||
				ExtractYear(inCard.Variation) != ""):
			edition = backend.Sets["SLC"].Name
		case len(MatchInSet(inCard.Name, "SLP")) != 0 &&
			(len(MatchInSet(inCard.Name, "SLD")) == 0 ||
				inCard.Contains("Showdown") ||
				inCard.Contains("Prize") ||
				inCard.Contains("Play")):
			edition = backend.Sets["SLP"].Name
		case len(MatchInSet(inCard.Name, "SLU")) != 0:
			edition = backend.Sets["SLU"].Name
		case len(MatchInSet(inCard.Name, "SLD")) != 0:
			edition = backend.Sets["SLD"].Name
		}

	// Untagged Planeshift Alternate Art - these could be solved with the
	// Promo handling, but they are not set as such in mtgjson/scryfall
	case (inCard.isGenericPromo() || inCard.isGenericAltArt()) && len(MatchInSet(inCard.Name, "PLS")) == 2:
		edition = "PLS"
		variation = "Alternate Art"

	// Rename the official name to the the more commonly used name
	case inCard.Edition == "Commander Legends" && inCard.isShowcase():
		variation = "Etched"

	// Planechase deduplication
	case inCard.Contains("Planechase") && len(MatchInSet(inCard.Name, "PDCI")) != 0 && (inCard.isRelease() || inCard.isDCIPromo() || inCard.isWPNGateway()):
		edition = backend.Sets["PDCI"].Name
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
		variation = strings.Replace(variation, "italian", "", 1)
		variation = strings.TrimSpace(variation)
	case strings.Contains(inCard.Edition, "Chronicles") && (inCard.Contains("Japanese") || inCard.Contains("FBB")):
		edition = "Chronicles Foreign Black Border"
		// This set has lots of variants, strip away any excess data
		variation = strings.ToLower(inCard.Variation)
		variation = strings.Replace(variation, "japanese", "", 1)
		variation = strings.TrimSpace(variation)
	case inCard.Edition == "Fourth Edition" && Contains(inCard.Variation, "Japanese"):
		edition = "Fourth Edition Foreign Black Border"

	// JPN promos from PRES
	case inCard.isIDWMagazineBook() && inCard.isJPN() && len(MatchInSet(inCard.Name, "PRES")) != 0:
		edition = backend.Sets["PRES"].Name

	// Separate timeshifted cards
	case inCard.Contains("Modern Horizons") &&
		(inCard.Contains("Retro Frame") || inCard.Contains("Timeshift")) &&
		len(MatchInSet(inCard.Name, "H1R")) != 0:
		edition = backend.Sets["H1R"].Name

	// Clash pack promos
	case (inCard.Contains("Clash") || inCard.isGenericPromo()) && len(MatchInSet(inCard.Name, "CP1")) == 1:
		edition = backend.Sets["CP1"].Name
	case (inCard.Contains("Clash") || inCard.isGenericPromo()) && len(MatchInSet(inCard.Name, "CP2")) == 1:
		edition = backend.Sets["CP2"].Name
	case (inCard.Contains("Clash") || inCard.isGenericPromo()) && len(MatchInSet(inCard.Name, "CP3")) == 1:
		edition = backend.Sets["CP3"].Name

	// Challenger decks promos
	case (inCard.Contains("Challenger Decks") || inCard.isGenericPromo()) && len(MatchInSet(inCard.Name, "Q06")) != 0:
		edition = backend.Sets["Q06"].Name

	// Open the Helvault oversized cards
	case (inCard.Contains("Oversize") || inCard.Contains("Helvault Promo") || inCard.isPrerelease()) && len(MatchInSet(inCard.Name, "PHEL")) == 1:
		edition = backend.Sets["PHEL"].Name
		variation = ""

	// All the oversized commander cards
	case inCard.Contains("Oversize") && !inCard.Contains("Plane") && !inCard.Contains("Phenomenon"):
		for _, tag := range []string{
			"OCM1", "PCMD", "OCMD", "OC13", "OC14", "OC15", "OC16", "OC17", "OC18", "OC19", "OC20",
		} {
			if inCard.Name == "Mayael the Anima" && !inCard.Contains("Arsenal") {
				edition = backend.Sets["OC13"].Name
				break
			} else if len(MatchInSet(inCard.Name, tag)) == 1 {
				edition = backend.Sets[tag].Name
				break
			}
		}

	// Lunar Year Promos
	case (inCard.isGenericPromo() || inCard.Contains("Lunar")) && len(MatchInSet(inCard.Name, "PL21")) == 1:
		edition = backend.Sets["PL21"].Name

	// Love Your LGS 2021, often confused with WPN
	case (inCard.isWPNGateway() || inCard.isGenericPromo()) && inCard.Contains("Retro Frame") && len(MatchInSet(inCard.Name, "PLG21")) == 1:
		edition = backend.Sets["PLG21"].Name

	// WPN 2021
	case inCard.Name != "Mind Stone" && inCard.isGenericPromo() && len(MatchInSet(inCard.Name, "PW21")) == 1:
		edition = backend.Sets["PW21"].Name

	// Unfinity Sticker Sheets
	case inCard.Edition == "Unfinity" && len(MatchInSet(inCard.Name, "SUNF")) == 1:
		edition = backend.Sets["SUNF"].Name

	// Move Release to Prerelease for Battlebond
	case inCard.isRelease() && strings.Contains(edition, "Battlebond") && len(MatchInSet(inCard.Name, "PBBD")) == 1:
		edition = backend.Sets["PBBD"].Name

	// Remove edition since the cards are either in ONE or in another set, but single printed
	case inCard.Contains("Phyrexia: All") && inCard.Contains("Concept"):
		switch inCard.Name {
		default:
			edition = "ignored"
		}

	// Decouple P30A from P30H and its Japanese version
	case inCard.Contains("30th Anniversary") && !inCard.Contains("Edition") && !inCard.Contains("Tokyo") && !inCard.Contains("Misc") && len(MatchInSet(inCard.Name, "P30H")) > 0:
		edition = backend.Sets["P30H"].Name
		if inCard.Contains("Japanese") {
			edition = backend.Sets["P30HJPN"].Name
		} else if inCard.Name == "Serra Angel" && (!inCard.Contains("History") || ExtractYear(inCard.Variation) != "") {
			edition = backend.Sets["P30A"].Name
		}

	// Oilslick lands may not have the bundle tag attached to them
	case inCard.isBasicLand() && inCard.isOilSlick() && !inCard.isBundle():
		variation += " Bundle"

	// Many providers don't tag these promos correctly
	case inCard.isRelease() && len(MatchInSet(inCard.Name, "PBBD")) == 1:
		edition = backend.Sets["PBBD"].Name
		variation = "Prerelease"

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
		case "Reya Dawnbringer":
			if inCard.isRelease() {
				edition = "P10E"
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
			if inCard.isDCIPromo() || inCard.isArena() || inCard.Contains("2HG") || inCard.Contains("Two-Headed Giant") {
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
			if inCard.isDCIPromo() || (inCard.Contains("Legend") && (inCard.Contains("Promo") || inCard.Contains("Member"))) {
				edition = "DCI Legend Membership"
			}
		case "Faerie Conclave", "Treetop Village":
			if inCard.isWPNGateway() || inCard.Contains("Summer") {
				edition = "Tenth Edition Promos"
			}
		case "Kamahl, Pit Fighter", "Char":
			if inCard.isDCIPromo() || inCard.Contains("15th Anniversary") || inCard.isGenericPromo() {
				edition = "15th Anniversary Cards"
			}
		case "Fling":
			if (inCard.isDCIPromo() || inCard.isWPNGateway()) && ExtractNumber(inCard.Variation) == "" {
				edition = backend.Sets["PDCI"].Name
				if inCard.isDCIPromo() {
					variation = "50"
				} else if inCard.isWPNGateway() {
					variation = "69"
				}
			}
		case "Sylvan Ranger":
			if (inCard.isDCIPromo() || inCard.isWPNGateway()) && ExtractNumber(inCard.Variation) == "" {
				edition = backend.Sets["PDCI"].Name
				if inCard.isDCIPromo() {
					variation = "51"
				} else if inCard.isWPNGateway() {
					variation = "70"
				}
			}
		case "Naya Sojourners":
			if inCard.isGenericPromo() {
				edition = backend.Sets["PDCI"].Name
			}
		case "Hall of Triumph":
			if inCard.isGenericPromo() {
				edition = "Journey into Nyx Promos"
			}
		case "Reliquary Tower":
			if inCard.Contains("League") {
				edition = backend.Sets["PM19"].Name
			} else if inCard.Contains("Bring a Friend") {
				edition = backend.Sets["PLG20"].Name
			}
		case "Bolas's Citadel":
			if inCard.isGenericPromo() {
				edition = backend.Sets["PWAR"].Name
			}
		case "Llanowar Elves":
			if inCard.isGenericPromo() {
				edition = backend.Sets["PDOM"].Name
			}
		case "Evolving Wilds":
			if inCard.isGenericPromo() {
				edition = backend.Sets["PRIX"].Name
			}
		case "Unquenchable Fury":
			if inCard.Edition == "Battle the Horde" {
				inCard.Name += " Token"
			}
		case "Pick Your Poison",
			"Red Herring":
			if strings.Contains(edition, "Playtest") {
				inCard.Name += " Playtest"
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
			if strings.HasSuffix(variation, "s") || strings.HasSuffix(variation, "p") {
				edition = backend.Sets["PM21"].Name
			}
		case "Mind Stone":
			switch edition {
			// Skip the check if this card already has the right edition
			case "DCI Promos",
				"Wizards Play Network 2021":
			default:
				if inCard.isWPNGateway() {
					edition = "Wizards Play Network 2021"
					if inCard.Contains("Gateway") {
						edition = "DCI Promos"
					}
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
		case "Diabolic Tutor":
			if inCard.isIDWMagazineBook() {
				edition = "Secret Lair Drop"
			}
		case "Magister of Worth":
			if inCard.isBaB() {
				variation = "Launch"
			}
		case "Arcane Signet":
			if inCard.Contains("Festival") || inCard.Contains("MagicFest") || inCard.Contains("30th") {
				edition = "30th Anniversary Misc Promos"
				if inCard.isEtched() {
					variation = "1F★"
				} else if inCard.isRetro() {
					variation = "1P"
				} else {
					variation = "1F"
				}
			}
		case "Hangarback Walker":
			if inCard.isReskin() || inCard.isGenericPromo() || strings.Contains(inCard.Edition, "LGS") {
				edition = backend.Sets["PLG20"].Name
			}
		// Sometimes these cards are not marked as prerelease because they are showcase
		case "Goro-Goro and Satoru", "Katilda and Lier", "Slimefoot and Squee":
			if inCard.isShowcase() && !inCard.isPrerelease() {
				variation += " Prerelease"
			}
		// There are three Prerelease editions across two editions
		case "Delighted Halfling",
			"Lobelia Sackville-Baggins",
			"Frodo Baggins",
			"Bilbo, Retired Burglar",
			"Gandalf, Friend of the Shire",
			"Wizard's Rockets":
			if inCard.isBorderless() && !inCard.isPrerelease() {
				variation += " Prerelease"
			}
		default:
			// Attempt a best effor match for known promotional tags if card or edition
			// wasn't found in previous steps
			if inCard.isGenericPromo() {
				logger.Printf("Precise matching for promo failed, attempting best effort")
				inCard.promoWildcard = true
			}
		}
	}
	inCard.Edition = edition
	inCard.Variation = variation

	// Adjust incorrect numbers sometimes used for Etched
	num := ExtractNumber(inCard.Variation)
	if num != "" && strings.HasSuffix(num, "e") && HasEtchedPrinting(inCard.Name, inCard.Edition) {
		fixedNum := strings.TrimSuffix(num, "e")
		variation = strings.Replace(variation, num, fixedNum, -1)
		if !Contains(variation, "Etched") {
			variation += " Etched"
		}
	}
	inCard.Variation = variation
}
