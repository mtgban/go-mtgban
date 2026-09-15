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

// tokenPairReport tallies why a tokenProducts pairing did or did not
// produce a derived entity, logged once so a fresh datastore's shape is
// visible without re-deriving these counts from scratch.
type tokenPairReport struct {
	entries          int // tokenProducts entries with exactly 2 tokenParts
	faceIDOnly       int // a face mtgjson never catalogued as its own card
	selfPair         int // both faces are the same uuid
	unresolvableUUID int // a face's uuid is not (or no longer) in this datastore
	alreadyModeled   int // one face is itself layout "double_faced_token"
	layoutExcluded   int // a face's layout is not token or emblem
	noUsableID       int // every id naming this pair is ambiguous or already claimed
	noCommonAncestor int // the two faces' sets share no ancestor
	derived          int // entities actually minted
}

func (r tokenPairReport) String() string {
	return fmt.Sprintf(
		"tokenProducts pairs: %d entries -> %d derived (excluded: %d faceId-only, %d self-pair, "+
			"%d unresolvable uuid, %d already modeled, %d layout, %d no usable id, %d no common ancestor)",
		r.entries, r.derived, r.faceIDOnly, r.selfPair, r.unresolvableUUID,
		r.alreadyModeled, r.layoutExcluded, r.noUsableID, r.noCommonAncestor)
}

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
func deriveTokenPairs(sets map[string]*Set, uuids map[string]*mtgmatcher.CardObject, tcgIDs map[string]string) ([]Card, tokenPairReport) {
	var report tokenPairReport

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
					report.entries++

					id := tp.Identifiers["tcgplayerProductId"]
					u1, u2 := tp.TokenParts[0].UUID, tp.TokenParts[1].UUID
					if id == "" || u1 == "" || u2 == "" {
						report.faceIDOnly++
						continue
					}
					if u1 == u2 {
						report.selfPair++
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

	var derived []Card
	for _, key := range keys {
		co1, found1 := uuids[key.a]
		co2, found2 := uuids[key.b]
		if !found1 || !found2 {
			report.unresolvableUUID++
			continue
		}
		if co1.Layout == "double_faced_token" || co2.Layout == "double_faced_token" {
			report.alreadyModeled++
			continue
		}
		if !isTokenPairLayout(co1.Layout) || !isTokenPairLayout(co2.Layout) {
			report.layoutExcluded++
			continue
		}

		var usableIDs []int
		for id := range pairIDs[key] {
			if len(idPairs[id]) > 1 {
				continue
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
			report.noUsableID++
			continue
		}
		sort.Ints(usableIDs)

		home := homeSet(sets, co1.SetCode, co2.SetCode)
		if home == "" {
			report.noCommonAncestor++
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

		report.derived++
	}

	return derived, report
}

// isTokenPairLayout reports whether a face's layout is one this loader
// knows how to pair. This is an allowlist, not a denylist: layout is
// upstream data mtgjson can add values to, and a denylist would let a new
// shape through unexamined. It deliberately excludes "art_series" (already
// dropped by skipSet) and "normal"/"flip"/"reversible_card" (the AFR
// dungeon sheets, a numbered card rather than a token on one side - a real
// pairing, but a different problem, left for its own follow-up). adjustTokens
// (mtgjson.go) already rewrites every set.Tokens entry with the right Types
// to "token" before this runs, so against today's data this rung and the
// "already modeled" one above it are not exercised - dungeon-type pairings
// are excluded by the multi-pair-id rung instead, since OAFR repeats AFR's
// own ids under its own uuids. Both stay as an explicit backstop for
// whatever upstream does next, not dead code to delete.
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
// usableIDs is sorted ascending and non-empty.
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
	number := first.Number + " // " + second.Number
	uuid := uuidLo + derivedTokenPairSuffix + uuidHi

	ids := make([]string, len(usableIDs))
	for i, n := range usableIDs {
		ids[i] = strconv.Itoa(n)
	}

	identifiers := map[string]string{
		"derivedTokenPair":    "true",
		"tokenPairPartA":      uuidLo,
		"tokenPairPartB":      uuidHi,
		"tcgplayerProductId":  ids[0],
		"tcgplayerProductIds": strings.Join(ids, ","),
		"tokenSetCode":        home,
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
