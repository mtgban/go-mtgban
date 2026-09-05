package coolstuffinc

import (
	"regexp"
	"strings"
)

// palworldNumberTail is a collector number with a rarity code written onto
// its end, "EBP01-025RR".
var palworldNumberTail = regexp.MustCompile(`^([A-Za-z]+[0-9]*-[0-9]+)([A-Za-z]+)$`)

// palworldBaseRarities are the rarity codes this game numbers no printing
// apart by, which is the whole of what the storefront may have glued to a
// number: the codes left out are the six the catalog files a collector
// number of its own under - OSR, SP, SR, SSP, TSP, TSR - and a number
// ending in one of those names that printing rather than restating the
// card's rarity, so the two must not be pulled apart.
//
// The list is the rarity facet this storefront publishes for the game,
// minus those six. It is enumerated rather than inverted so that a tier
// added to a later set is left alone: a code nobody here has heard of is a
// number's own tail until the catalog says otherwise, and reading it as a
// rarity would answer the base printing for a parallel.
var palworldBaseRarities = map[string]bool{
	"C":   true,
	"PO":  true,
	"PR":  true,
	"R":   true,
	"RR":  true,
	"SSS": true,
	"TD":  true,
	"TDR": true,
	"U":   true,
}

// palworldNotes writes a note's collector number and the rarity code this
// storefront sometimes glues to it back apart.
//
// The glue is this storefront's alone, and not even consistently its own:
// it files Victor's Strategy at "EBP01-046 RR" and Chillet at
// "EBP01-025RR" on the same shelf, and its buylist writes every number
// clean. Nobody else writes the code into the number either - Bushiroad's
// own card list numbers BP01-025 and carries RR in a rarity field,
// CardTrader splits the same two into collector_number and
// palworld_rarity, and TCGplayer, which the catalog takes its identity
// from, only ever suffixes the parallel tiers.
func palworldNotes(notes string) string {
	fields := strings.Fields(notes)
	for i, field := range fields {
		m := palworldNumberTail.FindStringSubmatch(field)
		if m == nil || !palworldBaseRarities[strings.ToUpper(m[2])] {
			continue
		}
		fields[i] = m[1] + " " + m[2]
		return strings.Join(fields, " ")
	}
	return notes
}

// palworldSpellings corrects the Palworld names this storefront misspells.
// The storefront's own shelves are what say it is a slip rather than a
// name: it sells EBP01-032 as "Fuack - Manic Wave Ripper" and the promo of
// that same Pal as "Fuak", where Bushiroad's list and the catalog both
// spell it Fuack.
//
// A replacer rather than a lookup because the name reaches here decorated -
// the storefront hangs the pack a promo came out of behind it - and spelled
// out rather than found by nearest match, for the reason csiSpellings
// gives.
var palworldSpellings = strings.NewReplacer(
	"Fuak - Manic Wave Ripper", "Fuack - Manic Wave Ripper",
)

// palworldName spells a Palworld card the way the catalog does, where this
// storefront has typed it wrong.
func palworldName(name string) string {
	return palworldSpellings.Replace(name)
}
