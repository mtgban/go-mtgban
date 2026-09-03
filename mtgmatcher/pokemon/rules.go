package pokemon

import (
	"maps"
	"regexp"
	"slices"
	"strings"

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
type Rules struct {
	mtgmatcher.DefaultRules

	// The sets indexed by the name left after the catalog's era-and-number
	// prefix, built once by Load because the sets do not change afterwards.
	// A Rules built without it still answers, by indexing on the spot.
	setsByTail map[string][]string
}

// NewRules returns the rules with whatever the backend lets them precompute.
func NewRules(b *mtgmatcher.Backend) Rules {
	return Rules{setsByTail: indexSetsByTail(b)}
}

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
	stripStorefrontTails(b, inCard)
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

// levelTailRe matches the level a storefront glues onto a Diamond &
// Pearl-era name ("Moltres Lv.33"). Digits only, on purpose: the catalog
// carries real "LV.X" names, and those must never be taken apart.
var levelTailRe = regexp.MustCompile(`(?i)\s+Lv\.\s*\d+$`)

// letterTailRe matches the single bracketed letter cardtrader tells the
// Unown apart with ("Unown [J]").
var letterTailRe = regexp.MustCompile(`\s+\[([A-Za-z])\]$`)

// deltaTail is the delta-species wording some storefronts spell into the
// EX-era names ("Deoxys δ Delta Species") where the catalog writes the bare
// name.
const deltaTail = " δ Delta Species"

// stripStorefrontTails takes off the decorations a storefront writes into
// the name itself, each only when the undecorated spelling is a name the
// catalog knows - a real name wearing the same shape is never touched. The
// stripped words move into the variation, where they can still tier the
// label.
//
// The dash subtitle is a respelling rather than a strip: the catalog
// brackets the trainer a Supporter names ("Boss's Orders [Cyrus]") where
// cardtrader writes a dash, and the bracketed spelling is tried before the
// bare head - the subtitle is what separates two Supporters sharing a
// number's shape, so it must not be thrown away when the catalog kept it.
func stripStorefrontTails(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	known := func(name string) bool {
		_, found := b.CanonicalNames[mtgmatcher.Normalize(name)]
		return found
	}

	if idx := strings.LastIndex(inCard.Name, " - "); idx > 0 {
		head, tail := inCard.Name[:idx], strings.TrimSpace(inCard.Name[idx+3:])
		if bracketed := head + " [" + tail + "]"; known(bracketed) {
			inCard.Name = bracketed
			return
		}
	}

	for {
		name := inCard.Name
		// The level stays out of the variation: its digits would read as
		// one more collector number and be asked before the real one.
		if tail := levelTailRe.FindString(name); tail != "" && known(strings.TrimSuffix(name, tail)) {
			inCard.Name = strings.TrimSuffix(name, tail)
		}
		if strings.HasSuffix(name, deltaTail) && known(strings.TrimSuffix(name, deltaTail)) {
			inCard.Name = strings.TrimSuffix(name, deltaTail)
			inCard.AddToVariant("Delta Species")
		}
		if m := letterTailRe.FindStringSubmatch(name); m != nil && known(strings.TrimSuffix(name, m[0])) {
			inCard.Name = strings.TrimSuffix(name, m[0])
			inCard.AddToVariant(m[1])
		}
		if inCard.Name == name {
			return
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
func (r Rules) AdjustEdition(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard) {
	edition := r.AliasEdition(b, inCard.Edition)
	// The sequin holo printings are the General Mills cereal promos, which
	// the catalog files in Miscellaneous Cards & Products under the promo
	// set's own numbering. The promo set a sequin listing names carries
	// only the plain printing, and landing there would file the sequin
	// price under the plain product's uuid.
	if mtgmatcher.SlugDescribes(inCard.Variation, "sequin") {
		edition = "Miscellaneous Cards & Products"
	}
	inCard.Edition = edition

	widenQualifiedName(b, inCard)
}

// AliasEdition spells an edition string toward a set name using the string
// alone. See mtgmatcher.GameRules.
func (r Rules) AliasEdition(b *mtgmatcher.Backend, edition string) string {
	edition = strings.TrimSpace(edition)
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
	// the rules to alias the edition, which is this function.
	if edition != "" {
		_, known := b.NormalizedSets[mtgmatcher.Normalize(edition)]
		if !known {
			_, err := b.GetSet(edition)
			known = err == nil
		}
		// A pooled name is left as it is: it spans two sets, so no single
		// rewrite can carry it, and FilterCards restricts on the name
		// itself instead.
		if _, pooled := normalizedPooledEditions()[mtgmatcher.Normalize(edition)]; !known && !pooled {
			name := normalizedEditionAliases()[mtgmatcher.Normalize(edition)]
			if name == "" {
				name = promoSetNamed(b, edition)
			}
			if name == "" {
				name = setNamedByHead(b, edition)
			}
			if name == "" {
				name = r.setNamedByTail(b, edition)
			}
			if name != "" {
				edition = name
			}
		}
	}
	return edition
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
	// An edition naming no set widens nothing. The storefronts carry whole
	// Japanese catalogs under set names ours has never heard of ("Plasma
	// Storm Promos", the JP "Scarlet & Violet Promos"), and widening there
	// is how a Japanese listing walked onto an English printing: the JP
	// promo's number is real, some qualified English name carries it, and
	// with no edition to gate on the rename went through.
	if code == "" {
		return
	}
	inEdition := func(co *mtgmatcher.CardObject) bool {
		return co.SetCode == code
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
func indexSetsByTail(b *mtgmatcher.Backend) map[string][]string {
	index := map[string][]string{}
	for _, set := range b.Sets {
		tail := mtgmatcher.Normalize(setTail(set.Name))
		index[tail] = append(index[tail], set.Name)
	}
	return index
}

func (r Rules) setNamedByTail(b *mtgmatcher.Backend, edition string) string {
	index := r.setsByTail
	if index == nil {
		index = indexSetsByTail(b)
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
	candidates := filterCandidates(b, inCard, cardSet)

	// A wording naming a stamp or a misprint prices a physical variant the
	// plain printing is not: "Pokemon Center Stamped", "Professor Program
	// Stamp", "Partial 1st Ed Stamp Misprint". When no surviving candidate
	// wears a stamp label, whatever survived is the unstamped product, and
	// pricing it would file the variant's price under the plain uuid - the
	// wording can even describe a label the candidate does wear ("Cosmos
	// Holo" beside the stamp) without the stamp being the one it wears.
	// Candidates that do wear stamp labels are left to the ordinary
	// tiering: the wording and the label can spell the same stamp
	// differently.
	if len(candidates) > 0 && demandsStamp(inCard.Variation) {
		stamped := false
		for _, card := range candidates {
			for _, promoType := range card.PromoTypes {
				if strings.Contains(promoType, "stamp") {
					stamped = true
				}
			}
		}
		if !stamped {
			return nil
		}
	}

	// A wording naming the Cosmos Holo outright has to reach a printing
	// that can be one: a candidate wearing the label, or at least selling
	// a holo entry the finish resolution can land on. What survives
	// otherwise is a plain printing that happens to share the number -
	// the treated printing lives in a set the edition does not admit -
	// and pricing it would file the holo's price under the plain uuid.
	// A wording naming the plain treatment too is the ambiguity the
	// tiering already surfaces, and is left alone.
	if len(candidates) > 0 && !describesPlain(inCard.Variation) &&
		mtgmatcher.SlugDescribes(inCard.Variation, "cosmosholo") {
		var cosmos []mtgmatcher.Card
		for _, card := range candidates {
			if wearsCosmosOrHolo(&card) {
				cosmos = append(cosmos, card)
			}
		}
		candidates = cosmos
	}
	return candidates
}

// wearsCosmosOrHolo reports whether the printing can be the Cosmos Holo a
// wording names: it wears a cosmos label, or it sells some holo entry.
func wearsCosmosOrHolo(card *mtgmatcher.Card) bool {
	for _, promoType := range card.PromoTypes {
		if strings.Contains(promoType, "cosmos") {
			return true
		}
	}
	for key := range card.FoilUUIDs {
		if strings.Contains(key, "holo") {
			return true
		}
	}
	return false
}

// demandsStamp reports whether the wording names a stamped or misprinted
// variant.
func demandsStamp(variation string) bool {
	variation = strings.ToLower(variation)
	return strings.Contains(variation, "stamp") || strings.Contains(variation, "misprint")
}

func filterCandidates(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cardSet map[string][]mtgmatcher.Card) []mtgmatcher.Card {
	// A pooled storefront name restricts the candidates to its pair of
	// sets outright: the edition kept the name, so nothing upstream could
	// narrow, and falling through past the pool would price a card the
	// pool does not carry as some same-numbered printing elsewhere.
	if pool, found := normalizedPooledEditions()[mtgmatcher.Normalize(inCard.Edition)]; found {
		pooled := map[string][]mtgmatcher.Card{}
		for _, name := range pool {
			if set, ok := b.NormalizedSets[mtgmatcher.Normalize(name)]; ok {
				if cards, carried := cardSet[set.Code]; carried {
					pooled[set.Code] = cards
				}
			}
		}
		cardSet = pooled
	}

	// A storefront can write more than one number into the wording: Cool
	// Stuff Inc prepends its own catalogue field, which is empty, "0" or
	// "001" for whole editions, and the number the listing prints is the
	// one Prefilter appends after it. Read them from the back, so the
	// number the card carries on it is asked first and the vendor's index
	// only answers when nothing else does.
	numbers := extractNumbers(inCard.Variation)
	number := ""
	var candidates []mtgmatcher.Card
	for _, number0 := range slices.Backward(numbers) {
		number = number0
		candidates = filterByNumber(b, inCard, cardSet, number)
		if len(candidates) > 0 {
			break
		}
	}
	// A lettered number the edition admits no printing of names a promo
	// printing filed in a set of its own. The storefronts sell the
	// alternate arts under the set the card was first printed in - Cool
	// Stuff Inc lists "Garbodor (Alt Art) - 51a/145" against "SM Guardians
	// Rising" - where the catalog files them in "Alternate Art Promos",
	// which the edition then keeps out. The number is the whole of what
	// says which printing this is, so it is asked of the catalog entire
	// before the letter is given up on.
	if len(candidates) == 0 {
		promo := letteredPromo(b, inCard, numbers)
		if len(promo) == 1 {
			return promo
		}
		// The catalog carries the lettered number more than once: TCGplayer
		// sells several of these promos twice, once at their own size and
		// once as an oversized jumbo, with the same name, number and finish
		// and nothing on the card to tell them apart. The set is what tells
		// them apart, and the wording names it - "(Alt Art)" against
		// "Alternate Art Promos" beside "Jumbo Cards".
		if len(promo) > 1 {
			if named := setNamed(b, inCard, promo); len(named) == 1 {
				return named
			}
			// It named neither, so the letter stands rather than being
			// dropped: answering with the plain card the promo reprints
			// would be a third card again.
			return nil
		}
	}
	// A letter hung off the end of a number the catalog carries without one
	// is the storefront's own marker for a printing it prices apart:
	// Strikezone numbers the Master Ball patterns "074M" beside the plain
	// "074", and the run-marked reprints "001A". The number as written is
	// asked first and every time, so the stripped form only ever answers
	// where the written one reached nothing, and the label tier below is
	// what picks the marked printing out of the ones it reaches.
	if len(candidates) == 0 {
		for _, number0 := range slices.Backward(numbers) {
			bare := strings.TrimRight(number0, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
			if bare == "" || bare == number0 {
				continue
			}
			number = bare
			candidates = filterByNumber(b, inCard, cardSet, bare)
			if len(candidates) > 0 {
				break
			}
		}
	}
	// The unnumbered pass answers only for a wording that wrote no number
	// at all: any number the wording did write overwrites it on the loop's
	// first turn, kept or not.
	if len(numbers) == 0 {
		candidates = filterByNumber(b, inCard, cardSet, "")
	}
	if len(candidates) <= 1 {
		return candidates
	}

	// A verbatim collector number beats a prefix-folded match, and so does
	// one whose set total agrees. The total is what separates a reprint
	// from its original - Cascoon is 44/130 in Diamond & Pearl and 44/127
	// in Platinum - and it is exactly the part the fold drops, which left
	// this tier inert for the nine printings in ten the catalog writes a
	// total on. An agreeing total also beats a verbatim bare number: the
	// wording that spells "13/147" names the 147-card set's printing, not
	// the promo the catalog numbers a bare "13".
	if number != "" {
		var plain, full []mtgmatcher.Card
		for _, card := range candidates {
			switch {
			case fullNumberMatches(inCard.Variation, card.OriginalNumber):
				full = append(full, card)
			case strings.EqualFold(number, card.OriginalNumber):
				plain = append(plain, card)
			}
		}
		exact := full
		if len(exact) == 0 {
			exact = plain
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

// setStopWords are the words a Pokemon set name shares with too many others
// to tell one from another.
var setStopWords = map[string]bool{
	"promo": true, "promos": true, "card": true, "cards": true,
	"set": true, "sets": true, "the": true, "and": true,
	"products": true, "collection": true,
}

// altArtSpellings are the ways a storefront writes the treatment the catalog
// files a whole set of, so the set's own name can be looked for in a wording
// that abbreviates it.
var altArtSpellings = strings.NewReplacer("alt art", "alternate art")

// setNamed keeps the printings whose set the wording names, by the words of
// that set's name which are its own.
//
// A set name shared down to nothing - "Miscellaneous Cards & Products" is
// every word a stop word but one - cannot be named by accident this way,
// and a wording naming none of the sets keeps them all, which the caller
// reads as the wording having failed to choose.
func setNamed(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, cards []mtgmatcher.Card) []mtgmatcher.Card {
	said := map[string]bool{}
	wording := altArtSpellings.Replace(strings.ToLower(inCard.Variation + " " + inCard.Edition))
	for field := range strings.FieldsSeq(wording) {
		if word := mtgmatcher.PromoTypeSlug(field); word != "" {
			said[word] = true
		}
	}
	var out []mtgmatcher.Card
	for _, card := range cards {
		set, found := b.Sets[card.SetCode]
		if !found {
			continue
		}
		whole, own := true, false
		for field := range strings.FieldsSeq(strings.ToLower(set.Name)) {
			word := mtgmatcher.PromoTypeSlug(field)
			if word == "" || setStopWords[word] {
				continue
			}
			own = true
			if !said[word] {
				whole = false
				break
			}
		}
		if own && whole {
			out = append(out, card)
		}
	}
	return out
}

// letteredNumberRe matches a collector number carrying the letter a promo
// printing is told apart by, "51a" and "182b" and "SM30a".
var letteredNumberRe = regexp.MustCompile(`(?i)^[A-Z]{0,4}[0-9]+[a-z]$`)

// letteredPromo collects the printings of this card that carry a lettered
// number the edition admits none of, wherever the catalog files them.
//
// The letter is the storefront's own marker and the catalog's alike: the
// alternate art promos are numbered for the set they reprint and lettered
// apart from it, so "51a" names one printing in the whole catalog where
// "51" names one per set. Reading it across sets is safe for exactly that
// reason, and only where the edition reached nothing - a lettered number
// the edition does admit has already answered.
//
// Nothing is stripped here. Where the letter names more than one printing
// the wording has not chosen, and the caller falls through to the tiers
// that read the rest of it.
func letteredPromo(b *mtgmatcher.Backend, inCard *mtgmatcher.InputCard, numbers []string) []mtgmatcher.Card {
	var out []mtgmatcher.Card
	seen := map[string]bool{}
	for _, number := range numbers {
		if !letteredNumberRe.MatchString(number) {
			continue
		}
		for _, uuid := range b.Hashes[mtgmatcher.Normalize(inCard.Name)] {
			co, found := b.UUIDs[uuid]
			if !found || co.Sealed || !numberMatches(number, co.Number) {
				continue
			}
			key := co.Card.Identifiers["tcgplayerProductId"]
			if key == "" {
				key = trimFinishSuffix(uuid)
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			out = append(out, co.Card)
		}
	}
	return out
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
		if totalDisagrees(inCard.Variation, card.OriginalNumber) {
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
		// A wording can spell two treatments at once: CardTrader files the
		// plain and the Cosmos Holo printing of a number under one
		// blueprint and writes "Non-Holo / Cosmos Holo" for both. The
		// listing does not say which it prices, so both stand and the
		// ambiguity surfaces rather than every copy repricing the holo.
		if len(base) > 0 && describesPlain(inCard.Variation) {
			return append(base, described...)
		}
		return described
	}
	if len(base) > 0 {
		return base
	}
	return labelled
}

// describesPlain reports whether the wording names the untreated printing:
// the "Non-Holo" the storefronts write it with.
func describesPlain(variation string) bool {
	variation = strings.ToLower(variation)
	return strings.Contains(variation, "non-holo") || strings.Contains(variation, "non holo")
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

// totalDisagrees reports whether the wording spells the card's own
// numerator with a set total that contradicts the card's. The total names
// the set as surely as the number names the card - "Non-Holo | 053/167" is
// the 167-card set's 53, and the fold that drops it is what let that
// wording price the 53/094 of another collection. Only agreement between
// two spelled-out totals can clear a contradiction, and a wording spelling
// the numerator with no total at all vetoes nothing.
func totalDisagrees(variation, number string) bool {
	numerator, total, found := strings.Cut(number, "/")
	if !found {
		return false
	}
	want := foldTotal(total)
	if want == "" {
		return false
	}
	saw := false
	for _, field := range strings.Fields(variation) {
		fieldNumerator, fieldTotal, cut := strings.Cut(field, "/")
		if !cut || foldNumber(fieldNumerator) != foldNumber(numerator) {
			continue
		}
		got := foldTotal(fieldTotal)
		if got == "" {
			continue
		}
		if got == want {
			return false
		}
		saw = true
	}
	return saw
}

// foldTotal reduces a set total to its digits, dropping the zero padding
// and whatever trails the digit run ("017" and "17h" both fold to "17").
// A total that does not open with a digit folds away entirely.
func foldTotal(total string) string {
	end := 0
	for end < len(total) && total[end] >= '0' && total[end] <= '9' {
		end++
	}
	return strings.TrimLeft(total[:end], "0")
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
