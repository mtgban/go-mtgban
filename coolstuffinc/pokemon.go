package coolstuffinc

import (
	"cmp"
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// pokemonNonHolo matches the bracket a Pokemon name states a plain printing
// with. The storefront sells a card printed in both finishes as two products
// telling them apart by that bracket alone - the note is empty and the foil
// flag is off on both - and read as the holo, a $2.00 Team Aqua's Kyogre
// would be served at the $80.00 one's price. The bracket's own case varies
// ("(NON-HOLO)" on Black & White prints), so the match has to as well.
//
// The rarity is what says whether the plain printing was ever made. A holo
// rare is sold holo and nothing else, so a bracket asking for its plain
// printing asks for one that does not exist and the row is refused. A plain
// rare is the opposite: the catalog holding no nonfoil for it is the catalog
// missing a printing rather than the storefront inventing one, and refusing
// those would drop 25 real listings to catch nothing.
var pokemonNonHolo = regexp.MustCompile(`(?i)\(Non-?\s?Holo\)`)

// pokemonNonHoloNote matches the note a Black & White deck exclusive carries
// where its name carries no bracket.
var pokemonNonHoloNote = regexp.MustCompile(`(?i)\*Non-?\s?Holo Version\b`)

// pokemonNonHoloDeckExclusive answers whether the catalog's Deck Exclusives
// shelf carries the plain printing a "(Non-Holo)" bracket asks for, taken
// only when the probe lands on PR-1840's own nonfoil at the listing's own
// number.
func pokemonNonHoloDeckExclusive(b *mtgmatcher.Backend, name, numbered string, foil bool) bool {
	num := cmp.Or(mtgmatcher.ExtractNumber(numbered), numbered)
	if num == "" {
		return false
	}
	id, err := b.Match(&mtgmatcher.InputCard{Name: name + " - " + numbered, Edition: "Deck Exclusives", Foil: foil})
	if err != nil {
		return false
	}
	co, err := b.GetUUID(id)
	return err == nil && co.SetCode == "PR-1840" && co.Finish == mtgmatcher.FinishNonfoil &&
		strings.TrimLeft(co.Number, "0") == num
}

// numberedListing reads a listing name apart from the collector number this
// storefront glues onto it, answering the name alone and what it took, or the
// name whole and nothing where the storefront named the card by itself.
//
// Only a tail opening on a number is taken, since the storefront also hangs
// plain wording off a dash ("Ancient Mew - Movie Promo") and the catalog
// spells some cards with one of its own. What is taken is set aside rather
// than thrown away: the printing is written in parentheses behind the number
// ("16/111 (Reverse Foil)") and neither side's foil column says so, which the
// matcher reads off the name it is handed.
func numberedListing(name string) (string, string) {
	head, tail, found := strings.Cut(name, " - ")
	if !found {
		return name, ""
	}
	number, _, _ := strings.Cut(tail, " ")
	if !buylistNumberWord.MatchString(number) {
		return name, ""
	}
	return head, tail
}

// pokemonListing reads a Pokemon listing the way the catalog names it. The
// Classic Collection reprints are sold under Celebrations with the
// collection in the note; the special energies are named "Special Metal
// Energy" where the catalog names the energy and labels it special; the
// Base Set Professor Oak is spelled "Imposter" the way Base Set 2 prints
// it, though Base Set printed "Impostor"; Nidoran is named without the sex
// the catalog names it by, which the number settles; the Elite Four
// cards of the Platinum sets are named "Alakazam 4" for the catalog's
// "Alakazam E4"; and the metal cards are named for the metal, "Metal Mew ex"
// for the catalog's Mew ex labelled a metal card.
func pokemonListing(b *mtgmatcher.Backend, name, edition, variation string, foil bool) *mtgmatcher.InputCard {
	if edition == "Pokemon Oversized Cards" {
		edition = "Jumbo Cards"
		name = promoProgrammeNumber.ReplaceAllString(strings.TrimSuffix(name, " Jumbo Size"), "$1")
	}
	name, numbered := numberedListing(pokemonRespellings.Replace(name))
	numbered = megaPromoNumber.ReplaceAllString(numbered, "MEP$1")
	variation = megaPromoNumber.ReplaceAllString(variation, "MEP$1")
	name = strings.TrimSpace(nonStampedName.ReplaceAllString(name, ""))
	card := &mtgmatcher.InputCard{Name: name, Edition: edition, Variation: variation, Foil: foil}
	m := basicEnergyName.FindStringSubmatch(name)
	if m != nil {
		bracket := strings.Trim(m[2], "()")
		fixed := pokemonBasicEnergy(b, m[1], bracket, edition, numbered, variation, foil)
		if fixed != nil {
			return fixed
		}
	}
	nonHolo := pokemonNonHolo.MatchString(name) || pokemonNonHolo.MatchString(numbered) || pokemonNonHoloNote.MatchString(variation)
	if nonHolo {
		strippedName := strings.TrimSpace(pokemonNonHolo.ReplaceAllString(name, ""))
		strippedNumbered := strings.TrimSpace(pokemonNonHolo.ReplaceAllString(numbered, ""))
		if pokemonNonHoloDeckExclusive(b, strippedName, strippedNumbered, foil) {
			card.Name = strippedName
			card.Edition = "Deck Exclusives"
			numbered = strippedNumbered
		}
	}
	if edition == "Celebrations" && (strings.Contains(variation, "Classic Collection") || strings.Contains(numbered, "Classic Collection")) {
		card.Edition = "Celebrations: Classic Collection"
		if m := classicNumber.FindStringSubmatch(variation); m != nil {
			card.Variation = m[1]
		}
	}
	// A reprint note ("25th Anniversary Stamp Base Set Reprint") reaches
	// retail's variation whole; clear it before "Stamp" reads as a demand.
	reprint := buylistReprintNote.MatchString(variation)
	if reprint {
		card.Variation = ""
	}
	if strings.HasPrefix(name, "Vivillon") {
		name = pokemonVivillonColors.Replace(name)
		card.Name = name
		numbered = pokemonVivillonColors.Replace(numbered)
	}
	// This storefront numbers the Alph Lithographs as secret rares past their
	// set's total, where the catalog numbers each of the four by its set.
	if name == "Alph Lithograph" {
		numbered, card.Variation = "", ""
	}
	shelved := false
	if !nonHolo {
		shelf := pokemonNoteShelf(b, numbered, card)
		if shelf != nil {
			card, numbered, shelved = shelf, strings.TrimSpace(pokemonStampTail.ReplaceAllString(numbered, "")), true
		}
	}
	// A "(Non-Holo)" listing is never a holo pull, so it is never this
	// redirect's business even when its note also carries one of
	// pokemonDeckHoloNotes' markers.
	redirected := ""
	if !nonHolo {
		redirected = pokemonDeckHoloRedirect(b, name, numbered, variation)
	}
	if redirected != "" {
		card.Edition = redirected
		card.Variation = "Cracked Ice Holo"
		card.Foil = true
	}
	m = goldStar.FindStringSubmatch(name)
	if m != nil {
		card.Name = m[1] + " Star"
	}
	m = unownListing.FindStringSubmatch(name)
	if m != nil {
		card.Name = "Unown"
		card.Variation = m[2] + "/" + m[3]
	}
	if m := specialEnergy.FindStringSubmatch(name); m != nil {
		card.Name = m[1]
		card.Variation = strings.TrimSpace("Special " + card.Variation)
	}
	// Only where the whole name is no card and the rest is one: "Metal
	// Energy" is a card, and so is none of "Energy (Secret Rare)".
	m = metalCardName.FindStringSubmatch(card.Name)
	if m != nil {
		_, whole := b.SearchEquals(card.Name)
		_, rest := b.SearchEquals(m[1])
		if whole != nil && rest == nil {
			card.Name = m[1]
			if !mtgmatcher.SlugDescribes(card.Variation, "metalcard") {
				card.Variation = strings.TrimSpace(card.Variation + " Metal Card")
			}
		}
	}
	if name == "Imposter Professor Oak" && strings.HasPrefix(edition, "Base Set") && !strings.HasPrefix(edition, "Base Set 2") {
		card.Name = "Impostor Professor Oak"
	}
	if m := eliteFour.FindStringSubmatch(name); m != nil && strings.HasPrefix(edition, "Platinum") {
		card.Name = m[1] + " E4"
	}
	// An energy or promo sold under a main set with the energy or promo
	// set's own number ("MEE007" on Ascended Heroes) is that set's.
	if m := prefixedNumber.FindStringSubmatch(variation); m != nil {
		if set, found := pokemonNumberSets[m[1]]; found {
			card.Edition = set
			// The catalog numbers the older promo sets with the
			// programme on ("SM190") and the newer ones without.
			if m[1] != "SM" && m[1] != "SWSH" {
				card.Variation = strings.Replace(variation, m[0], m[2], 1)
			}
		}
	}
	// The notes say a printing is the plain one ("Non-Stamped Version"),
	// name the illustrator, or name the stamp a promo carries; the plain
	// words come off, and a stamp names a promo shelf the catalog files
	// apart from the set the listing arrived on.
	card.Variation = strings.TrimSpace(plainWords.ReplaceAllString(card.Variation, " "))
	if !reprint && !shelved && stamped.MatchString(card.Variation) {
		card.Edition = "Promo"
	}
	// The Team Galactic inventions are named by their invention alone
	// where the catalog names the invention with its number.
	if _, err := b.SearchEquals(card.Name); err != nil {
		if invention := galacticInvention(b, card.Name); invention != "" {
			card.Name = invention
		}
	}
	// The catalog letters the type in a special energy's name ("Bubbly W
	// Energy") where the storefront spells it out, but not in every one
	// ("Magnetic Metal Energy"), so the spelled name is kept where the
	// catalog knows it.
	if m := typedEnergy.FindStringSubmatch(card.Name); m != nil {
		if _, err := b.SearchEquals(card.Name); err != nil {
			card.Name = m[1] + " " + energyLetters[m[2]] + " Energy"
		}
	}
	if name == "Nidoran" || name == "Nidoran?" {
		for _, sex := range []string{"Nidoran M", "Nidoran F"} {
			probe := *card
			probe.Name = sex
			if numbered != "" {
				probe.Name += " - " + numbered
			}
			if _, err := b.Match(&probe); err == nil {
				card.Name = sex
				break
			}
		}
	}
	if numbered != "" {
		card.Name = pokemonProfessor(b, card, numbered)
		card.Name += " - " + numbered
	}
	return card
}

// pokemonProfessor answers the name a Professor card goes by in the catalog
// where this storefront spells it the other way round ("Prof." for
// "Professor" and back, each the way some sets print it). The other spelling
// is taken only when the listing's own does not land at its number and the
// other does.
func pokemonProfessor(b *mtgmatcher.Backend, card *mtgmatcher.InputCard, numbered string) string {
	var swapped string
	switch {
	case strings.HasPrefix(card.Name, "Prof. "):
		swapped = "Professor " + strings.TrimPrefix(card.Name, "Prof. ")
	case strings.HasPrefix(card.Name, "Professor "):
		swapped = "Prof. " + strings.TrimPrefix(card.Name, "Professor ")
	default:
		return card.Name
	}
	num := mtgmatcher.ExtractNumber(numbered)
	if num == "" {
		return card.Name
	}
	landsAtNumber := func(name string) bool {
		probe := *card
		probe.Name = name + " - " + numbered
		id, err := b.Match(&probe)
		if err != nil {
			return false
		}
		co, err := b.GetUUID(id)
		return err == nil && strings.TrimLeft(co.Number, "0") == num
	}
	if landsAtNumber(card.Name) || !landsAtNumber(swapped) {
		return card.Name
	}
	return swapped
}

// pokemonBuylistCard reads a buylist row the way a sell listing is read, the
// print run included. The run rides in the shelf's title on both sides of the
// storefront: a buy row read with its shelf spelled whole matches the set of
// that name and is published against the unlimited printing at the first
// edition's price.
func pokemonBuylistCard(b *mtgmatcher.Backend, product CSIPriceEntry) (*mtgmatcher.InputCard, []string) {
	// CSI's "0" placeholder for an unnumbered year energy otherwise reads
	// as a number word of its own at the head of the variation.
	if product.Number == "0" {
		product.Number = ""
	}
	variation := catalogTreatment(buylistVariation(product))
	shelf, run := firstEditionShelf(product.ItemSet)
	shelf = pokemonPromoShelf(b, product.Name, shelf, product.RarityName, product.IsFoil == 1, variation)
	return pokemonListing(b, product.Name, shelf, variation, product.IsFoil == 1), run
}

// pokemonNumberSets are the sets a number's prefix names outright.
var pokemonNumberSets = map[string]string{
	"MEE":  "MEE: Mega Evolution Energies",
	"SVE":  "SVE: Scarlet & Violet Energies",
	"MEP":  "ME: Mega Evolution Promo",
	"SVP":  "SV: Scarlet & Violet Promo Cards",
	"SWSH": "SWSH: Sword & Shield Promo Cards",
	"SM":   "SM Promos",
}

// pokemonRespellings pairs the names this storefront misspells with the
// catalog's own. The head is rewritten in place, so the number or bracket
// behind it is kept; Kyurem's number is run into its name without a dash.
var pokemonRespellings = strings.NewReplacer(
	"Galatic HQ", "Galactic HQ",
	"Sprigattito", "Sprigatito",
	"Unit Energy GFW", "Unit Energy GRW",
	"Delta Species Rainbow Energy", "Delta Rainbow Energy",
	"Kyurem 43/113", "Kyurem - 43/113",
	"Vivilion", "Vivillon",
)

var (
	plainWords = regexp.MustCompile(`(?i)\bNon-?Stamped(?: Version)?\b|\bIllus\. [^,]+,`)
	stamped    = regexp.MustCompile(`(?i)\bStamp(?:ed)?\b`)

	// nonStampedName matches the same bracket as plainWords where it sits
	// in the name's own head instead - "Psyduck (Non-Stamped) - SM199".
	nonStampedName = regexp.MustCompile(`(?i)\s*\(Non-?Stamped\)`)
)

// galacticInvention names the Team Galactic invention the catalog files
// under its number, or nothing when no invention ends in the name given.
func galacticInvention(b *mtgmatcher.Backend, name string) string {
	found := ""
	for _, candidate := range b.Names(mtgmatcher.NameFormCanonical, false) {
		if strings.HasPrefix(candidate, "Team Galactic's Invention") && strings.HasSuffix(candidate, " "+name) {
			if found != "" {
				return ""
			}
			found = candidate
		}
	}
	return found
}

var energyLetters = map[string]string{
	"Grass": "G", "Fire": "R", "Water": "W", "Lightning": "L", "Psychic": "P",
	"Fighting": "F", "Darkness": "D", "Metal": "M", "Fairy": "Y", "Dragon": "N",
	"Colorless": "C",
}

var (
	typedEnergy    = regexp.MustCompile(`^(\w+) (Grass|Fire|Water|Lightning|Psychic|Fighting|Darkness|Metal|Fairy|Dragon|Colorless) Energy$`)
	prefixedNumber = regexp.MustCompile(`\b(MEE|SVE|MEP|SVP|SWSH|SM)(\d{3})\b`)
	classicNumber  = regexp.MustCompile(`Classic Collection (\d+)`)

	// megaPromoNumber matches the Mega Evolution promo number as this
	// storefront writes it, "ME099", where the catalog's own is "MEP099".
	megaPromoNumber = regexp.MustCompile(`\bME(\d{3})\b`)

	// promoProgrammeNumber matches the programme prefix a jumbo card's number
	// carries, which the catalog files its Jumbo Cards under without.
	promoProgrammeNumber = regexp.MustCompile(`\b(?:SVP|MEP|ME)(\d{3})\b`)
	specialEnergy        = regexp.MustCompile(`^Special ((?:Metal|Darkness) Energy)$`)
	eliteFour            = regexp.MustCompile(`^(.+) 4( LV\.X)?$`)

	// goldStar matches a Gold Star the way this storefront names it, "Mew *
	// (Star)" for the catalog's "Mew Star" - $1,800 of buylist refusals on
	// Ex Dragon Frontiers' Mew alone.
	goldStar = regexp.MustCompile(`^(.+) \* \(Star\)$`)

	// unownListing matches an EX Unseen Forces Unown the way this
	// storefront names it, the letter written into both the name and its
	// own index ("Unown A - A/28"), where the catalog names every one of
	// them "Unown" and numbers it by the letter alone.
	unownListing = regexp.MustCompile(`^Unown ([A-Z!?]) - ([A-Z!?])/(\d+)$`)

	// metalCardName matches an Ultra-Premium Collection metal card the way
	// this storefront names it, for the metal and the set it copies ("Metal
	// Mew ex", "Metal Base Set Charizard"), where the catalog names the card.
	metalCardName = regexp.MustCompile(`^Metal (?:Base Set )?(.+)$`)
)

// pokemonVivillonColors spells the two Vivillon colours this storefront
// shortens where the catalog's own promo label is longer than the colour
// alone - "(Pink)" for "(Meadow Pink)", the only thing that tells the XY
// 17/146 Pink Pattern apart from the identically-numbered Orange one.
var pokemonVivillonColors = strings.NewReplacer(
	"(Pink)", "(Meadow Pink)",
	"(Orange)", "(High Plains Orange)",
)

// pokemonNoteShelves names what a note says about the shelf a listing
// belongs on, for the printings this storefront files on a main or promo
// shelf while its note names the catalog's own: the edition to ask for, the
// variation to ask it with instead of the note ("" keeps the note), and the
// set code the listing has to land on to be trusted.
var pokemonNoteShelves = []struct {
	marker, edition, variation, wantSet string
}{
	{"Regional Championship", "League & Championship Cards", "", "PR-1539"},
	{"Build & Battle", "Miscellaneous Cards & Products", "Prerelease", "MCAP"},
	{"Prerelease", "XY Promos", "", "PR-1451"},
	{"From Dragon Vault Blister Pack", "Blister Exclusives", "", "BLE"},
}

// pokemonStampTail matches the stamp the name carries behind its number for
// the printings pokemonNoteShelves redirects: the catalog labels them by the
// event and not by a stamp, so the word would only demand one that is not
// there.
var pokemonStampTail = regexp.MustCompile(`(?i)\s*\([^)]*\bStamp(?:ed)?\)|\s+-\s+Dragon Vault Stamped Mirror Holo`)

// pokemonNoteShelf answers the listing re-asked on the shelf its note names,
// or nil where no note names one or the redirected listing does not land on
// that shelf's own set.
func pokemonNoteShelf(b *mtgmatcher.Backend, numbered string, card *mtgmatcher.InputCard) *mtgmatcher.InputCard {
	cleaned := strings.TrimSpace(pokemonStampTail.ReplaceAllString(numbered, ""))
	for _, r := range pokemonNoteShelves {
		if !strings.Contains(card.Variation, r.marker) {
			continue
		}
		redirected := *card
		redirected.Edition = r.edition
		if r.variation != "" {
			redirected.Variation = r.variation
		}
		probe := redirected
		if cleaned != "" {
			probe.Name += " - " + cleaned
		}
		id, err := b.Match(&probe)
		if err != nil {
			continue
		}
		co, err := b.GetUUID(id)
		if err != nil || co.SetCode != r.wantSet {
			continue
		}
		return &redirected
	}
	return nil
}

// pokemonDeckHoloNotes names the edition a print-run note belongs to, and
// the set code the redirect has to land on to be trusted.
var pokemonDeckHoloNotes = []struct {
	marker, edition, wantSet string
}{
	{"Theme Deck", "Deck Exclusives", "PR-1840"},
	{"EX Battle Stadium", "EX Battle Stadium", "BST"},
	{"Prism Holo", "Miscellaneous Cards & Products", "MCAP"},
}

// pokemonDeckHoloRedirect answers the edition for a pokemonDeckHoloNotes
// marker whose probe - asking for the "Cracked Ice Holo" label so a
// cracked-ice twin outranks the plain printing of the same number - lands
// on that marker's own set code, or "" otherwise.
func pokemonDeckHoloRedirect(b *mtgmatcher.Backend, name, numbered, notes string) string {
	for _, r := range pokemonDeckHoloNotes {
		if !strings.Contains(notes, r.marker) && !strings.Contains(numbered, r.marker) {
			continue
		}
		tail := strings.TrimSpace(strings.Replace(numbered, r.marker, "", 1))
		tail = strings.TrimSpace(strings.TrimSuffix(tail, "-"))
		probeName := name
		if tail != "" {
			probeName += " - " + tail
		}
		probe := &mtgmatcher.InputCard{Name: probeName, Edition: r.edition, Variation: "Cracked Ice Holo", Foil: true}
		id, err := b.Match(probe)
		if err != nil {
			continue
		}
		co, err := b.GetUUID(id)
		if err != nil || co.SetCode != r.wantSet {
			continue
		}
		return r.edition
	}
	return ""
}

// basicEnergyName matches this storefront's "<Type> Energy" and a treatment
// bracket it carries of its own ("(Cosmo Holo)"), apart from the number
// tail's. pokemonBasicEnergy gates the rename on the listing's own number.
var basicEnergyName = regexp.MustCompile(`^(Grass|Fire|Water|Lightning|Psychic|Fighting|Darkness|Metal|Fairy) Energy(?:\s+(\(.+\)))?$`)

// pokemonBasicEnergy answers a bare "<Type> Energy" listing the way the
// catalog spells its Scarlet & Violet / Mega Evolution era printings, read
// off the SVE/MEE-shaped product number glued onto the tail.
func pokemonBasicEnergy(b *mtgmatcher.Backend, energyType, bracket, edition, numbered, variation string, foil bool) *mtgmatcher.InputCard {
	name := "Basic " + energyType + " Energy"
	bracket = catalogTreatment(bracket)

	build := func(fromVariation bool, numberedTail string) *mtgmatcher.InputCard {
		card := &mtgmatcher.InputCard{Name: name, Edition: edition, Variation: bracket, Foil: foil}
		if fromVariation {
			card.Variation = strings.TrimSpace(bracket + " " + numberedTail)
		} else if numberedTail != "" {
			card.Name += " - " + numberedTail
		}
		return card
	}

	tail, fromVariation := csiTreatments.Replace(numbered), false
	pm := prefixedNumber.FindStringSubmatch(tail)
	if pm == nil {
		tail, fromVariation = csiTreatments.Replace(variation), true
		pm = prefixedNumber.FindStringSubmatch(tail)
	}
	if pm == nil {
		// A plain rename needs the tail's own number to say which
		// printing is meant, on the shelf's own set: the same gate
		// the on-shelf and SVE/MEE probes below both apply.
		num := mtgmatcher.ExtractNumber(numbered)
		if num == "" {
			return nil
		}
		shelf, err := b.GetSetByName(edition)
		if err != nil {
			return nil
		}
		plain := build(false, numbered)
		plain.Variation = strings.TrimSpace(bracket + " " + variation)
		id, err := b.Match(plain)
		if err != nil {
			return nil
		}
		co, err := b.GetUUID(id)
		if err != nil || co.SetCode != shelf.Code || strings.TrimLeft(co.Number, "0") != num {
			return nil
		}
		return plain
	}

	shelf, shelfErr := b.GetSetByName(edition)

	bare := strings.TrimLeft(pm[2], "0")
	if bare == "" {
		bare = "0"
	}
	onShelf := build(fromVariation, strings.TrimSpace(strings.Replace(tail, pm[0], bare, 1)))
	if shelfErr == nil {
		id, err := b.Match(onShelf)
		if err == nil {
			co, err := b.GetUUID(id)
			if err == nil && co.SetCode == shelf.Code {
				return onShelf
			}
		}
	}

	set, found := pokemonNumberSets[pm[1]]
	if !found {
		return nil
	}
	target, err := b.GetSetByName(set)
	if err != nil {
		return nil
	}
	redirected := build(fromVariation, strings.TrimSpace(strings.Replace(tail, pm[0], pm[2], 1)))
	redirected.Edition = set
	id, err := b.Match(redirected)
	if err != nil {
		return nil
	}
	co, err := b.GetUUID(id)
	if err != nil || co.SetCode != target.Code {
		return nil
	}
	return redirected
}

// pokemonPromoShelf answers the shelf a Pokemon listing belongs to, which is
// the one it arrived on unless the catalog files the card as a promo.
//
// A promo carrying a main set's number is sold here under that set, with only
// the rarity field saying otherwise: the Pokemon Day 2025 Eevee sits on SV
// Prismatic Evolutions at 074/131, where that set's own Eevee already stands.
// The two meet there, and the $2.50 promo would be priced as the card it was
// stamped from. The catalog keeps those on a promo shelf instead.
//
// The promo shelf only decides where it answers at all. Twenty of the fifty
// buylist listings this can reach name a printing no promo shelf holds - the
// Pokemon Rumble cards, the holo promos that are their set's own foil - and
// those keep the set they arrived on. The same shape reproduces on retail -
// the identical "Eevee - 074/131 (Pokemon Day 2025)" listing sits on the same
// shelf, with the same breadcrumb rarity "Promo" - so both sides call this
// with their own name for the rarity field: RarityName on the buylist, the
// scraped breadcrumb text on retail.
func pokemonPromoShelf(b *mtgmatcher.Backend, name, itemSet, rarityName string, foil bool, variation string) string {
	if rarityName != "Promo" ||
		strings.Contains(strings.ToLower(itemSet), "promo") {
		return itemSet
	}
	probe := &mtgmatcher.InputCard{
		Name:      name,
		Edition:   "Promo",
		Variation: variation,
		Foil:      foil,
	}
	_, err := b.Match(probe)
	if err != nil {
		return itemSet
	}
	return "Promo"
}
