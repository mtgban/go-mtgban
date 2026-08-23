package fleshandblood

import (
	"maps"
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for Flesh and Blood. A card is
// identified by name + collector number, with the variant label separating
// the printings that share a number (extended arts, marvels and the like).
// Finish never gates anything: every print run and treatment of a product
// matches, resolving to the default entry of the requested foilness — but a
// named finish routes the match onto the entry it names, whether it arrives
// in the finish field or as the words storefronts append ("Rainbow Foil",
// "Cold Foil", the print-run edition suffixes).
type Rules struct{ mtgmatcher.DefaultRules }

// fullNumberRe matches the game's collector number shapes: "WTR215",
// "1HP408", with an optional letter tail (cardtrader suffixes marvels
// "MST238m"). The dashed numbers ("MST158-A") are left to the variant
// wording, which carries the same tail.
var fullNumberRe = regexp.MustCompile(`^[0-9]?[A-Za-z]+[0-9]{1,4}[a-zA-Z]?$`)

// pairNumberRe matches the fused-card numbers ("WTR040 // WTR039",
// cardtrader's compact "UPR002//UPR165") so the pair survives extraction
// whole instead of field splitting cutting it at the separator.
var pairNumberRe = regexp.MustCompile(`[0-9]?[A-Za-z]+[0-9]{1,4}\s*/{1,2}\s*[0-9]?[A-Za-z]+[0-9]{1,4}`)

const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// Prefilter splits the parenthetical decorations off the name before the
// canonical-name lookup: storefronts write "Enigma, New Moon (Marvel)" for
// a variant the datastore labels beside the plain name. A full name that is
// itself canonical stays as it is — the pitch-color parentheticals ("Sink
// Below (Red)") are part of the name.
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	adjustQualifier(b, inCard)
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

// qualifierRe matches the parenthetical a Flesh and Blood name is qualified
// with. There are only four in the game: the three pitch colors and the
// marvel treatment.
var qualifierRe = regexp.MustCompile(`\s*\((?i:Red|Yellow|Blue|Marvel)\)$`)

// qualifiers are the spellings a qualified name is tried in, the empty one
// standing for the undecorated name.
var qualifiers = []string{"", " (Red)", " (Yellow)", " (Blue)", " (Marvel)"}

// adjustQualifier re-spells the name's qualifier as the printing the input's
// collector number names spells it.
//
// The datastore mirrors the storefront each set was catalogued from, and the
// storefronts do not agree with each other: the same card is "Hyper Driver" at
// ARC036 and "Hyper Driver (Red)" at DYN110, and "Bloodrot Trap" at ARA019 but
// "Bloodrot Trap (Red)" at OUT171. Whichever spelling a feed sends, it is a
// canonical name of the other set's printing, so the name lookup succeeds, the
// number then matches nothing under it, and the printing the feed actually
// named is unreachable — in both directions.
//
// The collector number is what makes the re-spelling safe, and it has to be a
// full one: a pitch color is part of the name, so "Sink Below (Red)" and "Sink
// Below (Blue)" are two different cards, and only a number carrying its set
// code names one of them. A bare "036" is every set's thirty-sixth card and
// must never license a rename. An input whose own spelling has a printing at
// the number is already right, and an ambiguous one is left alone. The
// parenthetical the re-spelling drops is handed to the wording, since the same
// shape also spells a treatment label rather than a piece of the name.
func adjustQualifier(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	number := extractNumber(inCard.Variation)
	if !fullNumberRe.MatchString(number) && !pairNumberRe.MatchString(number) {
		return
	}
	if numberedAs(b, inCard.Name, number) {
		return
	}
	base := qualifierRe.ReplaceAllString(inCard.Name, "")
	var match string
	for _, qualifier := range qualifiers {
		candidate := base + qualifier
		if !numberedAs(b, candidate, number) {
			continue
		}
		if match != "" {
			return
		}
		match = candidate
	}
	if match == "" {
		return
	}
	// The respelling swaps the input's parenthetical for the one the
	// numbered printing wears, and the two do not always say the same
	// thing. A pitch color belongs to the name, so swapping one for
	// another loses nothing; but the very same parenthetical also spells a
	// treatment label that tells the printings sharing a number apart —
	// "Golden Skull (Yellow)" is a name, and "(Marvel)" is the label on one
	// of the two printings filed under it. Handing the dropped
	// parenthetical to the wording keeps that printing reachable. Where no
	// printing at the number wears the label the wording describes nothing
	// and the tiering is the respelling's own.
	dropped := qualifierWord(inCard.Name)
	if dropped != "" && !strings.EqualFold(dropped, qualifierWord(match)) {
		inCard.AddToVariant(dropped)
	}
	inCard.Name = match
}

// qualifierWord returns the label inside a name's qualifier parenthetical,
// empty for a name carrying none.
func qualifierWord(name string) string {
	return strings.Trim(strings.TrimSpace(qualifierRe.FindString(name)), "()")
}

// numberedAs reports whether any printing filed under the name carries the
// collector number.
func numberedAs(b *mtgmatcher.Backend, name, number string) bool {
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		if numberMatches(number, co.Number) {
			return true
		}
	}
	return false
}

// AdjustName provides a prefix fallback for truncated feeds, adopting the
// one name among the prefix matches that carries the input's number. The
// pitch-color names resolve here: a bare "Breakneck Battery" extends into
// "Breakneck Battery (Red)" when the number picks the red pitch. Names
// compare normalized, so punctuation variants of one name are not read as
// ambiguity.
func (Rules) AdjustName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
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

// AdjustEdition trims the game-name prefix and the "Singles" suffix
// storefronts decorate set names with, and moves the print-run suffixes
// cardtrader spells its expansions with ("Welcome to Rathe - 1st Edition")
// into the variation: the datastore's sets carry no print run, but the
// wording now names the priced entry selectFinish resolves. An edition
// that still matches no set simply does not narrow the candidates.
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := strings.TrimSpace(inCard.Edition)
	for _, prefix := range []string{"Flesh and Blood", "Flesh & Blood"} {
		if strings.HasPrefix(edition, prefix) {
			edition = strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(edition, prefix), ":-"))
			break
		}
	}
	edition = strings.TrimSpace(strings.TrimSuffix(edition, "Singles"))
	for _, suffix := range []string{"1st Edition", "Unlimited Edition", "Unlimited", "Alpha Print Run"} {
		trimmed := strings.TrimSuffix(edition, suffix)
		if trimmed != edition {
			edition = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(trimmed), "-"))
			inCard.AddToVariant(suffix)
			break
		}
	}
	inCard.Edition = edition
}

// CanonicalFinish owns Flesh and Blood's finish vocabulary, which is the
// print run crossed with the treatment, as the catalog prices them ("Cold
// Foil", "1st Edition Rainbow Foil"). The combinations are data rather than
// a fixed list - a new treatment arrives with a set - so a name is
// normalized and handed back, and the lookup against the printing's own
// combinations is what decides. Nothing is special-cased: the game's own
// names normalize to themselves, and the shared names normalize onto the
// flag slots they already mean, so a source with a bare foilness reaches
// the same printing the flag would. The bare treatments stay finishes of
// their own rather than folding into the shared pair - a product sold in
// both a bare and a 1st Edition Normal keeps them apart - and a printing
// missing the bare one registers the alias that reaches its own.
func (Rules) CanonicalFinish(name string) string {
	return canonicalFinish(name)
}

func canonicalFinish(name string) string {
	return mtgmatcher.NormalizeFinish(name)
}

// FilterCards narrows candidates by edition, collector number and variant.
// The variant tiering mirrors the number sharing in the data: when the
// input's wording describes a variant label, the printings it describes
// win; a bare input keeps the base printing when one exists; the letter
// tail cardtrader appends to a marvel's number demands some variant without
// saying which.
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

		// A product's finish siblings all file under the name bucket; fold
		// them onto their shared product id so each candidate appears
		// exactly once, and output() picks the finish afterwards. The
		// loader writes each entry's uuid onto its Card, which rules the
		// uuid out as the folding key.
		key := card.Identifiers["tcgplayerProductId"]
		if key == "" {
			key = trimFinishSuffix(uuid)
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
		// An input naming a print run or a treatment re-keys the copy's
		// FoilUUIDs so the flag-driven resolution downstream lands on that
		// printing's entry. Both slots move together: a printing spans one
		// exact foilness, so the vendor's foil flag must not pull the
		// resolution off it.
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

	described, base, variants := tierByVariant(inCard, candidates)
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

// tierByVariant splits the candidates into the ones whose variant label the
// input's variation describes, the base printings, and the variant
// printings. Only the variation is consulted: set names carry the labels'
// words. When several labels are described, the finish tokens step aside —
// a label spelled from them alone ("Cold Foil" the label) defers to the
// label the rest of the wording still describes.
func tierByVariant(inCard *mtgmatcher.InputCard, candidates []mtgmatcher.Card) (described, base, variants []mtgmatcher.Card) {
	words := strings.Fields(strings.ToLower(inCard.Variation))
	for _, card := range candidates {
		if len(card.PromoTypes) == 0 {
			base = append(base, card)
			continue
		}
		variants = append(variants, card)
	}
	// The tags are tokens now, so the wording's words are joined back up a
	// run at a time to ask whether they name them.
	described = mtgmatcher.DescribedVariants(inCard.Variation, variants)
	if len(described) > 1 {
		var bare []string
		for _, word := range words {
			if !finishToken(word) {
				bare = append(bare, word)
			}
		}
		refined := mtgmatcher.DescribedVariants(strings.Join(bare, " "), described)
		if len(refined) > 0 {
			described = refined
		}
	}
	return
}

// finishToken reports whether selectFinish consumes the word.
func finishToken(word string) bool {
	switch word {
	case "1st", "first", "unlimited", "alpha", "rainbow", "cold", "normal":
		return true
	}
	return false
}

// wantsVariant reports whether the input demands some variant printing
// without describing which: the lowercase letter tail cardtrader appends to
// the collector number ("MST238m"). The uppercase dashed tails ("MST158-A")
// are real collector numbers, not demands.
func wantsVariant(number string) bool {
	if number == "" {
		return false
	}
	last := number[len(number)-1]
	return last >= 'a' && last <= 'z'
}

// finishUUID resolves the entry the input names to its uuid. The finish
// field speaks first and through the shared resolution, which is what walks
// the printing's aliases onto the print run it does carry, so a caller
// pricing one sku names the treatment rather than spelling it into the
// wording; the wording is the fallback for the storefronts that only ever
// had one field, and the only one that can name the two axes separately.
func finishUUID(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, card *mtgmatcher.Card) string {
	if uuid := b.FinishUUID(card, inCard.Finish); uuid != "" {
		return uuid
	}
	return card.FoilUUIDs[selectFinish(inCard, card)]
}

// selectFinish maps the input wording's print-run and treatment tokens
// onto one of the card's stored printings, so a listing spelling its
// finish out (TCGplayer's "1st Edition Rainbow Foil" printing name in the
// variation, cardtrader's "1st Edition" expansion suffix) resolves to that
// printing's entry instead of the flag-driven default. Only the variation
// speaks: set names carry the same words ("1st Strike"). The unsaid axis
// fills from the card itself — a bare print run is its plain printing, a
// bare treatment takes the plainest print run sold with it. A wording
// naming no finish, or a printing the product was not priced in, keeps the
// flag-driven default.
func selectFinish(inCard *mtgmatcher.InputCard, card *mtgmatcher.Card) string {
	var edition, treatment string
	for word := range strings.FieldsSeq(strings.ToLower(inCard.Variation)) {
		switch word {
		case "1st", "first", "alpha":
			// cardtrader's alpha print run sells as TCGplayer's 1st Edition
			edition = edition1st
		case "unlimited":
			edition = editionUnlimited
		case "rainbow":
			treatment = treatmentRainbowFoil
		case "cold":
			treatment = treatmentColdFoil
		case "normal":
			treatment = treatmentNormal
		}
	}
	if edition == "" && treatment == "" {
		return ""
	}
	if treatment == "" {
		treatment = treatmentNormal
	}
	editions := []string{edition}
	if edition == "" {
		editions = []string{editionBare, editionUnlimited, edition1st}
	}
	for _, prefix := range editions {
		if _, found := card.FoilUUIDs[prefix+treatment]; found {
			return prefix + treatment
		}
	}
	return ""
}

// extractNumber pulls the collector number out of the scraper-supplied
// Variation: a fused pair whole ("WTR040 // WTR039"), then the full
// "WTR215" shape, else the first digit-leading field ("215").
func extractNumber(variation string) string {
	pair := pairNumberRe.FindString(variation)
	if pair != "" {
		return pair
	}
	fields := strings.Fields(variation)
	for _, field := range fields {
		if fullNumberRe.MatchString(field) {
			return field
		}
	}
	for _, field := range fields {
		// The print-run ordinal reads digit-first but is never a number
		// ("1st Edition")
		if strings.EqualFold(field, "1st") {
			continue
		}
		if field[0] >= '0' && field[0] <= '9' {
			return strings.Split(field, "/")[0]
		}
	}
	return ""
}

// canonicalNumber folds the pair-separator spellings together ("WTR040 //
// WTR039", "UPR002//UPR165" and "MON219 / MON220" all become
// "WTR040/WTR039"-shaped) so the pairs compare by content.
func canonicalNumber(number string) string {
	fields := strings.FieldsFunc(number, func(r rune) bool {
		return r == ' ' || r == '/'
	})
	return strings.Join(fields, "/")
}

// numberMatches compares an input number against a printing's collector
// number: equal codes, the same number written with different padding, a
// letter-tailed input matching its base number, a pair matched by its front
// half, or a bare digit input (its own letter tail aside, "238m") matching
// the digit tail of the front code. Leading zeros never decide ("40"
// matches "WTR040").
func numberMatches(input, full string) bool {
	ci := canonicalNumber(input)
	cf := canonicalNumber(full)
	if strings.EqualFold(ci, cf) {
		return true
	}
	// The catalog pads a few numbers a digit wider than the printing wears
	// them - "JDG0077" for JDG077 - and a storefront writes whichever it was
	// given. Folding the padding away compares what the number says rather
	// than how wide it was written; no two numbers in a set fold together.
	if foldNumber(ci) == foldNumber(cf) {
		return true
	}
	trimmed := strings.TrimRight(ci, letters)
	if trimmed != ci && strings.EqualFold(trimmed, cf) {
		return true
	}
	front, _, _ := strings.Cut(cf, "/")
	if strings.Contains(cf, "/") && strings.EqualFold(ci, front) {
		return true
	}
	inFront, _, _ := strings.Cut(ci, "/")
	inFront = strings.TrimRight(inFront, letters)
	return isAllDigits(inFront) && canonicalTail(inFront) == canonicalTail(digitTail(front))
}

func isAllDigits(number string) bool {
	if number == "" {
		return false
	}
	for i := 0; i < len(number); i++ {
		if number[i] < '0' || number[i] > '9' {
			return false
		}
	}
	return true
}

// digitTail returns the trailing digit run of a collector code ("215" from
// "WTR215"), empty when the code ends in a letter ("MST158-A").
func digitTail(number string) string {
	i := len(number)
	for i > 0 && number[i-1] >= '0' && number[i-1] <= '9' {
		i--
	}
	return number[i:]
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
