package cardmarket

import (
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// yugiohExpansions maps the Cardmarket Yu-Gi-Oh expansions the matcher
// resolves to no set onto the sets their bridged products land in, by code
// where the catalog's own name is nothing the storefront would write. Each
// entry was read off the cardtrader bridge: every product of the expansion
// that carries a TCGplayer id lands in the set named. A shelf spelled with a
// language code ("Spell Ruler (SDM)", "Legend of Blue Eyes White Dragon
// (LDD)") is a print in another language and stays out on purpose.
var yugiohExpansions = map[string][]string{
	"Legend of Blue Eyes White Dragon":        {"LOB"},
	"2025 Mega-Pack Tin":                      {"MP25"},
	"2020 Tin of Lost Memories Mega Pack":     {"MP20"},
	"Yugi's Legendary Decks":                  {"YGLD"},
	"Shonen Jump Magazine":                    {"JUMP", "JMPS", "JMP"},
	"Lost Art Promos":                         {"LART"},
	"Gold Series 5: Haunted Mine":             {"GLD5"},
	"Gold Series 2":                           {"GLD2"},
	"Starter Deck: 2009":                      {"5DS2"},
	"Starter Deck: Syrus":                     {"YSDS"},
	"Starter Deck: GX 2006":                   {"YSD"},
	"Structure Deck: The Realm of Light":      {"SDLI"},
	"Yu-Gi-Oh! Championship Prize Cards 2025": {"25YC"},
	"Yu-Gi-Oh! Championship Series":           {"YCSW"},
	"Premium Collection":                      {"PRC1"},
	"Turbo Pack":                              {"TU01"},
	"McDonald's Promo Pack 2":                 {"MDP2"},
	"World Championship Celebration Promos":   {"WCJPP"},
	"Booster Pack Tin":                        {"BPT", "BPT-1341"},
	"3D Bonds Beyond Time Movie Pack":         {"YMP1"},
	"Master Collection Vol. 2":                {"MC2"},
	"The Falsebound Kingdom Promos":           {"TFK"},
	"Dawn of Destiny":                         {"DOD"},
	"Gameboy Worldwide Edition Promos":        {"GBI"},
	"Token Promos 1":                          {"TKN"},
	"Token Promos 2":                          {"TKN2"},
	"Token Promos 4":                          {"TKN", "JPRC"},
	"Promos":                                  {"MISC", "EFC1"},
	"Sneak Preview 2":                         {"SP2", "SP02"},
	"Mattel Action Figure Serie 1":            {"MF01"},
	"Mattel Action Figure Serie 2":            {"MF02"},
	"Yu-Gi-Oh! Early Days Collection":         {"EDC1"},
	"Duelist Pack Collection Tin 2011":        {"DPCT-DPC5"},
	"ZEXAL World Duel Carnival Promos":        {"ZDC1"},
	"World Championship 2004":                 {"WC4"},
	"World Championship 2005":                 {"WC5"},
	"5D's Over the Nexus Promotional Cards":   {"WC11"},
	"Yu-Gi-Oh! 5D's Wheelie Breakers Promos":  {"WB01"},
	"Hidden Arsenal: Special Edition":         {"HA04"},
	"Shonen Jump Championship Series":         {"G280", "SJCS"},
}

// yugiohShelfPatterns are the expansion families spelled one way per member,
// each family's members landing in the set of the same index: the Duelist
// League participation cards, the Champion and Astral packs, the yearly
// collector's tins and the Mega-Tin packs.
var yugiohShelfPatterns = []struct {
	re   *regexp.Regexp
	code string
}{
	{regexp.MustCompile(`^Duelist League (\d\d)$`), "DL$1"},
	{regexp.MustCompile(`^Champion Pack: Game (\w+)$`), "CP0%d"},
	{regexp.MustCompile(`^Astral Pack (\w+)$`), "AP0%d"},
	{regexp.MustCompile(`^Collector's Tins (\d{4})$`), "$1 Collectors Tin"},
	{regexp.MustCompile(`^(\d{4}) Mega-Tin Mega Pack$`), "$1 Mega-Tins Mega Pack"},
}

var yugiohOrdinals = map[string]int{
	"One": 1, "Two": 2, "Three": 3, "Four": 4, "Five": 5, "Six": 6, "Seven": 7, "Eight": 8, "Nine": 9,
}

// yugiohEditions answers the sets a Cardmarket expansion may hold, by name
// or code, the expansion's own name last.
func yugiohEditions(expansion string) []string {
	if sets, found := yugiohExpansions[expansion]; found {
		return sets
	}
	for _, pattern := range yugiohShelfPatterns {
		m := pattern.re.FindStringSubmatch(expansion)
		if m == nil {
			continue
		}
		if strings.Contains(pattern.code, "%d") {
			if n, found := yugiohOrdinals[m[1]]; found {
				return []string{strings.Replace(pattern.code, "%d", string(rune('0'+n)), 1)}
			}
			continue
		}
		return []string{pattern.re.ReplaceAllString(expansion, pattern.code)}
	}
	return []string{expansion}
}

// yugiohSameProduct reports whether two products are the same card sold
// twice: the same name once the version index is off it. The number is not
// asked, because the oldest sets are sold once more under a numbering
// shifted by a card or two, and two products named alike that reached one
// printing are that printing's however they are numbered.
func yugiohSameProduct(a, b *MKMProduct) bool {
	return mtgmatcher.Normalize(versionTail.ReplaceAllString(a.Name, "")) == mtgmatcher.Normalize(versionTail.ReplaceAllString(b.Name, ""))
}

// matchYugioh names a Yu-Gi-Oh product's printing from what the catalog
// says of it, held to the sets its expansion may hold. A number written with
// a region prefix names a print run of its own ("EN000" is the European
// print, "A000" the Asian), carried only where the catalog has a set for it;
// an expansion naming no set of ours, or a run we have no set for, is a
// catalog we do not carry rather than a product that named nothing.
func (mkm *Index) matchYugioh(product *MKMProduct) (string, error) {
	name := versionTail.ReplaceAllString(product.Name, "")
	var rarity string
	if fields := rarityTail.FindStringSubmatch(product.Name); fields != nil {
		rarity = fields[1]
	}
	region := ""
	if product.Number != "" {
		region = numberPrefix(product.Number)
	}
	tail := numberTail.FindString(product.Number)
	finishes := []string{yugiohRun(product), "Unlimited", ""}

	carried := false
	for _, edition := range yugiohEditions(product.ExpansionName) {
		set, err := mtgmatcher.GetSetByName(edition)
		if err != nil {
			continue
		}
		carried = true
		sets := []*mtgmatcher.Set{set}
		// The European print is a set of its own where the catalog has
		// one, numbered with the region the product writes.
		if region == "EN" {
			worldwide, err := mtgmatcher.GetSet(set.Code + "-EN")
			if err == nil {
				sets = []*mtgmatcher.Set{worldwide}
			}
		}
		for _, set := range sets {
			numbers := []string{product.Number}
			if tail != "" {
				base := strings.TrimSuffix(set.Code, "-EN")
				if region == "" {
					// The catalog writes the region into most sets'
					// numbers ("MP18-EN065") and Cardmarket leaves it
					// out; both spellings are asked.
					numbers = append(numbers, base+"-"+tail, base+"-EN"+tail)
				} else {
					numbers = append(numbers, base+"-"+region+tail)
				}
			}
			// A number the set gives to another card is the storefront's
			// mistake rather than the card's: the oldest sets are sold
			// once more under a numbering shifted by a card or two, and
			// the name is what still says which card it is. Only a
			// number so contradicted is set aside; a number the set
			// merely lacks stays a refusal.
			if tail != "" && mkm.yugiohNumberTaken(set.Code, numbers[1:], name) {
				numbers = append(numbers, "")
			}
			for _, number := range numbers {
				variation := strings.TrimSpace(number + " " + rarity)
				for _, finish := range finishes {
					id, err := mtgmatcher.Match(&mtgmatcher.InputCard{
						Name:      name,
						Edition:   set.Name,
						Variation: variation,
						Finish:    finish,
					})
					if err != nil {
						continue
					}
					co, err := mtgmatcher.GetUUID(id)
					if err != nil || !strings.EqualFold(co.SetCode, set.Code) {
						continue
					}
					if number != "" && otherPrintRun(product.Number, co.Number) {
						continue
					}
					return id, nil
				}
			}
		}
	}
	if !carried || region != "" {
		return "", errForeign
	}
	return "", errNoPrinting
}

// yugiohNumberTaken reports whether one of the numbers names a card of the
// set other than the one named. The index is built once, over the whole
// datastore, the first time a number is contradicted.
func (mkm *Index) yugiohNumberTaken(setCode string, numbers []string, name string) bool {
	mkm.numbersMu.Lock()
	defer mkm.numbersMu.Unlock()
	if mkm.numbers == nil {
		mkm.numbers = map[string]map[string]string{}
		for _, uuid := range mtgmatcher.GetUUIDs() {
			co, err := mtgmatcher.GetUUID(uuid)
			if err != nil {
				continue
			}
			index := mkm.numbers[co.SetCode]
			if index == nil {
				index = map[string]string{}
				mkm.numbers[co.SetCode] = index
			}
			index[strings.ToUpper(co.Number)] = mtgmatcher.Normalize(co.Name)
		}
	}
	index := mkm.numbers[setCode]
	for _, number := range numbers {
		holder, held := index[strings.ToUpper(number)]
		if held && holder != mtgmatcher.Normalize(name) {
			return true
		}
	}
	return false
}
