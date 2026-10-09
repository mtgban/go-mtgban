package coolstuffinc

import (
	"regexp"
	"strings"

	"github.com/mtgban/go-mtgban/mtgmatcher"
)

// yugiohCodes spells the set codes the buylist writes unpadded the way the
// catalog does: the Dinosaurs Rage deck is SD9 here and SD09 there, and the
// third Hobby League is HL3 where the catalog says HL03. Both the Number and
// the note carry them.
var yugiohCodes = strings.NewReplacer(
	"SD9-SS1", "SD09-ENSS1",
	"SD9-EN", "SD09-EN",
	"HL3EN", "HL03-EN",
	"HL3-EN", "HL03-EN",
)

// yugiohWording corrects the words a listing's name or note types the catalog
// cannot read. The matcher takes a bare number in the wording for the
// collector number, so "Version 1 - " and "4 Corners" are rewritten away from
// it; "No Stamp" says the opposite of the word the stamp promo type is read
// from; and GB1-001 is the number GBI-001 mistyped.
var yugiohWording = strings.NewReplacer(
	"Version 1 - ", "",
	"Version 2 - ", "",
	"Flowers in 4 Corners", "Flowers in Four Corners",
	"GB1-001", "GBI-001",
	"(No Stamp ", "(",
	" - No Stamp", "",
)

// yugiohListing answers the card a Yu-Gi-Oh listing describes. The variation
// is the note on the sell listing and the number and note on the buylist, and
// notes is the note alone, which tells the print runs apart.
func yugiohListing(name, edition, notes, variation, rarity string, foil bool) *mtgmatcher.InputCard {
	variation = jpArtWording(yugiohWording.Replace(variation))
	return &mtgmatcher.InputCard{
		Name:      catalogColor(catalogSpelling(jpArtWording(yugiohWording.Replace(name)))),
		Edition:   printRunEdition(edition, notes),
		Variation: strings.TrimSpace(variation + " " + catalogRarity(rarity)),
		Foil:      foil,
	}
}

// yugiohArtNotes names the artwork a note describes as the catalog's
// qualifier for it. The Limited Pack World Championship 2026 prints an
// alternate art of five Secret Rares, and the note describes its border and
// lettering where the catalog names it Emblazoned.
var yugiohArtNotes = map[string]string{
	"Red Inner Border, Japanese Letters in Art": "Emblazoned Alternate Art",
}

// yugiohArtCard answers a listing whose note is one of yugiohArtNotes, or nil.
// The qualifier names exactly one printing of the number, so the bracket the
// name carries and the rarity beside it are left behind: either one makes the
// matcher read the Emblazoned Secret Rare of the same number instead.
func yugiohArtCard(name, edition, number, notes string, foil bool) *mtgmatcher.InputCard {
	qualifier, found := yugiohArtNotes[strings.TrimSpace(notes)]
	if !found {
		return nil
	}
	head, _, _ := strings.Cut(name, " (")
	return &mtgmatcher.InputCard{
		Name:      catalogColor(catalogSpelling(head)),
		Edition:   printRunEdition(edition, notes),
		Variation: strings.TrimSpace(number + " " + qualifier),
		Foil:      foil,
	}
}

// yugiohImageCode matches the set code and number a Yu-Gi-Oh image sku
// carries: "BLMMEN053" is BLMM-EN053.
var yugiohImageCode = regexp.MustCompile(`^([A-Za-z0-9]{2,5}?)EN(\d{3})`)

// yugiohImageCard retries a refused buylist row with the code its own image
// sku carries in place of its Number, where the two disagree, and answers the
// id it lands on. The landing has to be the same card, so a sku naming
// another card's code reaches nothing.
func yugiohImageCard(b *mtgmatcher.Backend, card *mtgmatcher.InputCard, product CSIPriceEntry) string {
	match := yugiohImageCode.FindStringSubmatch(product.Image)
	if match == nil || product.Number == "" {
		return ""
	}
	code := match[1] + "-EN" + match[2]
	if strings.EqualFold(code, product.Number) {
		return ""
	}
	retried := *card
	retried.Variation = strings.Replace(card.Variation, product.Number, code, 1)
	id, err := b.Match(&retried)
	if err != nil {
		return ""
	}
	co, err := b.GetUUID(id)
	head, _, _ := strings.Cut(card.Name, " (")
	if err != nil || !mtgmatcher.Equals(co.Name, head) {
		return ""
	}
	return id
}
