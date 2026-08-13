package mtgmatcher

import (
	"slices"
	"strconv"
	"strings"
)

func MatchId(inputId string, finishes ...bool) (string, error) {
	return defaultBackend.MatchId(inputId, finishes...)
}

func Match(inCard *InputCard) (cardId string, err error) {
	return defaultBackend.Match(inCard)
}

// Return an array of Card containing all the cards with the exact
// same name as the input name in the Set identified by setCode.
// In case of combined card names (with '//' in their name), only the
// first chunk is considered
func MatchInSet(cardName string, setCode string) (outCards []Card) {
	return defaultBackend.MatchInSet(cardName, setCode)
}

// Return an array of Card containing all the cards with the exact
// same name as the input name in the Set identified by setCode with the
// specified collector number.
func MatchInSetNumber(cardName, setCode, number string) (outCards []Card) {
	return defaultBackend.MatchInSetNumber(cardName, setCode, number)
}

// Return an array of Card containing all the cards with the exact
// set code and collector number, using the name as hint (can be empty)
func MatchWithNumber(cardName, setCode, number string) (outCards []Card) {
	return defaultBackend.MatchWithNumber(cardName, setCode, number)
}

func (b *Backend) MatchId(inputId string, finishes ...bool) (string, error) {
	// Remove any extras after the underscore
	id := strings.Split(inputId, "_")[0]

	// Validate it's an actual uuid or a plain number for tcg id
	if !maybeUUID(id) {
		_, err := strconv.Atoi(id)
		if err != nil {
			return "", ErrCardUnknownId
		}
	}

	// Look up in one of the possible maps
	co, found := b.UUIDs[inputId]
	if !found {
		co, found = b.UUIDs[b.ExternalIdentifiers[inputId]]
	}
	if !found {
		return "", ErrCardUnknownId
	}

	isEtched := len(finishes) > 1 && finishes[1]
	isFoil := len(finishes) > 0 && finishes[0] && !isEtched

	// If the loaded card already matches the requested finishes
	// return the found id straight away
	if (co.Foil && isFoil) || (co.Etched && isEtched) ||
		(!co.Foil && !co.Etched && !isFoil && !isEtched) {
		return co.UUID, nil
	}

	outId := b.output(co.Card, finishes...)

	// Validate that what we found is correct
	co, found = b.UUIDs[outId]
	if !found {
		return "", ErrCardUnknownId
	}

	// If the input card was requested as foil, we should double check
	// if the original card has a foil under a separate id
	if co.Foil != isFoil || co.Etched != isEtched {
		// So we iterate over the Variations array and try outputting ids
		// until we find a perfect match in foiling status
		for _, variation := range co.Variations {
			// A missing key yields a nil pointer, not an empty card, so
			// every read of this map has to be checked before use
			altCo, found := b.UUIDs[variation]
			if !found {
				continue
			}
			// We assume that the collector number between the two version
			// stays the same, with a different suffix
			if ExtractNumberValue(co.Number) == ExtractNumberValue(altCo.Number) {
				maybeId := b.output(altCo.Card, isFoil, isEtched)
				altCo, found = b.UUIDs[maybeId]
				if !found {
					continue
				}

				// Make sure we're dealing with the same card
				// (this helps with promos that have similar numbers)
				// but different finish
				sameFinish := (co.HasFinish(FinishNonfoil) && altCo.HasFinish(FinishNonfoil)) ||
					(co.HasFinish(FinishFoil) && altCo.HasFinish(FinishFoil)) ||
					(co.HasFinish(FinishEtched) && altCo.HasFinish(FinishEtched))
				if sameFinish {
					continue
				}

				// If the alt card finish matches the expected one
				// then replace the final output uuid
				if altCo.Foil == isFoil && altCo.Etched == isEtched {
					outId = maybeId
					break
				}
			}
		}
	}
	return outId, nil
}

// hasFoilSubtype reports whether a printing registers a finish the caller's
// foil and etched flags cannot name - Lorcana's cold foil, rainbow pillars
// and the rest, which TCGplayer prices as skus of their own.
func hasFoilSubtype(co *CardObject) bool {
	for finish := range co.FoilUUIDs {
		switch finish {
		case FinishNonfoil, FinishFoil, FinishEtched:
		default:
			return true
		}
	}
	return false
}

// isFoilSubtype reports whether this entry is one of those sub-types itself,
// rather than a sibling that merely has one. The Finish field cannot answer
// it: a loader records the source's own foil name there for the primary foil
// too ("silver"), so only the key the uuid is filed under tells them apart.
func isFoilSubtype(co *CardObject) bool {
	for finish, uuid := range co.FoilUUIDs {
		switch finish {
		case FinishNonfoil, FinishFoil, FinishEtched:
		default:
			if uuid == co.UUID {
				return true
			}
		}
	}
	return false
}

// soleFinishOf returns the one id among the candidates that is a finish of
// the given printing, and an empty string when none or several are - only a
// single candidate identifies the printing the caller meant.
func soleFinishOf(co *CardObject, ids []string) string {
	var match string
	for _, id := range ids {
		isFinish := false
		for _, uuid := range co.FoilUUIDs {
			if uuid == id {
				isFinish = true
				break
			}
		}
		if !isFinish {
			continue
		}
		if match != "" && match != id {
			return ""
		}
		match = id
	}
	return match
}

func (b *Backend) Match(inCard *InputCard) (cardId string, err error) {
	if b.Sets == nil {
		return "", ErrDatastoreEmpty
	}

	// Adjust flag as needed
	if inCard.IsFoil() {
		inCard.Foil = true
	}

	// Set up language
	if inCard.Language != "" {
		lang, found := LanguageCode2LanguageTag[strings.ToLower(inCard.Language)]
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
	//
	// subtypeCo holds the printing the id resolved to where the sub-type case
	// below hands the finish to the wording, so the aliasing arm at the end of
	// the function can tell the wording's candidates apart.
	var subtypeCo *CardObject
	if inCard.Id != "" {
		Logger.Printf("Performing id lookup")
		outId, err := b.MatchId(inCard.Id, inCard.Foil, inCard.IsEtched())
		if err == nil {
			co := b.UUIDs[outId]
			Logger.Printf("Id found")

			// Validation step
			switch {
			// Only the default language is supported by id
			case inCard.Language != "" && !strings.Contains(co.Language, inCard.Language):
				Logger.Printf("Language validation failed, resetting card")
				inCard.Name = co.Name
				inCard.Edition = co.Edition
				inCard.Variation = co.Number
				inCard.Foil = co.Foil
				if co.Etched {
					inCard.AddToVariant("etched")
				}
			// Tokens are unsupported for broken ids in different languages
			case inCard.Language != "" && co.Layout == "token":
				return "", ErrUnsupported
			// This runs before b.rules is known non-nil, hence the check
			case b.rules != nil && b.rules.MissingPromoTag(b, inCard, co):
				Logger.Println("Missing necessary tag")
				return "", ErrUnsupported
			// MatchId resolves the finish from the caller's flags, which can
			// name the three finishes a flag has a bit for and no more. A
			// printing sold in a named foil sub-type beyond them has a
			// choice the flags cannot make, and answering with the plain
			// foil would file two of its sku prices under one id; the text
			// path below reads the sub-type out of the wording the caller
			// sent alongside. Without a name there is no text path to fall
			// to, and an id that named the sub-type outright has already
			// made the choice, so in both cases the id's answer stands.
			case inCard.Name != "" && hasFoilSubtype(co) && !isFoilSubtype(co):
				Logger.Println("Printing carries a foil sub-type, letting the wording pick")
				subtypeCo = co
			// Actually found id
			default:
				return outId, nil
			}
		}
		Logger.Printf("Id lookup failed, attempting full match")
	}

	// In case id lookup failed, an no more data is present
	if inCard.Name == "" {
		return "", ErrCardDoesNotExist
	}
	ogName := inCard.Name

	// A Backend without attached GameRules cannot match anything; check before
	// the prefilter below, which runs the game's name preprocessing.
	rules := b.rules
	if rules == nil {
		return "", ErrDatastoreEmpty
	}

	// Prefilter runs the game-specific name/variant preprocessing before the
	// canonical-name lookup: Magic splits bracketed editions and parenthesized
	// or dashed variants off the name, Lorcana only the parenthetical (its
	// names are "Character - Title"), plus each game's token/name fixups.
	rules.Prefilter(b, inCard)

	// Re-check foil in case prefilter moved a finish hint into the variant.
	if inCard.IsFoil() {
		inCard.Foil = true
	}
	if ogName != inCard.Name {
		Logger.Printf("Pre-adjusted name from '%s' to '%s' '%s'", ogName, inCard.Name, inCard.Variation)
	}

	// Skip unsupported sets
	if rules.IsUnsupported(b, inCard) {
		return "", ErrUnsupported
	}

	// Get the card basic info to retrieve the Printings array
	canonicalName, found := b.CanonicalNames[Normalize(inCard.Name)]
	if !found {
		ogName := inCard.Name
		// Fixup up the name and try again
		rules.AdjustName(b, inCard)
		if ogName != inCard.Name {
			inCard.OriginalName = ogName
			Logger.Printf("Adjusted name from '%s' to '%s'", ogName, inCard.Name)
		}

		canonicalName, found = b.CanonicalNames[Normalize(inCard.Name)]
		if !found {
			// Return a safe error if it's a token
			if b.IsToken(ogName) || Contains(inCard.Variation, "Oversize") {
				return "", ErrUnsupported
			}
			return "", ErrCardDoesNotExist
		}
	}

	// Restore the card to the canonical MTGJSON name
	ogName = inCard.Name
	inCard.Name = canonicalName

	// Fix up edition
	ogEdition := inCard.Edition
	rules.AdjustEdition(b, inCard)
	if ogName != inCard.Name {
		Logger.Printf("Re-adjusted name from '%s' to '%s'", ogName, inCard.Name)
	}
	if ogEdition != inCard.Edition {
		Logger.Printf("Adjusted edition from '%s' to '%s'", ogEdition, inCard.Edition)
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
	case rules.IsSpecificUnsupported(b, inCard):
		return "", ErrUnsupported
	}

	printings, err := b.Printings4Card(inCard.Name)
	if err != nil {
		Logger.Println("Printings error:", err)
		return "", err
	}

	// If there are multiple printings of the card, filter out to the
	// minimum common elements, using the rules defined.
	// Given that many tokens are not supported, make sure to filter
	// out unrelated editions.
	Logger.Println("Processing", inCard, printings)
	if len(printings) > 1 || strings.HasSuffix(ogName, "Token") {
		printings = rules.FilterPrintings(b, inCard, printings)
		Logger.Println("Filtered printings:", printings)

		// Filtering was too aggressive or wrong data fed,
		// in either case, nothing else to be done here.
		if len(printings) == 0 {
			// Return a safe error if it's a token
			if b.IsToken(ogName) || Contains(inCard.Variation, "Oversize") {
				return "", ErrUnsupported
			}
			return "", ErrCardNotInEdition
		}
	}

	// This map will contain the setCode and an array of possible matches for
	// each edition.
	cardSet := map[string][]Card{}

	// Only one printing, it *has* to be it
	if len(printings) == 1 {
		cardSet[printings[0]] = b.MatchInSet(inCard.Name, printings[0])
	} else if !inCard.PromoWildcard && !inCard.IsSecretLair() {
		// If multiple printing, try filtering to the closest name
		// described by the inCard.Edition.
		// This is skipped if we're in the wildcard Promo mode, as we
		// need as many editions as possible.
		Logger.Println("Several printings found, iterating over edition name")

		// First loop, search for a perfect match
		for _, setCode := range printings {
			// Perfect match, the card *has* to be present in the set
			if Equals(b.Sets[setCode].Name, inCard.Edition) {
				Logger.Println("Found a perfect match with", inCard.Edition, setCode)
				cardSet[setCode] = b.MatchInSet(inCard.Name, setCode)

				set := b.Sets[setCode]

				// In case it's a well known promo, consider the promo sets (or vice
				// versa for promo sets) in order to let filtering take care of them
				// JPN cards are skipped because they are well set usually
				if !inCard.IsJPN() && (inCard.IsPrerelease() || inCard.IsPromoPack() ||
					(inCard.IsBundle() && set.ReleaseDateTime.After(PromosForEverybodyYay)) ||
					(inCard.IsBaB() && set.ReleaseDateTime.After(BuyABoxInExpansionSetsDate))) {
					setName := b.Sets[setCode].Name
					if !strings.HasSuffix(setName, "Promos") {
						setCode = "P" + setCode
						set, found := b.Sets[setCode]
						if found {
							Logger.Println("Detected possible promo, adding edition", set.Name, setCode)
							cardSet[setCode] = b.MatchInSet(inCard.Name, setCode)
						}
					} else {
						setCode = strings.TrimPrefix(setCode, "P")
						set, found := b.Sets[setCode]
						if found {
							Logger.Println("Detected possible non-promo, adding edition", set.Name, setCode)
							cardSet[setCode] = b.MatchInSet(inCard.Name, setCode)
						}
					}
				}
			}
		}

		// Second loop, hope that a portion of the edition is in the set Name
		// This may result in false positives under certain circumstances.
		if len(cardSet) == 0 {
			Logger.Println("No perfect match found, trying with heuristics")
			for _, setCode := range printings {
				set := b.Sets[setCode]

				// Skip heuristics for WCD as short version would catch a lot
				if inCard.IsWorldChamp() {
					break
				}

				if Contains(set.Name, inCard.Edition) ||
					// If a card is promotional, only consider promotional sets
					(inCard.IsGenericPromo() && strings.HasSuffix(set.Name, "Promos")) ||
					// If it is Bundle or BaB, also consider base sets if recent enough
					(inCard.IsBundle() && !strings.HasSuffix(set.Name, "Promos") && set.ReleaseDateTime.After(PromosForEverybodyYay)) ||
					(inCard.IsBaB() && !strings.HasSuffix(set.Name, "Promos") && set.ReleaseDateTime.After(BuyABoxInExpansionSetsDate)) {
					Logger.Println("Found a possible match with", inCard.Edition, setCode)
					cardSet[setCode] = b.MatchInSet(inCard.Name, setCode)
				}
			}
		}
	}

	// Third loop, YOLO
	// Let's consider every edition and hope the second pass will filter
	// duplicates out. This may result in false positives of course.
	if len(cardSet) == 0 {
		Logger.Println("No loose match found, trying all")
		for _, setCode := range printings {
			cardSet[setCode] = b.MatchInSet(inCard.Name, setCode)
		}
	}

	// Log the candidate matches
	Logger.Println("Found these possible matches")
	for _, dupCards := range cardSet {
		for _, card := range dupCards {
			Logger.Println(card.SetCode, card.Name, card.Number)
		}
	}

	// Filter the candidates using all the input card details. The game's rules
	// own this step, so even a single candidate is validated rather than used
	// blindly (Lorcana enforces the collector number here, which the old
	// single-card shortcut skipped, returning a wrong-numbered card).
	Logger.Println("Now filtering...")
	outCards := rules.FilterCards(b, inCard, cardSet)

	Logger.Println("Post filtering status...")
	for _, card := range outCards {
		Logger.Println(card.SetCode, card.Name, card.Number)
	}

	// Just keep the first card found for gold-bordered sets
	if len(outCards) > 1 {
		if inCard.IsWorldChamp() {
			Logger.Println("Dropping a few extra entries...")
			Logger.Println(outCards[1:])
			outCards = []Card{outCards[0]}
		}
	}

	// Language check - out of filterCards to catch single cases too
	if inCard.Language != "" || len(outCards) > 1 {
		var filteredOutCards []Card
		for _, card := range outCards {
			if (inCard.Language == "" && card.Language != "English") ||
				!strings.Contains(card.Language, inCard.Language) {
				Logger.Println("Dropping different language prints...")
				Logger.Println(card.SetCode, card.Name, card.Number, card.Language)
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
		Logger.Println("No matches...")
		err = ErrCardWrongVariant
		if inCard.Variation == "" {
			err = ErrCardMissingVariant
		}
		if inCard.Language != "" {
			err = ErrUnsupported
		}
	// Victory
	case 1:
		Logger.Println("Found it!")

		cardId = b.output(outCards[0], inCard.Foil, inCard.IsEtched())

		co := b.UUIDs[cardId]
		Logger.Println(inCard, "->", co)

		// Validation step
		if rules.MissingPromoTag(b, inCard, co) {
			Logger.Println("...but it's invalid")
			return "", ErrUnsupported
		}
	// FOR SHAME
	default:
		Logger.Println("Aliasing...")
		alias := NewAliasingError()
		for i := range outCards {
			alias.Dupes = append(alias.Dupes, b.output(outCards[i], inCard.Foil, inCard.IsEtched()))
		}
		err = alias

		// The candidates carry the sub-type the wording picked but sit on
		// several printings, and the id named a printing but no sub-type:
		// when one of them is a finish of that printing, the two halves
		// answer together what neither did alone.
		if subtypeCo != nil {
			cardId = soleFinishOf(subtypeCo, alias.Dupes)
			if cardId != "" {
				Logger.Println("Aliasing resolved by the id's printing:", cardId)
				err = nil
			}
		}
	}

	return
}

func (b *Backend) MatchInSet(cardName string, setCode string) (outCards []Card) {
	set, found := b.Sets[setCode]
	if !found {
		return
	}
	for _, card := range set.Cards {
		// Cut rather than Split: only the front half is ever read, and
		// Split allocates a slice for every card in the set to hand it
		// over, whether or not the name has two halves at all
		front, _, _ := strings.Cut(card.Name, " // ")
		if cardName == card.Name || cardName == front {
			outCards = append(outCards, card)
		}
	}
	return
}

func (b *Backend) MatchInSetNumber(cardName, setCode, number string) (outCards []Card) {
	set, found := b.Sets[setCode]
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

func (b *Backend) MatchWithNumber(cardName, setCode, number string) (outCards []Card) {
	set, found := b.Sets[setCode]
	if !found {
		return
	}
	for _, card := range set.Cards {
		if Contains(card.Name, cardName) && card.Number == number {
			outCards = append(outCards, card)
		}
	}
	return
}
