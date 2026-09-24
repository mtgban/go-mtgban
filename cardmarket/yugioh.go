package cardmarket

import (
	"regexp"
	"slices"
	"strings"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// yugiohExpansions maps the Cardmarket Yu-Gi-Oh expansions the matcher
// resolves to no set onto the sets their bridged products land in, by code
// where the catalog's own name is nothing the storefront would write. Most
// entries were read off the cardtrader bridge: every product of the
// expansion that carries a TCGplayer id lands in the set named. The rest
// carry no bridged product at all and were read off the shelf's own
// Cardmarket expansion code equaling a datastore set code instead. A shelf
// spelled with a language code ("Spell Ruler (SDM)", "Legend of Blue Eyes
// White Dragon (LDD)") is a print in another language and stays out on
// purpose.
var yugiohExpansions = map[string][]string{
	"Legend of Blue Eyes White Dragon":             {"LOB"},
	"2025 Mega-Pack Tin":                           {"MP25"},
	"2020 Tin of Lost Memories Mega Pack":          {"MP20"},
	"Yugi's Legendary Decks":                       {"YGLD"},
	"Shonen Jump Magazine":                         {"JUMP", "JMPS", "JMP"},
	"Lost Art Promos":                              {"LART"},
	"Gold Series 5: Haunted Mine":                  {"GLD5"},
	"Gold Series 2":                                {"GLD2"},
	"Starter Deck: 2009":                           {"5DS2"},
	"Starter Deck: Syrus":                          {"YSDS"},
	"Starter Deck: GX 2006":                        {"YSD"},
	"Structure Deck: The Realm of Light":           {"SDLI"},
	"Yu-Gi-Oh! Championship Prize Cards 2025":      {"25YC"},
	"Yu-Gi-Oh! Championship Series":                {"YCSW"},
	"Premium Collection":                           {"PRC1"},
	"Turbo Pack":                                   {"TU01"},
	"McDonald's Promo Pack 2":                      {"MDP2"},
	"World Championship Celebration Promos":        {"WCJPP"},
	"Booster Pack Tin":                             {"BPT", "BPT-1341"},
	"3D Bonds Beyond Time Movie Pack":              {"YMP1"},
	"Master Collection Vol. 2":                     {"MC2"},
	"The Falsebound Kingdom Promos":                {"TFK"},
	"Dawn of Destiny":                              {"DOD"},
	"Gameboy Worldwide Edition Promos":             {"GBI"},
	"Token Promos 1":                               {"TKN"},
	"Token Promos 2":                               {"TKN2"},
	"Token Promos 4":                               {"TKN", "JPRC"},
	"Promos":                                       {"MISC", "EFC1"},
	"Sneak Preview 2":                              {"SP2", "SP02"},
	"Mattel Action Figure Serie 1":                 {"MF01"},
	"Mattel Action Figure Serie 2":                 {"MF02"},
	"Yu-Gi-Oh! Early Days Collection":              {"EDC1"},
	"Duelist Pack Collection Tin 2011":             {"DPCT-DPC5"},
	"ZEXAL World Duel Carnival Promos":             {"ZDC1"},
	"World Championship 2004":                      {"WC4"},
	"World Championship 2005":                      {"WC5"},
	"5D's Over the Nexus Promotional Cards":        {"WC11"},
	"Yu-Gi-Oh! 5D's Wheelie Breakers Promos":       {"WB01"},
	"Hidden Arsenal: Special Edition":              {"HA04"},
	"Shonen Jump Championship Series":              {"G280", "SJCS"},
	"Speed Duel Starter Decks: Ultimate Predators": {"SS03"},
	"Exclusive Pack 1":                             {"EP1"},
	"Mattel Action Figure Serie 3":                 {"MF03"},
	"5D's Tag Force 5 Promotional Cards":           {"TF05"},
	"Master Collection Vol. 1":                     {"MC1"},
	"Speed Duel: Event Pack":                       {"EVSD"},
	"Legendary Collection":                         {"LC01"},
	"Sneak Preview 1":                              {"SP1"},
	"Yu-Gi-Oh! The Movie":                          {"MOV"},
	"GX Tag Force 3":                               {"GX06"},
	"Nightmare Troubadour":                         {"NTR"},
	"GX Tag Force (PSP)":                           {"GX02"},
	"Duelist Pack 1&2 Special Edition":             {"DPK"},
	"Pharaoh Tour 2005":                            {"PT1"},
	"Pharaoh Tour 2006":                            {"PT02"},
	"Pharaoh Tour 2007":                            {"PT03"},
	"Ultimate Edition Promotional Cards: Series 2": {"UE02"},
	"Fire Fists Special Edition":                   {"FFSE"},
	"Duelist Pack Collection Tin 2010":             {"DPCT-192", "DPCT"},
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
func yugiohSameProduct(a, b *cm.Product) bool {
	return mtgmatcher.Normalize(versionTail.ReplaceAllString(a.Name, "")) == mtgmatcher.Normalize(versionTail.ReplaceAllString(b.Name, ""))
}

// yugiohVersionVariants names the printings a shelf sells under one name,
// one number and one rarity, in the order Cardmarket's version index counts
// them. Winner's Pack 2026-2027 hands the same forty cards out through three
// programmes and stamps each with the programme's mark; the datastore keeps
// the three apart by that stamp, and Cardmarket keeps them apart by nothing
// at all - V.1, V.2 and V.3 carry the same rarity, the same reprint count
// and a web address that only repeats the name. Without a label the three
// products alias onto the three rows and none of them prices.
//
// THE ORDER IS A DEFAULT, NOT A FACT. Nothing publishes it, and CardTrader,
// which arbitrates elsewhere, carries no WI26 expansion at all. What the
// published price guide says is that V.1 is the cheapest of the three on all
// 25 cards priced in every version, and that V.3 is the dearest on 21 of
// them - which fits OTS packs being the widely handed-out programme and the
// judge mark the scarce one, and is the whole of the evidence. The prices
// within one card run from 22.50 to 550, so a row found wrong is worth
// correcting here rather than reasoning about: swap two labels and the
// products follow.
var yugiohVersionVariants = map[string][]string{
	"WI26": {"OTS Stamp", "Regional Qualifier Stamp", "Judge Stamp"},
}

// yugiohOversized answers the oversized card a product names, which the
// shelf cannot. An oversized card is not the set's card: the marketplace
// files it under the deck it was handed out with - Machina Fortress under
// Structure Deck: Machina Mayhem, numbered SDMM-EN001 - and the datastore
// files it in the collector or value box it actually came in, which for that
// card is VBX. Set and number therefore disagree by construction, and the
// name and the tag are what is left.
//
// It answers only where exactly one printing of ours carries the name and
// the tag. Twenty-six of the thirty-eight oversized products have no such
// printing at all and stay refused, which is what they are: cards we do not
// carry.
func yugiohOversized(b *mtgmatcher.Backend, name string) (string, error) {
	uuids, err := b.SearchEquals(name)
	if err != nil {
		return "", errNoPrinting
	}
	var found string
	for _, uuid := range uuids {
		co, err := b.GetUUID(uuid)
		if err != nil || !slices.Contains(co.PromoTypes, "oversized") {
			continue
		}
		if found != "" {
			return "", errNoPrinting
		}
		found = uuid
	}
	if found == "" {
		return "", errNoPrinting
	}
	return found, nil
}

// yugiohOtherCard reports whether the bridged printing is of a card the
// product's name is not, under any name Konami gave it.
func (r *resolver) yugiohOtherCard(product *cm.Product, cardID string) bool {
	co, err := r.backend.GetUUID(cardID)
	if err != nil {
		return false
	}
	name := versionTail.ReplaceAllString(product.Name, "")
	konamiID := co.Identifiers["konamiId"]
	if konamiID == "" || mtgmatcher.Equals(co.Name, name) {
		return false
	}
	uuids, err := r.backend.SearchEquals(name)
	if err != nil {
		return false
	}
	for _, uuid := range uuids {
		named, err := r.backend.GetUUID(uuid)
		if err == nil && named.Identifiers["konamiId"] == konamiID {
			return false
		}
	}
	return true
}

// yugiohOtherNumber reports whether the bridged printing is numbered for
// another printing of the card: a special edition, box topper or European
// print that CardTrader links to the set's base row. Only the digits count,
// and the EN infix a European number ("EN086", "E001") needs; Cardmarket
// writes "SP02" where the datastore writes "ENS02".
func (r *resolver) yugiohOtherNumber(product *cm.Product, cardID string) bool {
	co, err := r.backend.GetUUID(cardID)
	if err != nil {
		return false
	}
	tail, rowTail := numberTail.FindString(product.Number), numberTail.FindString(co.Number)
	if tail == "" || rowTail == "" {
		return false
	}
	if !strings.EqualFold(strings.TrimLeft(tail, "0"), strings.TrimLeft(rowTail, "0")) {
		return true
	}
	return strings.HasPrefix(numberPrefix(product.Number), "E") && !strings.HasPrefix(numberPrefix(co.Number), "EN")
}

// yugiohWorded answers the printing the product's name and rarity reach,
// when the rarity names it and not the bridged printing.
func (r *resolver) yugiohWorded(product *cm.Product, cardID string) string {
	fields := rarityTail.FindStringSubmatch(product.Name)
	if fields == nil {
		return ""
	}
	co, err := r.backend.GetUUID(cardID)
	if err != nil || rarityNames(fields[1], co.Rarity) {
		return ""
	}
	id, err := r.matchYugioh(product)
	if err != nil || id == "" || id == cardID {
		return ""
	}
	alt, err := r.backend.GetUUID(id)
	if err != nil || !rarityNames(fields[1], alt.Rarity) {
		return ""
	}
	// Keep the bridged run, which the columns are laid out by.
	run, err := r.backend.MatchIDFinish(id, co.Finish)
	if err == nil {
		id = run
	}
	return id
}

// rarityNames reports whether every word of the storefront's rarity is a
// word of the datastore's, which may decorate it ("Prismatic Collector's").
func rarityNames(worded, rarity string) bool {
	words := strings.Fields(strings.ToLower(strings.ReplaceAll(rarity, "'", "")))
	for _, word := range strings.Fields(strings.ToLower(strings.ReplaceAll(worded, "'", ""))) {
		if !slices.Contains(words, word) {
			return false
		}
	}
	return true
}
