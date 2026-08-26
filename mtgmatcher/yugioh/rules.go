package yugioh

import (
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for Yu-Gi-Oh!. A card is identified
// by name + collector number + rarity: the same number exists as several
// products differing only by rarity, so the rarity wording (or the rarity
// suffix cardtrader appends to its collector numbers) is what tells them
// apart. The print run never gates anything: a product carries every run it
// was priced in, exactly as foil never gates One Piece. It does route: an
// input naming a run - in the finish field, or in the wording for the
// storefronts that have no such field - resolves to that run's entry
// instead of the default one.
type Rules struct{ mtgmatcher.DefaultRules }

// fullNumberRe matches the game's collector number shapes: "LOB-001",
// "RA01-EN019", "YGLD-ENA03", with an optional letter tail (cardtrader
// suffixes rarities "RA01-EN019qsec", Konami suffixes misprints
// "EOJ-EN004K").
var fullNumberRe = regexp.MustCompile(`^[A-Za-z0-9]+-[A-Za-z]*[0-9]+[a-zA-Z]*$`)

// suffixRarities maps the lowercase letter tail cardtrader appends to a
// collector number to the rarity it encodes; a further "a" tail marks the
// alternate-art printing.
var suffixRarities = map[string]string{
	"u":    "Ultra Rare",
	"sec":  "Secret Rare",
	"qsec": "Quarter Century Secret Rare",
	"cr":   "Collector's Rare",
	"ul":   "Ultimate Rare",
	"psec": "Platinum Secret Rare",
	"sh":   "Shatterfoil Rare",
}

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Prefilter splits the parenthetical decorations off the name before the
// canonical-name lookup: TCGplayer writes "Eldlich the Golden Lord (Quarter
// Century Secret Rare)" and "Harpie Lady (Original Artwork)". A full name
// that is itself canonical stays as it is — some catalog names carry their
// qualifier parenthetical ("Dark Magician (A)"). The name then takes the
// spelling its own edition files it under, when the two disagree.
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	_, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]
	if !found && strings.Contains(inCard.Name, "(") {
		vars := mtgmatcher.SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.AddToVariant(strings.Join(vars[1:], " "))
		}
	}
	respellName(b, inCard)
}

// respellName adopts the spelling the input's own edition files the card
// under, when the edition resolves to a set that does not carry the name as
// written: the storefront respellings nameRespellings pairs (cardtrader
// writes Magician's Force's "Vampire Orchis" as "Vampiric Orchis", the name
// of the real newer DASA card) and the token word-order flip in either
// direction ("Sky Striker Ace Token" against OTS Tournament Pack 8's
// "Token: Sky Striker Ace"). A token name the spellings cannot place is
// asked of the set's own token sheet by collector number ("Yugi & Dark
// Magician Token" T01 is Legendary Decks II's "Token: Yugi").
//
// The edition is the whole guard, three ways. A name the resolved set does
// carry is never touched, so the DASA "Vampiric Orchis" and the sets naming
// a token the storefront's way keep their own printing. A set carrying
// several distinct respellings, or several token names on one number (OTS
// Tournament Pack 9 prints three "Token: Mecha Phantom Beast" arts under
// 026), decides nothing and the name stays as written. And an edition
// naming no set at all decides nothing either — AdjustName's fallbacks own
// that input.
func respellName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	alternates := alternateNames(inCard.Name)
	token := isTokenName(inCard.Name)
	if len(alternates) == 0 && !token {
		return
	}
	set := editionSet(b, inCard.Edition)
	if set == nil {
		return
	}
	number := extractNumber(inCard.Variation)
	// A set printing the name as written settles it whatever the number
	// says: cardtrader numbers a few tokens one off ("Mask Token" 028 for
	// OTS Tournament Pack 19's EN029), and following such a number across
	// a respelling would price the neighbor.
	if editionCarries(set, inCard.Name, "") {
		return
	}
	var adopt string
	for _, alternate := range alternates {
		if !editionCarries(set, alternate, number) {
			continue
		}
		if adopt != "" && mtgmatcher.Normalize(adopt) != mtgmatcher.Normalize(alternate) {
			return
		}
		adopt = alternate
	}
	if adopt == "" {
		// A spelling the set prints under some other number is the same
		// name-against-number contradiction, so the number decides nothing
		// here either.
		for _, alternate := range alternates {
			if editionCarries(set, alternate, "") {
				return
			}
		}
		if token && number != "" {
			adopt = editionTokenAt(set, number)
		}
	}
	if adopt != "" {
		inCard.Name = adopt
	}
}

// alternateNames lists the other spellings a name might be filed under: its
// nameRespellings sibling, read in both directions, and the token word-order
// flip, either way around.
func alternateNames(name string) []string {
	var alternates []string
	normalized := mtgmatcher.Normalize(name)
	for _, pair := range nameRespellings {
		if mtgmatcher.Normalize(pair[0]) == normalized {
			alternates = append(alternates, pair[1])
		}
		if mtgmatcher.Normalize(pair[1]) == normalized {
			alternates = append(alternates, pair[0])
		}
	}
	if base, cut := strings.CutSuffix(name, " Token"); cut {
		alternates = append(alternates, "Token: "+base)
	}
	if base, cut := strings.CutPrefix(name, "Token: "); cut {
		alternates = append(alternates, base+" Token")
	}
	return alternates
}

// isTokenName reports whether a name names a token outright, in any of the
// word orders the catalog and the storefronts use — including the bare
// "Token" the sheets that never name their art are filed under.
func isTokenName(name string) bool {
	return strings.HasPrefix(name, "Token: ") || strings.HasSuffix(name, " Token") ||
		name == "Token"
}

// editionSet answers the set the input's edition names, through the same
// aliasing and decoration-trimming AdjustEdition applies, without touching
// the input.
func editionSet(b *mtgmatcher.Backend, edition string) *mtgmatcher.Set {
	edition = strings.TrimSpace(edition)
	named, found := namedSet(b, edition)
	if !found {
		named, found = namedSet(b, trimEditionDecorations(edition))
	}
	if !found {
		return nil
	}
	return b.NormalizedSets[mtgmatcher.Normalize(named)]
}

// editionCarries reports whether the set prints the name, under the given
// collector number when one is known.
func editionCarries(set *mtgmatcher.Set, name, number string) bool {
	normalized := mtgmatcher.Normalize(name)
	for i := range set.Cards {
		if mtgmatcher.Normalize(set.Cards[i].Name) != normalized {
			continue
		}
		if number == "" || numberMatches(number, set.Cards[i].Number) {
			return true
		}
	}
	return false
}

// editionTokenAt answers the one token the set prints under the number, or
// nothing when the number names none — or several, which no storefront
// wording tells apart.
func editionTokenAt(set *mtgmatcher.Set, number string) string {
	var name string
	for i := range set.Cards {
		card := &set.Cards[i]
		if !isTokenName(card.Name) || !numberMatches(number, card.Number) {
			continue
		}
		if name != "" && mtgmatcher.Normalize(name) != mtgmatcher.Normalize(card.Name) {
			return ""
		}
		name = card.Name
	}
	return name
}

// letteredNumber reads the letter-led numbers the token sheets and deck
// reprints use ("T01"), which extractNumber refuses everywhere else because
// outside a token's variation a letter-led field is a label word rather
// than a number.
func letteredNumber(variation string) string {
	for _, field := range strings.Fields(variation) {
		if len(field) < 2 || len(field) > 4 || !unicode.IsLetter(rune(field[0])) {
			continue
		}
		if !strings.ContainsFunc(field[1:], func(r rune) bool {
			return r < '0' || r > '9'
		}) {
			return field
		}
	}
	return ""
}

// AdjustName flips a token's name onto the word order the catalog files it
// under, and otherwise provides a prefix fallback for truncated feeds,
// adopting the one name among the prefix matches that carries the input's
// number. Names compare normalized, so punctuation variants of one name are
// not read as ambiguity.
func (Rules) AdjustName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	// The catalog names a token with the word first ("Token: Sheep") where
	// the storefronts write it last ("Sheep Token"). The flip only ever
	// answers for a name the datastore does not already know, above, so the
	// tokens a set does name the storefront's way keep their own printing.
	// It also only answers for an edition naming no set: respellName has
	// already asked a resolved edition for every spelling it carries, so
	// adopting the flip here anyway would hand the input another set's
	// token and the printing filter a name its own edition never prints.
	if base, cut := strings.CutSuffix(inCard.Name, " Token"); cut {
		flipped := "Token: " + base
		if _, found := b.CanonicalNames[mtgmatcher.Normalize(flipped)]; found &&
			editionSet(b, inCard.Edition) == nil {
			inCard.Name = flipped
			return
		}
	}
	uuids, err := b.SearchHasPrefix(inCard.Name)
	if err != nil {
		return
	}

	number := extractNumber(inCard.Variation)
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
	}
}

// AdjustEdition trims the game-name prefix and "Singles" suffix storefronts
// decorate set names with, and rewrites the names editionAliases carries. A
// handful of real set names carry the game name themselves ("Yu-Gi-Oh!
// Championship Series 2025 Prize Cards"), so an edition that already names a
// set is left alone - as it is asked, before the decorations come off and
// again after, the alias table only ever answering for a name no set has.
//
// An edition still naming no set is answered by the collector number, which
// carries the set code the wording never spelled.
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := strings.TrimSpace(inCard.Edition)
	if named, found := namedSet(b, edition); found {
		inCard.Edition = named
		return
	}
	edition = trimEditionDecorations(edition)
	if named, found := namedSet(b, edition); found {
		edition = named
	} else if set := numberSet(b, inCard, edition); set != nil {
		edition = set.Name
	}
	inCard.Edition = edition
}

// numberSet answers the set the input's collector number is filed in, for a
// bucket that names none.
//
// A bucket is an edition a storefront wrote and no set answers. An edition
// holding nothing is not one: a listing naming no set at all has claimed
// nothing to be wrong about, and the number alone must not stand in for the
// claim - a full code names one printing per set and several sets reprint it
// under exactly that code, so reading it as a set would hand back one of them
// where the whole pool is the honest answer.
//
// A Yu-Gi-Oh number opens with its set's code - "UBP1-EN005" is UBP1 - so a
// storefront filing a whole shelf of printings under one catch-all bucket
// still says, number by number, which set each of them lives in. Without
// that, Match has no set to search and falls through to every printing of the
// name, where the rarity and variant tiering pick whichever one the wording
// happens not to contradict: Cool Stuff Inc files three Exodias under
// "Promo", and the $170 Secret Rare and the $9 Quarter Century Rare were both
// answered with the $22 Ultra Rare.
//
// The code is read dashed or run together, because the feeds that need this
// most are the ones that drop the dash. The set is only adopted when it
// carries the card at that number: the number is the storefront's own claim,
// and one naming a set the card was never printed in says nothing about where
// the listing belongs. Nor when another set prints the card under that very
// number - the print-run twins are reissued under their original's numbers,
// and only the copyright date on the card tells those apart, which is the
// scraper's to read and not a code's to guess.
func numberSet(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, edition string) *mtgmatcher.Set {
	// Punctuation is what the normalizing drops first, so a bucket spelled
	// with nothing but it says as little as an absent one.
	if mtgmatcher.Normalize(edition) == "" {
		return nil
	}
	for _, field := range strings.Fields(inCard.Variation) {
		set := fieldSet(b, field)
		if set == nil {
			continue
		}
		full := editionNumberAt(set, inCard.Name, field)
		if full == "" || !onlySetAt(b, inCard.Name, full, set.Code) {
			continue
		}
		return set
	}
	return nil
}

// editionNumberAt answers the set's own spelling of the collector number the
// input fits, and nothing when the set prints the card at several numbers
// that fit - which decides no more than none at all.
func editionNumberAt(set *mtgmatcher.Set, name, number string) string {
	normalized := mtgmatcher.Normalize(name)
	var full string
	for i := range set.Cards {
		card := &set.Cards[i]
		if mtgmatcher.Normalize(card.Name) != normalized ||
			!numberMatches(number, card.Number) {
			continue
		}
		if full != "" && !strings.EqualFold(full, card.Number) {
			return ""
		}
		full = card.Number
	}
	return full
}

// onlySetAt reports whether the set is the only one printing the card under
// this collector number. The comparison is against the number as its own set
// spells it, never the storefront's: an input that dropped the dash compares
// on its digits alone, and every set numbering a reprint -005 would answer it.
func onlySetAt(b *mtgmatcher.Backend, name, full, code string) bool {
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed || co.SetCode == code {
			continue
		}
		if strings.EqualFold(co.Number, full) {
			return false
		}
	}
	return true
}

// numberTailRe matches what a collector number holds behind its set code: the
// language infix, the digits, and the letter tail a misprint reissue or a
// storefront's rarity suffix adds.
var numberTailRe = regexp.MustCompile(`^[A-Za-z]*[0-9]+[a-zA-Z]*$`)

// fieldSet reads one field of the variation as a collector number and answers
// the set its code names, or nothing when the field is not one. An undashed
// code is taken as long as it can be, so "UBP1EN005" is UBP1 rather than a
// shorter code that happens to open it. Longest wins is what keeps the answer
// off the map's iteration order too: two codes of one length cannot both open
// one field, so the longest fitting code is unique.
func fieldSet(b *mtgmatcher.Backend, field string) *mtgmatcher.Set {
	if prefix, tail, dashed := strings.Cut(field, "-"); dashed {
		if !numberTailRe.MatchString(tail) {
			return nil
		}
		return setByCode(b, prefix)
	}
	var best *mtgmatcher.Set
	var bestCode string
	for code, set := range b.Sets {
		if code == "" || len(code) >= len(field) ||
			!strings.EqualFold(field[:len(code)], code) ||
			!numberTailRe.MatchString(field[len(code):]) {
			continue
		}
		if best == nil || len(code) > len(bestCode) {
			best, bestCode = set, code
		}
	}
	return best
}

// setByCode answers the set filed under a code, however a storefront cased it.
func setByCode(b *mtgmatcher.Backend, code string) *mtgmatcher.Set {
	if set, found := b.Sets[code]; found {
		return set
	}
	for stored, set := range b.Sets {
		if strings.EqualFold(stored, code) {
			return set
		}
	}
	return nil
}

// trimEditionDecorations strips the game-name prefix and "Singles" suffix
// off a storefront's edition wording.
func trimEditionDecorations(edition string) string {
	for _, prefix := range []string{"Yu-Gi-Oh!", "Yu-Gi-Oh", "YuGiOh"} {
		if strings.HasPrefix(edition, prefix) {
			edition = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(edition, prefix), ":-"))
			break
		}
	}
	return strings.TrimSpace(strings.TrimSuffix(edition, "Singles"))
}

// namedSet answers the set an edition names: itself when the datastore
// carries it under that name, else whatever the alias table maps it onto.
func namedSet(b *mtgmatcher.Backend, edition string) (string, bool) {
	normalized := mtgmatcher.Normalize(edition)
	if _, found := b.NormalizedSets[normalized]; found {
		return edition, true
	}
	set, found := normalizedEditionAliases()[normalized]
	return set, found
}

// CanonicalFinish owns Yu-Gi-Oh's finish vocabulary, which is the print runs
// the catalog prices and nothing else. The runs are data rather than a fixed
// list - the datastore is the TCGplayer category, which is free to name a
// fourth - so a name is normalized and handed back, and the lookup against
// the printing's own runs is what decides. The shared names are the one
// refusal: the rarity is Yu-Gi-Oh's treatment and no product is sold as a
// foil, so nonfoil and foil name the flag slots the loader points at the
// default run rather than a finish anybody sells, and placing one would
// answer a bare foil flag with a print run it never asked for.
func (Rules) CanonicalFinish(name string) string {
	return canonicalFinish(name)
}

func canonicalFinish(name string) string {
	normalized := mtgmatcher.NormalizeFinish(name)
	if mtgmatcher.CanonicalFinish(normalized) != "" {
		return ""
	}
	return normalized
}

// FilterCards narrows candidates by edition, collector number, rarity and
// variant, in that order. Rarity only ever narrows on an explicit signal —
// the input wording spelling out a rarity, or the suffix map when only the
// number's tail speaks; a bare input facing several rarities keeps them all
// and surfaces as an aliasing error rather than a guess. The variant label
// tiering mirrors One Piece: a described label wins, a demanded-but-unnamed
// variant (the "a" tail) drops the base art, and a plain input keeps it.
func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	number := extractNumber(inCard.Variation)

	// A storefront writing the deck or volume index alone where the catalog
	// writes the whole set code - "5-001" for DL5-EN001, "05-001" for
	// YR05-EN001 - is saying the same number in fewer letters, and the strict
	// prefix compare throws it away. Reading the index as the code's own
	// digits recovers it, but the digits are a weak key on their own: "1-E002"
	// reads MC1-EN002 as readily as anything in DL. So the relaxation is
	// spent only inside the set the listing names, and a listing whose
	// edition names no set does not get it at all.
	var looseSet *mtgmatcher.Set
	if loosePrefixNumber(number) {
		looseSet = editionSet(b, inCard.Edition)
	}

	var candidates []mtgmatcher.Card
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		card := co.Card

		// A product's print-run siblings all file under the name bucket;
		// fold them onto their shared product id so each candidate appears
		// exactly once, and output() picks the run afterwards. The loader
		// writes each entry's uuid onto its Card, which rules the uuid out
		// as the folding key.
		key := card.Identifiers["tcgplayerProductId"]
		if key == "" {
			key = trimRunSuffix(uuid)
		}
		if seen[key] {
			continue
		}
		seen[key] = true

		if _, found := cardSet[card.SetCode]; !found {
			continue
		}
		if number != "" && !numberMatches(number, card.Number) &&
			!(looseSet != nil && strings.EqualFold(card.SetCode, looseSet.Code) &&
				loosePrefixMatches(number, card.Number)) {
			continue
		}
		// An input naming a print run re-keys the copy's FoilUUIDs so the
		// flag-driven resolution downstream lands on that run's entry. Both
		// slots move together: a run spans both foilnesses, so the vendor's
		// foil flag must not pull the resolution off it.
		uuid := finishUUID(b, inCard, &card)
		if uuid != "" {
			foilUUIDs := make(map[string]string, len(card.FoilUUIDs))
			maps.Copy(foilUUIDs, card.FoilUUIDs)
			foilUUIDs[mtgmatcher.FinishNonfoil] = uuid
			foilUUIDs[mtgmatcher.FinishFoil] = uuid
			card.FoilUUIDs = foilUUIDs
		}
		candidates = append(candidates, card)
	}
	if len(candidates) <= 1 {
		return candidates
	}

	// A verbatim collector number beats a suffix-stripped match, so the
	// "K" misprint reissues ("EOJ-EN004K") never pull in their base card.
	if number != "" {
		var exact []mtgmatcher.Card
		for _, card := range candidates {
			if strings.EqualFold(number, card.Number) {
				exact = append(exact, card)
			}
		}
		if len(exact) > 0 && len(exact) < len(candidates) {
			candidates = exact
		}
		if len(candidates) <= 1 {
			return candidates
		}
	}

	candidates = tierByRarity(inCard, candidates, number)
	if len(candidates) <= 1 {
		return candidates
	}
	return tierByVariant(inCard, candidates, number)
}

// tierByRarity keeps the candidates whose rarity the input's wording spells
// out, most specific description first ("Quarter Century Secret Rare" is not
// read as "Secret Rare"); with no described rarity, the collector number's
// suffix narrows through the suffix map. No signal keeps every candidate.
// Only the variation speaks: set names carry rarity words themselves
// ("McDonald's Promo").
func tierByRarity(inCard *mtgmatcher.InputCard, candidates []mtgmatcher.Card, number string) []mtgmatcher.Card {
	words := strings.Fields(strings.ToLower(inCard.Variation))

	described := map[string]bool{}
	for _, card := range candidates {
		rarity := strings.ToLower(card.Rarity)
		if !described[rarity] && allWordsIn(words, rarity) {
			described[rarity] = true
		}
	}
	for a := range described {
		for other := range described {
			if a != other && wordSubset(a, other) {
				delete(described, a)
				break
			}
		}
	}
	if len(described) > 0 {
		var out []mtgmatcher.Card
		for _, card := range candidates {
			if described[strings.ToLower(card.Rarity)] {
				out = append(out, card)
			}
		}
		return out
	}

	rarity := suffixRarity(number)
	if rarity == "" {
		return candidates
	}
	var out []mtgmatcher.Card
	for _, card := range candidates {
		if strings.EqualFold(card.Rarity, rarity) {
			out = append(out, card)
		}
	}
	return out
}

// tierByVariant splits the candidates into the ones whose variant label the
// input's variation describes, the base printings, and the variant
// printings. Only the variation is consulted: set names carry the color
// words the labels use ("Blue" against "Legend of Blue Eyes White Dragon").
func tierByVariant(inCard *mtgmatcher.InputCard, candidates []mtgmatcher.Card, number string) []mtgmatcher.Card {
	var base, variants []mtgmatcher.Card
	for _, card := range candidates {
		if len(card.PromoTypes) == 0 {
			base = append(base, card)
			continue
		}
		variants = append(variants, card)
	}
	// The tags are tokens now, so the wording's words are joined back up a
	// run at a time to ask whether they name them.
	described := mtgmatcher.DescribedVariants(inCard.Variation, variants)
	if len(described) > 0 {
		return described
	}
	if wantsVariant(number) {
		if len(variants) > 0 {
			return variants
		}
		return candidates
	}
	if len(base) > 0 {
		return base
	}
	return candidates
}

// allWordsIn reports whether the wording's words include every word of the
// label. Words compare whole: the one-letter artwork labels ("A") would
// otherwise hide inside almost any wording. They compare without their
// punctuation too, because the storefronts spell the game's premium tier
// both ways ("Collectors Rare" against the catalog's "Collector's Rare"),
// and a wording that names the rarity has to beat the number's suffix
// rather than fall through it onto every printing of the number.
func allWordsIn(words []string, label string) bool {
	labelWords := strings.Fields(strings.ToLower(label))
	if len(labelWords) == 0 {
		return false
	}
	for _, word := range labelWords {
		if !slices.ContainsFunc(words, func(spoken string) bool {
			return unpunctuated(spoken) == unpunctuated(word)
		}) {
			return false
		}
	}
	return true
}

// unpunctuated drops the marks a word is written with or without.
func unpunctuated(word string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, word)
}

// wordSubset reports whether a's words are a strict subset of b's.
func wordSubset(a, b string) bool {
	aWords := strings.Fields(a)
	bWords := strings.Fields(b)
	for _, word := range aWords {
		if !slices.Contains(bWords, word) {
			return false
		}
	}
	return len(bWords) > len(aWords)
}

// finishUUID resolves the entry the input names to its uuid. The finish
// field speaks first and through the shared resolution, so a caller pricing
// one sku names the run rather than spelling it into the wording; the
// wording is the fallback for the storefronts that only ever had one field.
func finishUUID(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, card *mtgmatcher.Card) string {
	if uuid := b.FinishUUID(card, inCard.Finish); uuid != "" {
		return uuid
	}
	return card.FoilUUIDs[selectFinish(inCard, card)]
}

// selectFinish maps the input wording's print-run tokens onto one of the
// card's stored runs, so a listing spelling its run out ("LOB-001 1st
// Edition") resolves to that run's entry instead of the default one. Only
// the variation speaks: set names carry the same words ("Limited Pack World
// Championship 2025"). Words compare whole, so "Unlimited" is never read as
// "Limited". A wording naming no run, or a run the product was not priced
// in, keeps the flag-driven default.
func selectFinish(inCard *mtgmatcher.InputCard, card *mtgmatcher.Card) string {
	var key string
	for word := range strings.FieldsSeq(strings.ToLower(inCard.Variation)) {
		switch word {
		case "1st", "first":
			key = finish1stEdition
		case "unlimited":
			key = finishUnlimited
		case "limited":
			key = finishLimited
		default:
			continue
		}
		break
	}
	if _, found := card.FoilUUIDs[key]; !found {
		return ""
	}
	return key
}

// extractNumber pulls the collector number out of the scraper-supplied
// Variation: the full "RA01-EN019" shape when present (rarity tail and
// all), else a cardtrader-style bare tail ("019qsec"). A bare digit run is
// only believed when it fits the game's three-digit numbering and any tail
// is a known rarity suffix, so the years variant labels start with ("2012
// Pre-registration") are never read as numbers.
//
// A letter-led tail is read last of all. The deck reprints number their
// cards by the deck the card came in - Legendary Decks II files
// Polymerization at LDK2-ENJ26 beside LDK2-ENK22 - and cardtrader publishes
// that tail bare, as "J26". Read after both other shapes, so a full number
// and a plain digit run keep their precedence, and only where nothing else
// answered: outside a number a letter-led field is a label word, and
// letteredNumber's own shape - one letter, then digits, four bytes at most -
// is what tells the two apart.
func extractNumber(variation string) string {
	fields := strings.Fields(variation)
	for _, field := range fields {
		if fullNumberRe.MatchString(field) {
			return field
		}
	}
	for _, field := range fields {
		if field[0] < '0' || field[0] > '9' {
			continue
		}
		digits := strings.TrimRight(field, letters)
		if len(digits) > 3 || strings.ContainsFunc(digits, func(r rune) bool {
			return r < '0' || r > '9'
		}) {
			continue
		}
		suffix := strings.ToLower(field[len(digits):])
		if suffix == "" || suffix == "a" || suffixRarity(field) != "" {
			return field
		}
	}
	return letteredNumber(variation)
}

// numberSuffix returns the lowercase letter tail of a collector number.
func numberSuffix(number string) string {
	return strings.ToLower(number[len(strings.TrimRight(number, letters)):])
}

// suffixRarity resolves a collector number's letter tail through the
// cardtrader suffix map, tolerating the further "a" alternate-art tail.
func suffixRarity(number string) string {
	suffix := numberSuffix(number)
	if rarity, found := suffixRarities[suffix]; found {
		return rarity
	}
	if strings.HasSuffix(suffix, "a") {
		if rarity, found := suffixRarities[strings.TrimSuffix(suffix, "a")]; found {
			return rarity
		}
	}
	return ""
}

// wantsVariant reports whether the collector number's tail demands some
// variant printing without describing which: the "a" cardtrader appends to
// its alternate arts, alone or after a rarity suffix.
func wantsVariant(number string) bool {
	suffix := numberSuffix(number)
	if !strings.HasSuffix(suffix, "a") {
		return false
	}
	if suffix == "a" {
		return true
	}
	_, found := suffixRarities[strings.TrimSuffix(suffix, "a")]
	return found
}

// numberMatches compares an input number against a printing's full
// collector number: equal full codes (rarity tail stripped or not), or a
// matching numeric tail — with equal set prefixes when the input carries
// one, and whatever the language infix ("LOB-EN001" matches "LOB-001").
// Leading zeros never decide ("19" matches "RA01-EN019").
func numberMatches(input, full string) bool {
	if strings.EqualFold(input, full) {
		return true
	}
	input = strings.TrimRight(input, letters)
	if strings.EqualFold(input, full) {
		return true
	}
	inPrefix, inTail, inDashed := strings.Cut(input, "-")
	if !inDashed {
		inTail = input
	}
	_, fullTail, _ := strings.Cut(full, "-")
	if inDashed {
		fullPrefix, _, _ := strings.Cut(full, "-")
		if !strings.EqualFold(inPrefix, fullPrefix) {
			return false
		}
	}
	inDigits := digitRun(inTail)
	fullDigits := digitRun(fullTail)
	return inDigits != "" && fullDigits != "" &&
		canonicalTail(inDigits) == canonicalTail(fullDigits)
}

// loosePrefixNumber reports whether an input number is dashed with a prefix
// made of digits alone, the shape a storefront writes a deck or volume index
// in where the catalog writes the whole set code ("5-001" for DL5-EN001).
func loosePrefixNumber(number string) bool {
	prefix, _, dashed := strings.Cut(number, "-")
	return dashed && prefix != "" && digitRun(prefix) == prefix
}

// loosePrefixMatches compares such a number against a printing's own, reading
// the input's prefix as the digits of the printing's set code and the tails as
// numberMatches reads them. The set code's letters are what it stops asking
// for, which is why the caller only asks inside one set.
func loosePrefixMatches(input, full string) bool {
	inPrefix, inTail, _ := strings.Cut(input, "-")
	fullPrefix, fullTail, dashed := strings.Cut(full, "-")
	if !dashed {
		return false
	}
	fullCode := digitRun(fullPrefix)
	if fullCode == "" || canonicalTail(inPrefix) != canonicalTail(fullCode) {
		return false
	}
	inDigits := digitRun(inTail)
	fullDigits := digitRun(fullTail)
	return inDigits != "" && fullDigits != "" &&
		canonicalTail(inDigits) == canonicalTail(fullDigits)
}

// digitRun extracts the trailing digit run of a collector number tail, the
// letter tail aside: "EN004K" yields "004".
func digitRun(tail string) string {
	tail = strings.TrimRight(tail, letters)
	idx := strings.LastIndexFunc(tail, func(r rune) bool {
		return r < '0' || r > '9'
	})
	return tail[idx+1:]
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
