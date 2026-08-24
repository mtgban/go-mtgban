package lorcana

import (
	"maps"
	"slices"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for Lorcana. Lorcana has no edition
// aliases, variant tables, or promo types: a card is identified by name +
// collector number + foil. So most hooks are no-ops and the real work is the
// number disambiguation in FilterCards (foil is honored downstream by output).
type Rules struct{ mtgmatcher.DefaultRules }

// Prefilter splits a trailing parenthetical variant off the name before the
// canonical-name lookup. Unlike Magic it leaves " - " intact, since Lorcana
// names are "Character - Title".
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if strings.Contains(inCard.Name, "(") {
		vars := mtgmatcher.SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.AddToVariant(strings.Join(vars[1:], " "))
		}
	}
}

// AdjustName provides a prefix fallback: scraper feeds sometimes truncate the
// "Character - Title" name. When the exact name is unknown, scan for cards
// whose name has the input as a prefix and let the collector number and finish
// narrow them; adopt the name only when exactly one survives. If several
// distinct names survive, the input stays unresolved and Match reports an
// unknown name (without a single name there is nothing to hand the pipeline).
func (Rules) AdjustName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	number := extractNumber(inCard.Variation)
	// A name no printing opens with has no prefix matches to tier, and the
	// number is all that is left to read it by.
	uuids, err := b.SearchHasPrefix(inCard.Name)
	if err != nil {
		uuids = nil
	}

	var fits, wrongFinish []string
	for _, uuid := range uuids {
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		// A sealed product shares the name buckets but is never the card
		// the truncated name is reaching for.
		if co.Sealed {
			continue
		}
		if number != "" && number != co.Number {
			continue
		}
		// A foil claim is a claim, and a printing sold in no foil cannot
		// answer it: the flag stands. Saying nothing is not the same claim,
		// so an unflagged input only sets a foil-only name aside.
		if inCard.Foil && !co.HasFinish(mtgmatcher.FinishFoil) {
			continue
		}
		if !inCard.Foil && !co.HasFinish(mtgmatcher.FinishNonfoil) {
			if !slices.Contains(wrongFinish, co.Name) {
				wrongFinish = append(wrongFinish, co.Name)
			}
			continue
		}
		if !slices.Contains(fits, co.Name) {
			fits = append(fits, co.Name)
		}
	}
	// The first tier holding anything decides, the shape FilterCards uses:
	// the names set aside only ever answer where the plain finish had
	// nothing to offer. Several names in that tier are genuinely ambiguous
	// and stay unresolved rather than falling through to a tie broken by
	// finish, which would answer with a name the tier itself often holds.
	for _, tier := range [][]string{fits, wrongFinish} {
		if len(tier) == 0 {
			continue
		}
		if len(tier) == 1 {
			inCard.Name = tier[0]
		}
		return
	}

	if name := nameAtNumber(b, inCard, number); name != "" {
		inCard.Name = name
	}
}

// nameAtNumber returns the datastore's spelling of the card a storefront
// misspelled, read off the set and collector number it wrote beside the name.
//
// A storefront that cannot spell a name still files the listing in a set and
// numbers it, and a number names one card of a set. Coolstuffinc doubles a
// letter that is single and singles one that is double ("Metalic Leader",
// "Gepetto"), drops one altogether ("Somone Will Lose His Head", "Valourous
// General") or swaps two ("Ambitious Entreperneur"), and every one of those
// listings was dropped with the number that says which card it is sitting in
// the same record.
//
// Two things gate it. The edition has to name one set, since a number is only
// an identifier within one. And exactly one of the cards wearing that number
// there has to be close to the misspelling - a piece missing off one end, or
// up to two letters wrong - because the number is the storefront's own claim
// and is sometimes another card's, so without the closeness test a stray
// number would rename a card into whatever it pointed at.
//
// A number is not an identifier on its own even inside a set: the promo pools
// are filed under the set code of the set they were handed out alongside and
// numbered from one within the pool, so the first set holds three cards
// numbered 2. Closeness is what tells them apart, and demanding it pick one
// is what keeps a stray number from reaching the other two.
func nameAtNumber(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, number string) string {
	if number == "" {
		return ""
	}
	setCode := soleSet(b, inCard.Edition)
	if setCode == "" {
		return ""
	}
	got := mtgmatcher.Normalize(inCard.Name)
	var match, matchNorm string
	for _, uuid := range b.AllUUIDs {
		co, err := b.GetUUID(uuid)
		if err != nil || co.Sealed || co.SetCode != setCode || co.Number != number {
			continue
		}
		norm := mtgmatcher.Normalize(co.Name)
		if norm == matchNorm || !closeName(got, norm) {
			continue
		}
		if match != "" {
			return ""
		}
		match, matchNorm = co.Name, norm
	}
	return match
}

// soleSet returns the code of the one set an edition names, or "" where none
// or several answer. The name is read the way Match reads it, whole or as a
// part, with the decorations a storefront hangs off it trimmed first.
func soleSet(b *mtgmatcher.Backend, edition string) string {
	edition = trimEdition(edition)
	if mtgmatcher.Normalize(edition) == "" {
		return ""
	}
	var found string
	for code, set := range b.Sets {
		if !mtgmatcher.Contains(set.Name, edition) {
			continue
		}
		if found != "" {
			return ""
		}
		found = code
	}
	return found
}

// closeName reports whether a storefront's normalized name is the datastore's
// own with a piece missing off one end or up to two letters wrong.
func closeName(got, want string) bool {
	if got == "" || want == "" {
		return false
	}
	if strings.HasPrefix(want, got) || strings.HasSuffix(want, got) {
		return true
	}
	return editDistance(got, want, 2) <= 2
}

// editDistance is the Levenshtein distance between two strings, giving up at
// limit: the caller only cares whether the two are within a couple of edits,
// and a name pair that is not stops being measured once the whole row of the
// table is past the limit.
func editDistance(a, b string, limit int) int {
	ar, br := []rune(a), []rune(b)
	if len(ar) > len(br) {
		ar, br = br, ar
	}
	if len(br)-len(ar) > limit {
		return limit + 1
	}
	prev := make([]int, len(ar)+1)
	cur := make([]int, len(ar)+1)
	for i := range prev {
		prev[i] = i
	}
	for j := 1; j <= len(br); j++ {
		cur[0] = j
		best := cur[0]
		for i := 1; i <= len(ar); i++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			cur[i] = min(prev[i]+1, cur[i-1]+1, prev[i-1]+cost)
			best = min(best, cur[i])
		}
		if best > limit {
			return limit + 1
		}
		prev, cur = cur, prev
	}
	return prev[len(ar)]
}

// AdjustEdition normalizes scraper edition strings toward LorcanaJSON set
// names: storefronts commonly prefix the game name ("Disney Lorcana: Rise of
// the Floodborn") or append catalog suffixes ("... Singles"). An edition that
// still matches no set name simply does not narrow the candidates (the Match
// skeleton falls back to every printing), so trimming can only help.
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := trimEdition(inCard.Edition)
	if mtgmatcher.IsPromoHeading(edition) {
		edition = ""
	}
	inCard.Edition = edition
}

// trimEdition strips the game name a storefront prefixes a set with and the
// catalog suffix it appends.
func trimEdition(edition string) string {
	edition = strings.TrimSpace(edition)
	for _, prefix := range []string{"Disney Lorcana", "Lorcana"} {
		if strings.HasPrefix(edition, prefix) {
			edition = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(edition, prefix), ":-"))
			break
		}
	}
	return strings.TrimSpace(strings.TrimSuffix(edition, "Singles"))
}

// IsUnsupported drops the non-card products TCGplayer files under its "Cards"
// product type, so they are skipped instead of reported as unknown names: the
// puzzle-piece inserts bundled with booster displays (sold per piece and as
// whole sets) and the multi-card promo lots. No Lorcana card carries either
// wording and none ever will, since neither is a card.
//
// Both tests are literal and read the name alone. Normalize erases every "s",
// which turns the lots' "Set of" into "etof" — a substring of "The Queen -
// Cruelest of All" and four more real names — so the normalized Contains is
// too lossy here. And the promotion behind the lot also names a real card's
// variant, "Mickey Mouse - True Friend (Disney Cruise Promo)", which the
// prefilter leaves in the variation: anchoring at the start of the name keeps
// the rule off every listing that merely mentions the promotion.
func (Rules) IsUnsupported(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	return strings.Contains(inCard.Name, "Puzzle Insert") ||
		strings.HasPrefix(inCard.Name, "Disney Cruise Promos")
}

// CanonicalFinish owns Lorcana's finish vocabulary. Lorcana's finish names
// are data rather than a fixed list - LorcanaJSON gives every printing the
// foil types it is sold in, and new ones keep arriving (Silver, Satin, Magma,
// FreeForm1, RainbowPillars, …) - so an unrecognized name is normalized and
// handed back rather than refused, and the lookup against the printing's own
// finishes is what decides. The named cases are the spellings that are not a
// foil type: LorcanaJSON's placeholder for a plain printing, and the name
// TCGplayer prices the standard foil under.
func (Rules) CanonicalFinish(name string) string {
	return canonicalFinish(name)
}

func canonicalFinish(name string) string {
	normalized := mtgmatcher.NormalizeFinish(name)
	switch normalized {
	case "none":
		return mtgmatcher.FinishNonfoil
	// Every Lorcana foil is a cold foil, whichever foil type the printing
	// is sold in, so the name names the printing's standard foil rather
	// than a type of its own
	case "coldfoil":
		return mtgmatcher.FinishFoil
	}
	if finish := mtgmatcher.CanonicalFinish(name); finish != "" {
		return finish
	}
	return normalized
}

// FilterCards narrows candidates by edition, collector number, and finish:
// candidates come from the name hash rather than the edition-keyed cardSet
// values, so case-variant spellings that normalize to the same canonical name
// stay reachable (three real pairs exist in the data) and iteration follows
// stable load order instead of random map order. The cardSet keys still
// matter: the Match skeleton fills them with the sets matching the input
// edition when one was supplied and resolves — falling back to every printing
// otherwise — so honoring them disambiguates a name+number shared across sets
// ("Let It Go" #163) while a missing or unrecognized edition changes nothing.
func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	number := extractNumber(inCard.Variation)
	// A letter hung off the end of the number may be the storefront's own
	// promo-series marker - Strikezone numbers its promos "010B", "003C" -
	// or the letter the data itself gives a same-numbered sibling ("4a"
	// through "4e"). Only the number as written can tell which, so the
	// stripped form is kept as a fallback and never as a substitute.
	bare := strings.TrimRight(number, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if bare == number {
		bare = ""
	}

	var out, wrongFinish, bareOut, bareWrongFinish []mtgmatcher.Card
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		// Foil printings (the primary "_f" and every foil sub-type suffix) are
		// stored under a suffixed uuid; fold them back onto the base card so
		// each candidate appears exactly once. Base uuids are numeric, so the
		// first underscore marks the start of any finish suffix.
		if idx := strings.IndexByte(uuid, '_'); idx >= 0 {
			uuid = uuid[:idx]
		}
		if seen[uuid] {
			continue
		}
		seen[uuid] = true

		co, found := b.UUIDs[uuid]
		if !found {
			continue
		}
		// Sealed products share the name buckets but never match as
		// cards; without this a sealed product named like a card would
		// read as an aliased printing of it
		if co.Sealed {
			continue
		}
		card := co.Card

		if _, found := cardSet[card.SetCode]; !found {
			continue
		}
		exact := number == "" || number == card.Number
		if !exact && (bare == "" || bare != card.Number) {
			continue
		}
		// Set aside the candidates sold in no plain finish. A card with both
		// passes either way; output() picks the uuid.
		finishFits := card.HasFinish(mtgmatcher.FinishNonfoil)
		if inCard.Foil {
			// A foil claim is a claim, and a printing sold in no foil
			// cannot answer it: the flag stands.
			if !card.HasFinish(mtgmatcher.FinishFoil) {
				continue
			}
			finishFits = true
		}
		// A listing naming a foil sub-type re-keys the copy's FoilUUIDs so
		// the flag-driven resolution downstream lands on that sub-type's uuid
		// instead of the primary foil's.
		if uuid := selectFinish(b, inCard, &card); uuid != "" {
			foilUUIDs := maps.Clone(card.FoilUUIDs)
			foilUUIDs[mtgmatcher.FinishFoil] = uuid
			card.FoilUUIDs = foilUUIDs
		}
		switch {
		case exact && finishFits:
			out = append(out, card)
		case exact:
			wrongFinish = append(wrongFinish, card)
		case finishFits:
			bareOut = append(bareOut, card)
		default:
			bareWrongFinish = append(bareWrongFinish, card)
		}
	}
	// A feed that never says "foil" must not lose a foil-only card outright:
	// the promotional printings are sold foil only, and a storefront listing
	// them plain had every one of them deleted here. Saying nothing is not
	// the same claim as saying nonfoil, which is why only this direction
	// falls back. The number as written and the plain finish both still
	// decide whenever they have candidates to choose between, so each
	// fallback only ever answers where the alternative was answering nothing.
	for _, tier := range [][]mtgmatcher.Card{out, wrongFinish, bareOut, bareWrongFinish} {
		if len(tier) > 0 {
			return tier
		}
	}
	return nil
}

// extractNumber pulls the collector number out of the scraper-supplied
// Variation. Core Match may append parenthetical chunks split off the input
// name ("205 Enchanted", or just "Enchanted" when no number was supplied),
// so only the first digit-leading field counts. The number is the part
// before '/' with leading zeros stripped — except a number the zeros are
// the whole of stays "0", with any letter it carries, so the genuine
// 0-numbered promo stays reachable.
func extractNumber(variation string) string {
	number := ""
	for field := range strings.FieldsSeq(variation) {
		if field[0] >= '0' && field[0] <= '9' {
			number = field
			break
		}
	}
	number = strings.Split(number, "/")[0]
	trimmed := strings.TrimLeft(number, "0")
	// TrimLeft stops at the first non-zero character of any kind, so a result
	// that is empty or no longer leads with a digit means the zeros were the
	// whole numeric part and one has to come back: "000B" is the 0-numbered
	// card written the way a storefront hangs its promo-series marker off a
	// number, not a number named "B".
	if number != "" && (trimmed == "" || trimmed[0] < '0' || trimmed[0] > '9') {
		trimmed = "0" + trimmed
	}
	return trimmed
}

// selectFinish maps the foil a listing names onto the uuid of the printing
// sold in it, so a sub-typed printing resolves to its own uuid instead of
// folding onto the primary foil. The caller's own finish answers first; a
// storefront that sends none still spells the sub-type in its wording, and
// the names to look for there are the ones the printing carries - its stored
// finishes and the vendor spellings the loader registered beside them, which
// is where "Holofoil means this printing's special treatment" lives. Anything
// else, a nonfoil included, keeps the flag-driven resolution.
func selectFinish(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, card *mtgmatcher.Card) string {
	if inCard.Finish != "" {
		uuid := b.FinishUUID(card, inCard.Finish)
		if uuid != "" && uuid != card.FoilUUIDs[mtgmatcher.FinishNonfoil] {
			return uuid
		}
	}

	// One wording can hold two of the names and map iteration is random, so
	// both sets are visited in sorted order, the printing's own finishes
	// before the spellings that only reach them.
	variation := mtgmatcher.NormalizeFinish(inCard.Variation)
	for _, finish := range slices.Sorted(maps.Keys(card.FoilUUIDs)) {
		if finish == mtgmatcher.FinishNonfoil || finish == mtgmatcher.FinishFoil {
			continue
		}
		if strings.Contains(variation, finish) {
			return card.FoilUUIDs[finish]
		}
	}
	for _, alias := range slices.Sorted(maps.Keys(card.FinishAliases)) {
		if strings.Contains(variation, alias) {
			return card.FoilUUIDs[card.FinishAliases[alias]]
		}
	}
	return ""
}
