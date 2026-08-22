package yugioh

import (
	"regexp"
	"slices"
	"strings"

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
// qualifier parenthetical ("Dark Magician (A)").
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	if strings.Contains(inCard.Name, "(") {
		vars := mtgmatcher.SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.AddToVariant(strings.Join(vars[1:], " "))
		}
	}
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
	if base, cut := strings.CutSuffix(inCard.Name, " Token"); cut {
		flipped := "Token: " + base
		if _, found := b.CanonicalNames[mtgmatcher.Normalize(flipped)]; found {
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
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := strings.TrimSpace(inCard.Edition)
	if named, found := namedSet(b, edition); found {
		inCard.Edition = named
		return
	}
	for _, prefix := range []string{"Yu-Gi-Oh!", "Yu-Gi-Oh", "YuGiOh"} {
		if strings.HasPrefix(edition, prefix) {
			edition = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(edition, prefix), ":-"))
			break
		}
	}
	edition = strings.TrimSpace(strings.TrimSuffix(edition, "Singles"))
	if named, found := namedSet(b, edition); found {
		edition = named
	}
	inCard.Edition = edition
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
		if number != "" && !numberMatches(number, card.Number) {
			continue
		}
		// An input naming a print run re-keys the copy's FoilUUIDs so the
		// flag-driven resolution downstream lands on that run's entry. Both
		// slots move together: a run spans both foilnesses, so the vendor's
		// foil flag must not pull the resolution off it.
		uuid := finishUUID(b, inCard, &card)
		if uuid != "" {
			foilUUIDs := make(map[string]string, len(card.FoilUUIDs))
			for k, v := range card.FoilUUIDs {
				foilUUIDs[k] = v
			}
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
// otherwise hide inside almost any wording.
func allWordsIn(words []string, label string) bool {
	labelWords := strings.Fields(strings.ToLower(label))
	if len(labelWords) == 0 {
		return false
	}
	for _, word := range labelWords {
		if !slices.Contains(words, word) {
			return false
		}
	}
	return true
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
	for _, word := range strings.Fields(strings.ToLower(inCard.Variation)) {
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
	return ""
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
