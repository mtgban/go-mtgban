// Package mtgmatcher resolves the loose card descriptions storefronts
// publish, a name and whatever qualifiers they chose, to the one printing
// they meant, and gives every printing a stable uuid to price against.
//
// Each game registers its own datastore loader and matching rules; see the
// mtgmatcher/<game> packages, or mtgmatcher/games to link them all in.
package mtgmatcher

import (
	"errors"
	"slices"
	"strings"
)

// MatchID resolves an identifier a storefront already knows to the uuid of a
// printing, using the default datastore. See the method.
func MatchID(inputID string, finishes ...bool) (string, error) {
	return defaultBackend.MatchID(inputID, finishes...)
}

// MatchIDFinish resolves an id to the uuid of the printing's sibling sold in
// the named finish, spelled however the caller's source spells it. See the
// method.
func MatchIDFinish(inputID, finish string) (string, error) {
	return defaultBackend.MatchIDFinish(inputID, finish)
}

// finishTwins reports whether two set-mates are one card filed as two
// finish-split entries: the same collector number - the foil twin only adds
// a suffix - with no primary finish sold by both, which is what tells a
// twin apart from a promo that happens to share the number. The three
// finishes named are the axis the flags can ask for, which every loader
// files its game's vocabulary into, not a magic idiom: a secondary
// treatment both entries sell - a signed art card - must not hide that
// the pair splits on the axis.
func finishTwins(co, altCo *CardObject) bool {
	if ExtractNumberValue(co.Number) != ExtractNumberValue(altCo.Number) {
		return false
	}
	sameFinish := (co.HasFinish(FinishNonfoil) && altCo.HasFinish(FinishNonfoil)) ||
		(co.HasFinish(FinishFoil) && altCo.HasFinish(FinishFoil)) ||
		(co.HasFinish(FinishEtched) && altCo.HasFinish(FinishEtched))
	return !sameFinish
}

// IsFinishTwin reports whether the printing is one of a pair the set filed
// as two entries for the one card, the twin's number differing only by a
// suffix. The star such a number ends in is the twin's own, and says nothing
// about a misprint.
func (b *Backend) IsFinishTwin(inputID string) bool {
	co, err := b.cardObject4Id(inputID)
	if err != nil {
		return false
	}
	for _, variation := range co.Variations {
		altCo, found := b.UUIDs[variation]
		if !found {
			continue
		}
		if finishTwins(co, altCo) {
			return true
		}
	}
	return false
}

// FinishSiblings answers every uuid the card behind the id is sold under,
// itself included: the printing's registered finishes first - the base, the
// shared ones, then the game's own vocabulary in sorted order - and, for the
// sets that filed a foil as a card of its own, the set-mates that are its
// finish twins. A sealed product or an unknown id answers with what it is.
func (b *Backend) FinishSiblings(inputID string) []string {
	co, err := b.cardObject4Id(inputID)
	if err != nil {
		return nil
	}

	var siblings []string
	seen := map[string]bool{}
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		_, found := b.UUIDs[id]
		if !found {
			return
		}
		seen[id] = true
		siblings = append(siblings, id)
	}

	addFinishes := func(co *CardObject) {
		for _, finish := range []string{FinishNonfoil, FinishFoil, FinishEtched} {
			add(co.FoilUUIDs[finish])
		}
		var extra []string
		for finish := range co.FoilUUIDs {
			switch finish {
			case FinishNonfoil, FinishFoil, FinishEtched:
				continue
			}
			extra = append(extra, finish)
		}
		slices.Sort(extra)
		for _, finish := range extra {
			add(co.FoilUUIDs[finish])
		}
		// A card without a registered map is its own only finish
		add(co.UUID)
	}
	addFinishes(co)

	// A twin is a whole card entry, so its registered finishes come along:
	// entering from the etched twin must still surface the base's foil.
	for _, variation := range co.Variations {
		altCo, found := b.UUIDs[variation]
		if !found {
			continue
		}
		if finishTwins(co, altCo) {
			addFinishes(altCo)
		}
	}
	return siblings
}

// FinishSiblings answers every uuid the card behind the id is sold under,
// against the default datastore. See the method.
func FinishSiblings(inputID string) []string {
	return defaultBackend.FinishSiblings(inputID)
}

// Match resolves a storefront's description of a card to the uuid of the one
// printing it names, using the default datastore. See the method.
func Match(inCard *InputCard) (cardID string, err error) {
	return defaultBackend.Match(inCard)
}

// MatchInSet returns every printing in the set whose name is exactly the one
// given, against the default datastore. A combined name is matched on its
// first half alone.
func MatchInSet(cardName string, setCode string) (outCards []Card) {
	return defaultBackend.MatchInSet(cardName, setCode)
}

// MatchInSetNumber returns every printing in the set with exactly this name
// and collector number, against the default datastore.
func MatchInSetNumber(cardName, setCode, number string) (outCards []Card) {
	return defaultBackend.MatchInSetNumber(cardName, setCode, number)
}

// MatchWithNumber returns every printing with this set code and collector
// number, against the default datastore. The name only narrows the result and
// may be empty.
func MatchWithNumber(cardName, setCode, number string) (outCards []Card) {
	return defaultBackend.MatchWithNumber(cardName, setCode, number)
}

// cardObject4Id resolves whatever identifier a caller sends - one of the
// matcher's own uuids, or an external product id - to the entry it names.
// The maps decide what an id is: only Magic spells its uuids the way mtgjson
// does and only some games number their products, so an id is looked up as
// it was sent rather than measured against either shape first.
func (b *Backend) cardObject4Id(inputID string) (*CardObject, error) {
	if inputID == "" {
		return nil, ErrCardUnknownID
	}

	// Look up as one of the matcher's own uuids first, then through the id
	// spaces in their fixed order
	co, found := b.UUIDs[inputID]
	for _, space := range idSpaceOrder {
		if found {
			break
		}
		co, found = b.UUIDs[b.ExternalIdentifiers[space][inputID]]
	}
	if !found {
		return nil, ErrCardUnknownID
	}
	return co, nil
}

// FinishUUID resolves a finish name, spelled the way any source spells it, to
// the uuid of the printing's sibling sold in it, and "" when the printing is
// not sold in it at all. The name goes through the game's vocabulary
// (GameRules.CanonicalFinish) and then through the printing's own aliases, so
// a vendor's spelling reaches the same uuid its canonical name does.
func (b *Backend) FinishUUID(card *Card, finish string) string {
	if b.rules == nil {
		return ""
	}
	canonical := b.rules.CanonicalFinish(finish)
	if canonical == "" {
		return ""
	}
	if alias, found := card.FinishAliases[canonical]; found {
		canonical = alias
	}
	return card.FoilUUIDs[canonical]
}

// MatchIDFinish answers an id with the uuid of the printing's sibling sold in
// the named finish, whichever sibling the id itself names: it promotes and
// demotes over the game's whole finish vocabulary the way the flags do over
// the three finishes they can name, so a plain uuid reaches its holofoil and
// a holofoil uuid reaches the plain one.
//
// A finish the printing is not sold in is an error, never another finish's
// uuid: the caller is pricing one sku, and answering with a sibling files two
// sku prices under one uuid, which is the whole point of a uuid per finish.
// It reports a datastore that does not carry a printing the vendor sells,
// where the flag form would have quietly clamped the price onto the finish it
// does carry. Unlike the flag form it answers with the printing's own
// siblings only, not with a foil Magic files as a printing of its own.
func (b *Backend) MatchIDFinish(inputID, finish string) (string, error) {
	co, err := b.cardObject4Id(inputID)
	if err != nil {
		return "", err
	}
	// A sealed product is one product in no finish at all, so a caller
	// naming one is describing the singles it holds, not the box.
	if co.Sealed {
		return co.UUID, nil
	}
	if b.rules == nil {
		return "", ErrDatastoreEmpty
	}
	if b.rules.CanonicalFinish(finish) == "" {
		Logger.Printf("Finish %q is not one this game names", finish)
		return "", ErrCardUnnamedFinish
	}
	outID := b.FinishUUID(&co.Card, finish)
	if outID == "" {
		canonical := b.rules.CanonicalFinish(finish)
		if !b.knownFinishes[canonical] {
			Logger.Printf("Finish %q is not one this datastore sells", finish)
			return "", ErrCardUnnamedFinish
		}
		Logger.Printf("Printing %s is not sold in finish %q", co.UUID, finish)
		return "", ErrCardWrongFinish
	}
	// Validate that what we found is correct
	if _, found := b.UUIDs[outID]; !found {
		return "", ErrCardUnknownID
	}
	return outID, nil
}

// matchIDFor answers an id the way the caller asked about it. The two forms
// differ in reach, not just in spelling: the flags may land on a foil Magic
// files as a printing of its own, while a named finish stays among the
// printing's own siblings, which is what lets it be loud about a finish the
// printing is not sold in.
func (b *Backend) matchIDFor(inCard *InputCard) (string, error) {
	if inCard.Finish != "" {
		return b.MatchIDFinish(inCard.ID, inCard.Finish)
	}
	return b.MatchID(inCard.ID, inCard.Foil, inCard.IsEtched())
}

// MatchID resolves an identifier a storefront already knows, one of the
// matcher's uuids or an external product id, to the uuid of a printing. The
// optional flags ask for the foil or etched sibling, and are answered only
// where the printing was sold in one.
func (b *Backend) MatchID(inputID string, finishes ...bool) (string, error) {
	co, err := b.cardObject4Id(inputID)
	if err != nil {
		return "", err
	}

	isEtched := len(finishes) > 1 && finishes[1]
	isFoil := len(finishes) > 0 && finishes[0] && !isEtched

	// If the loaded card already matches the requested finishes
	// return the found id straight away
	if (co.Foil && isFoil) || (co.Etched && isEtched) ||
		(!co.Foil && !co.Etched && !isFoil && !isEtched) {
		return co.UUID, nil
	}

	outID := b.output(co.Card, finishes...)

	// Validate that what we found is correct
	co, found := b.UUIDs[outID]
	if !found {
		return "", ErrCardUnknownID
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
			if !finishTwins(co, altCo) {
				continue
			}
			maybeID := b.output(altCo.Card, isFoil, isEtched)
			altCo, found = b.UUIDs[maybeID]
			if !found {
				continue
			}

			// If the alt card finish matches the expected one
			// then replace the final output uuid
			if altCo.Foil == isFoil && altCo.Etched == isEtched {
				outID = maybeID
				break
			}
		}
	}
	return outID, nil
}

// Match resolves a storefront's description of a card to the uuid of the one
// printing it names, reporting ErrAliasing when the description fits more than
// one. The input is normalized in place, so a caller can see what the matcher
// made of it.
func (b *Backend) Match(inCard *InputCard) (cardID string, err error) {
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
			for field := range strings.FieldsSeq(inCard.Language) {
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
	if inCard.ID != "" {
		Logger.Printf("Performing id lookup")
		outID, err := b.matchIDFor(inCard)
		// The wording cannot improve on a finish the printing does not
		// carry: it would answer from the same printing, and the only
		// answer it has is another finish's uuid. A name the game could not
		// place says nothing about the printing, so that one falls through.
		if errors.Is(err, ErrCardWrongFinish) {
			return "", err
		}
		if err == nil {
			co := b.UUIDs[outID]
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
			// Actually found id
			default:
				return outID, nil
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
	// A token carrying no Token in its name is filed under a key that says
	// it, leaving the plain one to whatever card normalizes the same way -
	// the Unsanctioned "Bat-" answers for "Bat". An edition that files
	// tokens is asking for the token, so let its key answer first; so is a
	// variation that says the word under a plain edition name, or the card
	// sharing the bucket would answer for the token beside it.
	var viaTokenKey bool
	if tokenName, ok := b.CanonicalNames[Normalize(inCard.Name)+"token"]; ok &&
		(b.editionFilesTokens(inCard.Edition) || Contains(inCard.Variation, "Token")) {
		canonicalName, found, viaTokenKey = tokenName, true, true
	}
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
	// For any unsupported set that wasn't processed previously
	case inCard.Contains("Oversize") && !b.hasOversizedPrinting(inCard.Name):
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
	// A name answered by the token key never passed through AdjustName, which
	// is what would have suffixed it and asked for the filter below. Ask for
	// it here instead, or a token carrying a single printing would be served
	// for whatever edition the listing named.
	if len(printings) > 1 || viaTokenKey || strings.HasSuffix(ogName, "Token") {
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

		cardID = b.output(outCards[0], inCard.Foil, inCard.IsEtched())

		co := b.UUIDs[cardID]
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
	}

	return
}

// MatchInSet returns every printing in the set whose name is exactly the one
// given. A combined name is matched on its first half alone.
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

// MatchInSetNumber returns every printing in the set with exactly this name
// and collector number.
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

// MatchWithNumber returns every printing with this set code and collector
// number. The name only narrows the result and may be empty.
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
