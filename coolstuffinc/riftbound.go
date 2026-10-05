package coolstuffinc

import (
	"regexp"
	"slices"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// riftboundNotePrefix matches the set code a Riftbound note opens with.
var riftboundNotePrefix = regexp.MustCompile(`^([A-Z]{2,4})-`)

// riftboundEventMarkers are the brackets a promo's name carries for the
// event that handed it out, which the catalog files on its Release Event
// Promos shelf, apart from the promo shelf's own printing at the same number.
var riftboundEventMarkers = []string{"(Prerelease)", "(Origins Stamp)"}

// riftboundNotes spells the notes this storefront words with a year the
// catalog's promo does not carry, which the matcher would read as a number.
var riftboundNotes = strings.NewReplacer("Worlds Bundle 2025 Promo", "Worlds Bundle Promo")

// riftboundShelf answers the set a Riftbound listing belongs to, which is the
// shelf it arrived on except where that shelf says only "Promo".
//
// The Nexus Night runes are sold under the promo shelf with the set that
// issued them written at the head of the note - "UNL-R05b", "SFD-R05b" - and
// the promo shelf holds a printing of its own at that number. All of them meet
// there, so a $5.00 Unleashed Chaos Rune and an $11.00 Spiritforged one would
// both price as the Organized Play printing they share a number with. The
// same three listings, word for word, sell on the retail search too.
//
// The note only decides where the set it names holds that printing. Vendetta
// issued no b-lettered rune of its own, so its six listings stay on the promo
// shelf, which is where the printing they mean actually is.
func riftboundShelf(b *mtgmatcher.Backend, itemSet, notes, name, variation string, foil bool) string {
	if itemSet != "Promo" {
		return itemSet
	}
	for _, marker := range riftboundEventMarkers {
		if !strings.Contains(name, marker) {
			continue
		}
		probe := &mtgmatcher.InputCard{Name: name, Edition: "Release Event Promos", Variation: variation, Foil: foil}
		id, err := b.Match(probe)
		if err != nil {
			continue
		}
		co, err := b.GetUUID(id)
		if err == nil && co.SetCode == "OPP" {
			return probe.Edition
		}
	}
	match := riftboundNotePrefix.FindStringSubmatch(notes)
	if match == nil {
		return itemSet
	}
	set, err := b.GetSet(match[1])
	if err != nil {
		return itemSet
	}
	probe := &mtgmatcher.InputCard{
		Name:      name,
		Edition:   set.Name,
		Variation: variation,
		Foil:      foil,
	}
	_, err = b.Match(probe)
	if err != nil {
		return itemSet
	}
	return set.Name
}

// riftboundImageStem reads the file name off a product image, whatever
// the storefront pictures in, and riftboundImageTCG the TCGplayer
// product id the oldest Origins products are pictured by.
var (
	riftboundImageStem = regexp.MustCompile(`(?i)/([^/]+)\.(?:jpe?g|png|webp|avif)$`)
	riftboundImageTCG  = regexp.MustCompile(`[0-9]{5,}`)

	riftboundImageTail = regexp.MustCompile(`(?i)(?:ovr|alt)?[_.]*(?:v[0-9]+)?$`)

	// riftboundImagePrintIndex strips a tally a sku tacks onto its own
	// number - "148_2" for Anivia. Anchored so it never eats a number
	// that is only ever "_<digits>" itself.
	riftboundImagePrintIndex = regexp.MustCompile(`^([0-9]+)_[0-9]+$`)

	// riftboundImageSig matches the tail the storefront hangs on a
	// signature printing's sku, which the catalog numbers with a star
	// instead ("SFD224SIG" for 224*). It is read rather than dropped:
	// the same set files an overnumbered printing of the same card at
	// that number without a star, so dropping it answers the wrong
	// printing at the wrong price rather than nothing.
	riftboundImageSig = regexp.MustCompile(`(?i)SIG$`)

	// The image pads its numbers to three digits where the catalog does
	// not ("VEN021" for 21, "UNLT01" for T1). PlainNumber reduces the
	// codes the catalog publishes and leaves a bare padded number as it
	// stands, so the padding comes off here.
	riftboundImagePad = regexp.MustCompile(`^([A-Za-z]*)0+([0-9])`)
)

// riftboundImageCard answers the card a listing's product image names.
//
// The storefront writes the champion's subtitle into the product name -
// "Akali - Deadly Weapon" - where the catalog files every one of a
// champion's cards under the champion alone. Vendetta's 021 and 038 are
// both just "Akali", so the listing's own words are the only thing
// telling them apart and the catalog's words are not, and the name
// reaches nothing. The image is the storefront's own sku and it carries
// the set code and the collector number, which says which printing the
// listing is when its wording cannot.
//
// It is read only where the wording was refused. Where both have an
// opinion the wording is the better one: a handful of images are a
// sibling printing's, reused, and the notes carry the exact number those
// listings are sold at.
func riftboundImageCard(b *mtgmatcher.Backend, imgURL string, foil bool) *mtgmatcher.InputCard {
	match := riftboundImageStem.FindStringSubmatch(imgURL)
	if match == nil {
		return nil
	}
	return riftboundSKUCard(b, match[1], foil)
}

// riftboundSKUCard answers the card one of the storefront's skus names.
// The sale listings carry it inside an image url and the buylist feed
// hands it over bare, in an Image field of its own.
func riftboundSKUCard(b *mtgmatcher.Backend, stem string, foil bool) *mtgmatcher.InputCard {
	if stem == "" {
		return nil
	}

	// A run of five digits or more is a TCGplayer product id rather than
	// a collector number, which runs to three. The catalog carries those
	// ids, so one that answers a card is the card; one that answers
	// nothing was never an id and the sku is read instead.
	for _, id := range riftboundImageTCG.FindAllString(stem, -1) {
		uuid := b.ConvertID(mtgmatcher.IDSpaceTCGplayer, id)
		if uuid == "" {
			continue
		}
		co, err := b.GetUUID(uuid)
		if err != nil {
			continue
		}
		card := riftboundCardAt(b, co.SetCode, co.Number, foil)
		if card != nil {
			return card
		}
	}

	// The set code sits somewhere in the name, behind a letter or two of
	// the storefront's own, and the number is what follows it. The codes
	// come from the datastore rather than a list here, so a set added
	// upstream is read without a change. Longest first: a short code is
	// a substring of longer ones.
	codes := slices.Clone(b.AllSets)
	slices.SortFunc(codes, func(a, b string) int {
		return len(b) - len(a)
	})

	upper := strings.ToUpper(stem)
	for _, code := range codes {
		index := strings.Index(upper, strings.ToUpper(code))
		if index < 0 {
			continue
		}
		number := upper[index+len(code):]
		var signature string
		if riftboundImageSig.MatchString(number) {
			number = riftboundImageSig.ReplaceAllString(number, "")
			signature = "*"
		}
		number = riftboundImageTail.ReplaceAllString(number, "")
		m := riftboundImagePrintIndex.FindStringSubmatch(number)
		if m != nil {
			number = m[1]
		}
		number = strings.Trim(number, "-_.")
		if number == "" {
			continue
		}
		number += signature
		card := riftboundCardAt(b, code, number, foil)
		if card != nil {
			return card
		}
	}
	return nil
}

// riftboundCardAt answers the one card a set files at a number, and
// nothing where the number names several cards or none. Several is not a
// miss to guess at: the number is being read to tell printings apart, so
// a number that does not is no answer.
func riftboundCardAt(b *mtgmatcher.Backend, code, number string, foil bool) *mtgmatcher.InputCard {
	set, err := b.GetSet(code)
	if err != nil {
		return nil
	}
	plain := riftboundImagePad.ReplaceAllString(b.PlainNumber(number), "${1}${2}")
	var name string
	for _, card := range set.Cards {
		if !strings.EqualFold(b.PlainNumber(card.Number), plain) {
			continue
		}
		if name != "" && name != card.Name {
			return nil
		}
		name = card.Name
	}
	if name == "" {
		return nil
	}
	return &mtgmatcher.InputCard{
		Name:      name,
		Edition:   set.Name,
		Variation: number,
		Foil:      foil,
	}
}
