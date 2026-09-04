package lorcana

import (
	"maps"
	"regexp"
	"slices"
	"strconv"
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
	// A name that is itself canonical stays whole: two catalog names carry
	// their qualifier inside them - the Errata Version of Bucky and of Elsa,
	// Gloves Off - and splitting those looked up the bare name instead, so
	// the errata printing was unreachable even when a listing spelled it out
	// exactly and answered with the original standing at the same number.
	_, canonical := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]
	if !canonical && strings.Contains(inCard.Name, "(") {
		vars := mtgmatcher.SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.AddToVariant(strings.Join(vars[1:], " "))
		}
	}
	adoptQualifiedName(b, inCard)
}

// qualifiedNameRe matches the trailing parenthetical the catalog keeps inside
// a name instead of distilling it into a label of its own.
var qualifiedNameRe = regexp.MustCompile(`\s*\(([^()]*)\)$`)

// adoptQualifiedName adopts the catalog's decorated spelling of a name the
// storefront wrote bare, where the wording says which decoration it means.
//
// Candidates are gathered by name, so a listing writing the bare name can
// never reach a printing the catalog files with its qualifier inside it. The
// Errata Version of Bucky, Squirrel Squeak Tutor stands at the same number as
// the original, and Cool Stuff Inc sells both under one name, telling them
// apart in a note - "3-Cost Errata, Foil No Ward" against "2-Cost w/ Ward".
// The errata row answered with the original and was served as its price.
//
// The qualifier is what licenses the swap, and three things keep it narrow.
// The number must be written, so the set and the printing are named rather
// than guessed. The wording must state a word of the qualifier, so a listing
// silent about it keeps the bare name the catalog also holds at that number.
// And two decorated siblings state nothing between them, so they refuse
// instead of choosing.
func adoptQualifiedName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	number := extractNumber(inCard.Variation)
	if number == "" {
		return
	}
	set, err := b.GetSetByName(inCard.Edition)
	if err != nil || set == nil {
		return
	}

	wording := strings.ToLower(inCard.Variation)
	var adopt string
	for i := range set.Cards {
		card := &set.Cards[i]
		match := qualifiedNameRe.FindStringSubmatchIndex(card.Name)
		if match == nil || card.Number != number ||
			!mtgmatcher.Equals(card.Name[:match[0]], inCard.Name) {
			continue
		}
		if !statesQualifier(wording, card.Name[match[2]:match[3]]) {
			continue
		}
		if adopt != "" && !mtgmatcher.Equals(adopt, card.Name) {
			return
		}
		adopt = card.Name
	}
	if adopt != "" {
		inCard.Name = adopt
	}
}

// statesQualifier reports whether the wording names the qualifier, which is
// any word of it the wording repeats. The short words are skipped: those are
// the ones a note carries for reasons of its own.
func statesQualifier(wording, qualifier string) bool {
	for _, word := range strings.Fields(strings.ToLower(qualifier)) {
		if len(word) < 4 {
			continue
		}
		if strings.Contains(wording, word) {
			return true
		}
	}
	return false
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
	var match, matchNorm string
	for _, uuid := range b.AllUUIDs {
		co, err := b.GetUUID(uuid)
		if err != nil || co.Sealed || co.SetCode != setCode || co.Number != number {
			continue
		}
		norm := mtgmatcher.Normalize(co.Name)
		if norm == matchNorm || !mtgmatcher.CloseName(inCard.Name, co.Name) {
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

// AdjustEdition normalizes scraper edition strings toward LorcanaJSON set
// names: storefronts commonly prefix the game name ("Disney Lorcana: Rise of
// the Floodborn") or append catalog suffixes ("... Singles"). An edition that
// still matches no set name simply does not narrow the candidates (the Match
// skeleton falls back to every printing), so trimming can only help.
func (r Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	inCard.Edition = r.AliasEdition(b, inCard.Edition)
}

// AliasEdition spells an edition string toward a set name using the string
// alone. See mtgmatcher.GameRules.
func (Rules) AliasEdition(b *mtgmatcher.Backend, edition string) string {
	edition = trimEdition(edition)
	if mtgmatcher.IsPromoHeading(edition) {
		edition = ""
	}
	return edition
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
// whole sets), the story inserts a set packs beside them, and the multi-card
// promo lots. No Lorcana card carries any of those wordings and none ever
// will, since none of them is a card.
//
// Both tests are literal and read the name alone. Normalize erases every "s",
// which turns the lots' "Set of" into "etof" — a substring of "The Queen -
// Cruelest of All" and four more real names — so the normalized Contains is
// too lossy here. And the promotion behind the lot also names a real card's
// variant, "Mickey Mouse - True Friend (Disney Cruise Promo)", which the
// prefilter leaves in the variation: anchoring at the start of the name keeps
// the rule off every listing that merely mentions the promotion.
//
// The story inserts are named for the set that packs them and end on the
// word - "Reign of Jafar - Lore Story Insert", "Azurite Sea Insert" - so the
// suffix is what reads them all without naming each set. No printing the
// datastore carries ends there: the puzzle inserts it does carry spell which
// piece they are after it, "Puzzle Insert (Top Left)".
func (Rules) IsUnsupported(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	return strings.Contains(inCard.Name, "Puzzle Insert") ||
		strings.HasSuffix(inCard.Name, "Insert") ||
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
	pool := extractPool(inCard.Variation)

	var out, wrongFinish, bareOut, bareWrongFinish, chaseOut, chaseWrongFinish []mtgmatcher.Card
	chased := false
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		// Every finish of a printing is stored under a uuid of its own,
		// suffixed with the finish it carries; fold them back onto the
		// printing so each candidate appears exactly once. Base uuids are
		// numeric, so the first underscore marks the start of the suffix.
		base := uuid
		if idx := strings.IndexByte(uuid, '_'); idx >= 0 {
			base = uuid[:idx]
		}
		if seen[base] {
			continue
		}
		seen[base] = true

		// A printing sold in no nonfoil has nothing under the bare uuid -
		// every finish it has is suffixed - so the entry that folded here
		// stands for it.
		co, found := b.UUIDs[base]
		if !found {
			co, found = b.UUIDs[uuid]
		}
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
		bareFits := !exact && bare != "" && bare == card.Number
		// A named chase tier is a claim about the printing, and it holds
		// wherever the number does not: a storefront that writes one down
		// has said which of two same-named printings the listing is even
		// when the number beside it reaches neither.
		chase := namesChaseRarity(inCard.Variation, card.Rarity)
		chased = chased || chase
		if !exact && !bareFits && !chase {
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
		case bareFits && finishFits:
			bareOut = append(bareOut, card)
		case bareFits:
			bareWrongFinish = append(bareWrongFinish, card)
		case finishFits:
			chaseOut = append(chaseOut, card)
		default:
			chaseWrongFinish = append(chaseWrongFinish, card)
		}
	}
	// A feed that never says "foil" must not lose a foil-only card outright:
	// the promotional printings are sold foil only, and a storefront listing
	// them plain had every one of them deleted here. Saying nothing is not
	// the same claim as saying nonfoil, which is why only this direction
	// falls back. The number as written and the plain finish both still
	// decide whenever they have candidates to choose between, so each
	// fallback only ever answers where the alternative was answering nothing.
	tiers := [][]mtgmatcher.Card{out, wrongFinish, bareOut, bareWrongFinish, chaseOut, chaseWrongFinish}
	// Once a tier is named by a printing that carries it, the printings that
	// do not are contradicted rather than merely unmentioned, and go out of
	// the tiers a number did not decide: the letter a storefront hangs off a
	// chase card's number is the one thing standing between the stripped
	// form and the plain card it numbers, and stripping it is a guess where
	// the named tier is a statement.
	//
	// A number that was written and landed exactly is the more specific
	// claim of the two and keeps its tiers, so a wording that names the tier
	// beside a number naming the printing outright never overrules it. A
	// number that was not written at all decided nothing, and there the
	// wording is all the listing said.
	first := 0
	if number != "" {
		first = 2
	}
	if chased {
		for i := first; i < len(tiers); i++ {
			tiers[i] = keepChaseRarity(inCard.Variation, tiers[i])
		}
	}
	for _, tier := range tiers {
		if len(tier) > 0 {
			return poolTiebreak(pool, tier)
		}
	}
	return nil
}

// chaseRarities are the tiers Lorcana keeps for the alternate art of a card
// already printed in the set, as the datastore spells them. They are the only
// rarities read: the rest of the ladder is never how a storefront tells two
// printings of one name apart, and "Promo" is a word half the promo wordings
// carry anyway.
var chaseRarities = []string{"enchanted", "epic", "iconic"}

// namesChaseRarity reports whether the variation claims the printing is the
// chase tier given. "Alternate Art" is the nickname a storefront writes for
// the same claim, and naming a tier no candidate carries reaches nothing, so
// a nickname read onto the wrong one of the three costs no printing.
func namesChaseRarity(variation, rarity string) bool {
	if !slices.Contains(chaseRarities, rarity) {
		return false
	}
	return spellsOut(variation, rarity) || spellsOut(variation, "alternate art")
}

// keepChaseRarity drops the printings whose tier the variation contradicts.
func keepChaseRarity(variation string, cards []mtgmatcher.Card) []mtgmatcher.Card {
	var kept []mtgmatcher.Card
	for _, card := range cards {
		if namesChaseRarity(variation, card.Rarity) {
			kept = append(kept, card)
		}
	}
	return kept
}

// spellsOut reports whether the text writes the phrase as consecutive whole
// words, so a tier spelled inside a longer word is not read as the tier.
func spellsOut(text, phrase string) bool {
	words := strings.Fields(strings.ToLower(text))
	wanted := strings.Fields(phrase)
	for i := 0; i+len(wanted) <= len(words); i++ {
		if slices.Equal(words[i:i+len(wanted)], wanted) {
			return true
		}
	}
	return false
}

// poolTiebreak narrows a tier to the printings numbered within the pool the
// storefront wrote behind the number.
//
// The datastore numbers each promo pool from one, so a card promoted twice
// carries the same number in both: "Maleficent - Monstrous Dragon" is card 5
// of the P1 pool and card 5 of the P3 one, and the number alone cannot tell
// the two apart. The pool can, and the storefront writes it where a set card
// writes its set size.
//
// A pool no candidate carries keeps the whole tier. The storefront's spelling
// of a pool is its own - it prints the one the card came from, which is not
// always the one the datastore numbered it in - and refusing a printing over
// it would price nothing where the number alone was answering.
func poolTiebreak(pool string, cards []mtgmatcher.Card) []mtgmatcher.Card {
	if pool == "" || len(cards) <= 1 {
		return cards
	}
	var pooled []mtgmatcher.Card
	for _, card := range cards {
		if slices.Contains(card.PromoTypes, pool) {
			pooled = append(pooled, card)
		}
	}
	if len(pooled) == 0 {
		return cards
	}
	return pooled
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

// extractPool pulls the promo pool out of the number a storefront wrote.
//
// A Lorcana number is written over what it is one of: "87/204" for the
// eighty-seventh of a set of two hundred and four, "5/P3" for the fifth card
// of the third promo pool. Only a tail that is not itself a count names a
// pool, and it is read as the token the pool's tag is stored under.
func extractPool(variation string) string {
	for field := range strings.FieldsSeq(variation) {
		if field[0] < '0' || field[0] > '9' {
			continue
		}
		_, tail, found := strings.Cut(strings.TrimRight(field, ","), "/")
		if !found || tail == "" {
			return ""
		}
		if _, err := strconv.Atoi(tail); err == nil {
			return ""
		}
		return mtgmatcher.PromoTypeSlug(tail)
	}
	return ""
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
