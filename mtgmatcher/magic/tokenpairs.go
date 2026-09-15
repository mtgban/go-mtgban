package magic

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// derivedTokenPairSuffix marks a uuid this loader minted for a two-sided
// token pairing - two token faces one physical card prints, sold as one
// TCGplayer product - rather than one mtgjson itself assigned. mtgjson
// already mints a combined printing of its own for the fixed pairings
// (layout "double_faced_token", e.g. Commander Legends: Battle for
// Baldur's Gate's Undercity // The Initiative); this covers the far more
// common shape, a random-feeling token sheet where TCGplayer sells every
// distinct pairing it prints as its own product, and mtgjson describes
// only via tokenProducts/tokenParts on each independent single-faced
// token, never as a combined entity.
//
// A derived card reaches Match only by its TCGplayer product id, never by
// name: the combined name is not unique inside its own set (many single-
// faced tokens pair with several different partners on the same sheet),
// so it is kept out of Hashes, CanonicalNames and set.Cards/set.Tokens on
// purpose. See deriveTokenPairs for the full exclusion ladder.
const derivedTokenPairSuffix = "_tp_"

// pairKey identifies a physical pairing by its two uuids, always ordered
// a < b so the same pairing hashes the same way regardless of which face's
// tokenProducts entry it was read from.
type pairKey struct {
	a, b string
}

func newPairKey(u1, u2 string) pairKey {
	if u1 < u2 {
		return pairKey{u1, u2}
	}
	return pairKey{u2, u1}
}

// deriveTokenPairs mints one combined Card per unique physical pairing a
// two-sided token sheet's tokenProducts describe. uuids is the fully built
// uuid index (every real printing already resolved); tcgIDs is
// ExternalIdentifiers[tcgplayer], read to refuse an id a real printing
// already owns and written to file each derived entity's own ids at its
// base sibling, the same convention every other identifier follows.
func deriveTokenPairs(sets map[string]*Set, uuids map[string]*mtgmatcher.CardObject, tcgIDs map[string]string) []Card {
	// pairIDs collects every id seen for a pairing; idPairs collects every
	// pairing seen for an id. An id naming more than one pairing describes
	// two different physical objects and answers for neither - the same
	// "an id claimed by more than one thing is a guess, refuse it" credential
	// crossSetProductIDs (tcgplayer/index.go) applies to TCGplayer's own
	// price scraper.
	pairIDs := map[pairKey]map[string]bool{}
	idPairs := map[string]map[pairKey]bool{}

	// A token this loader has already merged set.Tokens into set.Cards by
	// the time this runs (mtgjson.go's allCards/filteredCards step, for
	// every token bar art_series), so the same record's tokenProducts
	// would otherwise be read twice - once from each slice - inflating
	// every count below. Rather than assume that merge is exhaustive (a
	// set-specific filter downstream of it could still drop a token from
	// Cards alone), read both slices but skip a uuid already visited.
	seenCard := map[string]bool{}
	for _, set := range sets {
		for _, group := range [2][]Card{set.Cards, set.Tokens} {
			for _, card := range group {
				if card.UUID != "" {
					if seenCard[card.UUID] {
						continue
					}
					seenCard[card.UUID] = true
				}
				for _, tp := range card.TokenProducts {
					if len(tp.TokenParts) != 2 {
						continue
					}
					id := tp.Identifiers["tcgplayerProductId"]
					u1, u2 := tp.TokenParts[0].UUID, tp.TokenParts[1].UUID
					if id == "" || u1 == "" || u2 == "" {
						continue
					}
					if u1 == u2 {
						continue
					}

					key := newPairKey(u1, u2)
					if pairIDs[key] == nil {
						pairIDs[key] = map[string]bool{}
					}
					pairIDs[key][id] = true
					if idPairs[id] == nil {
						idPairs[id] = map[pairKey]bool{}
					}
					idPairs[id][key] = true
				}
			}
		}
	}

	// Deterministic order: map iteration would make a re-run's derived
	// uuid set identical but its log/slice order arbitrary.
	keys := make([]pairKey, 0, len(pairIDs))
	for key := range pairIDs {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].a != keys[j].a {
			return keys[i].a < keys[j].a
		}
		return keys[i].b < keys[j].b
	})

	// idCanonicalKey resolves a tcgplayerProductId claimed by more than one
	// pairKey to the one pairKey allowed to use it, for ids that aren't
	// genuinely ambiguous: every one of their claimants names the identical
	// two faces (order-independent), so this is mtgjson cataloguing one
	// real physical pairing under more than one uuid combination - a
	// memorabilia/oversized sibling set repeating a base set's own id
	// (OAFR duplicating AFR's own dungeon-card ids under its own uuids: 127
	// of 128 measured id collisions are this shape) - not two different
	// physical objects sharing an id by coincidence or error. A genuinely
	// ambiguous id (different name pairs among its claimants) gets no
	// entry and stays refused by every claimant, same as before this
	// existed.
	//
	// The canonical claimant is deliberately the one whose faces are NOT
	// themselves filed under a "memorabilia"-type set (an oversized or
	// reference sibling, never what a vendor's own scryfall_id or listing
	// wording actually resolves to), not an arbitrary pick: the pairing
	// index only ever indexes the uuids the winning claimant itself used, so
	// picking the memorabilia sibling here would silently leave a real
	// vendor listing - anchored on the ordinary set's own scryfall_id or
	// number - unable to find this pairing at all, even though a derived
	// entity for it exists. Ties (neither or both sides memorabilia) fall
	// back to sorted order, same as everywhere else non-determinism would
	// otherwise creep in.
	//
	// "Not genuinely ambiguous" is decided by face NAME agreement alone,
	// not by confirming the claimants are actually sibling-set uuids of
	// each other (no general "these two sets are reprint siblings" check
	// exists here beyond the memorabilia-preference tiebreak above). That
	// is sound for every case measured - 127 of 128 real id collisions in
	// today's datastore, all a memorabilia set repeating a base set's own
	// id - but it is a real, if narrower, assumption than "these uuids
	// are known duplicates of each other": two GENUINELY different real
	// printings that happen to share both face names by coincidence,
	// filed under two different non-memorabilia sets, would also read as
	// "not ambiguous" here and get silently collapsed to one canonical
	// pairing rather than refused. No such case has been found against
	// real data; if one ever is, this is where it would need a sharper
	// test than name equality (e.g. also requiring the claimant sets to
	// share an ancestor via homeSet).
	idCanonicalKey := map[string]pairKey{}
	for id, claimants := range idPairs {
		if len(claimants) <= 1 {
			continue
		}
		sorted := make([]pairKey, 0, len(claimants))
		for key := range claimants {
			sorted = append(sorted, key)
		}
		sort.Slice(sorted, func(i, j int) bool {
			if sorted[i].a != sorted[j].a {
				return sorted[i].a < sorted[j].a
			}
			return sorted[i].b < sorted[j].b
		})

		var namePair [2]string
		consistent := true
		var best pairKey
		bestScore := -1
		for i, key := range sorted {
			co1, found1 := uuids[key.a]
			co2, found2 := uuids[key.b]
			if !found1 || !found2 {
				consistent = false
				break
			}
			n1, n2 := co1.Card.Name, co2.Card.Name
			if n2 < n1 {
				n1, n2 = n2, n1
			}
			if i == 0 {
				namePair = [2]string{n1, n2}
			} else if namePair != [2]string{n1, n2} {
				consistent = false
				break
			}
			score := 0
			if !isMemorabiliaSet(sets, co1.SetCode) {
				score++
			}
			if !isMemorabiliaSet(sets, co2.SetCode) {
				score++
			}
			if score > bestScore {
				bestScore = score
				best = key
			}
		}
		if consistent {
			idCanonicalKey[id] = best
		}
	}

	var derived []Card
	for _, key := range keys {
		co1, found1 := uuids[key.a]
		co2, found2 := uuids[key.b]
		if !found1 || !found2 {
			continue
		}
		if co1.Layout == "double_faced_token" || co2.Layout == "double_faced_token" {
			continue
		}
		if !isTokenPairLayout(co1.Layout) || !isTokenPairLayout(co2.Layout) {
			continue
		}

		var usableIDs []int
		for id := range pairIDs[key] {
			if len(idPairs[id]) > 1 {
				if canon, ok := idCanonicalKey[id]; !ok || canon != key {
					continue
				}
			}
			if _, claimed := tcgIDs[id]; claimed {
				continue
			}
			n, err := strconv.Atoi(id)
			if err != nil {
				continue
			}
			usableIDs = append(usableIDs, n)
		}
		if len(usableIDs) == 0 {
			continue
		}
		sort.Ints(usableIDs)

		home := homeSet(sets, co1.SetCode, co2.SetCode)
		if home == "" {
			continue
		}

		card := buildDerivedCard(co1, co2, home, usableIDs)
		derived = append(derived, card)

		for _, n := range usableIDs {
			id := strconv.Itoa(n)
			if _, claimed := tcgIDs[id]; !claimed {
				tcgIDs[id] = card.UUID
			}
		}
	}

	return derived
}

// isTokenPairLayout reports whether a face's layout is one this loader
// knows how to pair. This is an allowlist, not a denylist: layout is
// upstream data mtgjson can add values to, and a denylist would let a new
// shape through unexamined. It deliberately excludes "art_series" (already
// dropped by skipSet - and the only art_series tokenProducts pairings
// found are a card paired with itself under two ids, not a genuine
// two-sided product) and "reversible_card" (one instance found, also
// self-paired, no vendor evidence). Both stay as an explicit backstop for
// whatever upstream does next, not dead code to delete.
//
// "normal" and "flip" don't need to be here even though AFR's dungeon
// cards (Dungeon of the Mad Mage, Lost Mine of Phandelver, Tomb of
// Annihilation - CK and SCG both sell these paired with their own tokens)
// carry "normal" upstream: adjustTokens (mtgjson.go) already rewrites
// every set.Tokens entry's own layout to "token" before this runs, and a
// set.Tokens entry is where a dungeon card lives, so this rung never
// actually sees "normal" for them. What was really excluding them is the
// multi-pair-id rung below, since OAFR (Forgotten Realms Oversized Cards,
// a memorabilia sibling of AFR) repeats AFR's own dungeon-card ids under
// its own uuids - see idCanonicalKey.
func isTokenPairLayout(layout string) bool {
	return layout == "token" || layout == "emblem"
}

// homeSet answers where a pairing belongs when its two faces come from
// different sets. Nothing in the data says which of TCGplayer's own
// groups sells it, so this answers the question a person would: the sheet
// of the release both faces belong to. The Commander decks, bonus sheets
// and promo sheets that ship with a set carry ParentCode back to it, so
// walking each side's chain to the first code they share lands on the
// main set - every cross-set pairing measured this session resolved this
// way, most in one step.
func homeSet(sets map[string]*Set, setA, setB string) string {
	if setA == setB {
		return tokenSetCodeOf(sets, setA)
	}

	chainA := parentChain(sets, setA)
	seen := make(map[string]bool, len(chainA))
	for _, code := range chainA {
		seen[code] = true
	}
	for _, code := range parentChain(sets, setB) {
		if seen[code] {
			return tokenSetCodeOf(sets, code)
		}
	}
	return ""
}

// tokenSetCodeOf resolves a set code to the code its tokens are filed
// under, falling back to the code itself when the set is unknown or names
// no token sheet of its own.
func tokenSetCodeOf(sets map[string]*Set, code string) string {
	if set, found := sets[code]; found && set.TokenSetCode != "" {
		return set.TokenSetCode
	}
	return code
}

// isMemorabiliaSet reports whether code names a "memorabilia"-type set -
// an oversized or reference sibling (Forgotten Realms Oversized Cards,
// OAFR, for AFR's own dungeon cards) mtgjson catalogues under its own
// uuids and, often, its own scryfall_id, but that a vendor's real listing
// never actually resolves through.
func isMemorabiliaSet(sets map[string]*Set, code string) bool {
	set, found := sets[code]
	return found && set.Type == "memorabilia"
}

// parentChain walks a set's ParentCode links up to the root, returning the
// codes visited starting with code itself. A cycle stops the walk rather
// than looping forever - defensively, mtgjson has never introduced one.
func parentChain(sets map[string]*Set, code string) []string {
	chain := []string{code}
	seen := map[string]bool{code: true}
	for {
		set, found := sets[code]
		if !found || set.ParentCode == "" || seen[set.ParentCode] {
			return chain
		}
		code = set.ParentCode
		chain = append(chain, code)
		seen[code] = true
	}
}

// buildDerivedCard assembles the combined Card for one surviving pairing.
// usableIDs is sorted ascending and may be empty for a pairing verified by
// vendor cross-checking rather than by an mtgjson tokenProducts id at all
// (see verifiedNoUpstreamPairs) - Identifiers["vendorVerifiedPair"] marks
// that case, since neither tcgplayerProductId key can be set without one.
func buildDerivedCard(co1, co2 *mtgmatcher.CardObject, home string, usableIDs []int) Card {
	uuidLo, uuidHi := co1.UUID, co2.UUID
	if uuidHi < uuidLo {
		uuidLo, uuidHi = uuidHi, uuidLo
	}

	// The uuid is built from lexical order so it survives mtgjson
	// renumbering a set; the display order below is independent and is
	// chosen to read like a real card, lower collector number first.
	first, second := co1, co2
	if numberLess(second.Number, first.Number) {
		first, second = second, first
	}

	name := first.Name + " // " + second.Name
	// For a cross-set pairing (665 of 5,993 measured, 36 distinct set-code
	// combinations) this mixes two different sets' own numbering into one
	// field - cosmetic only, never read for matching: a derived entity is
	// excluded from set.Cards/Tokens and every name/number index (see the
	// type's own comment above), so nothing indexed by Number can ever
	// return one, by construction, regardless of what this string holds.
	number := first.Number + " // " + second.Number
	uuid := uuidLo + derivedTokenPairSuffix + uuidHi

	ids := make([]string, len(usableIDs))
	for i, n := range usableIDs {
		ids[i] = strconv.Itoa(n)
	}

	identifiers := map[string]string{
		"derivedTokenPair": "true",
		"tokenPairPartA":   uuidLo,
		"tokenPairPartB":   uuidHi,
		"tokenSetCode":     home,
	}
	if len(ids) > 0 {
		identifiers["tcgplayerProductId"] = ids[0]
		identifiers["tcgplayerProductIds"] = strings.Join(ids, ",")
	} else {
		identifiers["vendorVerifiedPair"] = "true"
	}

	images := map[string]string{}
	if scryfallID := first.Identifiers["scryfallId"]; len(scryfallID) > 1 {
		for version, size := range map[string]string{"full": "normal", "thumbnail": "small", "crop": "art_crop"} {
			images[version] = scryfallImageURL(scryfallID, size)
		}
	}

	return Card{
		Name:        name,
		Number:      number,
		PlainNumber: plainNumber(number),
		SetCode:     home,
		Layout:      "token",
		Rarity:      "token",
		Language:    "English",
		Finishes:    unionFinishes(co1.Finishes, co2.Finishes),
		Identifiers: identifiers,
		Images:      images,
		UUID:        uuid,
	}
}

// mintVerifiedPairs mints the derived entity for every pairing in
// verifiedNoUpstreamPairs - the same buildDerivedCard deriveTokenPairs
// itself calls, just with no usable TCGplayer id (nil usableIDs), since
// these were confirmed real by vendor cross-verification rather than by
// mtgjson's own tokenProducts feed at all. alreadyDerived is the set of
// base uuids deriveTokenPairs already minted this same load: if mtgjson's
// own feed has since caught up with a pairing this table also names (both
// name the identical uuid pair, so buildDerivedCard's own uuid formula
// produces the identical uuid either way), the real, priced entity wins
// and this skips it rather than shadowing it with a worse, unpriced one.
// A face uuid the loaded datastore no longer carries, or a pair sharing
// no common ancestor set, is skipped rather than guessed - the same
// discipline deriveTokenPairs already applies to its own input.
func mintVerifiedPairs(sets map[string]*Set, uuids map[string]*mtgmatcher.CardObject, alreadyDerived map[string]bool) []Card {
	var cards []Card
	seen := map[[2]string]bool{}
	for _, p := range verifiedNoUpstreamPairs {
		key := [2]string{p.a, p.b}
		if p.b < p.a {
			key = [2]string{p.b, p.a}
		}
		if seen[key] {
			continue
		}
		seen[key] = true

		co1, found1 := uuids[p.a]
		co2, found2 := uuids[p.b]
		if !found1 || !found2 {
			continue
		}

		uuidLo, uuidHi := co1.UUID, co2.UUID
		if uuidHi < uuidLo {
			uuidLo, uuidHi = uuidHi, uuidLo
		}
		if alreadyDerived[uuidLo+derivedTokenPairSuffix+uuidHi] {
			continue
		}

		home := homeSet(sets, co1.SetCode, co2.SetCode)
		if home == "" {
			continue
		}
		cards = append(cards, buildDerivedCard(co1, co2, home, nil))
	}
	return cards
}

// scryfallImageURL mirrors generateImageURL, which takes this package's own
// Card rather than the mtgmatcher.CardObject a derived entity is built
// from; a derived entity has no scryfallId of its own; the image shown is
// honestly one face's, not a composite.
func scryfallImageURL(scryfallID, version string) string {
	return fmt.Sprintf("https://cards.scryfall.io/%s/front/%c/%c/%s.jpg", version, scryfallID[0], scryfallID[1], scryfallID)
}

// numberLess orders two collector numbers the way a person would: by their
// leading numeric run first, falling back to a plain string compare for
// numbers that do not start with (or share) one.
func numberLess(a, b string) bool {
	na, oka := leadingNumber(a)
	nb, okb := leadingNumber(b)
	if oka && okb && na != nb {
		return na < nb
	}
	return a < b
}

func leadingNumber(s string) (int, bool) {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(s[:i])
	return n, err == nil
}

// unionFinishes merges two finish lists, filtered to the two finishes a
// physical token sheet is sold in. 99 of the derived corpus's own pairs
// disagree between their two faces (one lists foil, the other does not);
// the union is the safe direction, since MatchIDFinish errors on a finish
// the card does not carry - a too-narrow set turns a real sku into a
// logged failure, a too-wide set is simply never asked about. Neither face
// of a two-sided token sheet was ever sold etched or signed.
func unionFinishes(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, list := range [2][]string{a, b} {
		for _, finish := range list {
			if finish != mtgmatcher.FinishNonfoil && finish != mtgmatcher.FinishFoil {
				continue
			}
			if seen[finish] {
				continue
			}
			seen[finish] = true
			out = append(out, finish)
		}
	}
	sort.Strings(out)
	return out
}

// tokenPairIndicesData is what one walk of the backend's own derived
// pairings builds for every index in this file keyed off of it - byFace's
// name-keyed shape, byUUIDPair's uuid-pair-keyed one and byBothNames' own
// name-pair-keyed one all need the identical set of pairings, gathered
// the identical way, so one walk builds all three rather than each
// walking the same backend again.
type tokenPairIndicesData struct {
	byFace map[string]map[string]string

	// byUUIDPair maps two face uuids (order-independent) directly to the
	// derived pairing's own tcgplayerProductId, for a caller that has
	// already anchored BOTH faces unambiguously by identity - typically
	// via MatchInSetNumber on each face's own filing set and number, the
	// same discipline MatchTokenPairingBySetNumber already trusts for one
	// face - rather than by name. Unlike byFace (keyed by one face's uuid
	// to the *other* face's own name, normalized), a genuine collision
	// here would mean two different derived Card entities claim the
	// identical unordered uuid pair - not expected, since deriveTokenPairs
	// already dedupes by exactly that unordered pair on the way in, so
	// this should never fire in practice. Blanked on one anyway, the same
	// "don't know, refuse" discipline as byFace's own collision handling,
	// rather than trusting that invariant silently: a last-write-wins map
	// would otherwise let a future change to deriveTokenPairs's own dedup
	// silently start returning an arbitrary pick between two real pairings
	// instead of refusing.
	byUUIDPair map[[2]string]string

	// byBothNames maps two normalized face names (order-independent)
	// directly to the derived pairing's own tcgplayerProductId, blanked on
	// a collision the identical way byFace and byUUIDPair already are: a
	// generic pairing name (measured against Card Trader's real catalog:
	// "Soldier // Spirit", "Bird // Myr", "Wolf // Treasure", ...) recurs
	// across more than one set's own token sheet, and two of those can
	// normalize to the identical unordered name pair - the same plurality
	// byFace's own doc comment describes for one anchored face, just on
	// both faces here since neither is anchored by identity at all.
	byBothNames map[[2]string]string
}

// buildTokenPairIndices walks the backend's own derived pairings once and
// builds every index above from it. byFace maps one face's uuid to every
// other face a vendor has been seen pairing it with, keyed by that other
// face's own name normalized (see NormalizeTokenFace), to the derived
// entity's own tcgplayerProductId (see above). Built once, by Load,
// against the backend that holds the pairings rather than per listing - a
// vendor's own catalog runs to six figures, and this side only needs
// looking up for the small fraction shaped like a two-sided token. The
// three land in Backend.TokenPairIndex, Backend.TokenPairIDByUUIDs and
// Backend.TokenPairIDByBothNames, which is what every vendor package
// resolving its own two-sided token listings reads (cardkingdom,
// starcitygames, ...): the indices themselves are vendor-agnostic, keyed
// only by that datastore's own uuids and names.
//
// One face commonly pairs with several different partners across a sheet -
// that plurality is the whole reason a vendor's wording has to disambiguate
// at all - and two of those partners can normalize to the identical key
// (measured against today's datastore: 5,783 derived pairings touch 1,865
// distinct faces across 5,768 (face, key) slots; 252 of those faces (13.5%)
// carry at least one colliding slot, and 297 of the slots themselves (5.1%)
// are actually blanked - a single face can collide on more than one key
// across different sheets, e.g. one "Beast" pairs with two different
// "Elemental" printings on two different sheets). A plain last-write-wins
// map would silently drop one pairing behind the other with no signal
// anything was lost, and a vendor listing that actually names the dropped
// one would then resolve to the wrong physical product under the
// survivor's id. addPairing tracks a collision per (face, key) and blanks
// the entry rather than letting either id win, so a colliding key resolves
// to "" - the caller's own fallback, not a guess between two real answers -
// exactly like every other ambiguous case this index already refuses on.
//
// Built by the loader onto the Backend it describes rather than memoized
// beside the package: two datastores in one process index different
// pairings, and a reload mints new uuids for the same sheets.
func buildTokenPairIndices(b *mtgmatcher.Backend) tokenPairIndicesData {
	byFace := map[string]map[string]string{}
	byUUIDPair := map[[2]string]string{}
	byUUIDPairAmbiguous := map[[2]string]bool{}
	byBothNames := map[[2]string]string{}
	byBothNamesAmbiguous := map[[2]string]bool{}
	ambiguous := map[string]map[string]bool{}

	addPairing := func(face, partnerName, id string) {
		key := NormalizeTokenFace(partnerName)
		if ambiguous[face] == nil {
			ambiguous[face] = map[string]bool{}
		}
		if ambiguous[face][key] {
			return
		}
		if byFace[face] == nil {
			byFace[face] = map[string]string{}
		}
		if existing, found := byFace[face][key]; found {
			if existing == id {
				return
			}
			delete(byFace[face], key)
			ambiguous[face][key] = true
			return
		}
		byFace[face][key] = id
	}

	seen := map[string]bool{}
	for uuid, co := range b.UUIDs {
		if co.Identifiers["derivedTokenPair"] != "true" || seen[uuid] {
			continue
		}
		seen[uuid] = true

		partA := co.Identifiers["tokenPairPartA"]
		partB := co.Identifiers["tokenPairPartB"]
		coA, errA := b.GetUUID(partA)
		coB, errB := b.GetUUID(partB)
		// A vendorVerifiedPair entity (see buildDerivedCard) carries no
		// tcgplayerProductId at all - its own uuid is what every index
		// below hands back instead, which tokenPairingFinishOK/MatchID
		// both resolve directly with no id-space conversion needed.
		id := co.Identifiers["tcgplayerProductId"]
		if id == "" {
			id = uuid
		}

		if errA == nil && errB == nil {
			addPairing(partA, coB.Card.Name, id)
			addPairing(partB, coA.Card.Name, id)
		}

		if partA != "" && partB != "" {
			key := [2]string{partA, partB}
			if partB < partA {
				key = [2]string{partB, partA}
			}
			if byUUIDPairAmbiguous[key] {
				continue
			}
			if existing, found := byUUIDPair[key]; found && existing != id {
				delete(byUUIDPair, key)
				byUUIDPairAmbiguous[key] = true
				continue
			}
			byUUIDPair[key] = id
		}

		if errA == nil && errB == nil {
			faceA, faceB := NormalizeTokenFace(coA.Card.Name), NormalizeTokenFace(coB.Card.Name)
			key := [2]string{faceA, faceB}
			if faceB < faceA {
				key = [2]string{faceB, faceA}
			}
			if byBothNamesAmbiguous[key] {
				continue
			}
			if existing, found := byBothNames[key]; found && existing != id {
				delete(byBothNames, key)
				byBothNamesAmbiguous[key] = true
				continue
			}
			byBothNames[key] = id
		}
	}
	return tokenPairIndicesData{byFace: byFace, byUUIDPair: byUUIDPair, byBothNames: byBothNames}
}

// NormalizeTokenFace reduces one face's name to the form a vendor's own
// wording and the datastore's own name compare equal by: no brace wrapping
// a vendor uses to mark a token name (harmless to strip where a vendor
// never uses it), no artist or variant parenthetical a vendor sometimes
// appends to tell two otherwise-identical tokens apart, no " Token" suffix
// (a vendor spells it, the datastore's own Card.Name from a derived pairing
// does not, since it is the pairing's own combined name split back apart),
// case-folded. The parenthetical must come off before the suffix - it sits
// after the suffix in a vendor's own wording ("X Token (Artist)"), so
// trimming the suffix first leaves it unable to ever match (it never finds
// " Token" at the end of "X Token (Artist)", only of "X Token").
func NormalizeTokenFace(name string) string {
	return strings.ToLower(strings.TrimSpace(CleanFaceName(name)))
}

// StripFaceWrapping removes the cosmetic wrapping a vendor puts on a raw
// face name - braces, an artist or variant parenthetical - while preserving
// case and the " Token" suffix, for a caller that needs the datastore's own
// properly-cased name for an exact-match lookup (MatchInSetNumber) and
// cannot assume the suffix is absent: most token Card.Names drop it, but
// some carry it as part of their own real name (e.g. a promotional token
// disambiguated from a same-named nontoken card, "Kobolds of Kher Keep
// Token"). See CleanFaceName for the form that also strips the suffix.
// Exported for a vendor package that needs to clean a raw face name
// itself - e.g. to anchor each half of a two-sided listing independently
// by its own set and number, rather than through MatchTokenPairing's own
// name-keyed lookup.
func StripFaceWrapping(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "{")
	if idx := strings.Index(name, "}"); idx >= 0 {
		name = name[:idx] + name[idx+1:]
	}
	name = strings.TrimSpace(name)
	if idx := strings.Index(name, " ("); idx >= 0 {
		name = name[:idx]
	}
	return strings.TrimSpace(name)
}

// CleanFaceName is StripFaceWrapping plus whatever type-name suffix a
// vendor spells that the datastore's own Card.Name does not: " Token" for
// an ordinary token (true for every derived pairing's own split-apart
// face, and for mtgjson's own natively combined "X // Y" names,
// MatchNativeTokenPair), " Dungeon" for a dungeon reference card (SCG's
// own listing convention - measured against SCG's real catalog: "Dungeon
// of the Mad Mage Dungeon", never the card's real name "Dungeon of the
// Mad Mage" alone). Neither suffix is ever part of a real Card.Name for
// anything this matches against, so stripping whichever one is present is
// unconditionally safe.
func CleanFaceName(name string) string {
	name = StripFaceWrapping(name)
	for _, suffix := range []string{" Token", " Dungeon"} {
		if trimmed := strings.TrimSuffix(name, suffix); trimmed != name {
			return trimmed
		}
	}
	return name
}

// SplitTokenPairName splits a two-sided token listing's name into its two
// faces the way a vendor spells it - "//" for cards a vendor treats as one
// printing with two names, "-" for a token sheet's own two names - reporting
// whether either separator was found at all. Tried in this order regardless
// of vendor: a name containing "//" is checked for it first, so a vendor
// whose own wording never uses "-" needs no per-vendor configuration to stay
// safe from a false split.
func SplitTokenPairName(name string) (first, second string) {
	for _, sep := range []string{" // ", " - "} {
		if before, after, found := strings.Cut(name, sep); found {
			return before, after
		}
	}
	return name, ""
}

// MatchTokenPairing resolves a two-sided token listing to the TCGplayer
// product id of the physical pairing it names, given the vendor's own
// externalID for one face (a Scryfall id, in every vendor this has been
// built for so far), its own name for the listing as a whole, and whether
// the listing is foil. Either the id or the name alone is not enough to
// place it safely: the id names a real printing, but that printing can pair
// with several different partners across a sheet (above), and the name
// alone is not unique to one set - "Soldier Token" says nothing about which
// Soldier. Together, the id anchors one face precisely and the wording only
// has to pick among the few pairings that exact face has, not guess a
// printing from free text alone.
//
// A caller asking for foil against a pairing that was never sold in foil
// gets "" back, not the nonfoil id: the pairing's own uuid carries no
// separate foil identity to fall back to the way a plain printing's does
// (InputCard.Foil on a bare .ID resolves through the same identifier the
// nonfoil request would, silently substituting the wrong finish's price
// unless this checks first), so the caller's own fallback chain - built for
// exactly this "requested finish doesn't exist" case on every other token
// path - is where the refusal belongs.
//
// Returns "" when the id, the name, or the requested finish don't jointly
// resolve to one derived pairing.
func MatchTokenPairing(b *mtgmatcher.Backend, externalID, listingName string, foil bool) string {
	if externalID == "" {
		return ""
	}
	first, second := SplitTokenPairName(listingName)
	if second == "" {
		return ""
	}
	uuid := b.ConvertID(mtgmatcher.IDSpaceScryfall, externalID)
	if uuid == "" {
		return ""
	}

	// A vendor usually carries the id against the first half of its own
	// listing name, with the second half naming the partner - but not
	// always. Where the id's own real name matches the second half instead,
	// the first half is the partner to look up.
	partner := second
	if co, err := b.GetUUID(uuid); err == nil {
		anchor := NormalizeTokenFace(co.Card.Name)
		if NormalizeTokenFace(second) == anchor && NormalizeTokenFace(first) != anchor {
			partner = first
		}
	}
	id := b.TokenPairIndex[uuid][NormalizeTokenFace(partner)]
	return tokenPairingFinishOK(b, id, foil)
}

// MatchTokenPairingBySetNumber is MatchTokenPairing's counterpart for a
// listing a vendor never publishes an external id for at all: it anchors
// the first face by its own filing set and number instead, through
// MatchInSetNumber - and the same len()==1-or-don't-guess discipline every
// other sku-driven resolution already trusts. Scoped to callers that have
// already confirmed the (setCode, number) pair is the real one a token
// sheet is filed under; unlike an external id, an unconfirmed set/number
// pair is a guess, not an anchor. See MatchTokenPairing for why foil is
// checked here rather than left to the caller.
func MatchTokenPairingBySetNumber(b *mtgmatcher.Backend, setCode, number, listingName string, foil bool) string {
	first, second := SplitTokenPairName(listingName)
	if second == "" || strings.Contains(second, " // ") || strings.Contains(second, " - ") {
		return ""
	}
	// Most token Card.Names drop the " Token" suffix a vendor spells, but
	// some carry it as part of their own real name - try both forms
	// rather than assume either.
	for _, face := range []string{StripFaceWrapping(first), CleanFaceName(first)} {
		cards := b.MatchInSetNumber(face, setCode, number)
		if len(cards) != 1 {
			continue
		}
		id := b.TokenPairIndex[cards[0].UUID][NormalizeTokenFace(second)]
		if id := tokenPairingFinishOK(b, id, foil); id != "" {
			return id
		}
	}
	return ""
}

// MatchNativeTokenPair resolves a two-sided token listing to a real,
// natively-combined mtgjson printing at a known set and number - a
// different case from MatchTokenPairingBySetNumber's synthetic
// tokenProducts-derived pairing (above): mtgjson sometimes already files a
// two-sided token sheet as one ordinary printing of its own, under a
// "X // Y" name, the same way it files a fixed double-faced token like
// Undercity // The Initiative - there is no derived entity to look up,
// only an ordinary printing this vendor's own two-part listing name has to
// be reconstructed to match. Tries both face orders, since a vendor's own
// listing order does not always agree with mtgjson's ("Weird Token //
// Goblin Token" names the same printing as mtgjson's own "Goblin //
// Weird"). Returns "" when neither order resolves to exactly one printing
// - callers apply their own finish handling (e.g. mtgmatcher.MatchID) to
// the uuid this returns, the same as any other set+number resolution.
func MatchNativeTokenPair(b *mtgmatcher.Backend, setCode, number, listingName string) string {
	first, second := SplitTokenPairName(listingName)
	if second == "" {
		return ""
	}
	// mtgjson's own combined name carries neither face's " Token" suffix
	// ("Copy // Horror", never "Copy Token // Horror Token"), the same as
	// a derived pairing's own split-apart faces - CleanFaceName strips it
	// alongside the wrapping, case preserved for this exact-match lookup.
	faceA, faceB := CleanFaceName(first), CleanFaceName(second)
	for _, combined := range []string{faceA + " // " + faceB, faceB + " // " + faceA} {
		out := b.MatchInSetNumber(combined, setCode, number)
		if len(out) == 1 {
			return out[0].UUID
		}
	}
	return ""
}

// tokenPairingFinishOK answers id unchanged when either the caller isn't
// asking for foil, or the derived pairing id names was sold in foil - and ""
// otherwise. A derived entity's uuid carries no separate foil identity a
// bare id lookup falls back to the way a plain printing's does, so asking
// for foil against a pairing that was never sold in one would otherwise
// silently resolve to (and price as) the nonfoil id.
func tokenPairingFinishOK(b *mtgmatcher.Backend, id string, foil bool) string {
	if id == "" || !foil {
		return id
	}
	// id is usually a tcgplayerProductId needing IDSpaceTCGplayer
	// conversion to reach the derived entity's own uuid - but for a
	// vendorVerifiedPair entity (no tcgplayerProductId at all, see
	// buildDerivedCard) every caller in this file hands back the derived
	// entity's own uuid directly instead, which GetUUID already resolves
	// with no conversion needed. Try that first: a real tcgplayerProductId
	// is purely numeric and never collides with this package's own
	// "_tp_"-infixed uuid format, so there is no ambiguity between the
	// two forms.
	uuid := id
	co, err := b.GetUUID(uuid)
	if err != nil {
		uuid = b.ConvertID(mtgmatcher.IDSpaceTCGplayer, id)
		if uuid == "" {
			return ""
		}
		co, err = b.GetUUID(uuid)
		if err != nil {
			return ""
		}
	}

	// co.Card.Finishes is deliberately the UNION of both faces' own finish
	// lists (unionFinishes, above), chosen so a caller asking
	// mtgmatcher.MatchIDFinish about a finish only one face happens to
	// carry gets an answer instead of a hard error - a fine tradeoff
	// there, since nothing IS asked about a finish nobody claims. Here the
	// question is the opposite: a vendor already claims this specific
	// two-sided PRODUCT was sold in the finish. HasFinish on the union
	// would accept that claim on the strength of either face alone, even
	// when the OTHER face was never sold that way - exactly the silent
	// wrong-finish substitution this check exists to prevent, just moved
	// one layer up. Require both source faces to agree instead: not proof
	// the combined product itself shipped in this finish, but a strictly
	// narrower bar than the union, and consistent with every other check
	// in this file preferring "don't know, refuse" over a guess built
	// from a half-agreeing signal.
	coA, errA := b.GetUUID(co.Identifiers["tokenPairPartA"])
	coB, errB := b.GetUUID(co.Identifiers["tokenPairPartB"])
	if errA != nil || errB != nil || !coA.Card.HasFinish("foil") || !coB.Card.HasFinish("foil") {
		return ""
	}
	return id
}

// VerifyTokenPairingFinish is tokenPairingFinishOK for a caller that has
// not already established id names a real derived pairing some other way.
// Every existing caller in this file only ever reaches tokenPairingFinishOK
// after a TokenPairIndex or uuid-pair lookup already confirmed that, so it
// trusts a nonfoil id unconditionally (id == "" || !foil short-circuits
// before ever checking) - correct there, wrong here: a caller with a bare,
// unverified id (Card Trader's own tcgplayerId, which sometimes already
// *is* a pairing's own product id with no name-splitting needed at all)
// needs the derivedTokenPair check to run regardless of the requested
// finish, or an ordinary card's id would pass straight through unchecked
// whenever foil is false.
func VerifyTokenPairingFinish(b *mtgmatcher.Backend, id string, foil bool) string {
	uuid := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, id)
	if uuid == "" {
		return ""
	}
	co, err := b.GetUUID(uuid)
	if err != nil || co.Identifiers["derivedTokenPair"] != "true" {
		return ""
	}
	return tokenPairingFinishOK(b, id, foil)
}

// MatchTokenPairingByUUIDs resolves a two-sided token listing given both
// faces' own uuids, each already anchored unambiguously by identity
// (e.g. via MatchInSetNumber on that face's own filing set and number)
// rather than guessed from a vendor's own wording. Returns "" when no
// derived pairing exists for this exact uuid pair (a real, verified
// pairing this vendor sells that TCGplayer's own tokenProducts feed
// simply has no product for - not a matching failure, a data gap one
// level up), or when the pairing was never sold in the requested finish
// (see tokenPairingFinishOK).
func MatchTokenPairingByUUIDs(b *mtgmatcher.Backend, uuidA, uuidB string, foil bool) string {
	if uuidA == "" || uuidB == "" {
		return ""
	}
	key := [2]string{uuidA, uuidB}
	if uuidB < uuidA {
		key = [2]string{uuidB, uuidA}
	}
	return tokenPairingFinishOK(b, b.TokenPairIDByUUIDs[key], foil)
}

// EditionTokenSetCode resolves an edition name to the set code its own
// tokens are filed under, walking ParentCode the way deriveTokenPairs'
// own homeSet does at mint time (that version is build-time only,
// operating on a map[string]*Set rather than the loaded backend - this is
// the same logic against the public GetSet/GetSetByName API instead, for
// a caller with only a vendor's own edition wording to start from - CK/SCG
// anchor by a sku's own embedded set code instead and have never needed
// this). Returns "" when the edition doesn't resolve to a real set, or
// that set's own chain never reaches one with a token sheet - refuse
// rather than guess, the same discipline as everywhere else in this file.
func EditionTokenSetCode(b *mtgmatcher.Backend, edition string) string {
	set, err := b.GetSetByName(edition)
	if err != nil {
		return ""
	}
	code := set.Code
	seen := map[string]bool{}
	for !seen[code] {
		seen[code] = true
		s, err := b.GetSet(code)
		if err != nil {
			return ""
		}
		if s.TokenSetCode != "" {
			return s.TokenSetCode
		}
		if s.ParentCode == "" {
			return ""
		}
		code = s.ParentCode
	}
	return ""
}

// MatchTokenPairingByNamesAndEdition resolves a two-sided token listing
// from both faces' own names plus the vendor's own claimed edition alone -
// for a listing that carries neither an id nor a filing set and number to
// anchor either face by identity at all (Card Trader's own no-id
// remainder: ~62% of its two-sided token catalog carries neither a
// scryfall_id nor a tcgplayer_id, and its own Token-category listings
// carry no collector number either, unlike Card Kingdom's or Star City
// Games's skus). Every other matcher in this file anchors at least one
// face by identity before trusting a name for the other; this trusts both
// purely from the vendor's own wording, a meaningfully weaker bar -
// Backend.TokenPairIDByBothNames' own collision-blanking catches two DIFFERENT
// pairings sharing the identical two names, but not one CURRENTLY-unique
// pairing that happens to share a generic name with a DIFFERENT real
// pairing the datastore simply hasn't modeled (yet, or never will): a
// listing naming a real, older set's own "Soldier // Spirit" would
// resolve to whichever set's "Soldier // Spirit" the datastore currently
// derives, silently wrong, without a second check.
//
// listingEdition is that second check: it must independently name the
// same set the byBothNames match resolved to (via EditionTokenSetCode),
// or this refuses rather than trust the name pair alone. Measured against
// Card Trader's real no-id catalog: of 970 blueprints whose two face names
// matched exactly one derived pairing, 831 (85.6%) had their own claimed
// edition agree with the match once checked this way, and 113 (11.6%)
// did not - a real rate of exactly the failure this check exists to
// catch, not a theoretical one.
func MatchTokenPairingByNamesAndEdition(b *mtgmatcher.Backend, listingName, listingEdition string, foil bool) string {
	first, second := SplitTokenPairName(listingName)
	if second == "" {
		return ""
	}
	faceA, faceB := NormalizeTokenFace(first), NormalizeTokenFace(second)
	key := [2]string{faceA, faceB}
	if faceB < faceA {
		key = [2]string{faceB, faceA}
	}
	id := b.TokenPairIDByBothNames[key]
	if id == "" {
		return ""
	}

	wantSet := EditionTokenSetCode(b, listingEdition)
	if wantSet == "" {
		return ""
	}
	uuid := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, id)
	co, err := b.GetUUID(uuid)
	if err != nil || !strings.EqualFold(co.SetCode, wantSet) {
		return ""
	}

	return tokenPairingFinishOK(b, id, foil)
}
