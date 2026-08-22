package pokemon

import (
	"regexp"
	"strings"
	"sync"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// Rules implements mtgmatcher.GameRules for Pokemon. A card is identified by
// name + collector number, with the qualifier label breaking ties: the same
// number recurs as several products differing only by the treatment or the
// promotion TCGplayer decorates the name with, and that label is the only
// thing between them.
//
// The foil treatment is real foilness here, unlike the Yu-Gi-Oh print runs:
// a storefront's foil flag has to reach the Holofoil printing. So the flag
// resolves through the loader's FoilUUIDs, and only an input naming a
// treatment outright re-keys onto the exact crossing it names.
type Rules struct{ mtgmatcher.DefaultRules }

// fullNumberRe matches the game's collector number shapes: "001/102",
// "SWSH001", "TG01/TG30", "H1/H32", with the set total that follows a slash
// kept out of the capture.
var fullNumberRe = regexp.MustCompile(`(?i)\b([A-Z]{0,4}\d+[a-z]?)(?:/\d+)?\b`)

// numberTailRe matches a collector number standing alone, which is the shape
// the storefronts writing one into the name leave behind once the name is
// split off it.
var numberTailRe = regexp.MustCompile(`(?i)^[A-Z]{0,4}\d+[a-z]?(?:/[A-Z]{0,4}\d+)?$`)

// Prefilter splits off the decorations storefronts write into the name: the
// parentheticals TCGplayer adds ("Pikachu (Cosmos Holo)", "Charizard
// (Prerelease)"), and the collector number Cool Stuff Inc glues on after a
// dash ("Wingull - 70/100"). A name that is itself canonical stays as it is,
// which is what keeps the real card names carrying either decoration - a
// parenthetical of their own, or the World Championship reprints named for
// the number they reprint - from being taken apart. The check runs again
// between the two splits, because removing the parenthetical is what makes
// such a name recognisable.
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	canonical := func() bool {
		_, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]
		return found
	}
	if canonical() {
		return
	}
	if strings.Contains(inCard.Name, "(") {
		vars := mtgmatcher.SplitVariants(inCard.Name)
		if len(vars) > 1 {
			inCard.Name = vars[0]
			inCard.AddToVariant(strings.Join(vars[1:], " "))
		}
	}
	if canonical() {
		return
	}
	index := strings.LastIndex(inCard.Name, " - ")
	if index < 0 {
		return
	}
	name := strings.TrimSpace(inCard.Name[:index])
	number := strings.TrimSpace(inCard.Name[index+len(" - "):])
	if name == "" || !numberTailRe.MatchString(number) {
		return
	}
	inCard.Name = name
	inCard.AddToVariant(number)
}

// AdjustName provides a prefix fallback for truncated feeds, adopting the
// one name among the prefix matches that carries the input's number. Names
// compare normalized, so punctuation variants of one name are not read as
// ambiguity.
func (Rules) AdjustName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	numbers := extractNumbers(inCard.Variation)
	if len(numbers) == 0 {
		return
	}
	uuids, err := b.SearchHasPrefix(inCard.Name)
	if err != nil {
		return
	}
	name := ""
	for _, uuid := range uuids {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		if !numbersMatchCard(b, numbers, &co.Card) {
			continue
		}
		if name != "" && mtgmatcher.Normalize(name) != mtgmatcher.Normalize(co.Name) {
			return
		}
		name = co.Name
	}
	if name != "" {
		inCard.Name = name
	}
}

// AdjustEdition trims the game-name prefix and "Singles" suffix storefronts
// decorate set names with, and drops the headings that name no set of ours.
func (Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := strings.TrimSpace(inCard.Edition)
	for _, prefix := range []string{"Pokemon TCG", "Pokemon", "Pokémon"} {
		if strings.HasPrefix(edition, prefix) {
			trimmed := strings.TrimSpace(strings.TrimLeft(strings.TrimPrefix(edition, prefix), ":-"))
			// A set really named for the game keeps its name: "Pokemon Go"
			// and "Pokemon Futsal Promos" are sets, not decorations.
			if _, err := b.GetSetByName(trimmed); err == nil {
				edition = trimmed
			}
			break
		}
	}
	edition = strings.TrimSpace(strings.TrimSuffix(edition, "Singles"))
	// The generic promo headings resolve to no set on purpose: the heading
	// spans every promotional set the game has - and it has more than
	// thirty - so it can gate the promo names but must not narrow the
	// candidates to whichever one happens to wear the name.
	if mtgmatcher.IsPromoHeading(edition) {
		edition = ""
	}
	// A storefront spelling the era but not the set's number within it
	// ("SWSH Darkness Ablaze") names a set the catalog calls something
	// longer ("SWSH03: Darkness Ablaze"); ask what the two have in common
	// rather than leaving the edition to narrow nothing.
	// The lookups here are the direct ones on purpose: GetSetByName asks
	// the rules to adjust the edition, which is this function.
	if edition != "" {
		_, known := b.NormalizedSets[mtgmatcher.Normalize(edition)]
		if !known {
			_, err := b.GetSet(edition)
			known = err == nil
		}
		if !known {
			name := setNamedByHead(b, edition)
			if name == "" {
				name = setNamedByTail(b, edition)
			}
			if name != "" {
				edition = name
			}
		}
	}
	inCard.Edition = edition

	widenQualifiedName(b, inCard)
}

// widenQualifiedName adopts the qualified spelling of a name the catalog
// bakes the disambiguator into. "Accelgor (Team Plasma)" is how the catalog
// writes the 8/101 of Plasma Blast and "Professor's Research [Professor
// Juniper]" the 085/086 of Black Bolt, while the storefront writes the bare
// name and the collector number - so the name hash holds nothing in the set
// the listing names and the row dies on the number instead.
//
// It runs once the edition is resolved, and only when no printing named
// exactly as the input carries that number there: a name the catalog already
// knows at that number is never rewritten. Two qualified spellings sharing
// the number mean the listing does not say which, so ambiguity is left for
// the caller to report rather than guessed at.
func widenQualifiedName(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	numbers := extractNumbers(inCard.Variation)
	if len(numbers) == 0 {
		return
	}
	code := editionSetCode(b, inCard.Edition)
	inEdition := func(co *mtgmatcher.CardObject) bool {
		return code == "" || co.SetCode == code
	}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed || !inEdition(co) {
			continue
		}
		if numbersMatchCard(b, numbers, &co.Card) {
			return
		}
	}
	uuids, err := b.SearchHasPrefix(inCard.Name)
	if err != nil {
		return
	}
	name := ""
	for _, uuid := range uuids {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed || !inEdition(co) {
			continue
		}
		if !numbersMatchCard(b, numbers, &co.Card) {
			continue
		}
		if name != "" && mtgmatcher.Normalize(name) != mtgmatcher.Normalize(co.Name) {
			return
		}
		name = co.Name
	}
	if name != "" {
		inCard.Name = name
	}
}

// editionSetCode answers the code of the set an already-adjusted edition
// names, and an empty string when it names none. The lookups are the direct
// ones because GetSetByName would ask the rules to adjust the edition again.
func editionSetCode(b *mtgmatcher.Backend, edition string) string {
	if edition == "" {
		return ""
	}
	if set, found := b.NormalizedSets[mtgmatcher.Normalize(edition)]; found {
		return set.Code
	}
	if set, err := b.GetSet(edition); err == nil {
		return set.Code
	}
	return ""
}

// setNamedByHead answers the set a storefront decorates with a trailing
// "Base Set", which is how three eras' first sets are headed: "Diamond and
// Pearl Base Set", "Platinum Base Set", "Expedition Base Set". Dropping the
// leading words instead lands every one of them on the set literally named
// "Base Set", 1999's, and the row then either dies or is priced as a card it
// is not.
//
// Only the trailing "Base Set" comes off, and only onto a name that is a set
// outright. Dropping trailing words in general is what must not happen:
// "Diamond and Pearl Stormfront" would resolve to "Diamond and Pearl", and
// measured that way it cost 335 previously correct entries.
func setNamedByHead(b *mtgmatcher.Backend, edition string) string {
	head := strings.TrimSpace(strings.TrimSuffix(edition, "Base Set"))
	if head == "" || head == edition {
		return ""
	}
	if _, known := b.NormalizedSets[mtgmatcher.Normalize(head)]; !known {
		return ""
	}
	return head
}

// setTails indexes each backend's sets by the name left after the catalog's
// era-and-number prefix, built once per datastore since the sets do not
// change after it is loaded.
var setTails sync.Map

// setTail drops the prefix the catalog decorates a set name with: both
// "SWSH03: Darkness Ablaze" and "XY - Steam Siege" are an era, a separator,
// and the name the set is actually known by.
func setTail(name string) string {
	if _, after, found := strings.Cut(name, ": "); found {
		return after
	}
	if _, after, found := strings.Cut(name, " - "); found {
		return after
	}
	return name
}

// setNamedByTail answers the set an era-prefixed spelling means. Storefronts
// write the era and the name where the catalog writes the era, the set's
// number within it, and the name, so what follows either prefix is what has
// to agree. Leading words are dropped one at a time, since how much of the
// prefix is era is not knowable in advance - "Diamond and Pearl Great
// Encounters" spends three words on it, and Cardmarket spends none at all,
// which is why the whole name is tried before any of it is dropped. A tail
// two sets share names neither.
func setNamedByTail(b *mtgmatcher.Backend, edition string) string {
	cached, found := setTails.Load(b)
	if !found {
		built := map[string][]string{}
		for _, set := range b.Sets {
			tail := mtgmatcher.Normalize(setTail(set.Name))
			built[tail] = append(built[tail], set.Name)
		}
		cached, _ = setTails.LoadOrStore(b, built)
	}
	index, ok := cached.(map[string][]string)
	if !ok {
		return ""
	}

	fields := strings.Fields(edition)
	for i := 0; i < len(fields); i++ {
		names := index[mtgmatcher.Normalize(strings.Join(fields[i:], " "))]
		if len(names) == 1 {
			return names[0]
		}
		if len(names) > 1 {
			return ""
		}
	}
	return ""
}

// FilterCards narrows candidates by edition, collector number and label, in
// that order. A bare input facing several labels keeps them all and surfaces
// as an aliasing error rather than a guess.
func (Rules) FilterCards(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	// A storefront can write more than one number into the wording: Cool
	// Stuff Inc prepends its own catalogue field, which is empty, "0" or
	// "001" for whole editions, and the number the listing prints is the
	// one Prefilter appends after it. Read them from the back, so the
	// number the card carries on it is asked first and the vendor's index
	// only answers when nothing else does.
	numbers := extractNumbers(inCard.Variation)
	number := ""
	candidates := filterByNumber(b, inCard, cardSet, "")
	for i := len(numbers) - 1; i >= 0; i-- {
		number = numbers[i]
		candidates = filterByNumber(b, inCard, cardSet, number)
		if len(candidates) > 0 {
			break
		}
	}
	if len(candidates) <= 1 {
		return candidates
	}

	// A verbatim collector number beats a prefix-folded match.
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

	return tierByLabel(inCard, candidates)
}

// filterByNumber collects the printings of the input's name that the edition
// admits and that carry the given collector number, one per product. An
// empty number asks for every printing the edition admits.
func filterByNumber(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card, number string) []mtgmatcher.Card {
	var candidates []mtgmatcher.Card
	seen := map[string]bool{}
	for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
		co, found := b.UUIDs[uuid]
		if !found || co.Sealed {
			continue
		}
		card := co.Card

		// A product's printing siblings all file under the name bucket;
		// fold them onto their shared product id so each candidate appears
		// exactly once, and output() picks the printing afterwards. The
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
		if number != "" && !numberMatchesCard(b, number, &card) {
			continue
		}
		// An input naming a treatment re-keys the copy's FoilUUIDs so the
		// flag-driven resolution downstream lands on that printing. Both
		// slots move together: the input named the printing outright, and
		// a storefront's foil flag is the less reliable of the two.
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
	return candidates
}

// tierByLabel splits the candidates into the ones whose label the input's
// variation describes, the base printings, and the labelled printings. Only
// the variation is consulted: set names carry the label words themselves
// ("Team Plasma" is a label and part of three set names).
func tierByLabel(inCard *mtgmatcher.InputCard, candidates []mtgmatcher.Card) []mtgmatcher.Card {
	var described, base, labelled []mtgmatcher.Card
	for _, card := range candidates {
		if len(card.PromoTypes) == 0 {
			base = append(base, card)
			continue
		}
		labelled = append(labelled, card)
		// The tag is a token, so the wording's words are joined back up a
		// run at a time to ask whether they name it.
		for _, promoType := range card.PromoTypes {
			if mtgmatcher.SlugDescribes(inCard.Variation, promoType) {
				described = append(described, card)
				break
			}
		}
	}
	if len(described) > 0 {
		return described
	}
	if len(base) > 0 {
		return base
	}
	return labelled
}

// finishUUID resolves the entry an input's finish names, from the finish
// field where a caller fills one in and from the variation wording where
// only the listing speaks.
func finishUUID(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, card *mtgmatcher.Card) string {
	if uuid := b.FinishUUID(card, inCard.Finish); uuid != "" {
		return uuid
	}
	return card.FoilUUIDs[selectFinish(inCard, card)]
}

// selectFinish maps the input wording's treatment tokens onto one of the
// card's stored printings, so a listing spelling its printing out ("Reverse
// Holo", "1st Edition") resolves to that entry instead of the default one.
//
// The wording names axes rather than a printing: it can say a run, a
// treatment, or both, and the printing that answers is the stored one naming
// everything the wording asked for and the least beside it. That is what
// lets "1st Edition" reach the 1st Edition Holofoil on a card sold in no
// other first-edition printing, while still preferring the plain 1st Edition
// on a card that has one.
//
// Only the variation speaks: set names carry the same words ("Unlimited" and
// "1st Edition" name the base-set reprints). A wording naming no axis, or a
// printing the product was not priced in, keeps the flag-driven default.
func selectFinish(inCard *mtgmatcher.InputCard, card *mtgmatcher.Card) string {
	words := strings.Fields(strings.ToLower(inCard.Variation))

	var wanted []string
	if hasAllTokens(words, []string{"reverse"}) {
		wanted = append(wanted, "reverse")
	}
	if hasAllTokens(words, []string{"1st"}) || hasAllTokens(words, []string{"first"}) {
		wanted = append(wanted, finish1stEdition)
	}
	if hasAllTokens(words, []string{"unlimited"}) {
		wanted = append(wanted, finishUnlimited)
	}
	if hasAllTokens(words, []string{"holo"}) {
		wanted = append(wanted, finishHolofoil)
	}
	if len(wanted) == 0 {
		return ""
	}

	best := ""
	for key := range card.FoilUUIDs {
		// The shared slots are the defaults this is trying to move off.
		if key == mtgmatcher.FinishNonfoil || key == mtgmatcher.FinishFoil {
			continue
		}
		named := true
		for _, axis := range wanted {
			if !strings.Contains(key, axis) {
				named = false
				break
			}
		}
		if !named {
			continue
		}
		if best == "" || len(key) < len(best) {
			best = key
		}
	}
	return best
}

// hasAllTokens reports whether every token is carried by some word of the
// wording. A token matches a word it opens, so "holo" is found in "Holofoil"
// and "1st" in "1st".
func hasAllTokens(words, tokens []string) bool {
	for _, token := range tokens {
		found := false
		for _, word := range words {
			if strings.HasPrefix(word, token) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// CanonicalFinish places the crossings of the two axes the catalog prices.
// The plain printings belong to the shared vocabulary — "Normal" is nonfoil
// everywhere TCGplayer writes it — and the treatments past it are this
// game's own, spelled with both axes so neither the run nor the treatment is
// lost.
func (Rules) CanonicalFinish(name string) string {
	return canonicalFinish(name)
}

func canonicalFinish(name string) string {
	normalized := mtgmatcher.NormalizeFinish(name)
	switch normalized {
	case finishHolofoil, finishReverseHolofoil,
		finish1stEdition, finishUnlimited,
		finish1stEditionHolo, finishUnlimitedHolo:
		return normalized
	}
	// "Reverse Holo" and "1st Edition Holo" are how storefronts abbreviate
	// two of them, and the abbreviation names no other printing.
	switch normalized {
	case "reverseholo":
		return finishReverseHolofoil
	case "holo":
		return finishHolofoil
	case "1steditionholo":
		return finish1stEditionHolo
	case "unlimitedholo":
		return finishUnlimitedHolo
	}
	return mtgmatcher.CanonicalFinish(normalized)
}

// extractNumbers pulls the collector numbers out of a storefront's variation
// wording, which mixes them with treatment words and set totals. Every token
// that looks like one is returned in the order it was written, because a
// wording carrying two numbers does not say which of them is the card's.
func extractNumbers(variation string) []string {
	var numbers []string
	for _, field := range strings.Fields(variation) {
		if m := fullNumberRe.FindStringSubmatch(field); m != nil {
			numbers = append(numbers, m[1])
		}
	}
	return numbers
}

// numberMatches compares a storefront's collector number against the
// catalog's, which carries the set total the storefront usually drops
// ("001/102" against "001", and either against "1").
func numberMatches(input, number string) bool {
	if number == "" {
		return false
	}
	return foldNumber(input) == foldNumber(number)
}

// numbersMatchCard reports whether any of the collector numbers a wording
// carries names the card's.
func numbersMatchCard(b *mtgmatcher.Backend, numbers []string, card *mtgmatcher.Card) bool {
	for _, number := range numbers {
		if numberMatchesCard(b, number, card) {
			return true
		}
	}
	return false
}

// numberMatchesCard compares a storefront's collector number against one
// card's, which is where the set the number belongs to is known.
func numberMatchesCard(b *mtgmatcher.Backend, input string, card *mtgmatcher.Card) bool {
	return numberMatches(input, card.Number) || numberMatchesPrefixed(b, input, card)
}

// numberMatchesPrefixed accepts the set code a storefront glues onto a
// number the catalog writes bare: the promo sets are numbered "001".."282"
// and the storefronts write "SVP001", "MEP003". The prefix has to be the
// set's own code and it has to stand in front of the whole number, so this
// can only ever admit the set the storefront already named - stripping
// letters freely would read "XY144" as the 144 of every set there is.
func numberMatchesPrefixed(b *mtgmatcher.Backend, input string, card *mtgmatcher.Card) bool {
	folded := foldNumber(card.Number)
	if folded == "" || strings.ContainsFunc(folded, func(r rune) bool { return r < '0' || r > '9' }) {
		return false
	}
	in := foldNumber(input)
	digits := strings.IndexFunc(in, func(r rune) bool { return r >= '0' && r <= '9' })
	if digits <= 0 || in[digits:] != folded {
		return false
	}
	set := b.Sets[card.SetCode]
	return set != nil && strings.EqualFold(in[:digits], set.Code)
}

// foldNumber reduces a collector number to the letters and digits that carry
// it, dropping the set total and the zeros each digit run is padded with.
func foldNumber(number string) string {
	number = strings.Split(number, "/")[0]
	var out strings.Builder
	digits := false
	for _, r := range strings.ToLower(number) {
		switch {
		case r >= '0' && r <= '9':
			if !digits && r == '0' {
				continue
			}
			digits = true
			out.WriteRune(r)
		case r >= 'a' && r <= 'z':
			digits = false
			out.WriteRune(r)
		}
	}
	return out.String()
}
