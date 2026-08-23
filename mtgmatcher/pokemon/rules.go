package pokemon

import (
	"maps"
	"regexp"
	"slices"
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
// the number they reprint - from being taken apart.
func (Rules) Prefilter(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	if _, found := b.CanonicalNames[mtgmatcher.Normalize(inCard.Name)]; found {
		return
	}
	name, tags := splitDecorations(b, inCard.Name)
	if name == inCard.Name {
		return
	}
	inCard.Name = name
	for _, tag := range tags {
		if tag != "" {
			inCard.AddToVariant(tag)
		}
	}
}

// splitDecorations peels a storefront's decorations off a product name and
// hands back the name with everything it took.
//
// Both halves have to keep what follows them. A parenthetical is lifted out
// in place rather than truncating the string, because the collector number
// is often written after it - "Greninja V-Union (Bottom Left) - SWSH157" is
// the only thing that tells four identically named quarter cards apart, and
// cutting at the parenthesis throws it away. The dashed segments are then
// read from the right, so a number with wording after it is still found
// ("Dragonite - 5/20 - Cosmos Holo") and the wording travels with it.
//
// A split whose head is a name the catalog knows is taken before the whole
// string is asked about. The catalog spells a handful of promos with the
// number inside the name ("Bouffalant -119/142"), which a decorated listing
// of an ordinary card collides with once its parenthetical is gone; the head
// says which of the two readings the listing meant. Names the catalog really
// spells with a dashed tail and nothing before it to recognise - the World
// Championship reprints, "Torchic - 2004" - are what the whole-string check
// is still there for.
func splitDecorations(b *mtgmatcher.Backend, raw string) (string, []string) {
	var tags []string
	name := raw
	for {
		begin := strings.Index(name, "(")
		if begin < 0 {
			break
		}
		end := strings.Index(name[begin:], ")")
		if end < 0 {
			break
		}
		tags = append(tags, strings.TrimSpace(name[begin+1:begin+end]))
		name = strings.Join(strings.Fields(name[:begin]+" "+name[begin+end+1:]), " ")
	}

	known := func(name string) bool {
		_, found := b.CanonicalNames[mtgmatcher.Normalize(name)]
		return found
	}
	parts := strings.Split(name, " - ")
	numbered := func(knownHead bool) (string, []string, bool) {
		for i := len(parts) - 1; i >= 1; i-- {
			segment := strings.TrimSpace(parts[i])
			if !numberTailRe.MatchString(segment) {
				continue
			}
			head := strings.TrimSpace(strings.Join(parts[:i], " - "))
			if head == "" || (knownHead && !known(head)) {
				continue
			}
			split := append([]string{}, tags...)
			for _, extra := range parts[i+1:] {
				split = append(split, strings.TrimSpace(extra))
			}
			return head, append(split, segment), true
		}
		return "", nil, false
	}

	if head, split, found := numbered(true); found {
		return head, split
	}
	if known(name) {
		return name, tags
	}
	if head, split, found := numbered(false); found {
		return head, split
	}
	return name, tags
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
		if reprintedByYear(inCard.Name, co.Name) {
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

// reprintYearRe matches what a World Championship reprint adds to the name
// it reprints: a dash and the year the deck was played.
var reprintYearRe = regexp.MustCompile(` - (?:19|20)\d{2}$`)

// reprintedByYear reports whether a name is another name spelled with the
// year it was played. The World Championship decks reprint 1,951 printings
// that way and keep the original's collector number, so a prefix search for
// a bare name nearly always turns one up - and since it carries the number
// too, it ties with the spelling the storefront actually meant and the
// ambiguity guard abandons the search.
//
// The dash is what makes this the World Championship spelling and not any
// name that merely ends in a year: 29 other printings do - the Victory
// Medals, the Code Cards, the Champion's Festivals - and every one of them
// writes the year without it, inside a parenthetical or glued to the words
// before. The head compares normalized, so punctuation variants of the name
// being reprinted still answer.
func reprintedByYear(name, candidate string) bool {
	tail := reprintYearRe.FindString(candidate)
	if tail == "" {
		return false
	}
	head := strings.TrimSuffix(candidate, tail)
	return mtgmatcher.Normalize(head) == mtgmatcher.Normalize(name)
}

// AdjustEdition trims the game-name prefix and "Singles" suffix storefronts
// decorate set names with, rewrites the names editionAliases carries, and
// drops the headings that name no set of ours.
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
			name := normalizedEditionAliases()[mtgmatcher.Normalize(edition)]
			if name == "" {
				name = promoSetNamed(b, edition)
			}
			if name == "" {
				name = setNamedByHead(b, edition)
			}
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

// promoSetNamed answers the promo set an era's heading means. The catalog
// names them with the era, a colon and a longer title - "SV: Scarlet &
// Violet Promo Cards", "SWSH: Sword & Shield Promo Cards" - while the
// storefronts head them "SV Promos" and "SWSH Promos", which agree on
// nothing after the era and so name no set at all. With no edition to narrow
// on, the row is matched against every printing of the name and aliases
// against the Jumbo and Deck Exclusive reprints, which carry the promo's own
// number.
//
// Only a single word can be the era, and only one promo set may answer to
// it: an era whose promos the catalog split across several sets does not say
// which one a listing means.
func promoSetNamed(b *mtgmatcher.Backend, edition string) string {
	era := strings.TrimSpace(strings.TrimSuffix(strings.TrimSuffix(edition, "Promos"), "Promo"))
	if era == "" || era == edition || strings.Contains(era, " ") {
		return ""
	}
	name := ""
	for _, set := range b.Sets {
		if set.Type != setTypePromo || !strings.HasPrefix(set.Name, era+": ") {
			continue
		}
		if name != "" {
			return ""
		}
		name = set.Name
	}
	return name
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
	for i := range fields {
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
	for _, number0 := range slices.Backward(numbers) {
		number = number0
		candidates = filterByNumber(b, inCard, cardSet, number)
		if len(candidates) > 0 {
			break
		}
	}
	if len(candidates) <= 1 {
		return candidates
	}

	// A verbatim collector number beats a prefix-folded match, and so does
	// one whose set total agrees. The total is what separates a reprint
	// from its original - Cascoon is 44/130 in Diamond & Pearl and 44/127
	// in Platinum - and it is exactly the part the fold drops, which left
	// this tier inert for the nine printings in ten the catalog writes a
	// total on.
	if number != "" {
		var exact []mtgmatcher.Card
		for _, card := range candidates {
			if strings.EqualFold(number, card.Number) || fullNumberMatches(inCard.Variation, card.Number) {
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

		if _, found := cardSet[card.SetCode]; !found && !subsetOf(b, cardSet, card.SetCode) {
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
			maps.Copy(foilUUIDs, card.FoilUUIDs)
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
	for field := range strings.FieldsSeq(variation) {
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

// subsetOf reports whether a set is one the edition already admits, filed
// under its own code. The catalog splits the collections printed inside a
// set out into a set of their own - "Legendary Treasures: Radiant
// Collection" beside "Legendary Treasures", the four Trainer Galleries
// beside their parents - while the storefronts file those cards under the
// parent, so the perfect-match loop builds the parent's code alone and the
// RC- and TG-numbered candidates are gated out.
//
// The suffixed code is not enough on its own: the same shape spells 32
// unrelated sets, from "Burger King Promos" under BKP to every POP series
// and every promo set under PR. The subset's name opening with its parent's
// is what tells the two apart, and it costs nothing to require - the eight
// real subsets all spell their parent out.
func subsetOf(b *mtgmatcher.Backend, cardSet map[string][]mtgmatcher.Card, code string) bool {
	parent, _, found := strings.Cut(code, "-")
	if !found {
		return false
	}
	if _, admitted := cardSet[parent]; !admitted {
		return false
	}
	set, subset := b.Sets[parent], b.Sets[code]
	return set != nil && subset != nil && strings.HasPrefix(subset.Name, set.Name)
}

// fullNumberMatches compares a storefront's collector number against the
// catalog's with the set total kept, which is what the ordinary fold drops.
// Both sides have to carry a total for the question to mean anything.
func fullNumberMatches(variation, number string) bool {
	numerator, total, found := strings.Cut(number, "/")
	if !found {
		return false
	}
	want := foldNumber(numerator) + "/" + strings.TrimLeft(total, "0")
	for field := range strings.FieldsSeq(variation) {
		numerator, total, found := strings.Cut(field, "/")
		if !found {
			continue
		}
		got := foldNumber(numerator) + "/" + strings.TrimLeft(strings.TrimRight(total, " ,."), "0")
		if strings.EqualFold(got, want) {
			return true
		}
	}
	return false
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
