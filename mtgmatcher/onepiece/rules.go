package onepiece

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for One Piece. A card is identified
// by name + collector number, with the variant label separating the
// printings that share a number (alternate arts, parallels, event
// printings). Foil never gates anything: a product is one printing whatever
// its stamping, and both flag values resolve to the same uuid.
type Rules struct{ mtgmatcher.DefaultRules }

// fullNumberRe matches the game's collector number shapes: "OP01-001",
// "ST01-001", "EB01-023", "P-001", with an optional letter tail
// (cardtrader suffixes alternate arts "OP01-001a").
var fullNumberRe = regexp.MustCompile(`^[A-Za-z]+[0-9]*-[0-9]+[a-zA-Z]*$`)

// dashNumberRe matches the collector number hung inside a name after a
// dash ("Monkey.D.Luffy - P-043").
var dashNumberRe = regexp.MustCompile(`\s+-\s+([A-Za-z]+[0-9]*-[0-9]+[a-zA-Z]*)\b`)

// dashTailRe matches the bare number coolstuffinc hangs off a name in
// place of the full code ("Trafalgar Law - 008"). It only matches at the
// end: two cards are named with a dash of their own, and no card name ends
// in one followed by a number.
var dashTailRe = regexp.MustCompile(`\s+-\s+([0-9]+[a-zA-Z]?)$`)

// trailingCodeRe matches the collector number a storefront writes behind a
// name in parentheses of its own ("Gloriosa (Grandma Nyon) (OP07-041)"). It
// only matches at the end, and only a full code: a parenthetical anywhere
// else may be part of the name.
var trailingCodeRe = regexp.MustCompile(`\s*\(([A-Za-z]+[0-9]*-[0-9]+[a-zA-Z]*)\)$`)

// Prefilter splits the parenthetical decorations off the name before the
// canonical-name lookup: storefronts write "Roronoa Zoro (OP01-001) (V.2)",
// "Shanks (001) (Parallel)" and "Monkey.D.Luffy - P-043 (Convention Promo
// 2024)". A full name that is itself canonical stays as it is — the epithet
// parentheticals ("Mr.2.Bon.Kurei (Bentham)") are part of the name.
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	splitDecorations(b, inCard)
	adoptNumberedName(b, inCard)
}

// splitDecorations moves the parenthetical qualifiers and the dash-hung
// collector number out of the name, leaving the bare spelling behind.
func splitDecorations(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	split := splitDashNumber(inCard)
	// A card whose own name ends in a parenthetical wears two of them once
	// the storefront writes the number behind it, and the split below cannot
	// tell which is which: it keeps the head and sweeps the rest into the
	// variation, so the epithet goes with the number and what is left is
	// another card's name ("Gloriosa" is OP17-046, not OP07-041). The number
	// is the one parenthetical never part of a name, so peel that alone and
	// ask again, and leave the name untouched unless the answer is yes.
	if fields := trailingCodeRe.FindStringSubmatch(inCard.Name); fields != nil {
		peeled := strings.TrimSuffix(inCard.Name, fields[0])
		if _, found := b.CanonicalNames[mtgmatcher.Normalize(peeled)]; found {
			inCard.Name = peeled
			inCard.AddToVariant(fields[1])
			return
		}
	}
	if strings.Contains(inCard.Name, "(") {
		vars := mtgmatcher.SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.AddToVariant(strings.Join(vars[1:], " "))
		}
	}
	// The same feed writes its qualifiers behind the number ("Trafalgar Law
	// - 008 (Parallel)"), where the bare tail is no longer at the end of the
	// name to be read; the parentheticals are off by now, so try again, on
	// the same terms - a name that is itself canonical is left alone.
	if split {
		return
	}
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	splitDashNumber(inCard)
}

// adoptNumberedName adopts the catalog's decorated spelling of a name the
// storefront wrote plain, where the number says the plain spelling is not
// the one meant.
//
// A handful of printings carry their occasion inside the name - both P-110
// printings are "Monkey.D.Luffy (4th Anniversary)", and the pre-release
// tournament cards read the same way - while the storefront writes the bare
// "Monkey.D.Luffy". That bare name is perfectly canonical, and 270 other
// printings answer to it, so the lookup succeeds and the number then deletes
// every one of them: the listing dies as an unknown variant with nothing
// having gone wrong with its variant. AdjustName's own reading of the number
// never runs, because it is only asked once the canonical lookup has failed.
//
// nameAtNumber does the reading, and its guards are what make it safe here: a
// code several names answer says nothing about which was meant, and the
// spelling adopted has to be close to the one written. A name with a printing
// at the number is already right, so it is never asked about.
func adoptNumberedName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; !found {
		return
	}
	number := extractNumber(inCard.Variation)
	if number == "" {
		return
	}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		if numberMatches(number, co.Number) {
			return
		}
	}
	if name := nameAtNumber(b, inCard.Name, number); name != "" {
		inCard.Name = name
	}
}

// splitDashNumber moves the collector number a storefront hangs off a name
// after a dash into the variation, reporting whether it found one. The
// number goes in front of whatever the variation already holds: the retry
// runs with the parenthetical qualifiers moved there, and those carry
// ordinals of their own ("Judge Pack Vol. 5", "1st Anniversary Set") that
// extractNumber's digit-leading fallback would read as the number instead.
func splitDashNumber(inCard *mtgmatcher.InputCard) bool {
	m := dashNumberRe.FindStringSubmatch(inCard.Name)
	if m != nil {
		inCard.Name = strings.Replace(inCard.Name, m[0], "", 1)
		inCard.Variation = strings.TrimSpace(m[1] + " " + inCard.Variation)
		return true
	}
	m = dashTailRe.FindStringSubmatch(inCard.Name)
	if m != nil {
		inCard.Name = strings.TrimSuffix(inCard.Name, m[0])
		inCard.Variation = strings.TrimSpace(m[1] + " " + inCard.Variation)
		return true
	}
	return false
}

// nameAliases maps a storefront's spelling of a card onto the name the
// datastore files it under, keyed by the normalized input name. Only
// spellings no canonical name answers belong here - the table is consulted
// after the canonical lookup has already failed.
var nameAliases = map[string]string{
	// The game's resource card is "DON!! Card" on the card and in the
	// catalog the datastore is built from; several storefronts call it by
	// the shout alone. The prefix fallback below cannot rescue it: every
	// character named Don reads as a prefix match too.
	"don": "DON!! Card",
}

// AdjustName provides a prefix fallback for truncated feeds, adopting the
// one name among the prefix matches that carries the input's number. Names
// compare normalized, so punctuation variants of one name are not read as
// ambiguity.
func (Rules) AdjustName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	if alias, found := nameAliases[mtgmatcher.Normalize(inCard.Name)]; found {
		if _, known := b.CanonicalNames[mtgmatcher.Normalize(alias)]; known {
			inCard.Name = alias
			return
		}
	}
	number := extractNumber(inCard.Variation)
	uuids, err := b.SearchHasPrefix(inCard.Name)
	if err == nil {
		var match, matchNorm string
		for _, uuid := range uuids {
			co, err := b.GetUUID(uuid)
			if err != nil || co.Sealed {
				continue
			}
			if number != "" && !numberMatches(number, co.Number) {
				continue
			}
			norm := mtgmatcher.Normalize(co.Name)
			if match != "" && matchNorm != norm {
				return
			}
			match, matchNorm = co.Name, norm
		}
		if match != "" {
			inCard.Name = match
			return
		}
	}
	if name := nameAtNumber(b, inCard.Name, number); name != "" {
		inCard.Name = name
	}
}

// nameAtNumber returns the datastore's spelling of the card a storefront
// misspelled, read off the collector number it wrote beside the name.
//
// A storefront that cannot spell a name still writes the number, and a full
// code names one printing outright, so the code says which card the listing
// is. Cardmarket types an acute accent where the card has an apostrophe
// ("Who´s.Who"), drops a letter ("Artifical Devil Fruit SMILE"), swaps two
// ("Crone Oli"), or names a character by the last part of their name alone
// ("Garp" for "Monkey.D.Garp").
//
// Two things have to hold before the spelling is adopted. Every printing
// wearing the code has to share one name: a code several names answer says
// nothing about which was meant. And the misspelling has to be close to
// that name - a prefix or a suffix of it, or within two edits, one typo or
// a transposition. The number is the storefront's own claim and it is
// sometimes another card's, so without the closeness test a stray number
// would rename a card into whatever it pointed at.
//
// Only a full code is read. A bare tail ("051") is the same number in every
// set of the game, and the edition that would say which set has not been
// resolved yet when a name is being fixed up.
func nameAtNumber(b *mtgmatcher.Backend, name, number string) string {
	if number == "" || !fullNumberRe.MatchString(number) {
		return ""
	}
	var match, matchNorm string
	for _, uuid := range b.GetUUIDs() {
		co, err := b.GetUUID(uuid)
		if err != nil || co.Sealed || !numberMatches(number, co.Number) {
			continue
		}
		norm := mtgmatcher.Normalize(co.Name)
		if match != "" && matchNorm != norm {
			return ""
		}
		match, matchNorm = co.Name, norm
	}
	if match == "" || !mtgmatcher.CloseName(name, match) {
		return ""
	}
	return match
}

// setCodePrefixRe matches a set code worn as an edition prefix: cardtrader
// spells its expansions "OP-01: Romance Dawn", coolstuffinc "OP03 - Pillars
// of Strength", with the compound codes ("OP15-EB04 - ...") in the same
// shape.
var setCodePrefixRe = regexp.MustCompile(`^[A-Za-z]+-?[0-9]+(?:-[A-Za-z]+[0-9]+)?\s*[-:]\s*`)

// AdjustEdition trims the game-name and set-code prefixes storefronts
// decorate set names with. An edition that still matches no set simply does
// not narrow the candidates.
//
// When the rest of the wording asks for a variant printing, the edition
// stops narrowing them: a variant usually shares its collector number with
// the base card while being filed in another set - the alternate arts in
// PRB-01, the event printings in OP-PR - and storefronts name the base
// card's set for all of them. Narrowing on it would delete the printing
// being asked for before FilterCards ever tiers the candidates, pricing a
// promo as the base common. The edition still reaches the tiering as
// wording, so it goes on describing a variant rather than selecting one.
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := strings.TrimSpace(inCard.Edition)
	for _, prefix := range []string{"One Piece Card Game", "One Piece TCG", "One Piece"} {
		if strings.HasPrefix(edition, prefix) {
			edition = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(edition, prefix), ":-"))
			break
		}
	}
	// The code the prefix spells is the storefront saying which set it
	// means, and it survives the trim: it is the only thing telling the
	// members of a set family apart once the shared name is all that is
	// left of the wording.
	code := strings.Trim(setCodePrefixRe.FindString(edition), " -:")
	edition = setCodePrefixRe.ReplaceAllString(edition, "")
	edition = strings.TrimSpace(strings.TrimSuffix(edition, "Singles"))

	// Storefronts spell a set's name their own way - dropping the ordinal
	// out of "Starter Deck 1: Straw Hat Crew", writing "500 Years into the
	// Future" for the set called "500 Years in the Future" - and a name
	// that is neither equal to nor contained in the datastore's selects
	// nothing at all. Snap it back to the set it describes when one set
	// describes it better than every other, keeping any promo line in
	// front so the event printings are still the ones being named.
	prefix := promoLineRe.FindString(edition)
	canon := canonicalEdition(b, edition[len(prefix):], code)
	if canon != "" {
		edition = prefix + canon
	}
	// A wording abbreviating an event's name is the storefront saying which
	// set the listing is in, however plainly its edition says the other one.
	if promo := promoSetBegun(b, inCard); promo != "" {
		edition = promo
	}
	inCard.Edition = edition

	// PromoWildcard is the flag Match already reads to skip edition
	// selection entirely, both the exact match and the looser one: a
	// storefront's "EB01 - Memorial Collection" does not equal the set's
	// "Extra Booster: Memorial Collection" but is contained in it, and
	// that alone was enough to delete the promo printing.
	if variantPointedAt(b, inCard) {
		inCard.PromoWildcard = true
	}
}

// promoSetBegun answers the promotional set a listing's wording names by the
// opening words of a printing's label, and "" when it names none.
//
// A storefront shortens the event a promo was handed out at - Cool Stuff Inc
// writes "(Championship 25-26)" for the catalog's "Championship 25 26
// Regionals Season 1" - and files the listing under the base card's set,
// which is the one set that printing is not in. The abbreviation is too short
// for the variant tiering to read as a label, so nothing at all says the
// listing is a promo and the base common answers it, wearing the promo's
// $1250 buy price. Nothing but the edition reaches the right card: the
// printing shares its number with the base card, so the number cannot pick
// it, and a tiering that never read the label cannot prefer it either.
//
// Four things hold the rule to that case. Only a full collector number is
// read, since a bare tail is the same number in every set of the game. The
// run has to open the label rather than sit anywhere in it, because a
// storefront shortening a name keeps its front. It has to be two words long,
// because one word is what a wording lands on by coincidence. And a wording
// that spells some label out in full is left alone, along with a run opening
// the labels of two different sets: the first is already the tiering's to
// answer and the second names nothing.
func promoSetBegun(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) string {
	number := inputNumber(b, inCard)
	if !fullNumberRe.MatchString(number) {
		return ""
	}
	var name, code string
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed || len(co.PromoTypes) == 0 ||
			!numberMatches(number, co.Number) {
			continue
		}
		if mtgmatcher.SlugDescribesAny(inCard.Variation, co.PromoTypes) {
			return ""
		}
		if !slugsBegunBy(inCard.Variation, co.PromoTypes) {
			continue
		}
		set, found := b.Sets[co.SetCode]
		if !found {
			continue
		}
		if code != "" && code != co.SetCode {
			return ""
		}
		name, code = set.Name, co.SetCode
	}
	if !setIsPromotional(name) {
		return ""
	}
	return name
}

// slugsBegunBy reports whether a wording opens any of the labels a printing
// wears, as a run of at least two whole words and short of the whole label.
// It is SlugDescribes stopped short of the end: the slug has lost its spaces,
// so the wording's words are joined back up a run at a time and the label
// asked whether it starts with them. A wording spelling a label out in full
// opens it too, along every shorter run, so it is promoSetBegun that tells
// the abbreviation from the full name.
func slugsBegunBy(wording string, slugs []string) bool {
	words := strings.Fields(strings.ToLower(wording))
	for _, slug := range slugs {
		if slug == "" {
			continue
		}
		for i := range words {
			var joined string
			for j := i; j < len(words); j++ {
				joined += mtgmatcher.PromoTypeSlug(words[j])
				if len(joined) >= len(slug) || !strings.HasPrefix(slug, joined) {
					break
				}
				if j > i {
					return true
				}
			}
		}
	}
	return false
}

// variantPointedAt reports whether the input's own wording asks for a
// variant printing: a letter tail or "(V.n)" index, or words naming the
// variant label of some printing of this card. Only the variation is read,
// never the edition - the edition is what this decides whether to trust.
func variantPointedAt(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) bool {
	number := inputNumber(b, inCard)
	if wantsVariant(inCard, number) {
		return true
	}
	wording := strings.ToLower(inCard.Variation)
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed || len(co.PromoTypes) == 0 {
			continue
		}
		// Only a printing of the number being asked about can say that this
		// input wants a variant. A label another number wears - the card
		// reprinted elsewhere under its own - would otherwise unpin the
		// edition for an input the edition was answering perfectly, and the
		// widened pool then hands back the base printing.
		if number != "" && !numberMatches(number, co.Number) {
			continue
		}
		if mtgmatcher.SlugDescribesAny(wording, co.PromoTypes) {
			return true
		}
	}
	return false
}

// CanonicalFinish adds nothing to the shared vocabulary: a One Piece product
// is sold plain or foil, spelled "Normal" and "Foil" by the catalog the
// datastore is built from, and both are names every game shares. The parallel
// and manga treatments are variants of a printing, each priced as a product
// of its own, not finishes of one.
func (Rules) CanonicalFinish(name string) string {
	return mtgmatcher.CanonicalFinish(name)
}

// FilterCards narrows candidates by edition, collector number and variant.
// The variant tiering mirrors the number sharing in the data: when the
// input's wording describes a variant label, the printings it describes
// win; a bare input keeps the base printing when one exists; cardmarket's
// positional "(V.n)" wording keeps the base for V.1 and the variants
// otherwise.
func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	number := inputNumber(b, inCard)

	var candidates []mtgmatcher.Card
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		card := co.Card

		// A dual-printing product files both its finish uuids under the
		// name bucket; fold the foil one back onto the bare id so each
		// candidate appears exactly once, and output() picks the finish
		// afterwards. The base id's own underscores rule out the first-
		// underscore cut riftbound uses, but the foil suffix is fixed.
		base := strings.TrimSuffix(uuid, "_foil")
		if seen[base] {
			continue
		}
		seen[base] = true

		if _, found := cardSet[card.SetCode]; !found {
			continue
		}
		if number != "" && !numberMatches(number, card.Number) {
			continue
		}
		candidates = append(candidates, card)
	}
	if len(candidates) <= 1 {
		return candidates
	}

	// A name whose printings wear no collector number has nothing but the
	// set and the label telling them apart, and the labels are prose the
	// catalog wrote rather than tags, so both are read here instead of by
	// the tiering below. Every other name is left to the tiering on
	// purpose: a variant shares its base card's number while being filed in
	// another set, and storefronts name the base card's set for it. The
	// DON!! cards are not reprinted that way - each set prints its own - so
	// an edition naming one of them names the set the listing is in, and a
	// wording that happens to describe another set's DON!! must not reach
	// past it.
	if !numbered(b, inCard.Name) {
		candidates = editionTiebreak(b, inCard, candidates)
		if len(candidates) <= 1 || !editionNamesSet(b, inCard.Edition) {
			return candidates
		}
		return labelOverlap(b, inCard, candidates, false)
	}

	// The letter tail cardtrader appends to a number ("OP01-001a") means a
	// variant printing without saying which; the V.n index says the same.
	// Either demand drops the base printing from consideration.
	described, base, variants := tierByVariant(inCard, candidates)
	if len(described) > 0 {
		return finishTiebreak(inCard, editionTiebreak(b, inCard, described))
	}
	if wantsVariant(inCard, number) {
		if len(variants) > 0 {
			// The edition decides first and the index only speaks for what
			// it leaves: cardmarket's index counts one shelf's products, so
			// reading it across sets would answer a storefront naming a
			// promo set with a booster set's parallel.
			narrowed := editionTiebreak(b, inCard, variants)
			if picked := variantAtIndex(inCard, editionTiebreak(b, inCard, base), narrowed); picked != nil {
				return picked
			}
			return narrowed
		}
		return candidates
	}
	if len(base) > 0 {
		return editionTiebreak(b, inCard, base)
	}
	// Every candidate wears a label, the wording spelled none of them out,
	// and the number has already said which card this is - so the label is
	// the only thing left the listing can be asking about. The catalog
	// prefixes an event label with the base set's code ("OP16 Release
	// Event", "OP16 Release Event Winner") and the storefront never writes
	// it, so the whole-label test the tiering above runs answers nothing and
	// both printings come back aliased. Counting the label's own words the
	// wording did say separates them, and refuses whatever it cannot.
	candidates = editionTiebreak(b, inCard, candidates)
	if len(candidates) <= 1 {
		return candidates
	}
	return labelOverlap(b, inCard, candidates, true)
}

// variantAtIndex narrows the variants tier to the one printing cardmarket's
// "(V.n)" index points at, when the datastore's own numbering says which that
// is. Bandai numbers a card's alternate printings by appending "_p1", "_p2"
// to its collector number, in the order cardmarket files them behind the base
// printing, so V.2 is "_p1" and V.3 is "_p2".
//
// The index is a position in cardmarket's own shelf, not a name for anything,
// so it only means this where the shelf is the plain one it describes: one
// base printing for V.1 to have been, and exactly one variant wearing the
// suffix asked for. Cardmarket files an un-indexed product beside indexed
// ones on its promo shelves, where V.1 is already an alternate rather than
// the base - and on those shelves every printing of ours carries a promo
// label, so there is no base tier and the index is left unread rather than
// spending V.2 on the printing V.1 stands for.
func variantAtIndex(inCard *mtgmatcher.InputCard, base, variants []mtgmatcher.Card) []mtgmatcher.Card {
	index := positionalIndex(inCard.Variation)
	if index < 2 || len(base) != 1 || base[0].Identifiers == nil {
		return nil
	}
	bandaiID := base[0].Identifiers["bandaiId"]
	if bandaiID == "" {
		return nil
	}
	want := bandaiID + "_p" + strconv.Itoa(index-1)
	var picked []mtgmatcher.Card
	for _, card := range variants {
		if card.Identifiers["bandaiId"] == want {
			picked = append(picked, card)
		}
	}
	if len(picked) != 1 {
		return nil
	}
	return picked
}

// finishTiebreak narrows a tier the wording described in full to the
// printings sold in the finish the storefront priced.
//
// The event printings a promo number carries are one card issued several
// times over, and a storefront names them all at once: "P-115 (OP15 Release
// Event - Winner)" says both the set's release-event card and the winner's
// copy of it, because the winner's label is the plain one with a word
// appended. What separates them is the stamping - the participation card is
// sold plain, the winner's foil - so the finish being priced is the last
// thing left that says which one the listing is.
//
// Only a tier the wording named throughout is narrowed this way. A wording
// naming none of them has not said the listing is one of these at all, and
// answering it with whichever printing happens to carry the right stamping
// would price a product the catalog does not hold as one it does.
func finishTiebreak(inCard *mtgmatcher.InputCard, cards []mtgmatcher.Card) []mtgmatcher.Card {
	if len(cards) <= 1 {
		return cards
	}
	finish := mtgmatcher.FinishNonfoil
	if inCard.Foil {
		finish = mtgmatcher.FinishFoil
	}
	var stamped []mtgmatcher.Card
	for _, card := range cards {
		if card.HasFinish(finish) {
			stamped = append(stamped, card)
		}
	}
	if len(stamped) == 0 {
		return cards
	}
	return stamped
}

// editionNamesSet reports whether a storefront's edition is a set of the
// game rather than a shelf it files a whole game under.
//
// The labels tell apart the printings of one set, and the set has to have
// been named for them to be read: cardmarket files half its promos under
// "One Piece Products", where the pool is every DON!! the game ever printed
// and one character name is spelled by half a dozen of them across as many
// sets. A listing naming a card the catalog does not hold - a convention
// DON!! nobody catalogued, an event set a year newer - would answer with a
// booster's card of the same character.
//
// The question is asked the way Match asks it, on the name or a part of it,
// with the promo line a storefront hangs a set name off read through.
func editionNamesSet(b *mtgmatcher.Backend, edition string) bool {
	if mtgmatcher.Normalize(edition) == "" {
		return false
	}
	base := promoLineRe.ReplaceAllString(edition, "")
	if mtgmatcher.Normalize(base) == "" {
		base = edition
	}
	for _, set := range b.Sets {
		if mtgmatcher.Contains(set.Name, edition) || mtgmatcher.Contains(set.Name, base) {
			return true
		}
	}
	return false
}

// editionTiebreak narrows a tier still holding several printings to the
// ones filed in the set the edition describes, with the exact name match
// preferred over the contained-in one, mirroring how Match itself selects
// editions. PromoWildcard trades the edition's pre-filtering for a wider
// pool so a cross-set variant survives to the tiering; this hands the
// edition back its say once the wording has done its work, so the same
// label printed in several sets - the starter-deck and Premium Booster
// Reprints share one number and one label - resolves instead of aliasing.
// An edition matching none of the tier keeps the whole tier: storefronts
// name the base card's set for printings filed elsewhere, and failing safe
// on genuine ambiguity is the point of the tiering.
func editionTiebreak(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cards []mtgmatcher.Card) []mtgmatcher.Card {
	if len(cards) <= 1 {
		return cards
	}
	if inCard.Edition == "" {
		return cards
	}

	// A storefront files a set's event printings under that set's promo
	// line: cardmarket's "Promos: Pillars of Strength" is the set the
	// datastore calls "Pillars of Strength Pre-Release Cards". Neither name
	// contains the other, so the marker has to be read as the selection it
	// is or the whole promo line prices as the base commons.
	// A marker with nothing behind it - a bare "Promos:" bucket - names no
	// set, and every set name contains the empty string: reading it as a
	// selection would hand back the event printing of whatever card it
	// decorated. Only a line naming a set selects one, and the emptiness to
	// refuse is the one Contains sees: a remainder of pure punctuation
	// ("Promos: -", "Promo: 's") normalizes away as thoroughly as an absent
	// one, and reads as the same empty needle.
	base := promoLineRe.ReplaceAllString(inCard.Edition, "")
	if mtgmatcher.Normalize(base) != "" && base != inCard.Edition {
		var event []mtgmatcher.Card
		for _, card := range cards {
			set, found := b.Sets[card.SetCode]
			if !found || !isEventSet(card.SetCode) {
				continue
			}
			if mtgmatcher.Contains(set.Name, base) {
				event = append(event, card)
			}
		}
		if len(event) > 0 {
			return event
		}
	}

	var equal, contained []mtgmatcher.Card
	for _, card := range cards {
		set, found := b.Sets[card.SetCode]
		if !found {
			continue
		}
		if mtgmatcher.Equals(set.Name, inCard.Edition) {
			equal = append(equal, card)
		} else if mtgmatcher.Contains(set.Name, inCard.Edition) {
			contained = append(contained, card)
		}
	}
	if len(equal) > 0 {
		return equal
	}
	if len(contained) > 0 {
		return contained
	}

	// The event printings a set's cards receive are filed in a set of their
	// own instead of wearing a variant label, so they share a name, a
	// number and an empty label with the card they reprint and nothing but
	// the set name tells them apart. A storefront names an event printing
	// when it stocks one, so an edition matching neither set is still a
	// claim that excludes them - unlike no edition at all, which claims
	// nothing and is left to alias.
	var regular []mtgmatcher.Card
	for _, card := range cards {
		if !isEventSet(card.SetCode) {
			regular = append(regular, card)
		}
	}
	if len(regular) > 0 {
		return regular
	}
	return cards
}

// promoLineRe matches the promo-line prefix a storefront hangs a set name
// off to name that set's event printings.
var promoLineRe = regexp.MustCompile(`(?i)^promos?\s*[:-]\s*`)

// canonicalEdition returns the name of the one set an edition describes
// better than every other, or "" when the wording picks no clear winner.
//
// The wording has to leave nothing of the set name unaccounted for, or
// share three words with it when it does: two words in common is the
// coincidence a vendor bucket like "One Piece Products" lands on, and
// answering that with a set would price a whole shelf as one card.
//
// The set code the storefront wore in front of the name overrides the
// wording: a family shares its name across volumes ("Premium Booster -The
// Best-" and its Vol. 2), so the code is the only thing that says which
// one the listing is filed in. An event set's code opens with its base
// set's - "OP14 RE" against "OP14" - and that opening field is the whole
// of what a storefront prefix spells, so the code cannot tell the pair
// apart and the wording has to. Only the wording being a name the
// storefront cut short unseats the code, and which words it holds does
// not say that: the markers the event sets append come out of one
// vocabulary, so a wording sharing every word with one of them is as
// likely a storefront hanging a decoration off the base name. A
// truncation loses a name's tail, so the wording that is one spells the
// winner's leading words in the winner's order - "The Azure Sea's Seven
// Release" against "The Azure Sea's Seven Release Event Cards", where
// "Pillars of Strength Cards" reaches past the marker of "Pillars of
// Strength Pre-Release Cards" and is decoration on the coded set instead.
func canonicalEdition(b *mtgmatcher.Backend, edition, code string) string {
	want := editionTokens(edition)
	if len(want) == 0 {
		return ""
	}
	// A set already naming itself needs no rewriting, and the set index
	// answers that without scoring a single name. The code still has its
	// say: it names the set outright when it disagrees with the wording.
	wantCode := foldSetCode(code)
	named, found := b.NormalizedSets[mtgmatcher.Normalize(edition)]
	if found && (wantCode == "" || foldSetCode(named.Code) == wantCode) {
		return ""
	}

	var best, runner, coded editionScore
	for setCode, set := range b.Sets {
		cur := scoreEdition(want, set.Name)
		cur.event = isEventSet(setCode)
		if cur.shared < 2 || cur.missing > 1 || (cur.missing > 0 && cur.shared < 3) {
			continue
		}
		if wantCode != "" && foldSetCode(setCode) == wantCode {
			coded = cur
		}
		if best.name == "" || cur.beats(best) {
			runner, best = best, cur
		} else if runner.name == "" || cur.beats(runner) {
			runner = cur
		}
	}
	// The coded set answers unless the winner both accounts for more of the
	// wording than it does and spells that wording as its own leading
	// words: only then is the wording a name the storefront cut short.
	if coded.name != "" {
		cut := best.missing < coded.missing && truncates(edition, best.name)
		if !cut {
			return coded.name
		}
	}
	if best.name == "" || !best.beats(runner) {
		return ""
	}
	return best.name
}

// foldSetCode drops the punctuation a storefront spells a set code with:
// cardtrader writes "OP-01" for the datastore's "OP01", coolstuffinc
// "EB01" for its "EB-01". The event sets keep their marker ("OP03 PRE"
// folds to OP03PRE), which no storefront prefix spells.
func foldSetCode(code string) string {
	return strings.Map(func(r rune) rune {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return -1
		}
		return unicode.ToUpper(r)
	}, code)
}

// editionScore counts how much of a set name an edition accounts for: the
// words the two share and the ones the set name carries that the edition
// left out, with the event sets flagged for the tie those two cannot
// break. Nothing else orders two sets - the words a set name spells beyond
// the wording say nothing about which one was meant, and answering a
// family's shared name with its shortest member invents a volume the
// storefront never wrote.
type editionScore struct {
	name            string
	shared, missing int
	event           bool
}

func scoreEdition(want map[string]bool, name string) editionScore {
	have := editionTokens(name)
	cur := editionScore{name: name}
	for token := range want {
		if have[token] {
			cur.shared++
		} else {
			cur.missing++
		}
	}
	return cur
}

func (s editionScore) beats(other editionScore) bool {
	if s.shared != other.shared {
		return s.shared > other.shared
	}
	if s.missing != other.missing {
		return s.missing < other.missing
	}
	// An event set wears its base set's whole name with a marker appended,
	// so a wording spelling the base name alone accounts for both exactly
	// as well and they would tie forever. A wording that spells the marker
	// too wins the event set the counts above, so a tie left here is the
	// marker gone unspelled, and the base set is the one being named.
	return !s.event && other.event
}

// editionFields splits a set name into the words that carry its identity,
// in the order it spells them. Normalize is no help here: it drops the
// spaces the words are counted by.
func editionFields(edition string) []string {
	return strings.FieldsFunc(strings.ToLower(edition), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func editionTokens(edition string) map[string]bool {
	out := map[string]bool{}
	for _, field := range editionFields(edition) {
		out[field] = true
	}
	return out
}

// truncates reports whether the wording is the set name with a tail cut
// off: its words are that name's leading ones, in the order the name
// spells them.
func truncates(edition, name string) bool {
	want := editionFields(edition)
	have := editionFields(name)
	if len(want) == 0 || len(want) > len(have) {
		return false
	}
	for i, field := range want {
		if field != have[i] {
			return false
		}
	}
	return true
}

// isEventSet reports whether a set code files the event printings of
// another set's cards. The datastore spells them as the base code with a
// marker appended - "OP03-PRE" for the pre-release cards, "OP10-RE" for the
// release event ones, "OP05-ANN" for the anniversary tournament ones.
//
// The marker is read off either separator: a set code cannot carry a space,
// since a search query is split on whitespace before a filter sees it, and
// the datastore wrote these with one until it stopped. Reading both spellings
// is what lets this land before the datastore is rebuilt.
func isEventSet(code string) bool {
	fields := strings.FieldsFunc(code, func(r rune) bool {
		return r == ' ' || r == '-'
	})
	if len(fields) < 2 {
		return false
	}
	switch fields[len(fields)-1] {
	case "PRE", "RE", "ANN":
		return true
	}
	return false
}

// tierByVariant splits the candidates into the ones whose variant label the
// input's wording describes, the base printings, and the variant printings.
func tierByVariant(inCard *mtgmatcher.InputCard, candidates []mtgmatcher.Card) (described, base, variants []mtgmatcher.Card) {
	wording := strings.ToLower(inCard.Variation + " " + inCard.Edition)
	for _, card := range candidates {
		if len(card.PromoTypes) == 0 {
			base = append(base, card)
			continue
		}
		variants = append(variants, card)
	}
	described = mtgmatcher.DescribedVariants(wording, variants)
	return
}

// doublePackRe matches the product line a Double Pack Set's two DON!!
// cards are filed under, in the storefront's spelling or the catalog's
// ("Double Pack Set Vol. 4", "Double Pack Volume 2", "Double Pack Vol 7").
var doublePackRe = regexp.MustCompile(`(?i)\bdouble\s*pack\b`)

// labelOverlap picks, among printings told apart by nothing but a label,
// the one whose label the storefront's wording names.
//
// The DON!! cards are the game's only such printings, and their labels are
// prose the catalog wrote rather than tags: "Trafalgar Law, Eustass Kid and
// Monkey.D.Luffy Double Pack Set Vol. 4" beside a plain "Alternate Art" in
// the same set. A storefront never spells one of those in full - it writes
// "(Law, Kid, Luffy) (Double Pack Set Vol. 4)" - so asking whether the
// wording names a whole label answers "Alternate Art" for every one of them,
// because every such listing also says the words "alternate art". Counting
// the label's own words the wording says asks the question the other way
// round and lets the fuller label win.
//
// The wording is not all identification, though. Beside the decorations a
// storefront hangs off the name it publishes a sentence describing the art -
// "Gold Hook", "Big Eye", "Sand Tornado" - and by the time a filter runs the
// two have been folded into one variation with nothing to tell them apart.
// Every guard below is there because that prose reached a label it had no
// business naming.
//
// A label word counts as said only when the wording spells it as a run of
// whole words, which is what lets a catalog's "GEAR5 Luffy" hear a
// storefront's "Gear 5 Luffy" and refuse the "Gear 4 Luffy" filed beside it.
// The winner must say more of its label than it leaves out, or an art
// sentence saying "Big Eye" would answer with the set's "Big Mom". And a
// winner that says no more words than the runner-up, only fewer wrong ones,
// has to say a word no other label carries: without it a wording naming a
// Double Pack volume and no character - "Sand Tornado", "Purple Smoke" -
// answers with whichever of the volume's two cards has the shorter name,
// and both of the volume's listings price as the same card.
//
// Only the variation is read. The edition is the same string for every
// candidate here - the set is what narrowed to them - so it can say nothing
// about which of their labels was meant, while the words of the set's own
// name would count as label evidence: "OP15 - Adventure On Kami's Island"
// hands the word "Island" to a "Sky Island Map" label no listing named.
func labelOverlap(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, candidates []mtgmatcher.Card, mustName bool) []mtgmatcher.Card {
	wording := inCard.Variation

	// A Double Pack Set's DON!! cards are filed under a label naming the
	// set, so a listing naming a volume is naming one of them and nothing
	// else in the set answers. Where the catalog holds no such card - Vol.
	// 1 is missing from it - the listing has no printing to price and must
	// not settle for the set's plain alternate art.
	if doublePackRe.MatchString(wording) {
		var packs []mtgmatcher.Card
		for _, card := range candidates {
			if doublePackRe.MatchString(promoLabel(b, card)) {
				packs = append(packs, card)
			}
		}
		if len(packs) == 0 {
			return nil
		}
		candidates = packs
	}

	groups, order := groupByIdentity(b, candidates)
	// One identity is one card printed plain and gold, and nothing has to
	// name it: the set was what narrowed to it, and the treatment is all
	// that is left to decide. A caller that demands the wording name
	// something is asking about the label rather than the treatment, so it
	// goes on to the scoring below, where a wording naming nothing settles
	// nothing.
	if len(order) == 1 && !mustName {
		return groups[order[0]].pick(inCard, treatmentSaid(inCard.Variation))
	}

	// A word two labels share says nothing about which of them was meant.
	carried := map[string]int{}
	for _, key := range order {
		for _, word := range uniqueWords(groups[key].words) {
			carried[word]++
		}
	}

	var best, runner *identityGroup
	for _, key := range order {
		group := groups[key]
		for _, word := range group.words {
			if mtgmatcher.SlugDescribes(wording, word) {
				group.named++
				if carried[word] == 1 {
					group.only = true
				}
			}
		}
		if best == nil || group.beats(best) {
			runner, best = best, group
		} else if runner == nil || group.beats(runner) {
			runner = group
		}
	}
	if best == nil || best.named <= len(best.words)-best.named {
		return candidates
	}
	if runner != nil && (!best.beats(runner) || (best.named == runner.named && !best.only)) {
		return candidates
	}
	return best.pick(inCard, treatmentSaid(inCard.Variation))
}

// identityGroup collects the printings of one label, the gold-treated half
// of a pair filed beside the plain one it is a treatment of, along with how
// much of that shared identity a wording said.
type identityGroup struct {
	words []string
	plain []mtgmatcher.Card
	gold  []mtgmatcher.Card
	named int
	only  bool
}

// beats orders two identities by the words of theirs a wording said, the
// label leaving fewest unsaid winning a tie. That tie-break is what keeps a
// bare wording on the bare printing where the treatment is not what tells
// the two apart: "Nami" says all of "Nami" and half of "Nami Manga".
func (g *identityGroup) beats(other *identityGroup) bool {
	if g.named != other.named {
		return g.named > other.named
	}
	return len(g.words)-g.named < len(other.words)-other.named
}

// pick answers with the treated half of the identity when the storefront
// asked for the treatment and the catalog holds one, and with the plain half
// otherwise. An identity holding only the half that was not asked for hands
// back what it has: the treatment is the storefront's claim, and refusing its
// own printing over it would price nothing at all. Handing back both is how
// this refuses to answer, leaving Match to report the aliasing.
//
// A storefront that indexes its products positionally has already said which
// of an identity's halves it means, and its index outranks any wording: this
// is cardmarket, which does not describe a DON!! beyond naming the character
// and files the plain printing as V.1 with the treated one behind it, the
// same reading the numbered cards get. An index past the pair names a product
// the catalog does not hold as a half of anything, and pricing it as either
// half would be a guess.
func (g *identityGroup) pick(inCard *mtgmatcher.InputCard, treated bool) []mtgmatcher.Card {
	both := append(append([]mtgmatcher.Card{}, g.plain...), g.gold...)
	if len(both) == 1 {
		return both
	}
	switch positionalIndex(inCard.Variation) {
	case 0:
	case 1:
		treated = false
	case 2:
		treated = true
	default:
		return both
	}
	if treated && len(g.gold) > 0 {
		return g.gold
	}
	if !treated && len(g.plain) > 0 {
		return g.plain
	}
	return both
}

// positionalIndex reads the "(V.n)" index cardmarket synthesizes to tell the
// products sharing a name apart, 0 where none was written.
func positionalIndex(variation string) int {
	for field := range strings.FieldsSeq(strings.ToLower(variation)) {
		tail, found := strings.CutPrefix(field, "v.")
		if !found {
			continue
		}
		index, err := strconv.Atoi(tail)
		if err != nil {
			continue
		}
		return index
	}
	return 0
}

// goldTreatment is the word a DON!! label ends in when the printing is the
// gold-bordered half of a pair: a third of the catalog's DON!! labels are
// another label of the same set with this word appended.
const goldTreatment = "gold"

// groupByIdentity buckets the candidates by the label they wear with the
// gold treatment taken off, so that the pair a set prints of one card is
// scored once, on the identity both halves share. Returned with the order
// the identities were first seen, so that nothing depends on map iteration.
func groupByIdentity(b *mtgmatcher.Backend, candidates []mtgmatcher.Card) (map[string]*identityGroup, []string) {
	groups := map[string]*identityGroup{}
	var order []string
	for _, card := range candidates {
		words := labelWords(promoLabel(b, card))
		// A label that is the treatment word alone is not a treated half of
		// anything: those are the coloured DON!! promos, filed under the name
		// of their colour.
		treated := len(words) > 1 && words[len(words)-1] == goldTreatment
		if treated {
			words = words[:len(words)-1]
		}
		key := strings.Join(words, " ")
		group, found := groups[key]
		if !found {
			group = &identityGroup{words: words}
			groups[key] = group
			order = append(order, key)
		}
		if treated {
			group.gold = append(group.gold, card)
		} else {
			group.plain = append(group.plain, card)
		}
	}
	return groups, order
}

// treatmentNeighbours are the words a storefront writes beside "gold" when
// it means the gold-bordered printing: the finish, the border, the text the
// border is drawn around, or the version of the card.
var treatmentNeighbours = map[string]bool{
	"foil": true, "border": true, "text": true, "ver": true, "version": true,
}

// treatmentSaid reports whether a wording claims the gold treatment rather
// than merely saying the word.
//
// A storefront claims it beside one of the words a treatment is named with
// ("Gold Foil", "Gold Border", "Gold text/Border", "Gold Ver."), or with the
// word standing at the end of what it wrote, which is where the decorations
// hung off a name end up. The art sentences say gold of something in the
// picture instead - a hook, a bell, a logo - and reading those as the claim
// priced a set's plain DON!! as its gold-bordered twin.
//
// Which word comes next is read off the punctuation as well as the spaces:
// a storefront runs its decorations together ("Gold text/Border", "(Gold
// Border)(We'll Have To...") and joining those into one word hid the
// treatment behind it.
func treatmentSaid(wording string) bool {
	words := strings.FieldsFunc(strings.ToLower(wording), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for i, word := range words {
		if word != goldTreatment {
			continue
		}
		if i == len(words)-1 || treatmentNeighbours[words[i+1]] {
			return true
		}
	}
	return false
}

// promoLabel returns the words a printing's single promo type was distilled
// from. A printing wearing none is the set's plain one and has no label.
func promoLabel(b *mtgmatcher.Backend, card mtgmatcher.Card) string {
	if len(card.PromoTypes) != 1 {
		return ""
	}
	return b.PromoTypeLabels[card.PromoTypes[0]]
}

// labelWords cuts a label or a wording into the words a comparison can be
// made of, each stripped of the punctuation the slugging drops so that
// "Law," and "Monkey.D.Luffy" read as they are written.
func labelWords(text string) []string {
	fields := strings.Fields(text)
	words := make([]string, 0, len(fields))
	for _, field := range fields {
		if word := mtgmatcher.PromoTypeSlug(field); word != "" {
			words = append(words, word)
		}
	}
	return words
}

// uniqueWords drops the repeats out of a label's words, so that a word a
// label spells twice is still one label carrying it.
func uniqueWords(words []string) []string {
	seen := map[string]bool{}
	out := words[:0:0]
	for _, word := range words {
		if seen[word] {
			continue
		}
		seen[word] = true
		out = append(out, word)
	}
	return out
}

// wantsVariant reports whether the input demands some variant printing
// without describing which: cardmarket's "(V.n)" with n past 1, or a letter
// tail on the collector number ("OP01-001a").
func wantsVariant(inCard *mtgmatcher.InputCard, number string) bool {
	if number != "" && number[len(number)-1] >= 'a' {
		return true
	}
	for field := range strings.FieldsSeq(strings.ToLower(inCard.Variation)) {
		if strings.HasPrefix(field, "v.") && field != "v.1" {
			return true
		}
	}
	return false
}

// inputNumber is extractNumber with the card being asked about in hand: the
// number a storefront wrote only narrows when the printings of that name
// wear collector numbers it could be one of.
func inputNumber(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) string {
	number := extractNumber(inCard.Variation)
	if number == "" || numbered(b, inCard.Name) {
		return number
	}
	return ""
}

// numbered reports whether any printing of a name wears a collector number
// with a digit in it. The DON!! cards are the game's one exception: every
// printing of them is filed under the literal code "DON" instead of a
// number, so nothing a storefront writes in its number field is one of
// theirs - coolstuffinc writes the set code there ("PRB-01"), cardmarket the
// number of the card the DON!! was packed with - and comparing it deleted
// every candidate the name had. A name with even one numbered printing is
// not that case: "Monkey.D.Luffy" wears "LEADER" on a few event printings
// and a real number everywhere else, and the number still has to tell those
// apart.
func numbered(b *mtgmatcher.Backend, name string) bool {
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		if strings.ContainsFunc(co.Number, unicode.IsDigit) {
			return true
		}
	}
	return false
}

// extractNumber pulls the collector number out of the scraper-supplied
// Variation: the full "OP01-001" shape when present, else the first
// digit-leading field ("001", "001/121").
func extractNumber(variation string) string {
	fields := strings.Fields(variation)
	for _, field := range fields {
		if fullNumberRe.MatchString(field) {
			return strings.Split(field, "/")[0]
		}
	}
	for _, field := range fields {
		if field[0] >= '0' && field[0] <= '9' {
			return strings.Split(field, "/")[0]
		}
	}
	return ""
}

// numberMatches compares an input number against a printing's full
// collector number: equal full codes, or a bare input matching the code's
// numeric tail. Leading zeros never decide ("1" matches "OP01-001"), and a
// letter tail on the input ("OP01-001a") matches its base number.
func numberMatches(input, full string) bool {
	if strings.EqualFold(input, full) {
		return true
	}
	input = strings.TrimRight(input, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
	if strings.EqualFold(input, full) {
		return true
	}
	inSet, inTail := splitNumber(input)
	fullSet, fullTail := splitNumber(full)
	// An input spelling its own set code has already named which printing's
	// number it is, and only that code may answer it: dropping to the tail
	// alone would let "OP07-002" take every card numbered -002 in the game,
	// aliasing a set's card against its every same-numbered reprint. A bare
	// input carries no code to disagree with, so it still matches on the
	// tail: that is the whole identification cardmarket's number field has.
	if inSet != "" && !strings.EqualFold(inSet, fullSet) {
		return false
	}
	return inTail != "" && canonicalTail(inTail) == canonicalTail(fullTail)
}

// splitNumber cuts a collector number into its set code and numeric tail,
// a number without a code being all tail.
func splitNumber(number string) (set, tail string) {
	idx := strings.LastIndexByte(number, '-')
	if idx >= 0 {
		return number[:idx], number[idx+1:]
	}
	return "", number
}

// canonicalTail strips leading zeros from a bare number, an all-zero run
// staying "0".
func canonicalTail(number string) string {
	trimmed := strings.TrimLeft(number, "0")
	if trimmed == "" && number != "" {
		return "0"
	}
	return trimmed
}
