package cardmarket

import (
	"regexp"
	"strings"

	cm "github.com/mtgban/go-cardmarket"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// fabPrintRuns names the print run a Cardmarket Flesh and Blood expansion
// spells into its own name, one expansion per run of the same set. Welcome
// to Rathe's first run is the only one the catalog calls Alpha; the
// datastore knows it as every other set's 1st Edition.
var fabPrintRuns = []struct{ suffix, run string }{
	{" - First", "1st Edition"},
	{" - Alpha", "1st Edition"},
	{" - Unlimited", "Unlimited Edition"},
}

// fabPrintRun splits a Flesh and Blood expansion name into the print run it
// names and the set name left over. The datastore's sets carry no run - it
// crosses the run with the treatment and gives each crossing its own
// printing - so the suffix has to come off before the set can be looked up.
// fabPromoPrefixes maps the expansions Cardmarket files promos under onto the
// prefix our one promo set numbers them by. Cardmarket sells the promos as
// seven programmes and the datastore carries them as a single set, where the
// programme survives as the collector number's prefix: "012" in FAB Promos is
// FAB012 in ours. The name alone cannot say which, since the same card is
// handed out by more than one programme.
var fabPromoPrefixes = map[string]string{
	"FAB Promos":      "FAB",
	"Hero Promos":     "HER",
	"Judge Promos":    "JDG",
	"LGS Promos":      "LGS",
	"LSS Promos":      "LSS",
	"Tournament Pack": "TNP",
	"XXX Promos":      "XXX",
}

// fabPromoSet is the one set the datastore files every promo programme in.
const fabPromoSet = "Flesh and Blood: Promo Cards"

// fabDeckRe matches the way Cardmarket names a deck product's expansion,
// "<set> - <hero> Blitz Deck" and "<set> - <hero> Hero Deck", which is the
// same information the datastore writes the other way round.
var fabDeckRe = regexp.MustCompile(`^(.+?) - (.+?) (Blitz|Hero) Deck$`)

// fabHistoryPackRe matches the History Pack decks, which Cardmarket numbers
// the way the datastore does but names "History" where the datastore names
// "Historic", and orders the other way round again.
var fabHistoryPackRe = regexp.MustCompile(`^History Pack (\d+) - (.+?) Blitz Deck$`)

// fabArchivePackRe matches the Archive packs, whose class is all the datastore
// keeps of the name.
var fabArchivePackRe = regexp.MustCompile(`^Archive Mastery Pack - (.+)$`)

// fabArmoryRe matches the Armory decks, which Cardmarket files under the line
// that issued them where the datastore names the hero alone - except for the
// Legends line, which the datastore keeps in the name.
var fabArmoryRe = regexp.MustCompile(`^Armory Deck (Origins|Legends): (.+?)(?:,.*)?$`)

// fabWelcomeRe matches the welcome decks, named the other way round.
var fabWelcomeRe = regexp.MustCompile(`^(.+) Welcome Deck$`)

// fabNumberDigits splits a collector number into the letters it opens on and
// the digits that follow, with the padding zeros dropped.
var fabNumberDigits = regexp.MustCompile(`^([A-Za-z]*)0*(\d+)(.*)$`)

// sameFabNumber reports whether two collector numbers name the same printing,
// the padding aside. The datastore writes four digits where the catalog writes
// three for the promos it renumbered late ("HER0160" against "160"), and the
// two are the same card - where FAB299 and LGS313 are not, which is the reason
// this compares at all rather than trusting the name.
func sameFabNumber(a, b string) bool {
	if strings.EqualFold(a, b) {
		return true
	}
	ma := fabNumberDigits.FindStringSubmatch(a)
	mb := fabNumberDigits.FindStringSubmatch(b)
	if ma == nil || mb == nil {
		return false
	}
	return strings.EqualFold(ma[1], mb[1]) && ma[2] == mb[2] && strings.EqualFold(ma[3], mb[3])
}

// fabEdition names the set of ours a Cardmarket expansion is, and the prefix
// its collector numbers need to be read with.
//
// The two catalogs agree on what a product is and disagree on how to say it:
// Cardmarket names a deck by its set and hero and the datastore by hero and
// set, it splits the promos into programmes the datastore keeps as number
// prefixes in one set, and it writes History where the datastore writes
// Historic. None of that is the matcher's business - it is what this
// marketplace calls things - so the translation lives here, and what reaches
// the matcher is a set name it knows.
func fabEdition(expansion string) (setName, numberPrefix string) {
	if prefix, found := fabPromoPrefixes[expansion]; found {
		return fabPromoSet, prefix
	}
	if m := fabHistoryPackRe.FindStringSubmatch(expansion); m != nil {
		return "Historic Pack " + m[1] + " Blitz Deck: " + m[2], ""
	}
	if m := fabArchivePackRe.FindStringSubmatch(expansion); m != nil {
		return "Mastery Pack " + m[1], ""
	}
	if m := fabArmoryRe.FindStringSubmatch(expansion); m != nil {
		if m[1] == "Legends" {
			return "Armory Deck: Legends " + m[2], ""
		}
		return "Armory Deck: " + m[2], ""
	}
	if m := fabWelcomeRe.FindStringSubmatch(expansion); m != nil {
		return "Welcome Deck: " + m[1], ""
	}
	if m := fabDeckRe.FindStringSubmatch(expansion); m != nil {
		// A hero deck is named for its hero alone, and Cardmarket writes
		// the epithet the card carries beside it ("Bravo, Showstopper");
		// a blitz deck keeps the set it was sold with.
		if m[3] == "Hero" {
			hero, _, _ := strings.Cut(m[2], ",")
			hero, _, _ = strings.Cut(hero, " ")
			return "Hero Deck: " + hero, ""
		}
		return "Blitz Deck: " + m[1] + " - " + m[2], ""
	}
	return expansion, ""
}

func fabPrintRun(expansion string) (run, setName string) {
	for _, printRun := range fabPrintRuns {
		trimmed := strings.TrimSuffix(expansion, printRun.suffix)
		if trimmed != expansion {
			return printRun.run, trimmed
		}
	}
	return "", expansion
}

// fabTreatments maps the treatment a Cardmarket Flesh and Blood
// parenthetical ends on to the finish the datastore spells it as.
var fabTreatments = []struct{ tail, finish string }{
	{"Regular", "Normal"},
	{"Rainbow Foil", "Rainbow Foil"},
	{"Cold Foil", "Cold Foil"},
}

// fabTreatment splits the treatment parenthetical off a Cardmarket Flesh
// and Blood product name ("Go Bananas (Rainbow Foil)"), returning the
// treatment as the datastore spells it and the card name left over. Any
// other parenthetical is part of the name ("Sink Below (Yellow)") and stays.
//
// A set selling one card in several arts spells the art into the
// parenthetical ahead of the treatment ("Display Loyalty (Extended Art
// Rainbow Foil)"), and the treatment is still the tail it ends on. Only the
// treatment comes off: the datastore keeps the art in a printing of its own
// and the treatment in the finish beside it, so the art has to stay on the
// name for the printing to be reachable at all.
func fabTreatment(name string) (treatment, card string) {
	if open := strings.LastIndex(name, " ("); open >= 0 && strings.HasSuffix(name, ")") {
		tail := name[open+2 : len(name)-1]
		for _, known := range fabTreatments {
			if tail != known.tail && !strings.HasSuffix(tail, " "+known.tail) {
				continue
			}
			card = name[:open]
			if art := strings.TrimSuffix(tail, known.tail); strings.TrimSpace(art) != "" {
				card += " (" + strings.TrimSpace(art) + ")"
			}
			return known.finish, card
		}
	}
	return "", name
}

// fabFinish names the printing a Cardmarket Flesh and Blood product is,
// from the two places the catalog says so: the print run in the expansion
// name ("Tales of Aria - First"), and the treatment in a parenthetical
// after the card's ("Go Bananas (Rainbow Foil)"). The datastore crosses the
// two and gives each crossing its own printing, so both have to be named to
// reach one. A product naming neither is left to the id alone.
func fabFinish(expansion, name string) string {
	run, _ := fabPrintRun(expansion)
	treatment, _ := fabTreatment(name)

	switch {
	case run != "" && treatment != "":
		return run + " " + treatment
	case treatment != "":
		return treatment
	}
	return ""
}

// fabWording matches a parenthetical Cardmarket writes past a Flesh and
// Blood card's own name: a treatment, the art ahead of it, a label the
// datastore keeps in the printing's name, or a pitch color.
var fabWording = regexp.MustCompile(`^(?:(?:Extended|Alternate) Art(?: (?:Regular|Rainbow Foil|Cold Foil))?|Regular|Rainbow Foil|Cold Foil(?: Golden)?|Marvel|Golden|Artist Proof|Red|Yellow|Blue)$`)

// fabArt matches the art parenthetical alone, the one a set may carry no
// printing of its own for.
var fabArt = regexp.MustCompile(` \((?:Extended|Alternate) Art\)$`)

// fabBaseName strips every wording parenthetical off a Cardmarket product
// name, leaving the card's own name, pitch included: what two products of
// one card share, and two products of different cards do not.
func fabBaseName(name string) string {
	for {
		open := strings.LastIndex(name, " (")
		if open < 0 || !strings.HasSuffix(name, ")") {
			return name
		}
		tail := name[open+2 : len(name)-1]
		if !fabWording.MatchString(tail) || fabPitch(tail) {
			return name
		}
		name = name[:open]
	}
}

// fabPitch reports whether a parenthetical names a pitch color, which
// belongs to the card's name: Sink Below (Red) and Sink Below (Blue) are
// two cards.
func fabPitch(word string) bool {
	return word == "Red" || word == "Yellow" || word == "Blue"
}

// fabDropArt answers a name without the art parenthetical fabTreatment
// left on it, for the sets that file the art under the plain name.
func fabDropArt(name string) string {
	return fabArt.ReplaceAllString(name, "")
}

// fabSameProduct reports whether two Cardmarket Flesh and Blood products
// are the same card sold twice: the same name once the treatment, art and
// labels are off it, at the same number - or one of them the unpitched
// listing of the other, the older spelling of a card the expansion sells
// pitched too.
func fabSameProduct(a, b *cm.Product) bool {
	// The same listing twice, whatever numbers the two copies wear
	if strings.EqualFold(a.Name, b.Name) {
		return true
	}
	if a.Number != "" && b.Number != "" && !sameFabNumber(a.Number, b.Number) {
		return false
	}
	return fabSameCard(fabBaseName(a.Name), fabBaseName(b.Name))
}

// fabSameCard reports whether two card names, wording already off them,
// name one card: the same name, or one the unpitched spelling of the
// other, which is how the older listings and the token-sized reprints
// write a card the set sells pitched.
func fabSameCard(nameA, nameB string) bool {
	if strings.EqualFold(nameA, nameB) {
		return true
	}
	return strings.EqualFold(unpitched(nameA), nameB) || strings.EqualFold(nameA, unpitched(nameB))
}

// fabPitchTail matches the pitch parenthetical ending a name.
var fabPitchTail = regexp.MustCompile(` \((?:Red|Yellow|Blue)\)$`)

func unpitched(name string) string {
	return fabPitchTail.ReplaceAllString(name, "")
}

// fabFaceOf reports whether a Cardmarket product names one face of the
// fused printing it is beside: the storefront sells a double-sided hero
// face by face, and the datastore files the card once under both faces.
func fabFaceOf(product *cm.Product, cardID string) bool {
	co, err := mtgmatcher.GetUUID(cardID)
	if err != nil || !strings.Contains(co.Name, "//") {
		return false
	}
	name := mtgmatcher.Normalize(fabBaseName(product.Name))
	for _, face := range strings.Split(co.Name, "//") {
		if mtgmatcher.Normalize(strings.TrimSpace(face)) == name {
			return true
		}
	}
	return false
}

// fabNamesPrinting reports whether a product names the printing an id
// landed it on: the same card, or one face of a fused card. The bridge
// speaks through another marketplace's links, and a link tied to the
// wrong product lands a card on its neighbour's printing.
func fabNamesPrinting(product *cm.Product, cardID string) bool {
	co, err := mtgmatcher.GetUUID(cardID)
	if err != nil {
		return false
	}
	if fabSameCard(fabBaseName(product.Name), fabBaseName(co.Name)) {
		return true
	}
	return fabFaceOf(product, cardID)
}

// fabNumberPrefix splits a collector number into the letters a set opens
// its numbers on and the digits after them.
var fabNumberPrefix = regexp.MustCompile(`^([A-Za-z0-9]*?[A-Za-z])(\d{3,4})[A-Za-z]*$`)

// fabSetPrefix answers the letters a set's collector numbers open on, or
// nothing for a set numbered by digits alone. Cardmarket writes the digits
// and the datastore the whole number, and a fused card answers to a face's
// number only when it is written whole.
func fabSetPrefix(set *mtgmatcher.Set) string {
	for _, card := range set.Cards {
		if fields := fabNumberPrefix.FindStringSubmatch(card.Number); fields != nil {
			return fields[1]
		}
	}
	return ""
}

// shelf is one set a product may be filed in, with the way its numbers
// are read there: the prefix a promo programme's numbers carry, and the
// print run the expansion names.
type shelf struct {
	set          *mtgmatcher.Set
	edition      string
	numberPrefix string
	printRun     string
}

// fabShelves names the sets a Cardmarket Flesh and Blood product may be
// filed in, in the order they are asked. Cardmarket sells each print run
// as its own expansion ("Monarch - First"), a name no set of ours carries:
// the run belongs to the printing, where fabFinish puts it, and the set is
// looked up without it. What Cardmarket calls the expansion is not always
// what we call the set, and translating is the fallback rather than the
// first move, so an expansion whose name we already know keeps answering
// for itself. A promo programme is asked of its own set before the one set
// every programme was once filed in, and an expansion no name places is
// asked of the set wearing its code, which is how the Silver Age decks and
// the Slingshot promos are filed.
func fabShelves(product *cm.Product) []shelf {
	printRun, edition := fabPrintRun(product.ExpansionName)
	var shelves []shelf
	if prefix, promo := fabPromoPrefixes[edition]; promo {
		programme, err := mtgmatcher.GetSet(prefix)
		if err == nil {
			shelves = append(shelves, shelf{set: programme, edition: programme.Name, numberPrefix: prefix, printRun: printRun})
		}
		set, err := mtgmatcher.GetSetByName(fabPromoSet)
		if err == nil {
			shelves = append(shelves, shelf{set: set, edition: fabPromoSet, numberPrefix: prefix, printRun: printRun})
		}
		return shelves
	}
	set, err := mtgmatcher.GetSetByName(edition)
	if err == nil {
		shelves = append(shelves, shelf{set: set, edition: edition, printRun: printRun})
	} else {
		translated, _ := fabEdition(edition)
		set, err = mtgmatcher.GetSetByName(translated)
		if err == nil {
			shelves = append(shelves, shelf{set: set, edition: translated, printRun: printRun})
		}
	}
	if product.ExpansionCode != "" {
		coded, cerr := mtgmatcher.GetSet(product.ExpansionCode)
		if cerr == nil && !shelved(shelves, coded) {
			shelves = append(shelves, shelf{set: coded, edition: coded.Name, printRun: printRun})
		}
	}
	return shelves
}

func shelved(shelves []shelf, set *mtgmatcher.Set) bool {
	for _, sh := range shelves {
		if sh.set.Code == set.Code {
			return true
		}
	}
	return false
}

// disownBridged takes the bridge's answer away from a product it landed on
// a card the same shelf sells, and prices, under another product's name,
// and lets the name answer instead. The bridge speaks through another
// marketplace's links, and a link tied to the neighbouring product lands
// a card on its neighbour's printing: Cardmarket's Herald of Ravages on
// the datastore's Herald of Rebirth, the red Lead with Heart on the
// yellow. A spelling the datastore does not share is not that - no priced
// product of the shelf claims the printing - and the id keeps its say
// over it, the misspelt listing of a card the shelf also sells refused
// included.
func (mkm *Index) disownBridged(results []resolved) {
	claimed := map[string]bool{}
	for _, r := range results {
		if r.err != nil || r.cardID == "" {
			continue
		}
		name := fabBaseName(r.product.Name)
		claimed[mtgmatcher.Normalize(name)] = true
		claimed[mtgmatcher.Normalize(unpitched(name))] = true
	}
	for i, r := range results {
		if r.err != nil || r.cardID == "" || r.byName || fabNamesPrinting(r.product, r.cardID) {
			continue
		}
		co, err := mtgmatcher.GetUUID(r.cardID)
		if err != nil || !claimed[mtgmatcher.Normalize(fabBaseName(co.Name))] {
			continue
		}
		cardID := mkm.matchFab(r.product)
		if cardID == "" {
			results[i] = resolved{product: r.product, err: errNoPrinting}
			continue
		}
		results[i] = resolved{product: r.product, cardID: cardID, cardIDFoil: cardID, byName: true}
	}
}

// fabNumbers answers the forms a Cardmarket number is asked in: whole,
// opening on the letters its set writes, which is how the datastore
// writes it and the only way a fused card answers to its faces' numbers;
// as written; and not at all, since a deck the two catalogs number
// differently still names its cards, and a name the set holds once is
// the card whatever number it was sold under.
func fabNumbers(prefix, number string) []string {
	numbers := []string{number}
	if prefix != "" && number != "" {
		parts := strings.Split(number, "/")
		for i, part := range parts {
			if part != "" && part[0] >= '0' && part[0] <= '9' {
				parts[i] = prefix + part
			}
		}
		if whole := strings.Join(parts, "//"); whole != number {
			numbers = append([]string{whole}, numbers...)
		}
	}
	return append(numbers, "")
}
