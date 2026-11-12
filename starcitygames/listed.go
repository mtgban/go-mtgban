package starcitygames

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/mtgban/go-mtgban/matcher"
)

// catalogSpellings counts how the catalog export words each finish and
// rarity the index numbers, on the products both of them list, so a product
// only the index carries can be worded the way the catalog words the rest.
type catalogSpellings struct {
	game         string
	finishes     map[string]map[string]int
	finishGroups map[string]map[string]int
	rarities     map[string]map[string]int

	// letterFinishes and letterGroups count the same wording by the finish
	// letter of the sku, for a finish the index numbers on no product the
	// catalog also lists.
	letterFinishes map[string]map[string]int
	letterGroups   map[string]map[string]int
}

func (s *catalogSpellings) alloc() {
	if s.finishes == nil {
		s.finishes = map[string]map[string]int{}
		s.finishGroups = map[string]map[string]int{}
		s.rarities = map[string]map[string]int{}
		s.letterFinishes = map[string]map[string]int{}
		s.letterGroups = map[string]map[string]int{}
	}
}

// learnLetter counts how a catalog single words the finish its sku letter
// names. A blank finish teaches nothing, since it would read as plain.
func (s *catalogSpellings) learnLetter(p CatalogProduct) {
	letter, ok := skuFinishLetter(p.SKU)
	if !ok || p.Finish == "" {
		return
	}
	s.alloc()
	s.game = p.Game
	countSpelling(s.letterFinishes, letter, p.Finish)
	countSpelling(s.letterGroups, letter, p.FinishGroup)
}

// letterFinish returns the catalog's word for the finish letter of a sku,
// only where every catalog single with that letter words it the same way:
// a letter shared by several finishes (Magic's F, Lorcana's A) names none.
func (s *catalogSpellings) letterFinish(sku string) (string, bool) {
	letter, ok := skuFinishLetter(sku)
	if !ok || len(s.letterFinishes[letter]) != 1 {
		return "", false
	}
	return spelling(s.letterFinishes, letter)
}

func (s *catalogSpellings) learn(hit sellListProduct, p CatalogProduct) {
	s.alloc()
	s.game = p.Game
	countSpelling(s.finishes, sellListID(hit.Finish), p.Finish)
	countSpelling(s.finishGroups, sellListID(hit.Finish), p.FinishGroup)
	countSpelling(s.rarities, sellListID(hit.Rarity), p.Rarity)
}

// sellListID is the index's number for a finish or rarity as a key, or empty
// where it gives none.
func sellListID(value any) string {
	n, ok := value.(float64)
	if !ok {
		return ""
	}
	return strconv.FormatFloat(n, 'f', -1, 64)
}

func countSpelling(counts map[string]map[string]int, id, spelling string) {
	if id == "" {
		return
	}
	if counts[id] == nil {
		counts[id] = map[string]int{}
	}
	counts[id][spelling]++
}

// spelling returns the word most of the products both list use for an id,
// which outvotes the odd product the catalog files under another.
func spelling(counts map[string]map[string]int, id string) (string, bool) {
	var best string
	var most int
	for word, n := range counts[id] {
		if n > most || n == most && word < best {
			best, most = word, n
		}
	}
	return best, most > 0
}

// catalogLanguage returns the catalog's name for a language the index codes.
func catalogLanguage(code string) (string, bool) {
	for name, c := range scgLanguageCodes {
		if c == code {
			return name, true
		}
	}
	return "", false
}

// listedProduct words a product only the index carries the way the catalog
// export would. It is false where the catalog spelled none of the products
// sharing its language or rarity this run, or sharing its finish unless the
// catalog words the sku's finish letter one way only, since a blank there
// would read as English or as plain.
func (scg *Starcitygames) listedProduct(sku string, hit sellListProduct) (CatalogProduct, bool) {
	language, ok := catalogLanguage(hit.Language)
	if !ok {
		return CatalogProduct{}, false
	}
	finish, ok := spelling(scg.spellings.finishes, sellListID(hit.Finish))
	group, _ := spelling(scg.spellings.finishGroups, sellListID(hit.Finish))
	if !ok {
		finish, ok = scg.spellings.letterFinish(sku)
		if !ok {
			return CatalogProduct{}, false
		}
		letter, _ := skuFinishLetter(sku)
		group, _ = spelling(scg.spellings.letterGroups, letter)
	}
	rarity, ok := spelling(scg.spellings.rarities, sellListID(hit.Rarity))
	if !ok {
		return CatalogProduct{}, false
	}
	return CatalogProduct{
		SKU:             sku,
		ProductType:     ProductTypeSingles,
		Name:            hit.Name,
		Game:            scg.spellings.game,
		Set:             hit.SetName,
		Finish:          finish,
		FinishGroup:     group,
		Language:        language,
		Rarity:          rarity,
		CollectorNumber: hit.CollectorNumber,
	}, true
}

// fileListed prices the singles the index bids on that the catalog export
// does not list at all, foreign printings and Flesh and Blood promos for the
// most part. With no vendor id they resolve from the sku and the wording, as
// a catalog product without one does, and they carry no retail.
func (scg *Starcitygames) fileListed() {
	var listed, unspelled, misplaced int
	for _, sku := range slices.Sorted(maps.Keys(scg.listed)) {
		if _, found := scg.catalogued[sku]; found || !strings.HasPrefix(sku, "SGL-") {
			continue
		}
		listed++
		p, ok := scg.listedProduct(sku, scg.listed[sku])
		if !ok {
			unspelled++
			continue
		}
		if scg.fileProduct(p, true) {
			misplaced++
		}
	}
	if listed > 0 {
		scg.printf("%d singles only the sell list carries, %d worded in a way the catalog never spelled, %d landed on another printing", listed, unspelled, misplaced)
	}
}

// listedLanding reports whether a product only the sell list carries landed
// on the printing its sku names. With no vendor id to vouch for it, a wording
// miss widens to any printing of the name, so the sku's finish letter has to
// agree, and a World Championship card has to stay in its year's set, its
// player's deck and the half of it the sku names.
func listedLanding(b *matcher.Backend, sku, cardID string) bool {
	co, err := b.GetUUID(cardID)
	if err != nil {
		return false
	}
	letter, ok := skuFinishLetter(sku)
	if !ok {
		return false
	}
	if (co.Foil || co.Etched) != (letter != "N") {
		return false
	}

	set, named := worldsSetFromSKU(sku)
	if !named {
		return true
	}
	initials, _, _ := strings.Cut(skuNumber(sku)[2:], "_")
	player := strings.TrimSuffix(co.Number, strings.TrimLeftFunc(co.Number, unicode.IsLetter))
	sideboard := strings.HasSuffix(skuNumber(sku), "s")
	return co.SetCode == set && player == strings.ToLower(initials) &&
		sideboard == strings.Contains(co.Number, "sb")
}
